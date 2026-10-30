package streams

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
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
	entries := []*contracts.StreamPublishEntry{{Payload: req.Payload, PayloadRef: req.PayloadRef}}

	if err := s.publish(ctx, req.Namespace, req.Topic, req.ProducerId, req.ProducerSeq, entries); err != nil {
		return nil, err
	}

	return &contracts.PublishStreamMessageResponse{}, nil
}

func (s *ServiceImpl) PublishBatch(ctx context.Context, req *contracts.PublishStreamMessagesRequest) (*contracts.PublishStreamMessagesResponse, error) {
	if len(req.Entries) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("entries is required"))
	}

	if err := s.publish(ctx, req.Namespace, req.Topic, req.ProducerId, req.FirstProducerSeq, req.Entries); err != nil {
		return nil, err
	}

	return &contracts.PublishStreamMessagesResponse{}, nil
}

// publish stores entries as the producer's messages firstSeq, firstSeq+1, ...
// in one transaction.
func (s *ServiceImpl) publish(ctx context.Context, namespace, topic, producerId string, firstSeq int64, entries []*contracts.StreamPublishEntry) error {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	payloadBytes := 0
	hasPayloadRef := false

	for _, e := range entries {
		payloadBytes += len(e.Payload)
		hasPayloadRef = hasPayloadRef || e.PayloadRef != ""
	}

	telemetry.WithAttributes(trace.SpanFromContext(ctx),
		telemetry.AttributeKV{Key: "tenant.id", Value: tenantId},
		telemetry.AttributeKV{Key: "stream.namespace", Value: namespace},
		telemetry.AttributeKV{Key: "stream.topic", Value: topic},
		telemetry.AttributeKV{Key: "stream.entries", Value: len(entries)},
		telemetry.AttributeKV{Key: "stream.payload_bytes", Value: payloadBytes},
		telemetry.AttributeKV{Key: "stream.payload_ref", Value: hasPayloadRef},
	)

	entitled, err := s.repo.TenantEntitlement().HasEntitlement(ctx, tenantId, v1.EntitlementDurableStreams)

	if err != nil {
		return err
	}

	if !entitled {
		return connect.NewError(connect.CodePermissionDenied, errNotEntitled)
	}

	if err := v1.ValidateStreamAddress(namespace, topic); err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	if producerId == "" {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("producer_id is required"))
	}

	if firstSeq < 0 {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("producer sequence must not be negative"))
	}

	msgs := make([]v1.TenantStreamMessage, len(entries))

	for i, e := range entries {
		payloadRef, err := s.validatePublishEntry(ctx, tenantId, e)

		if err != nil {
			if len(entries) > 1 {
				return wrapEntryError(i, err)
			}

			return err
		}

		msgs[i] = v1.TenantStreamMessage{
			TenantID: tenantId,
			Opts: v1.CreateOrderedStreamMessageOpts{
				Namespace:   namespace,
				Topic:       topic,
				Payload:     e.Payload,
				ProducerID:  producerId,
				ProducerSeq: firstSeq + int64(i),
				PayloadRef:  payloadRef,
			},
		}
	}

	ensureCtx, span := telemetry.NewSpan(ctx, "streams.publish.ensure-topic")
	err = s.repo.Streams().EnsureTopic(ensureCtx, tenantId, namespace, topic)
	recordSpanError(span, err)
	span.End()

	if err != nil {
		if errors.Is(err, v1.ErrResourceExhausted) {
			return connect.NewError(connect.CodeResourceExhausted, errors.New("resource exhausted: stream topic limit exceeded for tenant"))
		}

		return err
	}

	pre, post := s.repo.TenantLimit().Meter(ctx, nil, sqlcv1.LimitResourceSTREAMMESSAGE, tenantId, int32(len(msgs))) // nolint:gosec

	if err := pre(); err != nil {
		if errors.Is(err, v1.ErrResourceExhausted) {
			return connect.NewError(connect.CodeResourceExhausted, errors.New("resource exhausted: stream message limit exceeded for tenant"))
		}

		return err
	}

	results, err := s.publisher.publish(ctx, msgs...)

	if err != nil {
		return connect.NewError(connect.CodeUnavailable, fmt.Errorf("could not persist stream message"))
	}

	// a sequence at or below the watermark is a retry of a message already
	// stored; once one entry hits a gap every later one does too
	for i, res := range results {
		if seq := firstSeq + int64(i); !res.Inserted && res.CurrentSeq < seq {
			return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf(
				"producer %s skipped a sequence number: last stored %d, got %d", producerId, res.CurrentSeq, seq,
			))
		}
	}

	post()

	return nil
}

