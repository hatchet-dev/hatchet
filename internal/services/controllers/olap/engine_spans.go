package olap

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type engineSpanEvent struct {
	insertedAt         time.Time
	eventTimestamp     time.Time
	eventType          sqlcv1.V1EventTypeOlap
	stepReadableID     string
	additionalMetadata []byte
	taskID             int64
	retryCount         int32
	externalID         uuid.UUID
	workflowRunID      uuid.UUID
}

func (tc *OLAPControllerImpl) synthesizeEngineSpans(ctx context.Context, tenantId uuid.UUID, events []engineSpanEvent) {
	otelRepo := tc.repo.OTelCollector()
	if otelRepo == nil || len(events) == 0 {
		return
	}

	var queuedSpans []*v1.SpanData
	var terminalEvents []engineSpanEvent

	for i := range events {
		e := &events[i]

		switch e.eventType {
		case sqlcv1.V1EventTypeOlapSENTTOWORKER:
			span := tc.buildQueuedSpan(tenantId, e)
			if span != nil {
				queuedSpans = append(queuedSpans, span)
			}
		case sqlcv1.V1EventTypeOlapFINISHED,
			sqlcv1.V1EventTypeOlapFAILED,
			sqlcv1.V1EventTypeOlapCANCELLED,
			sqlcv1.V1EventTypeOlapTIMEDOUT:
			terminalEvents = append(terminalEvents, *e)
		}
	}

	stepRunSpans := tc.buildStepRunSpans(ctx, tenantId, terminalEvents)

	allSpans := make([]*v1.SpanData, 0, len(queuedSpans)+len(stepRunSpans))
	allSpans = append(allSpans, queuedSpans...)
	allSpans = append(allSpans, stepRunSpans...)
	if len(allSpans) == 0 {
		return
	}

	if err := otelRepo.CreateSpans(ctx, tenantId, &v1.CreateSpansOpts{
		TenantID: tenantId,
		Spans:    allSpans,
	}); err != nil {
		tc.l.Error().Ctx(ctx).Err(err).Msg("could not write engine spans")
	}
}

func (tc *OLAPControllerImpl) buildQueuedSpan(tenantId uuid.UUID, e *engineSpanEvent) *v1.SpanData {
	traceID, parentSpanID := parseTraceparent(e.additionalMetadata)
	if traceID == "" {
		return nil
	}

	spanID := generateSpanID()
	if spanID == "" {
		return nil
	}

	traceIDBytes, err := hex.DecodeString(traceID)
	if err != nil {
		return nil
	}
	spanIDBytes, err := hex.DecodeString(spanID)
	if err != nil {
		return nil
	}
	var parentBytes []byte
	if parentSpanID != "" {
		parentBytes, _ = hex.DecodeString(parentSpanID)
	}

	attrs, _ := json.Marshal(map[string]string{
		"hatchet.span_source":     "engine",
		"hatchet.step_run_id":     e.externalID.String(),
		"hatchet.workflow_run_id": e.workflowRunID.String(),
		"hatchet.step_name":       e.stepReadableID,
		"hatchet.retry_count":     fmt.Sprintf("%d", e.retryCount),
	})

	resourceAttrs, _ := json.Marshal(map[string]string{
		"service.name": "hatchet-engine",
	})

	return &v1.SpanData{
		TenantID:             tenantId,
		TraceID:              traceIDBytes,
		SpanID:               spanIDBytes,
		ParentSpanID:         parentBytes,
		Name:                 "hatchet.engine.queued",
		Kind:                 tracev1.Span_SPAN_KIND_INTERNAL,
		StartTimeUnixNano:    safeUint64(e.insertedAt.UnixNano()),
		EndTimeUnixNano:      safeUint64(e.eventTimestamp.UnixNano()),
		StatusCode:           tracev1.Status_STATUS_CODE_OK,
		Attributes:           attrs,
		ResourceAttributes:   resourceAttrs,
		InstrumentationScope: "hatchet-engine",
		TaskRunExternalID:    &e.externalID,
		WorkflowRunID:        &e.workflowRunID,
		RetryCount:           e.retryCount,
	}
}

