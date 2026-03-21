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
	RetryCount           int32
	StatusCode           tracev1.Status_StatusCode
	Kind                 tracev1.Span_SpanKind
}

type CreateSpansOpts struct {
	TenantID uuid.UUID `validate:"required"`
	Spans    []*SpanData
}

type OtelSpanRow struct {
	StartTime          pgtype.Timestamptz
	SpanName           string
	TraceID            string
	SpanKind           sqlcv1.V1OtelSpanKind
	ServiceName        string
	StatusCode         sqlcv1.V1OtelStatusCode
	SpanID             string
	ParentSpanID       pgtype.Text
	StatusMessage      pgtype.Text
	ResourceAttributes []byte
	SpanAttributes     []byte
	ScopeName          pgtype.Text
	ScopeVersion       pgtype.Text
	DurationNs         int64
	RetryCount         int32
}

type ListSpansResult struct {
	Rows  []*OtelSpanRow
	Total int64
}

type OTelCollectorRepository interface {
	CreateSpans(ctx context.Context, tenantId uuid.UUID, opts *CreateSpansOpts) error
	ListSpansByRunExternalID(ctx context.Context, tenantId uuid.UUID, taskRunExternalId, workflowRunExternalID *uuid.UUID, offset, limit int64) (*ListSpansResult, error)
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

	tenantIds := make([]uuid.UUID, len(opts.Spans))
	traceIds := make([][]byte, len(opts.Spans))
	spanIds := make([][]byte, len(opts.Spans))
	parentSpanIds := make([]string, len(opts.Spans))
	spanNames := make([]string, len(opts.Spans))
	spanKinds := make([]string, len(opts.Spans))
	serviceNames := make([]string, len(opts.Spans))
	statusCodes := make([]string, len(opts.Spans))
	statusMessages := make([]string, len(opts.Spans))
	durations := make([]int64, len(opts.Spans))
	resourceAttrs := make([][]byte, len(opts.Spans))
	spanAttrs := make([][]byte, len(opts.Spans))
	scopeNames := make([]string, len(opts.Spans))
	scopeVersions := make([]string, len(opts.Spans))
	taskRunExternalIDs := make([]uuid.UUID, len(opts.Spans))
	workflowRunExternalIDs := make([]uuid.UUID, len(opts.Spans))
	retryCounts := make([]int32, len(opts.Spans))
	startTimes := make([]pgtype.Timestamptz, len(opts.Spans))

	for i, sd := range opts.Spans {
		var parentSpanID string
		if len(sd.ParentSpanID) > 0 {
			parentSpanID = hex.EncodeToString(sd.ParentSpanID)
		}

		resourceAttr := []byte(sd.ResourceAttributes)
		if len(resourceAttr) == 0 {
			resourceAttr = []byte("{}")
		}

		spanAttr := []byte(sd.Attributes)
		if len(spanAttr) == 0 {
			spanAttr = []byte("{}")
		}

		var taskRunExternalID uuid.UUID
		if sd.TaskRunExternalID != nil && *sd.TaskRunExternalID != uuid.Nil {
			taskRunExternalID = *sd.TaskRunExternalID
		}

		var workflowRunExternalID uuid.UUID
		if sd.WorkflowRunID != nil && *sd.WorkflowRunID != uuid.Nil {
			workflowRunExternalID = *sd.WorkflowRunID
		}

		startTime := time.Unix(0, int64(sd.StartTimeUnixNano)) //nolint:gosec

		tenantIds[i] = tenantId
		traceIds[i] = sd.TraceID
		spanIds[i] = sd.SpanID
		parentSpanIds[i] = parentSpanID
		spanNames[i] = sd.Name
		spanKinds[i] = string(protoSpanKindToDB(sd.Kind))
		serviceNames[i] = extractServiceName(resourceAttr)
		statusCodes[i] = string(protoStatusCodeToDB(sd.StatusCode))
		statusMessages[i] = sd.StatusMessage
		durations[i] = int64(sd.EndTimeUnixNano - sd.StartTimeUnixNano)
		resourceAttrs[i] = resourceAttr
		spanAttrs[i] = spanAttr
		scopeNames[i] = sd.InstrumentationScope
		scopeVersions[i] = sd.InstrumentationScope
		taskRunExternalIDs[i] = taskRunExternalID
		workflowRunExternalIDs[i] = workflowRunExternalID
		retryCounts[i] = sd.RetryCount
		startTimes[i] = pgtype.Timestamptz{Time: startTime, Valid: true}
	}

	err := o.queries.InsertOtelSpans(ctx, o.pool, sqlcv1.InsertOtelSpansParams{
		Tenantids:              tenantIds,
		Traceids:               traceIds,
		Spanids:                spanIds,
		Parentspanids:          parentSpanIds,
		Spannames:              spanNames,
		Spankinds:              spanKinds,
		Servicenames:           serviceNames,
		Statuscodes:            statusCodes,
		Statusmessages:         statusMessages,
		Durationnss:            durations,
		Resourceattributes:     resourceAttrs,
		Spanattributes:         spanAttrs,
		Scopenames:             scopeNames,
		Scopeversions:          scopeVersions,
		Taskrunexternalids:     taskRunExternalIDs,
		Workflowrunexternalids: workflowRunExternalIDs,
		Retrycounts:            retryCounts,
		Starttimes:             startTimes,
	})

	if err != nil {
		return fmt.Errorf("error inserting otel spans: %w", err)
	}

	lookupTenantIds := make([]uuid.UUID, 0)
	lookupExternalIds := make([]uuid.UUID, 0)
	lookupRetryCounts := make([]int32, 0)
	lookupTraceIds := make([][]byte, 0)
	lookupStartTimes := make([]pgtype.Timestamptz, 0)

	for _, span := range opts.Spans {
		if span.TaskRunExternalID != nil {
			lookupTenantIds = append(lookupTenantIds, tenantId)
			lookupRetryCounts = append(lookupRetryCounts, span.RetryCount)
			lookupTraceIds = append(lookupTraceIds, span.TraceID)
			lookupStartTimes = append(lookupStartTimes, pgtype.Timestamptz{Time: time.Unix(0, int64(span.StartTimeUnixNano)), Valid: true})
			lookupExternalIds = append(lookupExternalIds, *span.TaskRunExternalID)
		}

		if span.WorkflowRunID != nil && span.TaskRunExternalID != nil && *span.TaskRunExternalID != *span.WorkflowRunID {
			// if both the task run and workflow run external ids are present and they're not the same, then we know
			// the task must be part of a DAG, so we should insert a lookup entry for the DAG itself in addition to the task run
			lookupTenantIds = append(lookupTenantIds, tenantId)
			lookupRetryCounts = append(lookupRetryCounts, span.RetryCount)
			lookupTraceIds = append(lookupTraceIds, span.TraceID)
			lookupStartTimes = append(lookupStartTimes, pgtype.Timestamptz{Time: time.Unix(0, int64(span.StartTimeUnixNano)), Valid: true})
			lookupExternalIds = append(lookupExternalIds, *span.WorkflowRunID)
		}
	}

	return o.queries.InsertOTelTraceLookup(ctx, o.pool, sqlcv1.InsertOTelTraceLookupParams{
		Tenantids:   lookupTenantIds,
		Externalids: lookupExternalIds,
		Retrycounts: lookupRetryCounts,
		Traceids:    lookupTraceIds,
		Starttimes:  lookupStartTimes,
	})
}

