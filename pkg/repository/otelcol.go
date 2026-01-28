package repository

import (
	"context"

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

	TaskRunExternalID *pgtype.UUID // from hatchet.step_run_id attribute
	WorkflowRunID     *pgtype.UUID // from hatchet.workflow_run_id attribute
	TenantID          pgtype.UUID  // from auth context

	InstrumentationScope string
}

type CreateSpansOpts struct {
	TenantID string `validate:"required,uuid"`
	Spans    []*SpanData
}

type OTelCollectorRepository interface {
	CreateSpans(ctx context.Context, tenantId string, opts *CreateSpansOpts) error
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
	// intentional no-op, intended to be overridden
	return nil
}
