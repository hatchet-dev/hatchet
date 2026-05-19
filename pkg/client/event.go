// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	eventcontracts "github.com/hatchet-dev/hatchet/internal/services/ingestor/contracts"
	"github.com/hatchet-dev/hatchet/pkg/config/client"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

// SourceInfo carries the workflow/step run context so the event client can
// inject hatchet__source_* keys into event metadata for cross-workflow tracing.
type SourceInfo struct {
	WorkflowRunID string
	StepRunID     string
}

type sourceInfoKeyType struct{}

var sourceInfoKey = sourceInfoKeyType{}

// WithSourceInfo stores source workflow/step run IDs in the context.
// Called by the opentelemetry middleware so Push/BulkPush can propagate them.
func WithSourceInfo(ctx context.Context, info SourceInfo) context.Context {
	return context.WithValue(ctx, sourceInfoKey, info)
}

// hatchetContextProvider lets us extract source IDs directly from a
// HatchetContext without importing pkg/worker (avoids import cycle).
type hatchetContextProvider interface {
	WorkflowRunId() string
	StepRunId() string
}

func getSourceInfo(ctx context.Context) (SourceInfo, bool) {
	if info, ok := ctx.Value(sourceInfoKey).(SourceInfo); ok {
		return info, true
	}

	if hCtx, ok := ctx.(hatchetContextProvider); ok {
		wfID := hCtx.WorkflowRunId()
		srID := hCtx.StepRunId()
		if wfID != "" || srID != "" {
			return SourceInfo{WorkflowRunID: wfID, StepRunID: srID}, true
		}
	}

	return SourceInfo{}, false
}

type pushOpt struct {
	additionalMetadata map[string]string
	priority           *int32
	scope              *string
}

type PushOpFunc func(*pushOpt) error

type BulkPushOpFunc func(*eventcontracts.BulkPushEventRequest) error

type streamEventOpts struct {
	index *int64
}

type StreamEventOption func(*streamEventOpts)

func WithStreamEventIndex(index int64) StreamEventOption {
	return func(opts *streamEventOpts) {
		opts.index = &index
	}
}

type EventClient interface {
	Push(ctx context.Context, eventKey string, payload interface{}, options ...PushOpFunc) error

	BulkPush(ctx context.Context, payloads []EventWithAdditionalMetadata, options ...BulkPushOpFunc) error

	PutLog(ctx context.Context, taskRunId, msg string, level *string, taskRetryCount *int32) error

	PutLogWithTimestamp(ctx context.Context, taskRunId, msg string, level *string, taskRetryCount *int32, createdAt *timestamppb.Timestamp) error

	PutStreamEvent(ctx context.Context, stepRunId string, message []byte, options ...StreamEventOption) error
}

type EventWithAdditionalMetadata struct {
	Event              interface{}       `json:"event"`
	AdditionalMetadata map[string]string `json:"metadata"`
	Key                string            `json:"key"`
	Priority           *int32            `json:"priority"`
	Scope              *string           `json:"scope"`
}

type eventClientImpl struct {
	client eventcontracts.EventsServiceClient

	tenantId string

	namespace string

	l *zerolog.Logger

	v validator.Validator

	ctx *contextLoader

	sharedMeta map[string]string
}

func newEvent(conn *grpc.ClientConn, opts *sharedClientOpts) EventClient {
	return &eventClientImpl{
		client:     eventcontracts.NewEventsServiceClient(conn),
		tenantId:   opts.tenantId,
		namespace:  opts.namespace,
		l:          opts.l,
		v:          opts.v,
		ctx:        opts.ctxLoader,
		sharedMeta: opts.sharedMeta,
	}
}

func WithEventMetadata(metadata map[string]string) PushOpFunc {
	return func(r *pushOpt) error {
		r.additionalMetadata = metadata

		return nil
	}
}

func WithEventPriority(priority *int32) PushOpFunc {
	return func(r *pushOpt) error {
		r.priority = priority
		return nil
	}
}

func WithFilterScope(scope *string) PushOpFunc {
	return func(r *pushOpt) error {
		r.scope = scope
		return nil
	}
}

