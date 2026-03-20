package repository

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
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
	ListSpansByTaskExternalID(ctx context.Context, tenantId, taskExternalID uuid.UUID, offset, limit int64) (*ListSpansResult, error)
	ListSpansByWorkflowRunExternalID(ctx context.Context, tenantId, workflowRunExternalID uuid.UUID, offset, limit int64) (*ListSpansResult, error)
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
			ParentSpanID:          pgtype.Text{String: parentSpanID, Valid: parentSpanID != ""},
			SpanName:              sd.Name,
			SpanKind:              protoSpanKindToDB(sd.Kind),
			ServiceName:           extractServiceName(sd.ResourceAttributes),
			StatusCode:            protoStatusCodeToDB(sd.StatusCode),
			StatusMessage:         pgtype.Text{String: sd.StatusMessage, Valid: sd.StatusMessage != ""},
			DurationNs:            int64(sd.EndTimeUnixNano - sd.StartTimeUnixNano), //nolint:gosec
			ResourceAttributes:    resourceAttrs,
			SpanAttributes:        spanAttrs,
			ScopeName:             pgtype.Text{String: sd.InstrumentationScope, Valid: sd.InstrumentationScope != ""},
			TaskRunExternalID:     taskRunExternalID,
			WorkflowRunExternalID: workflowRunExternalID,
			RetryCount:            sd.RetryCount,
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
	rows, err := o.queries.ListSpansByTaskExternalID(ctx, o.pool, sqlcv1.ListSpansByTaskExternalIDParams{
		Tenantid:       tenantId,
		Taskexternalid: taskExternalID,
		Spanoffset:     0,
		Spanlimit:      10000,
	})

	if err != nil {
		return nil, fmt.Errorf("error listing otel spans: %w", err)
	}

	var allRows []*OtelSpanRow
	var childWorkflowRunIDs []uuid.UUID
	seenSpanIDs := make(map[string]bool)

	for _, r := range rows {
		seenSpanIDs[r.SpanID] = true
		allRows = append(allRows, &OtelSpanRow{
			TraceID: r.TraceID, SpanID: r.SpanID, ParentSpanID: r.ParentSpanID,
			SpanName: r.SpanName, SpanKind: r.SpanKind, ServiceName: r.ServiceName,
			StatusCode: r.StatusCode, StatusMessage: r.StatusMessage, DurationNs: r.DurationNs,
			StartTime: r.StartTime, ResourceAttributes: r.ResourceAttributes,
			SpanAttributes: r.SpanAttributes, ScopeName: r.ScopeName, ScopeVersion: r.ScopeVersion,
			RetryCount: r.RetryCount,
		})

		childID := ExtractChildWorkflowRunID(r.SpanName, r.SpanAttributes)
		if childID != uuid.Nil {
			childWorkflowRunIDs = append(childWorkflowRunIDs, childID)
		}
	}

	total := int64(len(allRows))

	if offset >= total {
		return &ListSpansResult{Rows: nil, Total: total}, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return &ListSpansResult{Rows: allRows[offset:end], Total: total}, nil
}

func (o *otelCollectorRepositoryImpl) ListSpansByWorkflowRunExternalID(ctx context.Context, tenantId, workflowRunExternalID uuid.UUID, offset, limit int64) (*ListSpansResult, error) {
	// Fetch spans for the requested workflow run and all child workflow runs.
	// Instead of a slow recursive SQL CTE, we iteratively discover child workflow
	// run IDs from span attributes (hatchet.workflow_run_id on trigger spans).
	allRows, err := o.listSpansForWorkflowRunTree(ctx, tenantId, workflowRunExternalID, listSpansOpts{includeTraceIDLookup: true})
	if err != nil {
		return nil, err
	}

	total := int64(len(allRows))

	// Apply offset/limit in Go
	if offset >= total {
		return &ListSpansResult{Rows: nil, Total: total}, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}

	return &ListSpansResult{Rows: allRows[offset:end], Total: total}, nil
}

type listSpansOpts struct {
	includeTraceIDLookup bool
}

