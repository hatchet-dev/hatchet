package streams

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/services/shared/rpcstream"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// subscribeCatchUpBatchSize bounds a single page of the catch-up/tail keyset
// scan in Subscribe.
const subscribeCatchUpBatchSize = 500

// subscribeTailPollInterval is the fallback poll period in the shared
// topicPoller's tail loop (see topic_poller.go). The pubsub wake hint (see
// internal/msgqueue.StreamTopic) usually beats this, but Postgres is always
// re-queried as the source of truth regardless of which one fires.
const subscribeTailPollInterval = 1 * time.Second

// subscribeIdleHangupTimeout is how long a topicPoller waits with no new
// messages before hanging up every listener currently tailing it, so an idle
// topic doesn't hold its poll loop, listener goroutines, and HTTP/2 streams
// open indefinitely. A hung-up caller can always reconnect from the cursor
// it was sent.
const subscribeIdleHangupTimeout = 30 * time.Minute

func (s *ServiceImpl) Publish(ctx context.Context, req *contracts.PublishStreamMessageRequest) (*contracts.PublishStreamMessageResponse, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	if req.Topic == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("topic is required"))
	}

	if len(req.Payload) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("payload is required"))
	}

	if len(req.Payload) > v1.MaxStreamMessagePayloadBytes {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("payload exceeds maximum size of %d bytes", v1.MaxStreamMessagePayloadBytes))
	}

	if req.ProducerId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("producer_id is required"))
	}

	if err := s.repo.Streams().EnsureTopic(ctx, tenantId, req.Namespace, req.Topic); err != nil {
		if errors.Is(err, v1.ErrResourceExhausted) {
			return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("resource exhausted: stream topic limit exceeded for tenant"))
		}

		return nil, err
	}

	pre, post := s.repo.TenantLimit().Meter(ctx, nil, sqlcv1.LimitResourceSTREAMMESSAGE, tenantId, 1)

	if err := pre(); err != nil {
		if errors.Is(err, v1.ErrResourceExhausted) {
			return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("resource exhausted: stream message limit exceeded for tenant"))
		}

		return nil, err
	}

	now := time.Now()

	msg, err := msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDStreamMessage,
		true,
		true,
		tasktypes.StreamMessagePayload{
			Namespace:   req.Namespace,
			Topic:       req.Topic,
			Payload:     req.Payload,
			CreatedAt:   now,
			ProducerID:  req.ProducerId,
			ProducerSeq: req.ProducerSeq,
		},
	)

	if err != nil {
		return nil, err
	}

	// STREAMS_QUEUE requires publisher confirms, see Queue.RequiresPublishConfirm).
	// Return errors back to the publisher--firing and forgetting here can lead to the
	// SDK not being aware of failed messages, which will back up queue because inserting must be
	// in-order
	if err := s.pubBuffer.Pub(ctx, msgqueue.STREAMS_QUEUE, msg, true); err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("could not enqueue stream message: %w", err))
	}

	post()

	// Best-effort wake for any consumer currently tailing this topic. Never
	// the source of truth -- a dropped wake just means the tail loop's
	// fallback ticker (subscribeTailPollInterval) picks it up shortly after.
	if wakeMsg, err := msgqueue.NewTenantMessage(tenantId, msgqueue.MsgIDStreamMessage, true, false, struct{}{}); err == nil {
		_ = s.pubsub.Pub(ctx, msgqueue.StreamTopic(tenantId, req.Namespace, req.Topic), wakeMsg)
	}

	return &contracts.PublishStreamMessageResponse{}, nil
}

