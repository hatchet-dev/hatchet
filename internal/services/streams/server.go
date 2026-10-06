package streams

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/timestamppb"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/services/shared/rpcstream"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

// subscribeTailPollInterval is the poller's fallback when a wake is lost.
const subscribeTailPollInterval = 1 * time.Second

// subscribeIdleHangupTimeout frees an idle topic's poller and streams; a
// hung-up caller resumes from its last cursor.
const subscribeIdleHangupTimeout = 30 * time.Minute

func (s *ServiceImpl) Publish(ctx context.Context, req *contracts.PublishStreamMessageRequest) (*contracts.PublishStreamMessageResponse, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	telemetry.WithAttributes(trace.SpanFromContext(ctx),
		telemetry.AttributeKV{Key: "tenant.id", Value: tenantId},
		telemetry.AttributeKV{Key: "stream.namespace", Value: req.Namespace},
		telemetry.AttributeKV{Key: "stream.topic", Value: req.Topic},
		telemetry.AttributeKV{Key: "stream.payload_bytes", Value: len(req.Payload)},
		telemetry.AttributeKV{Key: "stream.payload_ref", Value: req.PayloadRef != ""},
	)

	entitled, err := s.repo.TenantEntitlement().HasEntitlement(ctx, tenantId, v1.EntitlementDurableStreams)

	if err != nil {
		return nil, err
	}

	if !entitled {
		return nil, connect.NewError(connect.CodePermissionDenied, errNotEntitled)
	}

	if err := v1.ValidateStreamAddress(req.Namespace, req.Topic); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var payloadRef *v1.StreamPayloadRef

	if req.PayloadRef != "" {
		if len(req.Payload) > 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("payload and payload_ref are mutually exclusive"))
		}

		ref, err := v1.DecodeStreamPayloadRef(req.PayloadRef)

		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		payloadRef = &ref
	} else if len(req.Payload) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("payload is required"))
	}

	if len(req.Payload) > v1.MaxStreamMessagePayloadBytes {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("payload exceeds maximum size of %d bytes", v1.MaxStreamMessagePayloadBytes))
	}

	if req.ProducerId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("producer_id is required"))
	}

	// readers would otherwise get a message whose payload can't be fetched
	if payloadRef != nil {
		checkCtx, span := telemetry.NewSpan(ctx, "streams.publish.check-payload-ref")
		err := s.repo.Streams().CheckStreamPayloadExists(checkCtx, tenantId, *payloadRef)
		recordSpanError(span, err)
		span.End()

		if err != nil {
			if errors.Is(err, v1.ErrStreamPayloadNotFound) {
				return nil, connect.NewError(connect.CodeInvalidArgument, err)
			}

			return nil, err
		}
	}

	ensureCtx, span := telemetry.NewSpan(ctx, "streams.publish.ensure-topic")
	err = s.repo.Streams().EnsureTopic(ensureCtx, tenantId, req.Namespace, req.Topic)
	recordSpanError(span, err)
	span.End()

	if err != nil {
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

	res, err := s.publisher.publish(ctx, v1.TenantStreamMessage{
		TenantID: tenantId,
		Opts: v1.CreateOrderedStreamMessageOpts{
			Namespace:   req.Namespace,
			Topic:       req.Topic,
			Payload:     req.Payload,
			ProducerID:  req.ProducerId,
			ProducerSeq: req.ProducerSeq,
			PayloadRef:  payloadRef,
		},
	})

	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("could not persist stream message: %w", err))
	}

	// a sequence at or below the watermark is a retry of a message already stored
	if !res.Inserted && res.CurrentSeq < req.ProducerSeq {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf(
			"producer %s skipped a sequence number: last stored %d, got %d", req.ProducerId, res.CurrentSeq, req.ProducerSeq,
		))
	}

	post()

	return &contracts.PublishStreamMessageResponse{}, nil
}

