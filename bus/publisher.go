// Package bus publishes domain events onto the shared olympsis RabbitMQ topic
// exchange, where invite-service and notif-service pick them up.
//
// Topology (owned jointly by all olympsis services): one durable topic exchange,
// `olympsis.events`, with `<entity>.<action>` routing keys. Each consuming
// service binds its own durable queue. The server is a pure PUBLISHER — it never
// consumes — so this package has no queue or consume loop.
//
// Two properties matter for how this is used:
//
//   - Publishing is BEST EFFORT. By the time we publish, the RSVP/comment/event
//     is already committed to Mongo. A broker hiccup must never turn a
//     successful write into a failed request, so use Emit, which logs and
//     swallows. Reserve Publish for the rare caller that genuinely wants the
//     error.
//   - The server must boot without the broker. If RabbitMQ is unreachable (or
//     unconfigured), the publisher runs disabled and keeps retrying in the
//     background; every other endpoint works normally in the meantime.
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

// publishTimeout bounds a single publish so a wedged broker connection can't
// hold an HTTP handler open.
const publishTimeout = 5 * time.Second

// ErrDisabled is returned by Publish when no broker URL is configured.
var ErrDisabled = errors.New("bus: publisher disabled (no RABBITMQ_URL)")

// Publisher owns the broker connection and the single publish channel.
//
// amqp Channels are NOT safe for concurrent use and HTTP handlers publish from
// many goroutines, so every publish is serialized by mu. mu also guards conn/ch
// against the background reconnect swapping them out mid-publish.
type Publisher struct {
	logger   *logrus.Logger
	url      string
	exchange string

	mu   sync.Mutex
	conn *amqp.Connection
	ch   *amqp.Channel

	// closed is signalled by amqp when the connection drops; the maintain
	// goroutine waits on it to trigger a reconnect.
	closed chan *amqp.Error

	stop chan struct{}
	once sync.Once
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
		stop:     make(chan struct{}),
	}
}

// Enabled reports whether a broker URL was configured.
func (p *Publisher) Enabled() bool { return p.url != "" }

// Connect dials the broker and starts the background reconnect watcher.
//
// It deliberately does NOT return a fatal error when the broker is down: it logs
// and lets the watcher keep retrying, so the server still starts. The returned
// error is informational — main logs it and carries on.
func (p *Publisher) Connect(ctx context.Context) error {
	if !p.Enabled() {
		p.logger.Warn("[Bus] RABBITMQ_URL not set — event publishing is disabled")
		return nil
	}

	err := p.dial()
	if err != nil {
		p.logger.Errorf("[Bus] initial connect failed, will keep retrying: %s", err.Error())
	} else {
		p.logger.Infof("[Bus] connected, publishing to exchange %q", p.exchange)
	}

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

	closed := make(chan *amqp.Error, 1)
	conn.NotifyClose(closed)

	p.mu.Lock()
	p.conn = conn
	p.ch = ch
	p.closed = closed
	p.mu.Unlock()
	return nil
}

// maintain reconnects whenever the connection drops. Unlike a consumer, a pure
// publisher has no delivery loop to notice a dead connection, so we watch
// NotifyClose explicitly — otherwise every publish after the first drop would
// fail silently forever.
func (p *Publisher) maintain(ctx context.Context) {
	for {
		p.mu.Lock()
		closed := p.closed
		p.mu.Unlock()

		if closed != nil {
			select {
			case <-ctx.Done():
				return
			case <-p.stop:
				return
			case amqpErr := <-closed:
				if amqpErr != nil {
					p.logger.Warnf("[Bus] connection closed (%s); reconnecting", amqpErr.Error())
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

// Publish sends body to the exchange under routingKey, returning any error.
// Most callers want Emit instead.
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

// Emit marshals payload to JSON and publishes it, logging any failure instead of
// returning it. This is the method handlers should call: the write it describes
// has already been committed, so a broker problem must not fail the request.
//
// The message body is the raw JSON of the domain struct with no envelope —
// the convention every olympsis service follows.
func (p *Publisher) Emit(ctx context.Context, routingKey string, payload any) {
	if !p.Enabled() {
		return
	}

	body, err := json.Marshal(payload)
	if err != nil {
		p.logger.Errorf("[Bus] failed to encode %s message: %s", routingKey, err.Error())
		return
	}

	if err := p.Publish(ctx, routingKey, body); err != nil {
		p.logger.Errorf("[Bus] failed to publish %s: %s", routingKey, err.Error())
	}
}

// Close stops the reconnect watcher and tears down the channel and connection.
// Call once on shutdown.
func (p *Publisher) Close() error {
	p.once.Do(func() { close(p.stop) })

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ch != nil {
		_ = p.ch.Close()
		p.ch = nil
	}
	if p.conn != nil {
		err := p.conn.Close()
		p.conn = nil
		return err
	}
	return nil
}