// listSpansForWorkflowRunTree fetches spans for a workflow run and all child
// workflow runs, discovering children by inspecting span attributes.
func (o *otelCollectorRepositoryImpl) listSpansForWorkflowRunTree(ctx context.Context, tenantId uuid.UUID, rootWorkflowRunID uuid.UUID, opts ...listSpansOpts) ([]*OtelSpanRow, error) {
	cfg := listSpansOpts{}
	if len(opts) > 0 {
		cfg = opts[0]
	}

	var allRows []*OtelSpanRow
	seenSpanIDs := make(map[string]bool)

	// BFS through workflow run tree
	queue := []uuid.UUID{rootWorkflowRunID}
	visited := map[uuid.UUID]bool{rootWorkflowRunID: true}

	for len(queue) > 0 {
		wfRunID := queue[0]
		queue = queue[1:]

		rows, err := o.queries.ListSpansByWorkflowRunExternalID(ctx, o.pool, sqlcv1.ListSpansByWorkflowRunExternalIDParams{
			Tenantid:              tenantId,
			Workflowrunexternalid: wfRunID,
			Spanoffset:            0,
			Spanlimit:             10000,
		})
		if err != nil {
			return nil, fmt.Errorf("error listing otel spans for workflow run %s: %w", wfRunID, err)
		}

		for _, r := range rows {
			if seenSpanIDs[r.SpanID] {
				continue
			}
			seenSpanIDs[r.SpanID] = true
			allRows = append(allRows, &OtelSpanRow{
				TraceID: r.TraceID, SpanID: r.SpanID, ParentSpanID: r.ParentSpanID,
				SpanName: r.SpanName, SpanKind: r.SpanKind, ServiceName: r.ServiceName,
				StatusCode: r.StatusCode, StatusMessage: r.StatusMessage, DurationNs: r.DurationNs,
				StartTime: r.StartTime, ResourceAttributes: r.ResourceAttributes,
				SpanAttributes: r.SpanAttributes, ScopeName: r.ScopeName, ScopeVersion: r.ScopeVersion,
				RetryCount: r.RetryCount,
			})

			// Check span attributes for child workflow run IDs
			childID := ExtractChildWorkflowRunID(r.SpanName, r.SpanAttributes)
			if childID != uuid.Nil && !visited[childID] {
				visited[childID] = true
				queue = append(queue, childID)
			}
		}
	}

	if !cfg.includeTraceIDLookup {
		return allRows, nil
	}

	// Second pass: fetch parent/trigger spans by trace_id.
	// Trigger spans (hatchet.run_workflow, hatchet.push_event) don't have
	// workflow_run_external_id set, so they're missed by the first query.
	traceIDs := make(map[string]bool)
	for _, row := range allRows {
		traceIDs[row.TraceID] = true
	}

	if len(traceIDs) > 0 {
		traceIDList := make([]string, 0, len(traceIDs))
		for tid := range traceIDs {
			traceIDList = append(traceIDList, tid)
		}

		traceRows, err := o.queries.ListSpansByTraceIDs(ctx, o.pool, sqlcv1.ListSpansByTraceIDsParams{
			Tenantid: tenantId,
			Traceids: traceIDList,
		})
		if err != nil {
			return nil, fmt.Errorf("error listing otel spans by trace_id: %w", err)
		}

		for _, r := range traceRows {
			if seenSpanIDs[r.SpanID] {
				continue
			}
			seenSpanIDs[r.SpanID] = true
			allRows = append(allRows, &OtelSpanRow{
				TraceID: r.TraceID, SpanID: r.SpanID, ParentSpanID: r.ParentSpanID,
				SpanName: r.SpanName, SpanKind: r.SpanKind, ServiceName: r.ServiceName,
				StatusCode: r.StatusCode, StatusMessage: r.StatusMessage, DurationNs: r.DurationNs,
				StartTime: r.StartTime, ResourceAttributes: r.ResourceAttributes,
				SpanAttributes: r.SpanAttributes, ScopeName: r.ScopeName, ScopeVersion: r.ScopeVersion,
				RetryCount: r.RetryCount,
			})
		}
	}

	return allRows, nil
}

// extractChildWorkflowRunID parses span attributes JSONB to find a child
// workflow_run_id, but only from trigger spans. Non-trigger spans (like
// hatchet.start_step_run) also carry hatchet.workflow_run_id but it refers
// to their OWN workflow run, not a child.
func ExtractChildWorkflowRunID(spanName string, attrs []byte) uuid.UUID {
	if len(attrs) == 0 {
		return uuid.Nil
	}

	// Only trigger/producer spans point to child workflow runs
	if !strings.HasPrefix(spanName, "hatchet.trigger") &&
		!strings.HasPrefix(spanName, "hatchet.run_workflow") {
		return uuid.Nil
	}

	var m map[string]interface{}
	if err := json.Unmarshal(attrs, &m); err != nil {
		return uuid.Nil
	}

	// Prefer the explicit child attribute; fall back to legacy attribute name
	val, ok := m["hatchet.child_workflow_run_id"]
	if !ok {
		val, ok = m["hatchet.workflow_run_id"]
		if !ok {
			return uuid.Nil
		}
	}

	str, ok := val.(string)
	if !ok {
		return uuid.Nil
	}

	id, err := uuid.Parse(str)
	if err != nil {
		return uuid.Nil
	}

	return id
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
