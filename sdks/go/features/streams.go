package features

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/client"
)

// StreamEvent is a stored message. Its Cursor resumes Events right after it.
type StreamEvent struct {
	Payload   []byte
	Cursor    string
	CreatedAt time.Time
}

// StreamTopicMetadata describes a topic's messages within the tenant's retention.
type StreamTopicMetadata = client.StreamTopicMetadata

type streamCallOpts struct {
	namespace string
	// Events only
	cursor *string
}

// StreamCallOpt configures a call to StreamsClient.Publish, StreamsClient.Events or StreamsClient.TopicMetadata.
type StreamCallOpt func(*streamCallOpts)

// WithNamespace sets the namespace; the default is "".
func WithNamespace(namespace string) StreamCallOpt {
	return func(o *streamCallOpts) {
		o.namespace = namespace
	}
}

// WithCursor resumes Events after the StreamEvent the cursor came from.
// Without it, Events starts at the oldest retained message. Ignored by Publish and TopicMetadata.
func WithCursor(cursor string) StreamCallOpt {
	return func(o *streamCallOpts) {
		o.cursor = &cursor
	}
}

// StreamsClient publishes to and reads from durable topics. Unlike
// ctx.PutStream streams, topics are stored, independent of any run, and
// readers can resume from a cursor.
type StreamsClient struct {
	v0Client client.Client
	l        *zerolog.Logger
}

func NewStreamsClient(v0Client client.Client) *StreamsClient {
	logger := v0Client.Logger()

	return &StreamsClient{
		v0Client: v0Client,
		l:        logger,
	}
}

// Publish returns once the message is stored. A topic is created on first
// publish. A client's messages to a topic are delivered in the order they were stored.
func (s *StreamsClient) Publish(ctx context.Context, topic string, message []byte, opts ...StreamCallOpt) error {
	o := &streamCallOpts{}

	for _, opt := range opts {
		opt(o)
	}

	return s.v0Client.Streams().Publish(ctx, o.namespace, topic, message)
}

// Events streams topic's messages from the start or WithCursor. The channel
// closes when ctx ends, the connection drops or the server hangs up; to
// continue, call Events again with the last event's Cursor.
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

// TopicMetadata returns a NotFound status error for a topic nothing was published to.
func (s *StreamsClient) TopicMetadata(ctx context.Context, topic string, opts ...StreamCallOpt) (*StreamTopicMetadata, error) {
	o := &streamCallOpts{}

	for _, opt := range opts {
		opt(o)
	}

	return s.v0Client.Streams().TopicMetadata(ctx, o.namespace, topic)
}
