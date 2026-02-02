package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type SpanData struct {
	WorkflowRunID        *pgtype.UUID
	TaskRunExternalID    *pgtype.UUID
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
	TenantID             pgtype.UUID
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