// validatePublishEntry returns the entry's payload ref, if it has one.
func (s *ServiceImpl) validatePublishEntry(ctx context.Context, tenantId uuid.UUID, e *contracts.StreamPublishEntry) (*v1.StreamPayloadRef, error) {
	if e.PayloadRef == "" {
		if len(e.Payload) == 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("payload is required"))
		}

		if len(e.Payload) > v1.MaxStreamMessagePayloadBytes {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("payload exceeds maximum size of %d bytes", v1.MaxStreamMessagePayloadBytes))
		}

		return nil, nil
	}

	if len(e.Payload) > 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("payload and payload_ref are mutually exclusive"))
	}

	ref, err := v1.DecodeStreamPayloadRef(e.PayloadRef)

	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// readers would otherwise get a message whose payload can't be fetched
	checkCtx, span := telemetry.NewSpan(ctx, "streams.publish.check-payload-ref")
	err = s.repo.Streams().CheckStreamPayloadExists(checkCtx, tenantId, ref)
	recordSpanError(span, err)
	span.End()

	if err != nil {
		if errors.Is(err, v1.ErrStreamPayloadNotFound) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		return nil, err
	}

	return &ref, nil
}

// wrapEntryError names the batch entry err is about, keeping its code.
func wrapEntryError(i int, err error) error {
	var connectErr *connect.Error

	if errors.As(err, &connectErr) {
		return connect.NewError(connectErr.Code(), fmt.Errorf("entry %d: %w", i, connectErr.Unwrap()))
	}

	return fmt.Errorf("entry %d: %w", i, err)
}

func (s *ServiceImpl) Subscribe(ctx context.Context, req *contracts.SubscribeStreamRequest, connectStream *connect.ServerStream[contracts.StreamMessage]) error {
	// the poller may send after this handler returns, which would panic on the raw stream
	sender := rpcstream.NewSender[contracts.StreamMessage](ctx, connectStream)
	defer sender.Close()

	return s.subscribe(ctx, req, sender)
}

// RegisterSubscribe serves Subscribe over channels to a caller in the same process (the
// in-process operator host), with every check the RPC makes: ctx must carry the tenant, as
// the auth middleware puts it, and a tenant without the durable streams entitlement is
// refused with PermissionDenied, which the stream's Err reports once it ends.
func (s *ServiceImpl) RegisterSubscribe(ctx context.Context, req *contracts.SubscribeStreamRequest) (*rpcstream.ChannelStream, error) {
	if _, ok := ctx.Value("tenant").(*sqlcv1.Tenant); !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("tenant not found on context"))
	}

	return rpcstream.ServeServer(ctx, func(ctx context.Context, sender *rpcstream.Sender[contracts.StreamMessage]) error {
		return s.subscribe(ctx, req, sender)
	}), nil
}

