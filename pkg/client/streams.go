package client

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sharedcontracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

type StreamMessage struct {
	Payload   []byte
	Cursor    string
	CreatedAt time.Time
}

type StreamsHandler func(msg StreamMessage) error

type StreamsClient interface {
	// Publish durably publishes a message to a topic. Topics are created
	// implicitly on first publish.
	Publish(ctx context.Context, namespace, topic string, payload []byte) error

	// Subscribe returns all existing messages from topic (from cursor-on) in batches
	// and then tails the topic, until handler returns an error,
	// the context is cancelled, or the server hangs up.
	Subscribe(ctx context.Context, namespace, topic string, cursor *string, handler StreamsHandler) error
}

// producerSeqState guards the producer identity and next producer_seq for one
// (namespace, topic): its lock is held for a whole Publish call, not just the
// increment, so a concurrent call on the same key can never reuse a seq that a
// still-in-flight call might yet succeed with.
type producerSeqState struct {
	mu         sync.Mutex
	producerID string
	seq        int64
}

type streamsClientImpl struct {
	client    sharedcontracts.V1StreamsClient
	ctx       *contextLoader
	mu        sync.Mutex
	seqStates map[string]*producerSeqState
}

func newStreams(conn *grpc.ClientConn, opts *sharedClientOpts) StreamsClient {
	return &streamsClientImpl{
		client:    sharedcontracts.NewV1StreamsClient(conn),
		ctx:       opts.ctxLoader,
		seqStates: make(map[string]*producerSeqState),
	}
}

// publishRejectedBeforeEnqueue reports whether err is one the server returns
// before the message could have reached the queue, so its seq was never used.
func publishRejectedBeforeEnqueue(err error) bool {
	switch status.Code(err) {
	case codes.InvalidArgument, codes.ResourceExhausted, codes.Unauthenticated, codes.PermissionDenied:
		return true
	default:
		return false
	}
}

func (s *streamsClientImpl) seqState(namespace, topic string) *producerSeqState {
	key := namespace + "\x00" + topic

	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.seqStates[key]

	if !ok {
		state = &producerSeqState{producerID: uuid.NewString()}
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
		ProducerId:  state.producerID,
		ProducerSeq: state.seq,
	})

	if err != nil {
		// the message may still land, so reusing its seq for a different
		// payload would get that payload dropped as a duplicate
		if !publishRejectedBeforeEnqueue(err) {
			state.producerID = uuid.NewString()
			state.seq = 0
		}

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