func (tc *OLAPControllerImpl) buildStepRunSpans(ctx context.Context, tenantId uuid.UUID, events []engineSpanEvent) []*v1.SpanData {
	if len(events) == 0 {
		return nil
	}

	taskIds := make([]int64, len(events))
	for i, e := range events {
		taskIds[i] = e.taskID
	}

	startedRows, err := tc.repo.OLAP().GetTaskStartedTimestamps(ctx, tenantId, taskIds)
	if err != nil {
		tc.l.Error().Ctx(ctx).Err(err).Msg("could not look up STARTED timestamps for step_run spans")
		return nil
	}

	type startedKey struct {
		taskID     int64
		retryCount int32
	}
	startedMap := make(map[startedKey]time.Time, len(startedRows))
	for _, row := range startedRows {
		if row.StartedAt.Valid {
			startedMap[startedKey{row.TaskID, row.RetryCount}] = row.StartedAt.Time
		}
	}

	var spans []*v1.SpanData
	for i := range events {
		e := &events[i]

		traceID, parentSpanID := parseTraceparent(e.additionalMetadata)
		if traceID == "" {
			continue
		}

		spanID := generateSpanID()
		if spanID == "" {
			continue
		}

		startTime, ok := startedMap[startedKey{e.taskID, e.retryCount}]
		if !ok {
			startTime = e.insertedAt
		}

		statusCode := tracev1.Status_STATUS_CODE_OK
		if e.eventType != sqlcv1.V1EventTypeOlapFINISHED {
			statusCode = tracev1.Status_STATUS_CODE_ERROR
		}

		traceIDBytes, err := hex.DecodeString(traceID)
		if err != nil {
			continue
		}
		spanIDBytes, err := hex.DecodeString(spanID)
		if err != nil {
			continue
		}
		var parentBytes []byte
		if parentSpanID != "" {
			parentBytes, _ = hex.DecodeString(parentSpanID)
		}

		attrs, _ := json.Marshal(map[string]string{
			"hatchet.span_source":     "engine",
			"hatchet.step_run_id":     e.externalID.String(),
			"hatchet.workflow_run_id": e.workflowRunID.String(),
			"hatchet.step_name":       e.stepReadableID,
			"hatchet.retry_count":     fmt.Sprintf("%d", e.retryCount),
		})

		resourceAttrs, _ := json.Marshal(map[string]string{
			"service.name": "hatchet-engine",
		})

		spans = append(spans, &v1.SpanData{
			TenantID:             tenantId,
			TraceID:              traceIDBytes,
			SpanID:               spanIDBytes,
			ParentSpanID:         parentBytes,
			Name:                 "hatchet.start_step_run",
			Kind:                 tracev1.Span_SPAN_KIND_INTERNAL,
			StartTimeUnixNano:    safeUint64(startTime.UnixNano()),
			EndTimeUnixNano:      safeUint64(e.eventTimestamp.UnixNano()),
			StatusCode:           statusCode,
			Attributes:           attrs,
			ResourceAttributes:   resourceAttrs,
			InstrumentationScope: "hatchet-engine",
			TaskRunExternalID:    &e.externalID,
			WorkflowRunID:        &e.workflowRunID,
			RetryCount:           e.retryCount,
		})
	}

	return spans
}

func parseTraceparent(additionalMetadata []byte) (traceID, parentSpanID string) {
	if len(additionalMetadata) == 0 {
		return "", ""
	}

	var meta map[string]interface{}
	if err := json.Unmarshal(additionalMetadata, &meta); err != nil {
		return "", ""
	}

	tp, ok := meta["traceparent"].(string)
	if !ok || tp == "" {
		return "", ""
	}

	parts := strings.SplitN(tp, "-", 4)
	if len(parts) < 3 {
		return "", ""
	}

	return parts[1], parts[2]
}

func safeUint64(v int64) uint64 {
	if v < 0 {
		return 0
	}
	return uint64(v) // nolint:gosec
}

func generateSpanID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
