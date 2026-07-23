package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	natsgo "github.com/nats-io/nats.go"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/pkg/logger"
)

// requiredMaxPayload is the minimum NATS server max_payload we accept. Task
// stream events routinely exceed the NATS default of 1MiB; the durable RabbitMQ
// path allows 16MiB, so the NATS server must match.
const requiredMaxPayload = 16 * 1024 * 1024

// PubSub implements msgqueue.PubSub over core NATS (no JetStream). Subjects are
// "hatchet.pubsub." + topic.Name(); delivery is best-effort at-most-once.
//
// INVARIANT: the PubSub owns its nats.Conn and never shares it with the durable
// MessageQueue, since Pub can be called from within durable-write paths.
type PubSub struct {
	nc *natsgo.Conn
	l  *zerolog.Logger
}

type PubSubOpt func(*PubSubOpts)

type PubSubOpts struct {
	l        *zerolog.Logger
	url      string
	username string
	password string
}

func defaultPubSubOpts() *PubSubOpts {
	l := logger.NewDefaultLogger("nats-pubsub")

	return &PubSubOpts{
		l: &l,
	}
}

// WithPubSubURL sets the NATS seed URL(s). Comma-separated lists are passed
// through to nats.go. Prefer bare hosts and set Username/Password so
// rediscovered cluster peers authenticate; URL-embedded user:pass still works
// for single-server/dev. Use the tls:// scheme for TLS.
func WithPubSubURL(url string) PubSubOpt {
	return func(opts *PubSubOpts) {
		opts.url = url
	}
}

// WithPubSubUsername sets the NATS username for nats.UserInfo. Use with
// WithPubSubPassword so auth applies on reconnect to gossiped cluster peers.
func WithPubSubUsername(username string) PubSubOpt {
	return func(opts *PubSubOpts) {
		opts.username = username
	}
}

// WithPubSubPassword sets the NATS password for nats.UserInfo.
func WithPubSubPassword(password string) PubSubOpt {
	return func(opts *PubSubOpts) {
		opts.password = password
	}
}

func WithPubSubLogger(l *zerolog.Logger) PubSubOpt {
	return func(opts *PubSubOpts) {
		opts.l = l
	}
}

// NewPubSub connects synchronously to NATS and returns a PubSub. Fails if the
// server is unreachable or if its max_payload is below 16MiB.
func NewPubSub(fs ...PubSubOpt) (func() error, *PubSub, error) {
	opts := defaultPubSubOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.url == "" {
		return nil, nil, fmt.Errorf("nats pubsub requires a URL to be set")
	}

	l := opts.l

	connectOpts := []natsgo.Option{
		natsgo.MaxReconnects(-1),
		natsgo.ReconnectBufSize(-1), // publishes fail during disconnect — no stale buffering
		natsgo.DisconnectErrHandler(func(_ *natsgo.Conn, err error) {
			if err != nil {
				l.Warn().Err(err).Msg("nats pubsub disconnected")
			} else {
				l.Warn().Msg("nats pubsub disconnected")
			}
		}),
		natsgo.ReconnectHandler(func(nc *natsgo.Conn) {
			l.Info().Str("url", nc.ConnectedUrl()).Msg("nats pubsub reconnected")
		}),
		natsgo.ClosedHandler(func(_ *natsgo.Conn) {
			l.Info().Msg("nats pubsub connection closed")
		}),
		natsgo.ErrorHandler(func(_ *natsgo.Conn, sub *natsgo.Subscription, err error) {
			subject := ""
			if sub != nil {
				subject = sub.Subject
			}
			l.Error().Err(err).Str("subject", subject).Msg("nats pubsub async error")
		}),
	}

	if opts.username != "" || opts.password != "" {
		connectOpts = append(connectOpts, natsgo.UserInfo(opts.username, opts.password))
	}

	nc, err := natsgo.Connect(opts.url, connectOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("could not connect to nats at %q: %w", opts.url, err)
	}

	if nc.MaxPayload() < requiredMaxPayload {
		nc.Close()
		return nil, nil, fmt.Errorf(
			"nats server max_payload is %d bytes; set max_payload: 16777216 in the NATS server config (default 1MiB is insufficient for task stream events)",
			nc.MaxPayload(),
		)
	}

	p := &PubSub{
		nc: nc,
		l:  l,
	}

	return func() error {
		nc.Close()
		return nil
	}, p, nil
}

