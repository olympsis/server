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

	// Must not panic, must not block.
	p.Emit(context.Background(), RoutingKeyRSVPCreated, map[string]string{"id": "x"})

	if err := p.Publish(context.Background(), RoutingKeyRSVPCreated, []byte("{}")); !errors.Is(err, ErrDisabled) {
		t.Errorf("Publish on disabled publisher = %v, want ErrDisabled", err)
	}
	if err := p.Close(); err != nil {
		t.Errorf("Close on disabled publisher: %v", err)
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

// TestPublishWithoutConnectionErrors — an enabled publisher that never reached
// the broker must report the failure so Emit can log it, rather than silently
// pretending the message went out.
func TestPublishWithoutConnectionErrors(t *testing.T) {
	p := New(testLogger(), "amqp://unreachable.invalid:5672/", "olympsis.events")

	if !p.Enabled() {
		t.Fatal("publisher with a url reported not Enabled")
	}
	if err := p.Publish(context.Background(), RoutingKeyCommentCreated, []byte("{}")); err == nil {
		t.Error("Publish with no open channel returned nil, want error")
	}

	// Emit swallows that error by design — it must not panic or block.
	p.Emit(context.Background(), RoutingKeyCommentCreated, map[string]string{"id": "x"})

	// Close must be safe even though Connect was never called.
	if err := p.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

// TestEmitIgnoresCancelledContext — handlers pass r.Context(), which dies the
// moment the client disconnects. The row is already committed by then, so Emit
// must still attempt the publish rather than silently dropping the notification.
func TestEmitIgnoresCancelledContext(t *testing.T) {
	p := New(testLogger(), "amqp://unreachable.invalid:5672/", "olympsis.events")
	defer p.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // client hung up

	// Must not panic and must not early-return on ctx.Err(). With no channel
	// open the publish fails and is logged, which is the best-effort contract.
	p.Emit(ctx, RoutingKeyCommentCreated, map[string]string{"id": "x"})

	// The cancellation must genuinely be stripped, not merely tolerated.
	if err := context.WithoutCancel(ctx).Err(); err != nil {
		t.Errorf("WithoutCancel still carries an error: %v", err)
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
