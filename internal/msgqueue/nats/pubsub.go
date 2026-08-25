package nats

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"slices"

	natsgo "github.com/nats-io/nats.go"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/pkg/logger"
)

// defaultSubjectPrefix is used when WithPubSubSubjectPrefix is unset or empty.
const defaultSubjectPrefix = "hatchet.pubsub"

// PubSub implements msgqueue.PubSub over core NATS. Subjects are
// subjectPrefix + "." + topic.Name() (default prefix "hatchet.pubsub"),
// delivery is best-effort at-most-once.
type PubSub struct {
	nc            *natsgo.Conn
	l             *zerolog.Logger
	subjectPrefix string
}

type PubSubOpt func(*PubSubOpts)

type PubSubOpts struct {
	l             *zerolog.Logger
	url           string
	username      string
	password      string
	tlsEnabled    bool
	tlsRootCAFile string
	subjectPrefix string
}

func defaultPubSubOpts() *PubSubOpts {
	l := logger.NewDefaultLogger("nats-pubsub")

	return &PubSubOpts{
		l: &l,
	}
}

// WithPubSubURL sets the NATS seed URL(s). Comma-separated lists are passed
// through to nats.go. Prefer bare hosts and set Username/Password so
// rediscovered cluster peers authenticate.
func WithPubSubURL(url string) PubSubOpt {
	return func(opts *PubSubOpts) {
		opts.url = url
	}
}

// WithPubSubUsername sets the NATS username for nats.UserInfo.
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

// WithPubSubTLSEnabled requires TLS with a TLS-first handshake (the server
// must enable handshake_first). Verification uses the system roots unless a
// root CA file is set.
func WithPubSubTLSEnabled(enabled bool) PubSubOpt {
	return func(opts *PubSubOpts) {
		opts.tlsEnabled = enabled
	}
}

// WithPubSubTLSRootCAFile sets a PEM CA bundle for server verification.
// Requires WithPubSubTLSEnabled(true).
func WithPubSubTLSRootCAFile(path string) PubSubOpt {
	return func(opts *PubSubOpts) {
		opts.tlsRootCAFile = path
	}
}

// WithPubSubSubjectPrefix sets the NATS subject prefix (default
// "hatchet.pubsub"). Empty falls back to the default. No trimming or
// validation: a bad prefix fails loudly via nats ErrBadSubject at startup.
func WithPubSubSubjectPrefix(prefix string) PubSubOpt {
	return func(opts *PubSubOpts) {
		opts.subjectPrefix = prefix
	}
}

func WithPubSubLogger(l *zerolog.Logger) PubSubOpt {
	return func(opts *PubSubOpts) {
		opts.l = l
	}
}