func (a *eventClientImpl) Push(ctx context.Context, eventKey string, payload interface{}, options ...PushOpFunc) error {
	sourceInfo, _ := getSourceInfo(ctx)

	tracer := otel.Tracer("github.com/hatchet-dev/hatchet/pkg/client")
	ctx, span := tracer.Start(ctx, "hatchet.push_event",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("instrumentor", "hatchet"),
			attribute.String("hatchet.event_key", eventKey),
		),
	)
	defer span.End()

	key := client.ApplyNamespace(eventKey, &a.namespace)

	request := eventcontracts.PushEventRequest{
		Key:            key,
		EventTimestamp: timestamppb.Now(),
	}

	payloadBytes, err := json.Marshal(payload)

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	request.Payload = string(payloadBytes)

	opts := &pushOpt{}

	for _, optionFunc := range options {
		err = optionFunc(opts)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}

	if opts.additionalMetadata == nil {
		opts.additionalMetadata = make(map[string]string)
	}
	injectTraceContext(ctx, opts.additionalMetadata, sourceInfo)

	additionalMetaBytes, err := a.getAdditionalMetaBytes(&opts.additionalMetadata)

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	additionalMetaString := string(additionalMetaBytes)

	request.AdditionalMetadata = &additionalMetaString
	request.Priority = opts.priority
	request.Scope = opts.scope

	_, err = a.client.Push(a.ctx.newContext(ctx), &request)

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

func (a *eventClientImpl) BulkPush(ctx context.Context, payload []EventWithAdditionalMetadata, options ...BulkPushOpFunc) error {
	sourceInfo, _ := getSourceInfo(ctx)

	tracer := otel.Tracer("github.com/hatchet-dev/hatchet/pkg/client")
	ctx, span := tracer.Start(ctx, "hatchet.bulk_push_event",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("instrumentor", "hatchet"),
			attribute.Int("hatchet.num_events", len(payload)),
		),
	)
	defer span.End()

	request := eventcontracts.BulkPushEventRequest{}

	var events []*eventcontracts.PushEventRequest

	for _, p := range payload {

		ePayload, err := json.Marshal(p.Event)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		md := p.AdditionalMetadata
		if md == nil {
			md = make(map[string]string)
		}
		injectTraceContext(ctx, md, sourceInfo)
		eMetadata, err := a.getAdditionalMetaBytes(&md)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		eMetadataString := string(eMetadata)

		events = append(events, &eventcontracts.PushEventRequest{
			Key:                a.namespace + p.Key,
			EventTimestamp:     timestamppb.Now(),
			Payload:            string(ePayload),
			AdditionalMetadata: &eMetadataString,
			Priority:           p.Priority,
			Scope:              p.Scope,
		})
	}

	request.Events = events

	_, err := a.client.BulkPush(a.ctx.newContext(ctx), &request)

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

func (a *eventClientImpl) PutLog(ctx context.Context, taskRunId, msg string, level *string, taskRetryCount *int32) error {
	return a.PutLogWithTimestamp(ctx, taskRunId, msg, level, taskRetryCount, timestamppb.Now())
}

func (a *eventClientImpl) PutLogWithTimestamp(ctx context.Context, taskRunId, msg string, level *string, taskRetryCount *int32, createdAt *timestamppb.Timestamp) error {
	_, err := a.client.PutLog(a.ctx.newContext(ctx), &eventcontracts.PutLogRequest{
		CreatedAt:         createdAt,
		TaskRunExternalId: taskRunId,
		Message:           msg,
		Level:             level,
		TaskRetryCount:    taskRetryCount,
	})

	return err
}

func (a *eventClientImpl) PutStreamEvent(ctx context.Context, taskRunId string, message []byte, options ...StreamEventOption) error {
	opts := &streamEventOpts{}

	for _, optionFunc := range options {
		optionFunc(opts)
	}

	request := &eventcontracts.PutStreamEventRequest{
		CreatedAt:         timestamppb.Now(),
		TaskRunExternalId: taskRunId,
		Message:           message,
	}

	if opts.index != nil {
		request.EventIndex = opts.index
	}

	_, err := a.client.PutStreamEvent(a.ctx.newContext(ctx), request)

	return err
}

// injectTraceContext adds traceparent and source workflow/step run IDs into the
// metadata map so triggered workflow runs inherit the trace and can be linked
// back to the emitting step.
func injectTraceContext(ctx context.Context, meta map[string]string, info SourceInfo) {
	carrier := propagation.MapCarrier(meta)
	propagation.TraceContext{}.Inject(ctx, carrier)

	if info.WorkflowRunID != "" {
		meta["hatchet__source_workflow_run_id"] = info.WorkflowRunID
	}
	if info.StepRunID != "" {
		meta["hatchet__source_step_run_id"] = info.StepRunID
	}
}

func (e *eventClientImpl) getAdditionalMetaBytes(opt *map[string]string) ([]byte, error) {
	additionalMeta := make(map[string]string)

	for key, value := range e.sharedMeta {
		additionalMeta[key] = value
	}

	if opt != nil {
		for key, value := range *opt {
			additionalMeta[key] = value
		}
	}

	metadataBytes, err := json.Marshal(additionalMeta)

	if err != nil {
		return nil, fmt.Errorf("could not marshal additional metadata: %w", err)
	}

	return metadataBytes, nil
}
