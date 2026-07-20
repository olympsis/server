// Package bus publishes domain events onto the shared olympsis RabbitMQ topic
// exchange, where invite-service and notif-service pick them up.
//
// Topology (owned jointly by all olympsis services): one durable topic exchange,
// `olympsis.events`, with `<entity>.<action>` routing keys. Each consuming
// service binds its own durable queue. The server is a pure PUBLISHER — it never
// consumes — so this package has no queue or consume loop.
//
// Three properties matter for how this is used:
//
//   - Publishing is BEST EFFORT and ASYNCHRONOUS. By the time we publish, the
//     RSVP/comment/event is already committed to Mongo, so a broker problem must
//     never fail — or even slow down — the request. Emit hands the message to a
//     bounded queue drained by a background worker and returns immediately.
//     Handlers never touch the socket. See the note on publishTimeout for why
//     this queue is load-bearing rather than a nicety.
//   - The server must boot without the broker. If RabbitMQ is unreachable (or
//     unconfigured), the publisher runs disabled and keeps retrying in the
//     background; every other endpoint works normally in the meantime.
//   - The queue is bounded and lossy. If the broker stays wedged, messages are
//     dropped with a log rather than accumulating without limit. Notifications
//     are not worth an OOM on a shared 16 GB box.
package bus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

// Routing keys the server publishes. Keep these in sync with the bindings in
// invite-service (event.*/team.*) and notif-service (rsvp.*/comment.*).
const (
	RoutingKeyEventCreated   = "event.created"   // -> invite-service
	RoutingKeyTeamCreated    = "team.created"    // -> invite-service
	RoutingKeyRSVPCreated    = "rsvp.created"    // -> notif-service
	RoutingKeyCommentCreated = "comment.created" // -> notif-service
)

// publishTimeout is applied to each publish, but understand what it does NOT do:
// amqp091-go only checks the context BEFORE it starts writing, and it sets no
// write deadline on the socket. If the broker raises a memory/disk alarm and
// stops reading, the underlying WriteFrame blocks indefinitely and no context
// can interrupt it.
//
// That is precisely why Emit is asynchronous. A wedged broker can stall the
// single worker goroutine; it cannot stall an HTTP handler, because handlers
// only ever hand off to the queue.
const publishTimeout = 5 * time.Second

// queueSize bounds the in-memory backlog. Sized for a burst while the broker
// reconnects, not for a long outage — see the lossy note in the package doc.
const queueSize = 256

// shutdownDrain caps how long Close waits for the worker to finish. A worker
// stuck in a blocked WriteFrame must not hold shutdown open forever.
const shutdownDrain = 5 * time.Second

// ErrDisabled is returned by Publish when no broker URL is configured.
var ErrDisabled = errors.New("bus: publisher disabled (no RABBITMQ_URL)")

// message is one queued publish.
type message struct {
	routingKey string
	body       []byte
}

// Publisher owns the broker connection and the single publish channel.
//
// amqp Channels are NOT safe for concurrent use. Only the worker goroutine
// publishes, but the reconnect goroutine swaps the channel out underneath it, so
// conn/ch and the close listeners are all guarded by mu.
type Publisher struct {
	logger   *logrus.Logger
	url      string
	exchange string

	queue chan message

	mu   sync.Mutex
	conn *amqp.Connection
	ch   *amqp.Channel
	// Signalled by amqp when the connection or the channel dies; maintain waits
	// on both. Kept separate because amqp closes each registered listener
	// itself — sharing one channel across the two NotifyClose calls would close
	// it twice and panic the process on shutdown.
	connClosed chan *amqp.Error
	chanClosed chan *amqp.Error
	closing    bool // set by Close; stops dial from installing a new conn

	stop chan struct{}
	once sync.Once
	wg   sync.WaitGroup
}