// subscribe is Subscribe over any sender: it holds the stream open on the topic's poller
// until ctx ends, a send fails, the topic idles out or the service shuts down.
func (s *ServiceImpl) subscribe(ctx context.Context, req *contracts.SubscribeStreamRequest, sender *rpcstream.Sender[contracts.StreamMessage]) error {
	tenant := ctx.Value("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	entitled, err := s.repo.TenantEntitlement().HasEntitlement(ctx, tenantId, v1.EntitlementDurableStreams)

	if err != nil {
		return err
	}

	if !entitled {
		return connect.NewError(connect.CodePermissionDenied, errNotEntitled)
	}

	resumed := req.Cursor != nil && *req.Cursor != ""

	if req.Topic == "" && !resumed {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("topic is required unless cursor is supplied"))
	}

	if req.Topic != "" {
		if err := v1.ValidateStreamAddress(req.Namespace, req.Topic); err != nil {
			return connect.NewError(connect.CodeInvalidArgument, err)
		}
	}

	var cursor v1.StreamCursor

	if resumed {
		cursor, err = s.resolveCursor(ctx, tenantId, req.Namespace, req.Topic, *req.Cursor)
	} else {
		cursor, err = s.repo.Streams().TopicStartCursor(ctx, tenantId, req.Namespace, req.Topic)
	}

	if err != nil {
		return err
	}

	namespace, topic := cursor.Namespace, cursor.Topic

	telemetry.WithAttributes(trace.SpanFromContext(ctx),
		telemetry.AttributeKV{Key: "tenant.id", Value: tenantId},
		telemetry.AttributeKV{Key: "stream.namespace", Value: namespace},
		telemetry.AttributeKV{Key: "stream.topic", Value: topic},
		telemetry.AttributeKV{Key: "stream.resumed", Value: resumed},
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

	if err != nil {
		return err
	}

	defer unregister()

	// a topic has no end: this returns on client cancel, a failed send, idle
	// hangup or shutdown
	<-ctx.Done()

	return nil
}

// Reset makes subscribers without a cursor start after req.Cursor, or the topic's
// latest message. Subscriptions already open, and any with a cursor, are unaffected.
func (s *ServiceImpl) Reset(ctx context.Context, req *contracts.ResetStreamTopicRequest) (*contracts.ResetStreamTopicResponse, error) {
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

	var cursor *v1.StreamCursor

	if req.Cursor != nil && *req.Cursor != "" {
		decoded, err := s.resolveCursor(ctx, tenantId, req.Namespace, req.Topic, *req.Cursor)

		if err != nil {
			return nil, err
		}

		cursor = &decoded
	}

	if err := s.repo.Streams().ResetTopic(ctx, tenantId, req.Namespace, req.Topic, cursor); err != nil {
		return nil, err
	}

	return &contracts.ResetStreamTopicResponse{}, nil
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

// resolveCursor decodes raw, a client's cursor for namespace and topic, and checks
// it's still retained; an empty topic takes the cursor's own address. Errors are
// connect errors.
func (s *ServiceImpl) resolveCursor(ctx context.Context, tenantId uuid.UUID, namespace, topic, raw string) (v1.StreamCursor, error) {
	cursor, err := v1.DecodeStreamCursor(raw)

	if err != nil {
		return v1.StreamCursor{}, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if topic == "" {
		// a cursor is client-supplied, so the address decoded from it is checked too
		if err := v1.ValidateStreamAddress(cursor.Namespace, cursor.Topic); err != nil {
			return v1.StreamCursor{}, connect.NewError(connect.CodeInvalidArgument, err)
		}
	} else if cursor.Namespace != namespace || cursor.Topic != topic {
		return v1.StreamCursor{}, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf(
			"cursor belongs to namespace %q topic %q, not namespace %q topic %q",
			cursor.Namespace, cursor.Topic, namespace, topic,
		))
	}

	err = s.repo.Streams().CheckCursorRetained(ctx, tenantId, cursor)

	var expired *v1.StreamCursorExpiredError

	if errors.As(err, &expired) {
		return v1.StreamCursor{}, connect.NewError(connect.CodeOutOfRange, expired)
	}

	if err != nil {
		return v1.StreamCursor{}, err
	}

	return cursor, nil
}