func (s *ServiceImpl) Subscribe(ctx context.Context, req *contracts.SubscribeStreamRequest, connectStream *connect.ServerStream[contracts.StreamMessage]) error {
	// other goroutines could in principle send on this stream after this
	// handler returns; Sender rejects those sends instead of racing the
	// HTTP/2 server's panic-on-write-after-handler-return.
	sender := rpcstream.NewSender[contracts.StreamMessage](ctx, connectStream)
	defer sender.Close()

	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	namespace, topic, cursor, err := resolveSubscribeAddressAndCursor(req)

	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	deregister := s.streamSessions.Register(cancel)
	defer deregister()

	// sendPage runs the keyset query from the current cursor, sending and
	// paging until a batch returns fewer than subscribeCatchUpBatchSize rows.
	// This same loop is used for both the initial catch-up (which may replay
	// a large backlog if a real cursor was supplied) and every subsequent
	// tail wake-up (which is normally a no-op since nothing new has arrived)
	// -- there is deliberately no separate branch for "no cursor supplied":
	// see the cursor default resolved above.
	sendPage := func() error {
		for {
			msgs, err := s.repo.Streams().ListMessagesAfterCursor(ctx, tenantId, v1.ListStreamMessagesOpts{
				Namespace: namespace,
				Topic:     topic,
				Cursor:    cursor,
				Limit:     subscribeCatchUpBatchSize,
			})

			if err != nil {
				return err
			}

			entries := make([]*contracts.StreamEntry, 0, len(msgs))

			for _, m := range msgs {
				next := v1.StreamCursor{Namespace: namespace, Topic: topic, CreatedAt: m.InsertedAt.Time, ID: m.ID}

				encodedCursor, err := v1.EncodeStreamCursor(next)

				if err != nil {
					return err
				}

				entries = append(entries, &contracts.StreamEntry{
					Payload:   m.Payload,
					Cursor:    encodedCursor,
					CreatedAt: timestamppb.New(m.InsertedAt.Time),
				})

				cursor = next
			}

			for _, chunk := range chunkStreamEntries(entries) {
				if err := sender.Send(&contracts.StreamMessage{Entries: chunk}); err != nil {
					return err
				}
			}

			if len(msgs) < subscribeCatchUpBatchSize {
				return nil
			}
		}
	}

	if err := sendPage(); err != nil {
		return err
	}

	// all tails on the same topic are pooled together
	unregister, err := s.topicPollers.Join(ctx, topicPollerKey{
		tenantId:  tenantId,
		namespace: namespace,
		topic:     topic,
	}, cursor, &topicListener{
		send:   sender.Send,
		cancel: cancel,
	})

	if err != nil {
		return err
	}

	defer unregister()

	// A topic has no natural end -- unlike workflow-run event streams, this
	// RPC only returns via ctx.Done() (client cancel, a failed send on the
	// shared poller calling cancel on our behalf, or graceful shutdown via
	// CancelStreamSessions).
	<-ctx.Done()

	return nil
}

// resolveSubscribeAddressAndCursor determines which (namespace, topic) to
// subscribe to and the effective starting cursor for a Subscribe call.
func resolveSubscribeAddressAndCursor(req *contracts.SubscribeStreamRequest) (namespace, topic string, cursor v1.StreamCursor, err error) {
	// default cursor starts from the beginning of the topic; ordering is by id
	// alone, so 0 is before every real row.
	decoded := v1.StreamCursor{
		Namespace: req.Namespace,
		Topic:     req.Topic,
		CreatedAt: time.Time{},
		ID:        0,
	}

	if req.Cursor != nil && *req.Cursor != "" {
		c, err := v1.DecodeStreamCursor(*req.Cursor)

		if err != nil {
			return "", "", v1.StreamCursor{}, err
		}

		decoded = c
	}

	if req.Topic == "" {
		if decoded.Topic == "" {
			return "", "", v1.StreamCursor{}, fmt.Errorf("topic is required unless cursor is supplied")
		}

		return decoded.Namespace, decoded.Topic, decoded, nil
	}

	if req.Cursor != nil && *req.Cursor != "" && (decoded.Namespace != req.Namespace || decoded.Topic != req.Topic) {
		return "", "", v1.StreamCursor{}, fmt.Errorf(
			"cursor belongs to namespace %q topic %q, not namespace %q topic %q; omit topic to resume the cursor's own topic",
			decoded.Namespace, decoded.Topic, req.Namespace, req.Topic,
		)
	}

	return req.Namespace, req.Topic, decoded, nil
}
