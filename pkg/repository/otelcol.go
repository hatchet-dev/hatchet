package repository

import (
	"context"

	"github.com/google/uuid"
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

	TaskRunExternalID *uuid.UUID // from hatchet.step_run_id attribute
	WorkflowRunID     *uuid.UUID // from hatchet.workflow_run_id attribute
	TenantID          uuid.UUID  // from auth context

	InstrumentationScope string
}

type CreateSpansOpts struct {
	TenantID uuid.UUID `validate:"required"`
	Spans    []*SpanData
}

type OTelCollectorRepository interface {
	CreateSpans(ctx context.Context, tenantId uuid.UUID, opts *CreateSpansOpts) error
}

type otelCollectorRepositoryImpl struct {
	*sharedRepository
}

func newOTelCollectorRepository(s *sharedRepository) OTelCollectorRepository {
	return &otelCollectorRepositoryImpl{
		sharedRepository: s,
	}
}

func (o *otelCollectorRepositoryImpl) CreateSpans(ctx context.Context, tenantId uuid.UUID, opts *CreateSpansOpts) error {
	// intentional no-op, intended to be overridden
	return nil
}
