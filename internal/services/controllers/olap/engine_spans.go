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
	taskInsertedAt     time.Time
	eventTimestamp     time.Time
	eventType          sqlcv1.V1EventTypeOlap
	stepReadableID     string
	additionalMetadata []byte
	actionID           string
	displayName        string
	eventMessage       string
	taskID             int64
	retryCount         int32
	externalID         uuid.UUID
	workflowRunID      uuid.UUID
	workflowID         uuid.UUID
	workflowVersionID  uuid.UUID
	stepID             uuid.UUID
}

type taskRetryKey struct {
	taskID     int64
	retryCount int32
}

func (tc *OLAPControllerImpl) synthesizeEngineSpans(ctx context.Context, tenantId uuid.UUID, events []engineSpanEvent) {
	otelRepo := tc.repo.OTelCollector()
	if otelRepo == nil || len(events) == 0 {
		return
	}

	var queuedSpans []*v1.SpanData
	var terminalEvents []engineSpanEvent
	batchStartedTimes := make(map[taskRetryKey]time.Time)

	for _, e := range events {
		// FIXME: add spans for other event types
		switch e.eventType {
		case sqlcv1.V1EventTypeOlapSTARTED:
			batchStartedTimes[taskRetryKey{e.taskID, e.retryCount}] = e.eventTimestamp
		case sqlcv1.V1EventTypeOlapSENTTOWORKER:
			span := tc.buildQueuedSpan(tenantId, e)
			if span != nil {
				queuedSpans = append(queuedSpans, span)
			}
		case sqlcv1.V1EventTypeOlapFINISHED,
			sqlcv1.V1EventTypeOlapFAILED,
			sqlcv1.V1EventTypeOlapCANCELLED,
			sqlcv1.V1EventTypeOlapTIMEDOUT:
			terminalEvents = append(terminalEvents, e)
		}
	}

	stepRunSpans := tc.buildStepRunSpans(ctx, tenantId, terminalEvents, batchStartedTimes)

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

func (tc *OLAPControllerImpl) buildQueuedSpan(tenantId uuid.UUID, e engineSpanEvent) *v1.SpanData {
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
		"hatchet.span_source":         "engine",
		"hatchet.step_run_id":         e.externalID.String(),
		"hatchet.workflow_run_id":     e.workflowRunID.String(),
		"hatchet.step_name":           e.stepReadableID,
		"hatchet.retry_count":         fmt.Sprintf("%d", e.retryCount),
		"hatchet.action_id":           e.actionID,
		"hatchet.task_name":           e.stepReadableID,
		"hatchet.workflow_id":         e.workflowID.String(),
		"hatchet.workflow_version_id": e.workflowVersionID.String(),
		"hatchet.step_id":             e.stepID.String(),
	})

	resourceAttrs, _ := json.Marshal(map[string]string{
		"service.name": "hatchet-engine",
	})

	externalID := e.externalID
	workflowRunID := e.workflowRunID

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
		TaskRunExternalID:    &externalID,
		WorkflowRunID:        &workflowRunID,
		RetryCount:           e.retryCount,
	}
}

func (tc *OLAPControllerImpl) buildStepRunSpans(ctx context.Context, tenantId uuid.UUID, events []engineSpanEvent, batchStartedTimes map[taskRetryKey]time.Time) []*v1.SpanData {
	if len(events) == 0 {
		return nil
	}

	taskIds := make([]int64, len(events))
	taskInsertedAts := make([]time.Time, len(events))
	retryCounts := make([]int32, len(events))
	for i, e := range events {
		taskIds[i] = e.taskID
		taskInsertedAts[i] = e.taskInsertedAt
		retryCounts[i] = e.retryCount
	}

	startedRows, err := tc.repo.OLAP().GetTaskStartedTimestamps(ctx, tenantId, taskIds, taskInsertedAts, retryCounts)
	if err != nil {
		tc.l.Error().Ctx(ctx).Err(err).Msg("could not look up STARTED timestamps for step_run spans")
		return nil
	}

	startedMap := make(map[taskRetryKey]time.Time, len(startedRows)+len(batchStartedTimes))

	// NOTE: seed from the in-memory batch first so DB rows take precedence
	// when both are present (the DB value has already been persisted).
	for k, t := range batchStartedTimes {
		startedMap[k] = t
	}
	for _, row := range startedRows {
		if row.StartedAt.Valid {
			startedMap[taskRetryKey{row.TaskID, row.RetryCount}] = row.StartedAt.Time
		}
	}

	var spans []*v1.SpanData
	for _, e := range events {
		traceID, parentSpanID := parseTraceparent(e.additionalMetadata)
		if traceID == "" {
			continue
		}

		spanID := generateSpanID()
		if spanID == "" {
			continue
		}

		startTime, ok := startedMap[taskRetryKey{e.taskID, e.retryCount}]
		if !ok {
			startTime = e.insertedAt
		}

		statusCode := tracev1.Status_STATUS_CODE_OK
		var statusMessage string
		if e.eventType != sqlcv1.V1EventTypeOlapFINISHED {
			statusCode = tracev1.Status_STATUS_CODE_ERROR
			statusMessage = e.eventMessage
			if statusMessage == "" {
				switch e.eventType {
				case sqlcv1.V1EventTypeOlapCANCELLED:
					statusMessage = "task cancelled"
				case sqlcv1.V1EventTypeOlapTIMEDOUT:
					statusMessage = "task timed out"
				case sqlcv1.V1EventTypeOlapFAILED:
					statusMessage = "task failed"
				}
			}
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
			"hatchet.span_source":         "engine",
			"hatchet.step_run_id":         e.externalID.String(),
			"hatchet.workflow_run_id":     e.workflowRunID.String(),
			"hatchet.step_name":           e.stepReadableID,
			"hatchet.retry_count":         fmt.Sprintf("%d", e.retryCount),
			"hatchet.action_id":           e.actionID,
			"hatchet.task_name":           e.stepReadableID,
			"hatchet.workflow_id":         e.workflowID.String(),
			"hatchet.workflow_version_id": e.workflowVersionID.String(),
			"hatchet.step_id":             e.stepID.String(),
		})

		resourceAttrs, _ := json.Marshal(map[string]string{
			"service.name": "hatchet-engine",
		})

		externalID := e.externalID
		workflowRunID := e.workflowRunID

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
			StatusMessage:        statusMessage,
			Attributes:           attrs,
			ResourceAttributes:   resourceAttrs,
			InstrumentationScope: "hatchet-engine",
			TaskRunExternalID:    &externalID,
			WorkflowRunID:        &workflowRunID,
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