func (s *ServiceImpl) Subscribe(ctx context.Context, req *contracts.SubscribeStreamRequest, connectStream *connect.ServerStream[contracts.StreamMessage]) error {
	// the poller may send after this handler returns, which would panic on the raw stream
	sender := rpcstream.NewSender[contracts.StreamMessage](ctx, connectStream)
	defer sender.Close()

	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	entitled, err := s.repo.TenantEntitlement().HasEntitlement(ctx, tenantId, v1.EntitlementDurableStreams)

	if err != nil {
		return err
	}

	if !entitled {
		return connect.NewError(connect.CodePermissionDenied, errNotEntitled)
	}

	namespace, topic, cursor, err := resolveSubscribeAddressAndCursor(req)

	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	telemetry.WithAttributes(trace.SpanFromContext(ctx),
		telemetry.AttributeKV{Key: "tenant.id", Value: tenantId},
		telemetry.AttributeKV{Key: "stream.namespace", Value: namespace},
		telemetry.AttributeKV{Key: "stream.topic", Value: topic},
		telemetry.AttributeKV{Key: "stream.resumed", Value: cursor.ID > 0},
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	deregister := s.streamSessions.Register(cancel)
	defer deregister()

	key := topicPollerKey{
		tenantId:  tenantId,
		namespace: namespace,
		topic:     topic,
	}
	unregister, err := s.topicPollers.Join(ctx, key, cursor, &topicListener{
		send:   sender.Send,
		cancel: cancel,
	})

	var expired *v1.StreamCursorExpiredError

	if errors.As(err, &expired) {
		return connect.NewError(connect.CodeOutOfRange, fmt.Errorf("%w; subscribe without a cursor to start from the oldest retained message", expired))
	}

	if err != nil {
		return err
	}

	defer unregister()

	// a topic has no end: this returns on client cancel, a failed send, idle
	// hangup or shutdown
	<-ctx.Done()

	return nil
}

func (s *ServiceImpl) GetTopicMetadata(ctx context.Context, req *contracts.GetStreamTopicMetadataRequest) (*contracts.StreamTopicMetadata, error) {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	telemetry.WithAttributes(trace.SpanFromContext(ctx),
		telemetry.AttributeKV{Key: "tenant.id", Value: tenantId},
		telemetry.AttributeKV{Key: "stream.namespace", Value: req.Namespace},
		telemetry.AttributeKV{Key: "stream.topic", Value: req.Topic},
	)

	entitled, err := s.repo.TenantEntitlement().HasEntitlement(ctx, tenantId, v1.EntitlementDurableStreams)

	if err != nil {
		return nil, err
	}

	if !entitled {
		return nil, connect.NewError(connect.CodePermissionDenied, errNotEntitled)
	}

	if err := v1.ValidateStreamAddress(req.Namespace, req.Topic); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	md, err := s.repo.Streams().GetTopicMetadata(ctx, tenantId, req.Namespace, req.Topic)

	if errors.Is(err, v1.ErrStreamTopicNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	if err != nil {
		return nil, err
	}

	resp := &contracts.StreamTopicMetadata{
		Namespace:    req.Namespace,
		Topic:        req.Topic,
		TenantId:     tenantId.String(),
		MessageCount: md.MessageCount,
	}

	if md.LatestCursor != nil {
		cursor, err := v1.EncodeStreamCursor(*md.LatestCursor)

		if err != nil {
			return nil, err
		}

		resp.LatestCursor = &cursor
		resp.LastPublishedAt = timestamppb.New(md.LatestCursor.CreatedAt)
	}

	return resp, nil
}

func recordSpanError(span trace.Span, err error) {
	if err == nil {
		return
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

func resolveSubscribeAddressAndCursor(req *contracts.SubscribeStreamRequest) (namespace, topic string, cursor v1.StreamCursor, err error) {
	// offsets start at 1, so ID 0 is the start of the topic
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

		if err := v1.ValidateStreamAddress(decoded.Namespace, decoded.Topic); err != nil {
			return "", "", v1.StreamCursor{}, err
		}

		return decoded.Namespace, decoded.Topic, decoded, nil
	}

	if req.Cursor != nil && *req.Cursor != "" && (decoded.Namespace != req.Namespace || decoded.Topic != req.Topic) {
		return "", "", v1.StreamCursor{}, fmt.Errorf(
			"cursor belongs to namespace %q topic %q, not namespace %q topic %q; omit topic to resume the cursor's own topic",
			decoded.Namespace, decoded.Topic, req.Namespace, req.Topic,
		)
	}

	if err := v1.ValidateStreamAddress(req.Namespace, req.Topic); err != nil {
		return "", "", v1.StreamCursor{}, err
	}

	return req.Namespace, req.Topic, decoded, nil
}
