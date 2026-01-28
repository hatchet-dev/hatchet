package otelcol

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	collectortracev1 "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonv1 "go.opentelemetry.io/proto/otlp/common/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const (
	// keep these in sync with attributes sent from the sdks
	AttrHatchetTaskRunID     = "hatchet.step_run_id"     // Task run external ID from SDK
	AttrHatchetWorkflowRunID = "hatchet.workflow_run_id" // Workflow run ID from SDK
)

type otelCollectorImpl struct {
	collectortracev1.UnimplementedTraceServiceServer

	repo repository.Repository
	l    *zerolog.Logger
}

func (oc *otelCollectorImpl) Export(ctx context.Context, req *collectortracev1.ExportTraceServiceRequest) (*collectortracev1.ExportTraceServiceResponse, error) {
	tenant, ok := ctx.Value("tenant").(*sqlcv1.Tenant)
	if !ok {
		oc.l.Warn().Msg("no tenant in context for trace export")
		return &collectortracev1.ExportTraceServiceResponse{}, nil
	}

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	otelColRepo := oc.repo.OTelCollector()
	if otelColRepo == nil {
		oc.l.Debug().Msg("otel collector repository not configured, discarding spans")
		return &collectortracev1.ExportTraceServiceResponse{}, nil
	}

	spans := oc.convertOTLPToSpanData(req.GetResourceSpans(), tenant.ID)

	if len(spans) == 0 {
		return &collectortracev1.ExportTraceServiceResponse{}, nil
	}

	err := otelColRepo.CreateSpans(ctx, tenantId, &repository.CreateSpansOpts{
		TenantID: tenantId,
		Spans:    spans,
	})

	if err != nil {
		oc.l.Error().Err(err).Msg("failed to store spans")
		return &collectortracev1.ExportTraceServiceResponse{
			PartialSuccess: &collectortracev1.ExportTracePartialSuccess{
				RejectedSpans: int64(len(spans)),
				ErrorMessage:  err.Error(),
			},
		}, nil
	}

	oc.l.Debug().Int("span_count", len(spans)).Str("tenant_id", tenantId).Msg("stored spans")

	return &collectortracev1.ExportTraceServiceResponse{}, nil
}

func (oc *otelCollectorImpl) convertOTLPToSpanData(resourceSpans []*tracev1.ResourceSpans, tenantID pgtype.UUID) []*repository.SpanData {
	var spans []*repository.SpanData

	for _, rs := range resourceSpans {
		resourceAttrs := oc.serializeAttributes(rs.GetResource().GetAttributes())

		for _, ss := range rs.GetScopeSpans() {
			scopeName := ss.GetScope().GetName()

			for _, span := range ss.GetSpans() {
				spanData := &repository.SpanData{
					TraceID:              span.GetTraceId(),
					SpanID:               span.GetSpanId(),
					ParentSpanID:         span.GetParentSpanId(),
					Name:                 span.GetName(),
					Kind:                 int32(span.GetKind()),
					StartTimeUnixNano:    span.GetStartTimeUnixNano(),
					EndTimeUnixNano:      span.GetEndTimeUnixNano(),
					StatusCode:           int32(span.GetStatus().GetCode()),
					StatusMessage:        span.GetStatus().GetMessage(),
					Attributes:           oc.serializeAttributes(span.GetAttributes()),
					Events:               oc.serializeEvents(span.GetEvents()),
					Links:                oc.serializeLinks(span.GetLinks()),
					ResourceAttributes:   resourceAttrs,
					TenantID:             tenantID,
					InstrumentationScope: scopeName,
				}

				oc.extractHatchetCorrelation(span.GetAttributes(), spanData)

				spans = append(spans, spanData)
			}
		}
	}

	return spans
}

