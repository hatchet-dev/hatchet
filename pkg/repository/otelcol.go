package repository

import (
	"context"

	"github.com/google/uuid"
)

type SpanData struct {
	WorkflowRunID        *uuid.UUID
	TaskRunExternalID    *uuid.UUID
	StatusMessage        string
	InstrumentationScope string
	Name                 string
	ResourceAttributes   []byte
	Attributes           []byte
	Events               []byte
	Links                []byte
	TraceID              []byte
	ParentSpanID         []byte
	SpanID               []byte
	EndTimeUnixNano      uint64
	StartTimeUnixNano    uint64
	StatusCode           int32
	Kind                 int32
	TenantID             uuid.UUID
}

type CreateSpansOpts struct {
	Spans    []*SpanData
	TenantID uuid.UUID `validate:"required"`
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