// New builds a publisher. A blank url disables it — every Emit becomes a no-op,
// which is how a dev box without RabbitMQ keeps working. Call Connect next.
func New(logger *logrus.Logger, url, exchange string) *Publisher {
	if exchange == "" {
		exchange = "olympsis.events"
	}
	return &Publisher{
		logger:   logger,
		url:      url,
		exchange: exchange,
		queue:    make(chan message, queueSize),
		stop:     make(chan struct{}),
	}
}

// Enabled reports whether this publisher will do anything. Nil-safe so a Service
// constructed without a Bus degrades to a no-op instead of panicking a handler.
func (p *Publisher) Enabled() bool { return p != nil && p.url != "" }

// Connect dials the broker, then starts the publish worker and the reconnect
// watcher.
//
// It deliberately does NOT return a fatal error when the broker is down: it logs
// and lets the watcher keep retrying, so the server still starts. The returned
// error is informational — main logs it and carries on.
func (p *Publisher) Connect(ctx context.Context) error {
	if !p.Enabled() {
		if p != nil {
			p.logger.Warn("[Bus] RABBITMQ_URL not set — event publishing is disabled")
		}
		return nil
	}

	err := p.dial()
	if err != nil {
		p.logger.Errorf("[Bus] initial connect failed, will keep retrying: %s", err.Error())
	} else {
		p.logger.Infof("[Bus] connected, publishing to exchange %q", p.exchange)
	}

	p.wg.Add(2)
	go p.worker()
	go p.maintain(ctx)
	return err
}

// dial opens a fresh connection + channel and declares the shared exchange.
// Declaring is idempotent, so it is safe on every reconnect.
func (p *Publisher) dial() error {
	conn, err := amqp.Dial(p.url)
	if err != nil {
		return fmt.Errorf("dial rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("open channel: %w", err)
	}

	// durable=true so the exchange survives a broker restart. Every service
	// declares it with these same arguments; mismatched args would error here.
	if err := ch.ExchangeDeclare(p.exchange, "topic", true, false, false, false, nil); err != nil {
		conn.Close()
		return fmt.Errorf("declare exchange %q: %w", p.exchange, err)
	}

	// Watch BOTH the connection and the channel. A channel exception (the
	// exchange being deleted and re-declared with different args, an
	// ACCESS_REFUSED, any broker-initiated channel close) kills the channel while
	// leaving the connection healthy. Watching only the connection would leave a
	// dead channel in place and silently drop every message until a restart.
	//
	// These MUST be two separate Go channels. amqp closes every registered
	// listener when its owner shuts down, so handing the same channel to both
	// NotifyClose calls gets it closed twice — a "close of closed channel" panic
	// that takes down the process on every clean shutdown.
	connClosed := make(chan *amqp.Error, 1)
	chanClosed := make(chan *amqp.Error, 1)
	conn.NotifyClose(connClosed)
	ch.NotifyClose(chanClosed)

	p.mu.Lock()
	defer p.mu.Unlock()

	// Close may have run while we were dialing. Don't install a connection
	// nothing will ever tear down.
	if p.closing {
		conn.Close()
		return errors.New("bus: publisher closing")
	}

	p.conn = conn
	p.ch = ch
	p.connClosed = connClosed
	p.chanClosed = chanClosed
	return nil
}

// worker is the only goroutine that publishes. Serializing here is what lets
// handlers hand off without blocking on the broker.
func (p *Publisher) worker() {
	defer p.wg.Done()
	for {
		select {
		case <-p.stop:
			return
		case msg := <-p.queue:
			if err := p.Publish(context.Background(), msg.routingKey, msg.body); err != nil {
				p.logger.Errorf("[Bus] failed to publish %s: %s", msg.routingKey, err.Error())
			}
		}
	}
}

// maintain reconnects whenever the connection or channel drops. Unlike a
// consumer, a pure publisher has no delivery loop to notice a dead connection,
// so we watch NotifyClose explicitly.
func (p *Publisher) maintain(ctx context.Context) {
	defer p.wg.Done()
	for {
		p.mu.Lock()
		connClosed, chanClosed := p.connClosed, p.chanClosed
		p.mu.Unlock()

		if connClosed != nil || chanClosed != nil {
			// Whichever fires first wins; a dropped connection takes its channel
			// with it, so both may fire and the loser is simply discarded when
			// dial() installs fresh listeners.
			select {
			case <-ctx.Done():
				return
			case <-p.stop:
				return
			case amqpErr := <-connClosed:
				if amqpErr != nil {
					p.logger.Warnf("[Bus] connection closed (%s); reconnecting", amqpErr.Error())
				}
			case amqpErr := <-chanClosed:
				if amqpErr != nil {
					p.logger.Warnf("[Bus] channel closed (%s); reconnecting", amqpErr.Error())
				}
			}
		}

		// Retry with exponential backoff, capped, until we're connected again.
		backoff := time.Second
		for {
			select {
			case <-ctx.Done():
				return
			case <-p.stop:
				return
			case <-time.After(backoff):
			}

			if err := p.dial(); err != nil {
				p.logger.Errorf("[Bus] reconnect failed: %s", err.Error())
				if backoff < 30*time.Second {
					backoff *= 2
				}
				continue
			}
			p.logger.Info("[Bus] reconnected")
			break
		}
	}
}

// Publish sends body to the exchange under routingKey, returning any error. It
// blocks on the broker, so it is called only from the worker goroutine — use
// Emit instead.
func (p *Publisher) Publish(ctx context.Context, routingKey string, body []byte) error {
	if !p.Enabled() {
		return ErrDisabled
	}

	ctx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()

	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ch == nil {
		return errors.New("bus: publisher channel not open")
	}

	return p.ch.PublishWithContext(ctx, p.exchange, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent, // survive a broker restart
		Timestamp:    time.Now(),
	})
}