func (oc *otelCollectorImpl) extractHatchetCorrelation(attrs []*commonv1.KeyValue, spanData *repository.SpanData) {
	for _, attr := range attrs {
		switch attr.GetKey() {
		case AttrHatchetTaskRunID:
			if strVal := attr.GetValue().GetStringValue(); strVal != "" {
				uuid := sqlchelpers.UUIDFromStr(strVal)
				spanData.TaskRunExternalID = &uuid
			}
		case AttrHatchetWorkflowRunID:
			if strVal := attr.GetValue().GetStringValue(); strVal != "" {
				uuid := sqlchelpers.UUIDFromStr(strVal)
				spanData.WorkflowRunID = &uuid
			}
		}
	}
}

func (oc *otelCollectorImpl) serializeAttributes(attrs []*commonv1.KeyValue) []byte {
	if len(attrs) == 0 {
		return nil
	}

	attrMap := make(map[string]any, len(attrs))
	for _, kv := range attrs {
		attrMap[kv.GetKey()] = oc.anyValueToInterface(kv.GetValue())
	}

	data, err := json.Marshal(attrMap)
	if err != nil {
		oc.l.Warn().Err(err).Msg("failed to serialize attributes")
		return nil
	}

	return data
}

func (oc *otelCollectorImpl) serializeEvents(events []*tracev1.Span_Event) []byte {
	if len(events) == 0 {
		return nil
	}

	eventList := make([]map[string]any, 0, len(events))
	for _, event := range events {
		eventMap := map[string]any{
			"name":                     event.GetName(),
			"time_unix_nano":           event.GetTimeUnixNano(),
			"dropped_attributes_count": event.GetDroppedAttributesCount(),
		}

		if len(event.GetAttributes()) > 0 {
			attrMap := make(map[string]any, len(event.GetAttributes()))
			for _, kv := range event.GetAttributes() {
				attrMap[kv.GetKey()] = oc.anyValueToInterface(kv.GetValue())
			}
			eventMap["attributes"] = attrMap
		}

		eventList = append(eventList, eventMap)
	}

	data, err := json.Marshal(eventList)
	if err != nil {
		oc.l.Warn().Err(err).Msg("failed to serialize events")
		return nil
	}

	return data
}

func (oc *otelCollectorImpl) serializeLinks(links []*tracev1.Span_Link) []byte {
	if len(links) == 0 {
		return nil
	}

	linkList := make([]map[string]any, 0, len(links))
	for _, link := range links {
		linkMap := map[string]any{
			"trace_id":                 link.GetTraceId(),
			"span_id":                  link.GetSpanId(),
			"trace_state":              link.GetTraceState(),
			"dropped_attributes_count": link.GetDroppedAttributesCount(),
		}

		if len(link.GetAttributes()) > 0 {
			attrMap := make(map[string]any, len(link.GetAttributes()))
			for _, kv := range link.GetAttributes() {
				attrMap[kv.GetKey()] = oc.anyValueToInterface(kv.GetValue())
			}
			linkMap["attributes"] = attrMap
		}

		linkList = append(linkList, linkMap)
	}

	data, err := json.Marshal(linkList)
	if err != nil {
		oc.l.Warn().Err(err).Msg("failed to serialize links")
		return nil
	}

	return data
}

func (oc *otelCollectorImpl) anyValueToInterface(v *commonv1.AnyValue) interface{} {
	if v == nil {
		return nil
	}

	switch val := v.GetValue().(type) {
	case *commonv1.AnyValue_StringValue:
		return val.StringValue
	case *commonv1.AnyValue_BoolValue:
		return val.BoolValue
	case *commonv1.AnyValue_IntValue:
		return val.IntValue
	case *commonv1.AnyValue_DoubleValue:
		return val.DoubleValue
	case *commonv1.AnyValue_ArrayValue:
		arr := make([]any, 0, len(val.ArrayValue.GetValues()))
		for _, item := range val.ArrayValue.GetValues() {
			arr = append(arr, oc.anyValueToInterface(item))
		}
		return arr
	case *commonv1.AnyValue_KvlistValue:
		m := make(map[string]any, len(val.KvlistValue.GetValues()))
		for _, kv := range val.KvlistValue.GetValues() {
			m[kv.GetKey()] = oc.anyValueToInterface(kv.GetValue())
		}
		return m
	case *commonv1.AnyValue_BytesValue:
		return val.BytesValue
	default:
		return nil
	}
}
