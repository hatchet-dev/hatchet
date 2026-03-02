package repository

import (
	"context"
	"time"

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
	TenantID uuid.UUID `validate:"required"`
	Spans    []*SpanData
}

type OtelSpanRow struct {
	TraceID            string
	SpanID             string
	ParentSpanID       string
	SpanName           string
	SpanKind           string
	ServiceName        string
	StatusCode         string
	StatusMessage      string
	Duration           uint64
	CreatedAt          time.Time
	ResourceAttributes map[string]string
	SpanAttributes     map[string]string
	ScopeName          string
	ScopeVersion       string
}

type OTelCollectorRepository interface {
	CreateSpans(ctx context.Context, tenantId uuid.UUID, opts *CreateSpansOpts) error
	ListSpansByWorkflowRunID(ctx context.Context, tenantId, workflowRunExternalId uuid.UUID) ([]*OtelSpanRow, error)
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

func (o *otelCollectorRepositoryImpl) ListSpansByWorkflowRunID(ctx context.Context, tenantId, workflowRunExternalId uuid.UUID) ([]*OtelSpanRow, error) {
	// intentional no-op, intended to be overridden
	return nil, nil
}
