package repository

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"
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
	CreatedAt          time.Time
	SpanAttributes     map[string]string
	ResourceAttributes map[string]string
	SpanName           string
	SpanKind           string
	ServiceName        string
	StatusCode         string
	StatusMessage      string
	TraceID            string
	ParentSpanID       string
	SpanID             string
	ScopeName          string
	ScopeVersion       string
	Duration           uint64
}

type OTelCollectorRepository interface {
	CreateSpans(ctx context.Context, tenantId uuid.UUID, opts *CreateSpansOpts) error
	ListSpansByTaskExternalID(ctx context.Context, tenantId, taskExternalID uuid.UUID) ([]*OtelSpanRow, error)
}

type otelCollectorRepositoryImpl struct {
	*sharedRepository
}

func newOTelCollectorRepository(s *sharedRepository) OTelCollectorRepository {
	return &otelCollectorRepositoryImpl{
		sharedRepository: s,
	}
}

// transformedSpan holds pre-processed span data ready for insertion.
type transformedSpan struct {
	startTime             time.Time
	taskRunExternalID     *uuid.UUID
	workflowRunExternalID *uuid.UUID
	statusMessage         string
	scopeVersion          string
	spanKind              string
	serviceName           string
	statusCode            string
	traceID               string
	spanID                string
	parentSpanID          string
	spanName              string
	scopeName             string
	spanAttributes        []byte
	resourceAttributes    []byte
	durationNs            int64
	tenantID              uuid.UUID
}

// spanCopyFromSource implements pgx.CopyFromSource for batch inserts.
type spanCopyFromSource struct {
	spans []transformedSpan
	idx   int
}

func (s *spanCopyFromSource) Next() bool {
	s.idx++
	return s.idx < len(s.spans)
}

func (s *spanCopyFromSource) Values() ([]interface{}, error) {
	span := s.spans[s.idx]
	return []interface{}{
		span.tenantID,
		span.traceID,
		span.spanID,
		span.parentSpanID,
		span.spanName,
		span.spanKind,
		span.serviceName,
		span.statusCode,
		span.statusMessage,
		span.durationNs,
		span.resourceAttributes,
		span.spanAttributes,
		span.scopeName,
		span.scopeVersion,
		span.taskRunExternalID,
		span.workflowRunExternalID,
		span.startTime,
	}, nil
}

func (s *spanCopyFromSource) Err() error {
	return nil
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

	transformed := make([]transformedSpan, 0, len(opts.Spans))
	for _, sd := range opts.Spans {
		ts := transformedSpan{
			tenantID:      tenantId,
			traceID:       hex.EncodeToString(sd.TraceID),
			spanID:        hex.EncodeToString(sd.SpanID),
			spanName:      sd.Name,
			spanKind:      spanKindToString(sd.Kind),
			serviceName:   extractServiceName(sd.ResourceAttributes),
			statusCode:    spanStatusCodeToString(sd.StatusCode),
			statusMessage: sd.StatusMessage,
			durationNs:    int64(sd.EndTimeUnixNano - sd.StartTimeUnixNano), //nolint:gosec
			scopeName:     sd.InstrumentationScope,
			startTime:     time.Unix(0, int64(sd.StartTimeUnixNano)), //nolint:gosec
		}

		if len(sd.ParentSpanID) > 0 {
			ts.parentSpanID = hex.EncodeToString(sd.ParentSpanID)
		}

		ts.resourceAttributes = jsonBytesToJSONB(sd.ResourceAttributes)
		ts.spanAttributes = jsonBytesToJSONB(sd.Attributes)

		if sd.TaskRunExternalID != nil && *sd.TaskRunExternalID != uuid.Nil {
			id := *sd.TaskRunExternalID
			ts.taskRunExternalID = &id
		}

		if sd.WorkflowRunID != nil && *sd.WorkflowRunID != uuid.Nil {
			id := *sd.WorkflowRunID
			ts.workflowRunExternalID = &id
		}

		transformed = append(transformed, ts)
	}

	_, err := o.pool.CopyFrom(
		ctx,
		pgx.Identifier{"v1_otel_traces"},
		[]string{
			"tenant_id", "trace_id", "span_id", "parent_span_id",
			"span_name", "span_kind", "service_name", "status_code",
			"status_message", "duration_ns", "resource_attributes",
			"span_attributes", "scope_name", "scope_version",
			"task_run_external_id", "workflow_run_external_id", "start_time",
		},
		&spanCopyFromSource{spans: transformed, idx: -1},
	)

	if err != nil {
		return fmt.Errorf("error copying spans to v1_otel_traces: %w", err)
	}

	return nil
}