func (p *PubSub) IsReady() bool {
	return p.nc.IsConnected()
}

// Pub publishes a message to the topic. Delivery is best-effort: if no
// subscriber is listening, the message is dropped. No flush per message.
// Oversized multi-payload messages are chunked like rabbitmq/pubsub.go.
func (p *PubSub) Pub(ctx context.Context, topic msgqueue.Topic, msg *msgqueue.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	subject := "hatchet.pubsub." + topic.Name()

	body, err := json.Marshal(msg)
	if err != nil {
		p.l.Error().Ctx(ctx).Err(err).Msg("error marshaling pubsub message")
		return err
	}

	// Use the server-advertised limit (constructor already requires >= 16MiB).
	maxPayload := p.nc.MaxPayload()

	if int64(len(body)) > maxPayload {
		if len(msg.Payloads) == 1 {
			return fmt.Errorf("message size %d bytes exceeds maximum allowed size of %d bytes", len(body), maxPayload)
		}

		// split the payloads in half and publish recursively until each chunk is
		// under the max size (same strategy as rabbitmq/pubsub.go)
		payloadsPerChunk := max(len(msg.Payloads)/2, 1)

		for chunk := range slices.Chunk(msg.Payloads, payloadsPerChunk) {
			err := p.Pub(ctx, topic, &msgqueue.Message{
				ID:                msg.ID,
				Payloads:          chunk,
				TenantID:          msg.TenantID,
				ImmediatelyExpire: msg.ImmediatelyExpire,
				Persistent:        msg.Persistent,
				OtelCarrier:       msg.OtelCarrier,
				Retries:           msg.Retries, // nolint: staticcheck
				Compressed:        msg.Compressed,
			})

			if err != nil {
				return err
			}
		}

		return nil
	}

	if err := p.nc.Publish(subject, body); err != nil {
		p.l.Error().Ctx(ctx).Err(err).Str("subject", subject).Msg("error publishing pubsub message")
		return err
	}

	return nil
}

// Sub subscribes to a topic with plain Subscribe (fan-out to every subscriber).
// Delivery is at-most-once: handler errors are logged, never redelivered.
func (p *PubSub) Sub(topic msgqueue.Topic, handler msgqueue.MsgHandler) (func() error, error) {
	subject := "hatchet.pubsub." + topic.Name()

	sub, err := p.nc.Subscribe(subject, func(natsMsg *natsgo.Msg) {
		msg := &msgqueue.Message{}

		if err := json.Unmarshal(natsMsg.Data, msg); err != nil {
			p.l.Error().Err(err).Msg("error unmarshalling pubsub message")
			return
		}

		// The durable RabbitMQ path may compress payloads in place
		// (msg.Compressed=true, gzipped Payloads) before the same *Message
		// pointer reaches pubsub.Pub via PubTenantMessage. Mirror rabbitmq/pubsub.go
		// Sub and transparently decompress so handlers always see plain payloads.
		if msg.Compressed {
			decompressed, err := msgqueue.DecompressPayloads(msg.Payloads)
			if err != nil {
				p.l.Error().Err(err).Msg("error decompressing pubsub payloads")
				return
			}

			msg.Payloads = decompressed
		}

		if err := handler(msg); err != nil {
			p.l.Error().Err(err).Msgf("error handling pubsub message %s", msg.ID)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("could not subscribe to %s: %w", subject, err)
	}

	// Flush so interest is established before Sub returns.
	if err := p.nc.Flush(); err != nil {
		_ = sub.Unsubscribe()
		return nil, fmt.Errorf("could not flush after subscribe to %s: %w", subject, err)
	}

	return func() error {
		return sub.Unsubscribe()
	}, nil
}
