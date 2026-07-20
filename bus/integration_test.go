package bus

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/olympsis/models"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Integration coverage for the publisher against a REAL broker.
//
// These skip automatically when RabbitMQ is unreachable, so CI and a laptop
// without the dev stack stay green — the same convention invite-service and
// notif-service use. Point them somewhere else with TEST_RABBITMQ_URL.
//
// They publish to a dedicated `olympsis.events.test` exchange rather than the
// shared one, so a test run can never deliver a bogus notification to a real
// consumer queue.
const testExchange = "olympsis.events.test"

func testBrokerURL() string {
	if u := os.Getenv("TEST_RABBITMQ_URL"); u != "" {
		return u
	}
	return "amqp://olympsis:olympsis@localhost:5672/"
}

// dialOrSkip gives us a control connection for asserting what actually landed
// on the broker, and skips the test when there is no broker to talk to.
func dialOrSkip(t *testing.T) *amqp.Connection {
	t.Helper()
	conn, err := amqp.DialConfig(testBrokerURL(), amqp.Config{
		Dial: amqp.DefaultDial(2 * time.Second),
	})
	if err != nil {
		t.Skipf("skipping: no RabbitMQ at %s (%v)", testBrokerURL(), err)
	}
	return conn
}

// bindTempQueue creates an exclusive, auto-deleted queue bound to the test
// exchange for one routing key, and returns its name.
func bindTempQueue(t *testing.T, conn *amqp.Connection, routingKey string) (*amqp.Channel, string) {
	t.Helper()
	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("open control channel: %v", err)
	}
	if err := ch.ExchangeDeclare(testExchange, "topic", true, false, false, false, nil); err != nil {
		t.Fatalf("declare test exchange: %v", err)
	}
	// exclusive + autoDelete: disappears with the connection, so runs don't leak queues.
	q, err := ch.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		t.Fatalf("declare temp queue: %v", err)
	}
	if err := ch.QueueBind(q.Name, routingKey, testExchange, false, nil); err != nil {
		t.Fatalf("bind temp queue: %v", err)
	}
	return ch, q.Name
}

// TestPublisherDeliversToBroker is the end-to-end proof: Emit -> worker ->
// exchange -> bound queue, with the body arriving as the raw domain struct.
func TestPublisherDeliversToBroker(t *testing.T) {
	conn := dialOrSkip(t)
	defer conn.Close()

	ctrl, queue := bindTempQueue(t, conn, RoutingKeyRSVPCreated)
	defer ctrl.Close()

	p := New(testLogger(), testBrokerURL(), testExchange)
	if err := p.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer p.Close()

	sent := models.RSVPCreatedMessage{
		ID:        "participant-1",
		UserID:    "auth0|joiner",
		EventID:   "event-1",
		Status:    models.RSVPYes,
		CreatedAt: time.Unix(0, 0).UTC(),
	}
	p.Emit(context.Background(), RoutingKeyRSVPCreated, sent)

	deliveries, err := ctrl.Consume(queue, "", true, false, false, false, nil)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}

	select {
	case d := <-deliveries:
		if d.ContentType != "application/json" {
			t.Errorf("content type = %q, want application/json", d.ContentType)
		}
		if d.DeliveryMode != amqp.Persistent {
			t.Errorf("delivery mode = %d, want persistent", d.DeliveryMode)
		}

		// The body must be the bare domain struct — no envelope — and status
		// must be the STRING form, which is what notif-service parses.
		var raw map[string]any
		if err := json.Unmarshal(d.Body, &raw); err != nil {
			t.Fatalf("body is not JSON: %v (%s)", err, d.Body)
		}
		if raw["status"] != "YES" {
			t.Errorf("status on the wire = %v, want \"YES\"", raw["status"])
		}
		if raw["user_id"] != sent.UserID {
			t.Errorf("user_id = %v, want %q", raw["user_id"], sent.UserID)
		}

		var got models.RSVPCreatedMessage
		if err := json.Unmarshal(d.Body, &got); err != nil {
			t.Fatalf("decode into message: %v", err)
		}
		if got.Status != models.RSVPYes || got.ID != sent.ID {
			t.Errorf("round-trip = %+v, want %+v", got, sent)
		}

	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the published message")
	}
}

// TestPublisherRecoversFromChannelClose covers the failure this package got
// wrong once already: a channel dies while the connection stays healthy. The
// publisher must notice and reconnect rather than silently dropping everything.
func TestPublisherRecoversFromChannelClose(t *testing.T) {
	conn := dialOrSkip(t)
	defer conn.Close()

	ctrl, queue := bindTempQueue(t, conn, RoutingKeyCommentCreated)
	defer ctrl.Close()

	p := New(testLogger(), testBrokerURL(), testExchange)
	if err := p.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer p.Close()

	// Kill the publisher's channel out from under it, leaving its connection up.
	p.mu.Lock()
	ch := p.ch
	p.mu.Unlock()
	if ch == nil {
		t.Fatal("publisher has no channel after Connect")
	}
	_ = ch.Close()

	// The reconnect watcher backs off ~1s before its first retry.
	deadline := time.Now().Add(20 * time.Second)
	for {
		p.mu.Lock()
		live := p.ch != nil && p.ch != ch
		p.mu.Unlock()
		if live {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("publisher never re-established its channel after a channel close")
		}
		time.Sleep(200 * time.Millisecond)
	}

	// And it must actually deliver again.
	p.Emit(context.Background(), RoutingKeyCommentCreated, models.CommentCreatedMessage{
		ID: "comment-1", UserID: "auth0|author", EventID: "event-1", Text: "see you there",
	})

	deliveries, err := ctrl.Consume(queue, "", true, false, false, false, nil)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	select {
	case d := <-deliveries:
		var got models.CommentCreatedMessage
		if err := json.Unmarshal(d.Body, &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.ID != "comment-1" {
			t.Errorf("id = %q, want comment-1", got.ID)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("no delivery after recovery — the channel-close path is broken")
	}
}
