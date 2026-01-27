package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type SpanData struct {
	TraceID           []byte // 16 bytes
	SpanID            []byte // 8 bytes
	ParentSpanID      []byte // optional
	Name              string
	Kind              int32
	StartTimeUnixNano uint64
	EndTimeUnixNano   uint64
	StatusCode        int32
	StatusMessage     string

	Attributes         []byte
	Events             []byte
	Links              []byte
	ResourceAttributes []byte

	TaskRunExternalID *pgtype.UUID // from hatchet.task_run_external_id attribute
	WorkflowRunID     *pgtype.UUID // from hatchet.workflow_run_id attribute
	TenantID          pgtype.UUID  // from auth context

	InstrumentationScope string
}

type CreateSpansOpts struct {
	TenantID string `validate:"required,uuid"`
	Spans    []*SpanData
}

type ListSpansOpts struct {
	StartTime        *time.Time
	EndTime          *time.Time
	TaskExternalID   *string
	WorkflowRunID    *string
	TraceID          []byte
	Limit            *int `validate:"omitnil,min=1,max=10000"`
	Offset           *int
	OrderByDirection *string `validate:"omitempty,oneof=ASC DESC"`
}

type OTelCollectorRepository interface {
	CreateSpans(ctx context.Context, tenantId string, opts *CreateSpansOpts) error

	ListSpansByTask(ctx context.Context, tenantId, taskExternalId string, opts *ListSpansOpts) ([]*SpanData, error)

	ListSpansByWorkflowRun(ctx context.Context, tenantId, workflowRunId string, opts *ListSpansOpts) ([]*SpanData, error)

	ListSpansByTraceID(ctx context.Context, tenantId string, traceId []byte, opts *ListSpansOpts) ([]*SpanData, error)
}

type otelCollectorRepositoryImpl struct {
	*sharedRepository
}

func newOTelCollectorRepository(s *sharedRepository) OTelCollectorRepository {
	return &otelCollectorRepositoryImpl{
		sharedRepository: s,
	}
}

func (o *otelCollectorRepositoryImpl) CreateSpans(ctx context.Context, tenantId string, opts *CreateSpansOpts) error {
	if err := o.v.Validate(opts); err != nil {
		return err
	}

	// TODO: Implement CreateSpans to store spans in database
	// For now, just log that we received spans
	o.l.Debug().
		Int("span_count", len(opts.Spans)).
		Str("tenant_id", tenantId).
		Msg("received spans for storage (not yet implemented)")

	return nil
}

func (o *otelCollectorRepositoryImpl) ListSpansByTask(ctx context.Context, tenantId, taskExternalId string, opts *ListSpansOpts) ([]*SpanData, error) {
	// TODO: Implement ListSpansByTask
	return nil, nil
}

func (o *otelCollectorRepositoryImpl) ListSpansByWorkflowRun(ctx context.Context, tenantId, workflowRunId string, opts *ListSpansOpts) ([]*SpanData, error) {
	// TODO: Implement ListSpansByWorkflowRun
	return nil, nil
}

func (o *otelCollectorRepositoryImpl) ListSpansByTraceID(ctx context.Context, tenantId string, traceId []byte, opts *ListSpansOpts) ([]*SpanData, error) {
	// TODO: Implement ListSpansByTraceID
	return nil, nil
}
