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

	TaskExternalID *pgtype.UUID // hatchet.task_external_id
	WorkflowRunID  *pgtype.UUID // hatchet.workflow_run_id
	TenantID       pgtype.UUID  // hatchet.tenant_id

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