func (o *otelCollectorRepositoryImpl) ListSpansByRunExternalID(ctx context.Context, tenantId uuid.UUID, taskRunExternalID, workflowRunExternalId *uuid.UUID, offset, limit int64) (*ListSpansResult, error) {
	rows, err := o.queries.ListSpansByExternalID(ctx, o.pool, sqlcv1.ListSpansByExternalIDParams{
		Tenantid:              tenantId,
		TaskRunExternalId:     taskRunExternalID,
		WorkflowRunExternalId: workflowRunExternalId,
		Spanoffset:            offset,
		Spanlimit:             limit,
	})

	if err != nil {
		return nil, fmt.Errorf("error listing otel spans: %w", err)
	}

	allRows := make([]*OtelSpanRow, 0, len(rows))

	for _, r := range rows {
		allRows = append(allRows, &OtelSpanRow{
			TraceID:            hex.EncodeToString(r.TraceID),
			SpanID:             hex.EncodeToString(r.SpanID),
			ParentSpanID:       r.ParentSpanID,
			SpanName:           r.SpanName,
			SpanKind:           r.SpanKind,
			ServiceName:        r.ServiceName,
			StatusCode:         r.StatusCode,
			StatusMessage:      r.StatusMessage,
			DurationNs:         r.DurationNs,
			StartTime:          r.StartTime,
			ResourceAttributes: r.ResourceAttributes,
			SpanAttributes:     r.SpanAttributes,
			ScopeName:          r.ScopeName,
			ScopeVersion:       r.ScopeVersion,
			RetryCount:         r.RetryCount,
		})
	}

	return &ListSpansResult{Rows: allRows, Total: int64(len(allRows))}, nil
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
