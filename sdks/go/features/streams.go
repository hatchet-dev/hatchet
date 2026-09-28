package features

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/client"
)

// StreamEvent is a single durably-persisted message read back from a durable
// stream topic. Cursor is this event's own position, so a caller can
// checkpoint after any individual event, not just at the start of a call to
// Events.
type StreamEvent struct {
	Payload   []byte
	Cursor    string
	CreatedAt time.Time
}

type streamCallOpts struct {
	namespace string
	// cursor is only read by Events; ignored by Publish.
	cursor *string
}

// StreamCallOpt configures a call to StreamsClient.Publish or StreamsClient.Events.
type StreamCallOpt func(*streamCallOpts)

// WithNamespace scopes a Publish or Events call to a namespace other than the
// default (empty-string) namespace.
func WithNamespace(namespace string) StreamCallOpt {
	return func(o *streamCallOpts) {
		o.namespace = namespace
	}
}

// WithCursor resumes Events from a cursor previously seen on a StreamEvent --
// a cursor is a client-side construction read off a received event, not
// something Publish produces. Omitting it starts delivery from "now" -- i.e.
// only messages published from this point forward, not a backfill of
// previously retained history. Has no effect on Publish.
func WithCursor(cursor string) StreamCallOpt {
	return func(o *streamCallOpts) {
		o.cursor = &cursor
	}
}

// StreamsClient provides methods for publishing to and reading from durable,
// topic-based streams. This is distinct from the ephemeral, per-run streaming
// exposed by ctx.PutStream / RunsClient.SubscribeToStream, which is
// fanout-only and never persisted: a durable stream topic is independent of
// any workflow run, and a late-connecting reader can resume from a cursor.
type StreamsClient struct {
	v0Client client.Client
	l        *zerolog.Logger
}

// NewStreamsClient creates a new client for publishing to and reading from
// durable stream topics.
func NewStreamsClient(v0Client client.Client) *StreamsClient {
	logger := v0Client.Logger()

	return &StreamsClient{
		v0Client: v0Client,
		l:        logger,
	}
}

// Publish durably publishes a message to a topic. Topics are created
// implicitly on first publish. Messages from this client instance are
// delivered to Events in the order Publish was called, even under concurrent
// calls or network/queue reordering.
func (s *StreamsClient) Publish(ctx context.Context, topic string, message []byte, opts ...StreamCallOpt) error {
	o := &streamCallOpts{}

	for _, opt := range opts {
		opt(o)
	}

	return s.v0Client.Streams().Publish(ctx, o.namespace, topic, message)
}

// Events returns a channel of messages published to topic, starting from the
// given cursor (WithCursor) or from "now" if none is supplied. The channel is
// closed when the underlying Subscribe call ends, whether because the context
// was cancelled or the connection dropped -- callers wanting automatic
// reconnection should re-call Events with the last-seen event's Cursor.
func (s *StreamsClient) Events(ctx context.Context, topic string, opts ...StreamCallOpt) <-chan StreamEvent {
	o := &streamCallOpts{}

	for _, opt := range opts {
		opt(o)
	}

	ch := make(chan StreamEvent)

	go func() {
		defer close(ch)

		err := s.v0Client.Streams().Subscribe(ctx, o.namespace, topic, o.cursor, func(msg client.StreamMessage) error {
			select {
			case ch <- StreamEvent{
				Payload:   msg.Payload,
				Cursor:    msg.Cursor,
				CreatedAt: msg.CreatedAt,
			}:
			case <-ctx.Done():
				return ctx.Err()
			}

			return nil
		})

		if err != nil && ctx.Err() == nil {
			s.l.Error().Ctx(ctx).Err(err).Str("topic", topic).Msg("stream consumption ended")
		}
	}()

	return ch
}