// NewPubSub connects synchronously to NATS and returns a PubSub. Fails if the
// server is unreachable or if its max_payload is below msgqueue.MaxMessageSize.
func NewPubSub(fs ...PubSubOpt) (func() error, *PubSub, error) {
	opts := defaultPubSubOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.url == "" {
		return nil, nil, fmt.Errorf("nats pubsub requires a URL to be set")
	}

	if opts.tlsRootCAFile != "" && !opts.tlsEnabled {
		return nil, nil, fmt.Errorf("nats pubsub tlsRootCAFile is set but tlsEnabled is false; a private CA bundle only takes effect with tlsEnabled: true (SERVER_MSGQUEUE_PUBSUB_NATS_TLS_ENABLED)")
	}

	l := opts.l

	connectOpts := []natsgo.Option{
		// Reconnect behavior is deliberately fixed rather than configurable.
		// MaxReconnects(-1) retries forever: any finite limit permanently
		// closes the connection once exhausted, leaving the engine without
		// pub/sub until a process restart (the rabbitmq backend also retries
		// indefinitely).
		natsgo.MaxReconnects(-1),
		// ReconnectBufSize(-1) disables the client-side buffer that would
		// otherwise queue publishes during a disconnect and flush them on
		// reconnect. These signals are latency optimizations over their
		// consumers' polling paths (schedulers poll their queues, the
		// dispatcher polls for unacked finished runs), so by the time a
		// buffered publish flushed, polling would already have covered it;
		// Pub fails fast while disconnected instead.
		natsgo.ReconnectBufSize(-1),
		// Empty credentials never reach the wire (the CONNECT payload omits
		// empty user/pass fields), so UserInfo is safe to set unconditionally.
		// Whether a username requires a password is the server's decision,
		// enforced by the synchronous Connect below.
		natsgo.UserInfo(opts.username, opts.password),
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

	if opts.tlsEnabled {
		// Non-nil config: bare Secure() skips server verification per its
		// doc comment.
		connectOpts = append(connectOpts,
			natsgo.Secure(&tls.Config{MinVersion: tls.VersionTLS13}),
			natsgo.TLSHandshakeFirst(),
		)

		if opts.tlsRootCAFile != "" {
			// RootCAs fails Connect on a missing or non-PEM file before any dial.
			connectOpts = append(connectOpts, natsgo.RootCAs(opts.tlsRootCAFile))
		}
	}

	nc, err := natsgo.Connect(opts.url, connectOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("could not connect to nats at %q: %w", opts.url, err)
	}

	// The server must accept Hatchet's message-size contract: Pub chunks
	// multi-payload messages down to the server's max_payload, but a single
	// payload (e.g. one task stream event) cannot be split and may approach
	// msgqueue.MaxMessageSize. The NATS default max_payload is 1MiB, so
	// refuse to start against a misconfigured server rather than dropping
	// oversized publishes at runtime.
	if nc.MaxPayload() < msgqueue.MaxMessageSize {
		nc.Close()
		return nil, nil, fmt.Errorf(
			"nats server max_payload is %d bytes but hatchet requires at least %d; raise max_payload in the NATS server config",
			nc.MaxPayload(), msgqueue.MaxMessageSize,
		)
	}

	prefix := opts.subjectPrefix
	if prefix == "" {
		prefix = defaultSubjectPrefix
	}

	p := &PubSub{
		nc:            nc,
		l:             l,
		subjectPrefix: prefix,
	}

	return func() error {
		nc.Close()
		return nil
	}, p, nil
}

func (p *PubSub) IsReady() bool {
	return p.nc.IsConnected()
}

func (p *PubSub) subject(topic msgqueue.Topic) string {
	return p.subjectPrefix + "." + topic.Name()
}

// Pub publishes a message to the topic.
// Oversized multi-payload messages are chunked like rabbitmq/pubsub.go.
func (p *PubSub) Pub(ctx context.Context, topic msgqueue.Topic, msg *msgqueue.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	subject := p.subject(topic)

	body, err := json.Marshal(msg)
	if err != nil {
		p.l.Error().Ctx(ctx).Err(err).Msg("error marshaling pubsub message")
		return err
	}

	maxPayload := p.nc.MaxPayload()

	if int64(len(body)) > maxPayload {
		if len(msg.Payloads) == 1 {
			return fmt.Errorf("message size %d bytes exceeds maximum allowed size of %d bytes", len(body), maxPayload)
		}

		// split the payloads in half and publish recursively until each chunk is
		// under the max size (same strategy as rabbitmq/pubsub.go)
		payloadsPerChunk := max(len(msg.Payloads)/2, 1)

		for chunk := range slices.Chunk(msg.Payloads, payloadsPerChunk) {
			msgCp := msg.Clone()
			msgCp.Payloads = chunk

			err := p.Pub(ctx, topic, msgCp)
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
	subject := p.subject(topic)

	sub, err := p.nc.Subscribe(subject, func(natsMsg *natsgo.Msg) {
		msg := &msgqueue.Message{}

		if err := json.Unmarshal(natsMsg.Data, msg); err != nil {
			p.l.Error().Err(err).Msg("error unmarshalling pubsub message")
			return
		}

		// NATS Pub never compresses, so Compressed is expected to be false here.
		// We still honour the flag: compression is a per-message wire property
		// that any publisher on this subject may set, and handlers require plain
		// payloads either way (same contract as rabbitmq/pubsub.go Sub).
		if msg.Compressed {
			decompressed, err := msgqueue.DecompressPayloads(msg.Payloads)
			if err != nil {
				p.l.Error().Err(err).Msg("error decompressing pubsub payloads")
				return
			}

			msg.Payloads = decompressed
			msg.Compressed = false
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