func (o *otelCollectorRepositoryImpl) ListSpansByTaskExternalID(ctx context.Context, tenantId, taskExternalID uuid.UUID) ([]*OtelSpanRow, error) {
	query := `
		SELECT
			trace_id, span_id, parent_span_id, span_name, span_kind,
			service_name, status_code, status_message, duration_ns, start_time,
			resource_attributes, span_attributes, scope_name, scope_version
		FROM v1_otel_traces
		WHERE tenant_id = $1 AND trace_id IN (
			SELECT DISTINCT trace_id FROM v1_otel_traces
			WHERE tenant_id = $1 AND task_run_external_id = $2
		)
		ORDER BY start_time ASC
		LIMIT 1000
	`

	rows, err := o.pool.Query(ctx, query, tenantId, taskExternalID)
	if err != nil {
		return nil, fmt.Errorf("error querying v1_otel_traces: %w", err)
	}
	defer rows.Close()

	var result []*OtelSpanRow
	for rows.Next() {
		row := &OtelSpanRow{}
		var durationNs int64
		var resourceAttrsJSON, spanAttrsJSON []byte

		if err := rows.Scan(
			&row.TraceID,
			&row.SpanID,
			&row.ParentSpanID,
			&row.SpanName,
			&row.SpanKind,
			&row.ServiceName,
			&row.StatusCode,
			&row.StatusMessage,
			&durationNs,
			&row.CreatedAt,
			&resourceAttrsJSON,
			&spanAttrsJSON,
			&row.ScopeName,
			&row.ScopeVersion,
		); err != nil {
			return nil, fmt.Errorf("error scanning v1_otel_traces row: %w", err)
		}

		row.Duration = uint64(durationNs) //nolint:gosec
		row.ResourceAttributes = jsonbToStringMap(resourceAttrsJSON)
		row.SpanAttributes = jsonbToStringMap(spanAttrsJSON)

		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating v1_otel_traces rows: %w", err)
	}

	return result, nil
}

// Helper functions for data transformation

func extractServiceName(resourceAttrsJSON []byte) string {
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

func jsonBytesToJSONB(jsonBytes []byte) []byte {
	if len(jsonBytes) == 0 {
		return []byte("{}")
	}
	// Validate it's valid JSON; if not, return empty object
	var raw json.RawMessage
	if err := json.Unmarshal(jsonBytes, &raw); err != nil {
		return []byte("{}")
	}
	return jsonBytes
}

func jsonbToStringMap(jsonBytes []byte) map[string]string {
	if len(jsonBytes) == 0 {
		return make(map[string]string)
	}

	var jsonMap map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &jsonMap); err != nil {
		return make(map[string]string)
	}

	result := make(map[string]string, len(jsonMap))
	for k, v := range jsonMap {
		switch val := v.(type) {
		case string:
			result[k] = val
		case nil:
			result[k] = ""
		default:
			b, err := json.Marshal(v)
			if err != nil {
				result[k] = fmt.Sprintf("%v", v)
			} else {
				result[k] = string(b)
			}
		}
	}

	return result
}

func spanKindToString(kind int32) string {
	switch tracev1.Span_SpanKind(kind) {
	case tracev1.Span_SPAN_KIND_UNSPECIFIED:
		return "UNSPECIFIED"
	case tracev1.Span_SPAN_KIND_INTERNAL:
		return "INTERNAL"
	case tracev1.Span_SPAN_KIND_SERVER:
		return "SERVER"
	case tracev1.Span_SPAN_KIND_CLIENT:
		return "CLIENT"
	case tracev1.Span_SPAN_KIND_PRODUCER:
		return "PRODUCER"
	case tracev1.Span_SPAN_KIND_CONSUMER:
		return "CONSUMER"
	default:
		return "UNKNOWN"
	}
}

func spanStatusCodeToString(code int32) string {
	switch tracev1.Status_StatusCode(code) {
	case tracev1.Status_STATUS_CODE_UNSET:
		return "UNSET"
	case tracev1.Status_STATUS_CODE_OK:
		return "OK"
	case tracev1.Status_STATUS_CODE_ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}