// Emit marshals payload to JSON and queues it for the worker. It never blocks
// and never returns an error: the write it describes is already committed, so a
// broker problem must not fail the request.
//
// The message body is the raw JSON of the domain struct with no envelope — the
// convention every olympsis service follows.
//
// ctx is accepted for symmetry with the handler call sites, but is intentionally
// NOT used to cancel the publish: handlers pass r.Context(), which dies the
// moment the client disconnects, and a user who closes the app right after
// commenting would otherwise lose the notification for a comment that IS saved.
func (p *Publisher) Emit(_ context.Context, routingKey string, payload any) {
	if !p.Enabled() {
		return
	}

	body, err := json.Marshal(payload)
	if err != nil {
		p.logger.Errorf("[Bus] failed to encode %s message: %s", routingKey, err.Error())
		return
	}

	select {
	case p.queue <- message{routingKey: routingKey, body: body}:
	default:
		// Backlog full — the broker has been unreachable for a while. Drop with a
		// log rather than growing the queue without bound.
		p.logger.Errorf("[Bus] queue full, dropping %s message", routingKey)
	}
}

// Close stops the worker and reconnect goroutines and tears down the broker
// connection. Safe to call more than once.
func (p *Publisher) Close() error {
	if p == nil {
		return nil
	}
	p.once.Do(func() { close(p.stop) })

	// Tear the connection down BEFORE waiting on the goroutines: if the worker is
	// stuck in a blocked write, closing the socket is what unblocks it.
	p.mu.Lock()
	p.closing = true
	ch, conn := p.ch, p.conn
	p.ch, p.conn = nil, nil
	p.mu.Unlock()

	if ch != nil {
		_ = ch.Close()
	}
	var err error
	if conn != nil {
		err = conn.Close()
	}

	// Bounded wait — a worker wedged on an unresponsive socket must not hold
	// shutdown open indefinitely.
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(shutdownDrain):
		p.logger.Warn("[Bus] timed out waiting for publisher goroutines to exit")
	}

	return err
}
