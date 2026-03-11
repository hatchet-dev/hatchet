package repository

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type SpanData struct {
	TenantID             uuid.UUID
	WorkflowRunID        *uuid.UUID
	TaskRunExternalID    *uuid.UUID
	StatusMessage        string
	InstrumentationScope string
	Name                 string
	ResourceAttributes   json.RawMessage
	Attributes           json.RawMessage
	Events               json.RawMessage
	Links                json.RawMessage
	TraceID              []byte
	ParentSpanID         []byte
	SpanID               []byte
	EndTimeUnixNano      uint64
	StartTimeUnixNano    uint64
	StatusCode           tracev1.Status_StatusCode
	Kind                 tracev1.Span_SpanKind
}

type CreateSpansOpts struct {
	TenantID uuid.UUID `validate:"required"`
	Spans    []*SpanData
}

type ListSpansResult struct {
	Rows  []*sqlcv1.ListSpansByTaskExternalIDRow
	Total int64
}

type OTelCollectorRepository interface {
	CreateSpans(ctx context.Context, tenantId uuid.UUID, opts *CreateSpansOpts) error
	ListSpansByTaskExternalID(ctx context.Context, tenantId, taskExternalID uuid.UUID, offset, limit int64) (*ListSpansResult, error)
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
	if opts == nil {
		return fmt.Errorf("opts cannot be nil")
	}

	if err := o.v.Validate(opts); err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	if len(opts.Spans) == 0 {
		return nil
	}

	params := make([]sqlcv1.InsertOtelSpansParams, len(opts.Spans))
	for i, sd := range opts.Spans {
		var parentSpanID string
		if len(sd.ParentSpanID) > 0 {
			parentSpanID = hex.EncodeToString(sd.ParentSpanID)
		}

		resourceAttrs := []byte(sd.ResourceAttributes)
		if len(resourceAttrs) == 0 {
			resourceAttrs = []byte("{}")
		}

		spanAttrs := []byte(sd.Attributes)
		if len(spanAttrs) == 0 {
			spanAttrs = []byte("{}")
		}

		var taskRunExternalID *uuid.UUID
		if sd.TaskRunExternalID != nil && *sd.TaskRunExternalID != uuid.Nil {
			id := *sd.TaskRunExternalID
			taskRunExternalID = &id
		}

		var workflowRunExternalID *uuid.UUID
		if sd.WorkflowRunID != nil && *sd.WorkflowRunID != uuid.Nil {
			id := *sd.WorkflowRunID
			workflowRunExternalID = &id
		}

		startTime := time.Unix(0, int64(sd.StartTimeUnixNano)) //nolint:gosec

		params[i] = sqlcv1.InsertOtelSpansParams{
			TenantID:              tenantId,
			TraceID:               hex.EncodeToString(sd.TraceID),
			SpanID:                hex.EncodeToString(sd.SpanID),
			ParentSpanID:          parentSpanID,
			SpanName:              sd.Name,
			SpanKind:              protoSpanKindToDB(sd.Kind),
			ServiceName:           extractServiceName(sd.ResourceAttributes),
			StatusCode:            protoStatusCodeToDB(sd.StatusCode),
			StatusMessage:         sd.StatusMessage,
			DurationNs:            int64(sd.EndTimeUnixNano - sd.StartTimeUnixNano), //nolint:gosec
			ResourceAttributes:    resourceAttrs,
			SpanAttributes:        spanAttrs,
			ScopeName:             sd.InstrumentationScope,
			TaskRunExternalID:     taskRunExternalID,
			WorkflowRunExternalID: workflowRunExternalID,
			StartTime:             pgtype.Timestamptz{Time: startTime, Valid: true},
		}
	}

	_, err := o.queries.InsertOtelSpans(ctx, o.pool, params)
	if err != nil {
		return fmt.Errorf("error inserting otel spans: %w", err)
	}

	return nil
}

func (o *otelCollectorRepositoryImpl) ListSpansByTaskExternalID(ctx context.Context, tenantId, taskExternalID uuid.UUID, offset, limit int64) (*ListSpansResult, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, o.pool, o.l)
	if err != nil {
		return nil, fmt.Errorf("error starting transaction: %w", err)
	}
	defer rollback()

	total, err := o.queries.CountSpansByTaskExternalID(ctx, tx, sqlcv1.CountSpansByTaskExternalIDParams{
		Tenantid:       tenantId,
		Taskexternalid: taskExternalID,
	})
	if err != nil {
		return nil, fmt.Errorf("error counting otel spans: %w", err)
	}

	rows, err := o.queries.ListSpansByTaskExternalID(ctx, tx, sqlcv1.ListSpansByTaskExternalIDParams{
		Tenantid:       tenantId,
		Taskexternalid: taskExternalID,
		Spanoffset:     offset,
		Spanlimit:      limit,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing otel spans: %w", err)
	}

	if err := commit(ctx); err != nil {
		return nil, fmt.Errorf("error committing transaction: %w", err)
	}

	return &ListSpansResult{Rows: rows, Total: total}, nil
}

func extractServiceName(resourceAttrsJSON json.RawMessage) string {
	if len(resourceAttrsJSON) == 0 {
		return "unknown"
	}

	var attrs map[string]interface{}
	if err := json.Unmarshal(resourceAttrsJSON, &attrs); err != nil {
		return "unknown"
	}

	if serviceName, ok := attrs["service.name"].(string); ok {
		return serviceName
	}

	return "unknown"
}

func protoSpanKindToDB(kind tracev1.Span_SpanKind) sqlcv1.V1OtelSpanKind {
	switch kind {
	case tracev1.Span_SPAN_KIND_INTERNAL:
		return sqlcv1.V1OtelSpanKindINTERNAL
	case tracev1.Span_SPAN_KIND_SERVER:
		return sqlcv1.V1OtelSpanKindSERVER
	case tracev1.Span_SPAN_KIND_CLIENT:
		return sqlcv1.V1OtelSpanKindCLIENT
	case tracev1.Span_SPAN_KIND_PRODUCER:
		return sqlcv1.V1OtelSpanKindPRODUCER
	case tracev1.Span_SPAN_KIND_CONSUMER:
		return sqlcv1.V1OtelSpanKindCONSUMER
	default:
		return sqlcv1.V1OtelSpanKindUNSPECIFIED
	}
}

func protoStatusCodeToDB(code tracev1.Status_StatusCode) sqlcv1.V1OtelStatusCode {
	switch code {
	case tracev1.Status_STATUS_CODE_OK:
		return sqlcv1.V1OtelStatusCodeOK
	case tracev1.Status_STATUS_CODE_ERROR:
		return sqlcv1.V1OtelStatusCodeERROR
	default:
		return sqlcv1.V1OtelStatusCodeUNSET
	}
}
