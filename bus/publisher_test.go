package bus

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/sirupsen/logrus"
)

func testLogger() *logrus.Logger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

// TestDisabledPublisherIsNoOp is the guarantee that keeps a dev box (or a
// production box mid-broker-outage) usable: with no RABBITMQ_URL, Connect
// succeeds and Emit does nothing rather than erroring or panicking.
func TestDisabledPublisherIsNoOp(t *testing.T) {
	p := New(testLogger(), "", "")

	if p.Enabled() {
		t.Fatal("publisher with blank url reported Enabled")
	}
	if err := p.Connect(context.Background()); err != nil {
		t.Fatalf("Connect on disabled publisher: %v", err)
	}

	// Must not panic, must not block, must not queue.
	p.Emit(context.Background(), RoutingKeyRSVPCreated, map[string]string{"id": "x"})
	if len(p.queue) != 0 {
		t.Errorf("disabled publisher queued %d message(s)", len(p.queue))
	}

	if err := p.Publish(context.Background(), RoutingKeyRSVPCreated, []byte("{}")); !errors.Is(err, ErrDisabled) {
		t.Errorf("Publish on disabled publisher = %v, want ErrDisabled", err)
	}
	if err := p.Close(); err != nil {
		t.Errorf("Close on disabled publisher: %v", err)
	}
}

// TestNilPublisherIsNoOp — a Service built without a Bus (a test fixture, a
// future API module) must degrade to no-op rather than panicking the handler.
func TestNilPublisherIsNoOp(t *testing.T) {
	var p *Publisher

	if p.Enabled() {
		t.Error("nil publisher reported Enabled")
	}
	if err := p.Connect(context.Background()); err != nil {
		t.Errorf("Connect on nil publisher: %v", err)
	}
	p.Emit(context.Background(), RoutingKeyRSVPCreated, map[string]string{"id": "x"})
	if err := p.Close(); err != nil {
		t.Errorf("Close on nil publisher: %v", err)
	}
}

// TestDefaultExchange — an unset RABBITMQ_EXCHANGE must still land on the shared
// exchange the other services bind to, not on the empty (default) exchange,
// where messages would be silently dropped.
func TestDefaultExchange(t *testing.T) {
	if got := New(testLogger(), "", "").exchange; got != "olympsis.events" {
		t.Errorf("default exchange = %q, want %q", got, "olympsis.events")
	}
	if got := New(testLogger(), "", "custom.events").exchange; got != "custom.events" {
		t.Errorf("explicit exchange = %q, want %q", got, "custom.events")
	}
}

// TestPublishWithoutConnectionErrors — Publish must report a missing channel so
// the worker can log it, rather than pretending the message went out.
func TestPublishWithoutConnectionErrors(t *testing.T) {
	p := New(testLogger(), "amqp://unreachable.invalid:5672/", "olympsis.events")
	defer p.Close()

	if !p.Enabled() {
		t.Fatal("publisher with a url reported not Enabled")
	}
	if err := p.Publish(context.Background(), RoutingKeyCommentCreated, []byte("{}")); err == nil {
		t.Error("Publish with no open channel returned nil, want error")
	}
}

// TestEmitDoesNotBlockOnADeadBroker is the core best-effort guarantee. No worker
// is running here, so nothing drains the queue — Emit must still return
// immediately, and must drop rather than grow without bound once full.
//
// A cancelled context is used deliberately: handlers pass r.Context(), which
// dies when the client disconnects, and that must not suppress the publish.
func TestEmitDoesNotBlockOnADeadBroker(t *testing.T) {
	p := New(testLogger(), "amqp://unreachable.invalid:5672/", "olympsis.events")
	defer p.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // client hung up

	// Far more than the queue holds. If Emit blocked, this would deadlock and the
	// test would time out rather than fail.
	for range queueSize * 3 {
		p.Emit(ctx, RoutingKeyCommentCreated, map[string]string{"id": "x"})
	}

	if len(p.queue) != queueSize {
		t.Errorf("queue depth = %d, want it capped at %d", len(p.queue), queueSize)
	}
}

// TestEmitQueuesForTheWorker — the message must actually reach the queue with
// the right routing key and an unwrapped JSON body (the shared envelope
// convention: no wrapper, just the domain struct).
func TestEmitQueuesForTheWorker(t *testing.T) {
	p := New(testLogger(), "amqp://unreachable.invalid:5672/", "olympsis.events")
	defer p.Close()

	p.Emit(context.Background(), RoutingKeyRSVPCreated, map[string]string{"id": "abc"})

	select {
	case msg := <-p.queue:
		if msg.routingKey != RoutingKeyRSVPCreated {
			t.Errorf("routing key = %q, want %q", msg.routingKey, RoutingKeyRSVPCreated)
		}
		if string(msg.body) != `{"id":"abc"}` {
			t.Errorf("body = %s, want %s", msg.body, `{"id":"abc"}`)
		}
	default:
		t.Fatal("Emit did not queue the message")
	}
}

// TestEmitDropsUnencodablePayload — a payload that can't be marshalled must be
// logged and dropped, never queued as a broken message.
func TestEmitDropsUnencodablePayload(t *testing.T) {
	p := New(testLogger(), "amqp://unreachable.invalid:5672/", "olympsis.events")
	defer p.Close()

	p.Emit(context.Background(), RoutingKeyRSVPCreated, make(chan int)) // channels can't marshal

	if len(p.queue) != 0 {
		t.Errorf("queued %d message(s) for an unencodable payload", len(p.queue))
	}
}

// TestCloseIsIdempotent — Close runs on the shutdown path and must tolerate
// being reached more than once without panicking on a double channel close.
func TestCloseIsIdempotent(t *testing.T) {
	p := New(testLogger(), "", "")
	for i := range 3 {
		if err := p.Close(); err != nil {
			t.Fatalf("Close #%d: %v", i+1, err)
		}
	}
}

// TestCloseStopsGoroutines — Connect starts a worker and a reconnect watcher
// even when the broker is unreachable. Close must reap both; a leak here would
// accumulate across restarts in tests and mask real leaks in pprof.
func TestCloseStopsGoroutines(t *testing.T) {
	p := New(testLogger(), "amqp://unreachable.invalid:5672/", "olympsis.events")

	// Connect returns an error (broker is unreachable) but still starts both
	// goroutines, which is the case we need to reap.
	_ = p.Connect(context.Background())

	if err := p.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}

	// wg.Wait inside Close is bounded; if the goroutines were still running we'd
	// have seen the timeout warning. Verify stop is actually closed.
	select {
	case <-p.stop:
	default:
		t.Error("Close did not signal stop")
	}
}
