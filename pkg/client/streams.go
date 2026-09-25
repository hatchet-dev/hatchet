package client

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	sharedcontracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// StreamMessage is a single durably-persisted message read back from a
// durable stream topic. Cursor is this message's own position, so a caller
// can checkpoint after any individual message, not just at the start of a
// Subscribe call.
type StreamMessage struct {
	Payload   []byte
	Cursor    string
	CreatedAt time.Time
}

// StreamsHandler processes one StreamMessage. Returning an error stops the
// enclosing Subscribe call.
type StreamsHandler func(msg StreamMessage) error

// StreamsClient is the low-level client for the durable, topic-based streams
// feature (see api-contracts/v1/streams.proto). It is distinct from the
// legacy per-workflow-run stream events exposed by EventClient.PutStreamEvent
// and SubscribeClient.Stream, which are fanout-only and never persisted.
type StreamsClient interface {
	// Publish durably publishes a message to a topic. Topics are created
	// implicitly on first publish. A cursor is purely a client-side
	// construction read off a message actually received from Subscribe, so
	// there is nothing for Publish to return beyond success or failure.
	Publish(ctx context.Context, namespace, topic string, payload []byte) error

	// Subscribe performs a keyset catch-up from cursor (or from "now" if cursor
	// is nil) and then tails the topic, invoking handler once per message
	// (the server may batch several messages into one wire frame during
	// catch-up, transparently to handler), until handler returns an error,
	// the context is cancelled, or the server hangs up.
	Subscribe(ctx context.Context, namespace, topic string, cursor *string, handler StreamsHandler) error
}

// producerSeqState guards the next producer_seq for one (namespace, topic):
// its lock is held for a whole Publish call, not just the increment, so a
// failed call never advances seq (a retry reuses it) while still making it
// impossible for a concurrent call on the same key to reuse a seq that a
// still-in-flight call might yet succeed with.
type producerSeqState struct {
	mu  sync.Mutex
	seq int64
}

type streamsClientImpl struct {
	client sharedcontracts.V1StreamsClient
	ctx    *contextLoader

	// producerID identifies this client instance to the server for ordering
	// purposes (see api-contracts/v1/streams.proto).
	producerID string
	mu         sync.Mutex
	seqStates  map[string]*producerSeqState
}

func newStreams(conn *grpc.ClientConn, opts *sharedClientOpts) StreamsClient {
	return &streamsClientImpl{
		client:     sharedcontracts.NewV1StreamsClient(conn),
		ctx:        opts.ctxLoader,
		producerID: uuid.NewString(),
		seqStates:  make(map[string]*producerSeqState),
	}
}

func (s *streamsClientImpl) seqState(namespace, topic string) *producerSeqState {
	key := namespace + "\x00" + topic

	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.seqStates[key]

	if !ok {
		state = &producerSeqState{}
		s.seqStates[key] = state
	}

	return state
}

func (s *streamsClientImpl) Publish(ctx context.Context, namespace, topic string, payload []byte) error {
	state := s.seqState(namespace, topic)

	state.mu.Lock()
	defer state.mu.Unlock()

	_, err := s.client.Publish(s.ctx.newContext(ctx), &sharedcontracts.PublishStreamMessageRequest{
		Namespace:   namespace,
		Topic:       topic,
		Payload:     payload,
		ProducerId:  s.producerID,
		ProducerSeq: state.seq,
	})

	if err != nil {
		return err
	}

	state.seq++

	return nil
}

func (s *streamsClientImpl) Subscribe(ctx context.Context, namespace, topic string, cursor *string, handler StreamsHandler) error {
	stream, err := s.client.Subscribe(s.ctx.newContext(ctx), &sharedcontracts.SubscribeStreamRequest{
		Namespace: namespace,
		Topic:     topic,
		Cursor:    cursor,
	})

	if err != nil {
		return err
	}

	for {
		msg, err := stream.Recv()

		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

			return err
		}

		if msg.Hangup {
			return nil
		}

		for _, entry := range msg.Entries {
			var createdAt time.Time

			if entry.CreatedAt != nil {
				createdAt = entry.CreatedAt.AsTime()
			}

			if err := handler(StreamMessage{
				Payload:   entry.Payload,
				Cursor:    entry.Cursor,
				CreatedAt: createdAt,
			}); err != nil {
				return err
			}
		}
	}
}
