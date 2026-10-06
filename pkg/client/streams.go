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

// StreamTopicMetadata covers only messages within the tenant's retention.
type StreamTopicMetadata struct {
	Namespace       string
	Topic           string
	TenantID        string
	MessageCount    int64
	LatestCursor    *string
	LastPublishedAt *time.Time
}

type StreamsClient interface {
	// Publish returns once the message is stored. A topic is created on first publish.
	Publish(ctx context.Context, namespace, topic string, payload []byte) error

	// Subscribe delivers every message after cursor (or from the oldest), then
	// tails the topic until handler errors, ctx ends, or the server hangs up.
	Subscribe(ctx context.Context, namespace, topic string, cursor *string, handler StreamsHandler) error

	// TopicMetadata returns a NotFound status error for a topic nothing was published to.
	TopicMetadata(ctx context.Context, namespace, topic string) (*StreamTopicMetadata, error)
}

// producerSeqState is locked for a whole Publish, so a concurrent publish
// can't reuse a seq an in-flight one may still store.
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

// publishRejectedBeforeStoring: the server stored nothing, so the seq is unused.
func publishRejectedBeforeStoring(err error) bool {
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

	for attempt := 0; ; attempt++ {
		_, err := s.client.Publish(s.ctx.newContext(ctx), &sharedcontracts.PublishStreamMessageRequest{
			Namespace:   namespace,
			Topic:       topic,
			Payload:     payload,
			ProducerId:  state.producerID,
			ProducerSeq: state.seq,
		})

		if err == nil {
			state.seq++
			return nil
		}

		// it may still land, and a different payload under its seq would be dropped as a duplicate
		if !publishRejectedBeforeStoring(err) {
			state.producerID = uuid.NewString()
			state.seq = 0
		}

		// a gap stored nothing (e.g. the watermark passed cursor retention), so resend as the new producer
		if status.Code(err) == codes.FailedPrecondition && attempt == 0 {
			continue
		}

		return err
	}
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

func (s *streamsClientImpl) TopicMetadata(ctx context.Context, namespace, topic string) (*StreamTopicMetadata, error) {
	resp, err := s.client.GetTopicMetadata(s.ctx.newContext(ctx), &sharedcontracts.GetStreamTopicMetadataRequest{
		Namespace: namespace,
		Topic:     topic,
	})

	if err != nil {
		return nil, err
	}

	md := &StreamTopicMetadata{
		Namespace:    resp.Namespace,
		Topic:        resp.Topic,
		TenantID:     resp.TenantId,
		MessageCount: resp.MessageCount,
		LatestCursor: resp.LatestCursor,
	}

	if resp.LastPublishedAt != nil {
		publishedAt := resp.LastPublishedAt.AsTime()
		md.LastPublishedAt = &publishedAt
	}

	return md, nil
}
