package olap

import (
	"bytes"
	"context"
	"encoding/binary"
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
	eventMessage       string
	stepReadableID     string
	actionID           string
	displayName        string
	eventType          sqlcv1.V1EventTypeOlap
	additionalMetadata []byte
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

type eventEmittedAccumulator struct {
	eventSeenAt                 time.Time
	eventKey                    string
	triggeredRunExternalIDs     []string
	sourceWorkflowRunExternalID uuid.UUID
	sourceStepRunExternalID     uuid.UUID
	eventExternalID             uuid.UUID
}

func (tc *OLAPControllerImpl) writeEngineSpans(ctx context.Context, tenantId uuid.UUID, spans []*v1.SpanData, label string) {
	if len(spans) == 0 {
		return
	}

	olapRepo := tc.repo.OLAP()
	if olapRepo == nil {
		return
	}

	opts := &v1.CreateSpansOpts{TenantID: tenantId, Spans: spans}

	if err := olapRepo.CreateSpans(ctx, tenantId, opts); err != nil {
		tc.l.Error().Ctx(ctx).Err(err).Str("kind", label).Msg("could not write engine spans")
	}

	if err := olapRepo.CreateSpanLookupTableEntries(ctx, tenantId, opts); err != nil {
		tc.l.Error().Ctx(ctx).Err(err).Str("kind", label).Msg("could not write engine span lookup entries")
	}
}

var engineResourceAttrs []byte

func init() {
	engineResourceAttrs, _ = json.Marshal(map[string]string{
		"service.name": "hatchet-engine",
	})
}

func newEngineSpan(tenantId uuid.UUID, name string, traceID, spanID, parentSpanID []byte, startNano, endNano uint64, attrs []byte) *v1.SpanData {
	return &v1.SpanData{
		TenantID:             tenantId,
		TraceID:              traceID,
		SpanID:               spanID,
		ParentSpanID:         parentSpanID,
		Name:                 name,
		Kind:                 tracev1.Span_SPAN_KIND_INTERNAL,
		StartTimeUnixNano:    startNano,
		EndTimeUnixNano:      endNano,
		StatusCode:           tracev1.Status_STATUS_CODE_OK,
		Attributes:           attrs,
		ResourceAttributes:   engineResourceAttrs,
		InstrumentationScope: "hatchet-engine",
	}
}

func deriveEventSpanID(eventExternalID, workflowRunExternalID uuid.UUID) []byte {
	return v1.DeriveIDBytes(8, []byte("hatchet-engine-evt-span:"), eventExternalID[:], workflowRunExternalID[:])
}

func deriveStepRunSpanID(stepRunExternalID uuid.UUID, retryCount int32, spanType string) []byte {
	rc := make([]byte, 4)
	binary.BigEndian.PutUint32(rc, uint32(retryCount)) // nolint:gosec
	return v1.DeriveIDBytes(8, []byte("hatchet-engine-sr-span:"), stepRunExternalID[:], rc, []byte(spanType))
}

func parseSourceInfo(additionalMetadata []byte) (wfRunID, stepRunID uuid.UUID, ok bool) {
	if len(additionalMetadata) == 0 {
		return
	}

	var meta map[string]interface{}
	if err := json.Unmarshal(additionalMetadata, &meta); err != nil {
		return
	}

	wfStr, _ := meta["hatchet__source_workflow_run_id"].(string)
	stepStr, _ := meta["hatchet__source_step_run_id"].(string)
	if wfStr == "" || stepStr == "" {
		return
	}

	var err error
	if wfRunID, err = uuid.Parse(wfStr); err != nil {
		return
	}
	if stepRunID, err = uuid.Parse(stepStr); err != nil {
		return
	}

	ok = true
	return
}

type parentInfo struct {
	wfRunID   uuid.UUID
	stepRunID uuid.UUID
	isChild   bool
	isEvent   bool
	eventID   uuid.UUID
}

func parseParentInfo(additionalMetadata []byte) *parentInfo {
	if len(additionalMetadata) == 0 {
		return nil
	}

	var meta map[string]interface{}
	if err := json.Unmarshal(additionalMetadata, &meta); err != nil {
		return nil
	}

	if wfStr, ok := meta["hatchet__parent_workflow_run_id"].(string); ok && wfStr != "" {
		wfID, err := uuid.Parse(wfStr)
		if err != nil {
			return nil
		}
		stepStr, _ := meta["hatchet__parent_step_run_id"].(string)
		stepID, _ := uuid.Parse(stepStr)
		return &parentInfo{wfRunID: wfID, stepRunID: stepID, isChild: true}
	}

	if wfStr, ok := meta["hatchet__source_workflow_run_id"].(string); ok && wfStr != "" {
		wfID, err := uuid.Parse(wfStr)
		if err != nil {
			return nil
		}
		stepStr, _ := meta["hatchet__source_step_run_id"].(string)
		stepID, _ := uuid.Parse(stepStr)
		info := &parentInfo{wfRunID: wfID, stepRunID: stepID, isEvent: true}
		if evtStr, ok := meta["hatchet__event_id"].(string); ok {
			info.eventID, _ = uuid.Parse(evtStr)
		}
		return info
	}

	// Externally-pushed event: no source workflow, but the engine stamped
	// hatchet__event_id on the triggered run's metadata.
	if evtStr, ok := meta["hatchet__event_id"].(string); ok && evtStr != "" {
		evtID, err := uuid.Parse(evtStr)
		if err != nil {
			return nil
		}
		return &parentInfo{isEvent: true, eventID: evtID}
	}

	return nil
}

func resolveTraceIDFromParent(pi *parentInfo, workflowRunExternalID uuid.UUID) []byte {
	if pi != nil && pi.wfRunID != uuid.Nil {
		return v1.DeriveWorkflowRunTraceID(pi.wfRunID)
	}
	return v1.DeriveWorkflowRunTraceID(workflowRunExternalID)
}

// traceIDFromTraceparent extracts the trace_id bytes from a traceparent stored
// in additionalMetadata. Returns nil when no valid traceparent is present.
func traceIDFromTraceparent(additionalMetadata []byte) []byte {
	if len(additionalMetadata) == 0 {
		return nil
	}

	var meta map[string]interface{}
	if err := json.Unmarshal(additionalMetadata, &meta); err != nil {
		return nil
	}

	tp, _ := meta["traceparent"].(string)
	if tp == "" {
		return nil
	}

	parts := strings.Split(tp, "-")
	if len(parts) != 4 || len(parts[1]) != 32 {
		return nil
	}

	b, err := hex.DecodeString(parts[1])
	if err != nil {
		return nil
	}

	return b
}

// spanIDFromTraceparent extracts the span_id bytes from a traceparent in
// metadata. Used to recover the SDK producer span_id for parent linkage.
func spanIDFromTraceparent(additionalMetadata []byte) []byte {
	if len(additionalMetadata) == 0 {
		return nil
	}

	var meta map[string]interface{}
	if err := json.Unmarshal(additionalMetadata, &meta); err != nil {
		return nil
	}

	tp, _ := meta["traceparent"].(string)
	if tp == "" {
		return nil
	}

	parts := strings.Split(tp, "-")
	if len(parts) != 4 || len(parts[2]) != 16 {
		return nil
	}

	b, err := hex.DecodeString(parts[2])
	if err != nil {
		return nil
	}

	return b
}

func resolveTraceID(additionalMetadata []byte, workflowRunExternalID uuid.UUID) []byte {
	if traceID := traceIDFromTraceparent(additionalMetadata); traceID != nil {
		return traceID
	}
	return resolveTraceIDFromParent(parseParentInfo(additionalMetadata), workflowRunExternalID)
}

func buildWorkflowRunRootSpan(
	tenantId uuid.UUID,
	workflowRunExternalID uuid.UUID,
	workflowID uuid.UUID,
	displayName string,
	insertedAt time.Time,
	additionalMetadata []byte,
) *v1.SpanData {
	pi := parseParentInfo(additionalMetadata)

	traceID := resolveTraceIDFromParent(pi, workflowRunExternalID)
	spanID := v1.DeriveWorkflowRunSpanID(workflowRunExternalID)

	var parentSpanID []byte
	if pi != nil {
		if pi.isChild {
			// Parent the child workflow_run under the spawning task's
			// engine step_run span. The traceparent in additionalMetadata
			// is unreliable here: the SDK forwards ctx_additional_metadata
			// from ancestor tasks, so the traceparent may belong to any
			// ancestor workflow — not an SDK OTel producer span.
			// FIXME: this is incrementally better than the previous behavior, but still not perfect.
			// We will not nest this under the run span from the sdk.
			parentSpanID = deriveStepRunSpanID(pi.stepRunID, 0, "step_run")
		} else if pi.isEvent && pi.eventID != uuid.Nil {
			// When the SDK injected a traceparent, the bridge event span
			// uses the triggered run's own ID (buildEventSpan path).
			// Without a traceparent, match the event_emitted span which
			// uses the source run's ID.
			evtWfRunID := pi.wfRunID
			if evtWfRunID == uuid.Nil || traceIDFromTraceparent(additionalMetadata) != nil {
				evtWfRunID = workflowRunExternalID
			}
			parentSpanID = deriveEventSpanID(pi.eventID, evtWfRunID)
		}
	}

	// If the SDK injected a traceparent, use its trace_id and set the SDK
	// producer span as the parent of this engine root span.
	if sdkTraceID := traceIDFromTraceparent(additionalMetadata); sdkTraceID != nil {
		traceID = sdkTraceID

		if parentSpanID == nil {
			if sdkParent := spanIDFromTraceparent(additionalMetadata); sdkParent != nil && !bytes.Equal(sdkParent, spanID) {
				parentSpanID = sdkParent
			}
		}
	}

	attrs, _ := json.Marshal(map[string]string{
		"hatchet.span_source":     "engine",
		"hatchet.workflow_run_id": workflowRunExternalID.String(),
		"hatchet.workflow_id":     workflowID.String(),
		"hatchet.workflow_name":   displayName,
	})

	ts := safeUint64(insertedAt.UnixNano())
	span := newEngineSpan(tenantId, "hatchet.engine.workflow_run", traceID, spanID, parentSpanID, ts, ts, attrs)
	wfRunID := workflowRunExternalID
	span.WorkflowRunID = &wfRunID
	return span
}

func buildEventSpan(
	tenantId uuid.UUID,
	eventExternalID uuid.UUID,
	eventKey string,
	eventSeenAt time.Time,
	workflowRunExternalID uuid.UUID,
	eventAdditionalMetadata []byte,
) *v1.SpanData {
	attrs, _ := json.Marshal(map[string]string{
		"hatchet.span_source": "engine",
		"hatchet.event_key":   eventKey,
		"hatchet.event_id":    eventExternalID.String(),
	})

	traceID := v1.DeriveWorkflowRunTraceID(workflowRunExternalID)
	var parentSpanID []byte

	if sdkTraceID := traceIDFromTraceparent(eventAdditionalMetadata); sdkTraceID != nil {
		traceID = sdkTraceID
		parentSpanID = spanIDFromTraceparent(eventAdditionalMetadata)
	}

	ts := safeUint64(eventSeenAt.UnixNano())
	span := newEngineSpan(
		tenantId, "hatchet.engine.event",
		traceID,
		deriveEventSpanID(eventExternalID, workflowRunExternalID),
		parentSpanID, ts, ts, attrs,
	)
	wfRunID := workflowRunExternalID
	span.WorkflowRunID = &wfRunID
	return span
}

func buildEventEmittedSpan(
	tenantId uuid.UUID,
	sourceWorkflowRunExternalID uuid.UUID,
	sourceStepRunExternalID uuid.UUID,
	eventExternalID uuid.UUID,
	eventKey string,
	eventSeenAt time.Time,
	triggeredRunExternalIDs []string,
) *v1.SpanData {
	attrMap := map[string]string{
		"hatchet.span_source":     "engine",
		"hatchet.event_key":       eventKey,
		"hatchet.event_id":        eventExternalID.String(),
		"hatchet.workflow_run_id": sourceWorkflowRunExternalID.String(),
	}
	if len(triggeredRunExternalIDs) > 0 {
		triggered, _ := json.Marshal(triggeredRunExternalIDs)
		attrMap["hatchet.triggered_workflow_run_ids"] = string(triggered)
	}
	attrs, _ := json.Marshal(attrMap)

	ts := safeUint64(eventSeenAt.UnixNano())
	span := newEngineSpan(
		tenantId, "hatchet.engine.event_emitted",
		v1.DeriveWorkflowRunTraceID(sourceWorkflowRunExternalID),
		deriveEventSpanID(eventExternalID, sourceWorkflowRunExternalID),
		deriveStepRunSpanID(sourceStepRunExternalID, 0, "step_run"),
		ts, ts, attrs,
	)
	wfRunID := sourceWorkflowRunExternalID
	span.WorkflowRunID = &wfRunID
	return span
}

func (tc *OLAPControllerImpl) synthesizeEngineSpans(ctx context.Context, tenantId uuid.UUID, events []engineSpanEvent) {
	if len(events) == 0 {
		return
	}

	var queuedSpans []*v1.SpanData
	var terminalEvents []engineSpanEvent
	batchStartedTimes := make(map[taskRetryKey]time.Time)

	for _, e := range events {
		switch e.eventType {
		case sqlcv1.V1EventTypeOlapSTARTED:
			batchStartedTimes[taskRetryKey{e.taskID, e.retryCount}] = e.eventTimestamp
		case sqlcv1.V1EventTypeOlapSENTTOWORKER:
			if span := tc.buildQueuedSpan(tenantId, e); span != nil {
				queuedSpans = append(queuedSpans, span)
			}
		case sqlcv1.V1EventTypeOlapFINISHED,
			sqlcv1.V1EventTypeOlapFAILED,
			sqlcv1.V1EventTypeOlapCANCELLED,
			sqlcv1.V1EventTypeOlapTIMEDOUT,
			sqlcv1.V1EventTypeOlapSCHEDULINGTIMEDOUT:
			terminalEvents = append(terminalEvents, e)
		}
	}

	stepRunSpans := tc.buildStepRunSpans(ctx, tenantId, terminalEvents, batchStartedTimes)

	allSpans := make([]*v1.SpanData, 0, len(queuedSpans)+len(stepRunSpans))
	allSpans = append(allSpans, queuedSpans...)
	allSpans = append(allSpans, stepRunSpans...)

	tc.writeEngineSpans(ctx, tenantId, allSpans, "task")
}

func (tc *OLAPControllerImpl) buildQueuedSpan(tenantId uuid.UUID, e engineSpanEvent) *v1.SpanData {
	attrs := buildEngineSpanAttributes(e)
	traceID := resolveTraceID(e.additionalMetadata, e.workflowRunID)

	span := newEngineSpan(
		tenantId, "hatchet.engine.queued",
		traceID,
		deriveStepRunSpanID(e.externalID, e.retryCount, "queued"),
		v1.DeriveWorkflowRunSpanID(e.workflowRunID),
		safeUint64(e.insertedAt.UnixNano()),
		safeUint64(e.eventTimestamp.UnixNano()),
		attrs,
	)

	externalID := e.externalID
	workflowRunID := e.workflowRunID
	span.TaskRunExternalID = &externalID
	span.WorkflowRunID = &workflowRunID
	span.RetryCount = e.retryCount
	return span
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
		startTime, ok := startedMap[taskRetryKey{e.taskID, e.retryCount}]
		if !ok {
			startTime = e.insertedAt
		}

		statusCode := tracev1.Status_STATUS_CODE_OK
		var statusMessage string
		if e.eventType != sqlcv1.V1EventTypeOlapFINISHED {
			statusCode = tracev1.Status_STATUS_CODE_ERROR
			statusMessage = stepRunStatusMessage(e)
		}

		attrs := buildEngineSpanAttributes(e)
		traceID := resolveTraceID(e.additionalMetadata, e.workflowRunID)

		span := newEngineSpan(
			tenantId, "hatchet.engine.start_step_run",
			traceID,
			deriveStepRunSpanID(e.externalID, e.retryCount, "step_run"),
			v1.DeriveWorkflowRunSpanID(e.workflowRunID),
			safeUint64(startTime.UnixNano()),
			safeUint64(e.eventTimestamp.UnixNano()),
			attrs,
		)

		span.StatusCode = statusCode
		span.StatusMessage = statusMessage

		externalID := e.externalID
		workflowRunID := e.workflowRunID
		span.TaskRunExternalID = &externalID
		span.WorkflowRunID = &workflowRunID
		span.RetryCount = e.retryCount

		spans = append(spans, span)
	}

	return spans
}

func buildEngineSpanAttributes(e engineSpanEvent) []byte {
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
	return attrs
}

func stepRunStatusMessage(e engineSpanEvent) string {
	if e.eventMessage != "" {
		return e.eventMessage
	}
	switch e.eventType {
	case sqlcv1.V1EventTypeOlapCANCELLED:
		return "task cancelled"
	case sqlcv1.V1EventTypeOlapTIMEDOUT:
		return "task timed out"
	case sqlcv1.V1EventTypeOlapSCHEDULINGTIMEDOUT:
		return "scheduling timed out"
	case sqlcv1.V1EventTypeOlapFAILED:
		return "task failed"
	default:
		return ""
	}
}

func safeUint64(v int64) uint64 {
	if v < 0 {
		return 0
	}
	return uint64(v) // nolint:gosec
}
