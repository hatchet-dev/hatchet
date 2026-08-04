package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// ReadableStatusForOlapEvent maps a monitoring event type to the readable run
// status it implies. Unknown or informational events map to QUEUED.
func ReadableStatusForOlapEvent(eventType sqlcv1.V1EventTypeOlap) sqlcv1.V1ReadableStatusOlap {
	switch eventType {
	case sqlcv1.V1EventTypeOlapRETRYING,
		sqlcv1.V1EventTypeOlapREASSIGNED,
		sqlcv1.V1EventTypeOlapRETRIEDBYUSER,
		sqlcv1.V1EventTypeOlapCREATED,
		sqlcv1.V1EventTypeOlapQUEUED,
		sqlcv1.V1EventTypeOlapREQUEUEDNOWORKER,
		sqlcv1.V1EventTypeOlapREQUEUEDRATELIMIT,
		sqlcv1.V1EventTypeOlapBATCHBUFFERED,
		sqlcv1.V1EventTypeOlapWAITINGFORBATCH:
		return sqlcv1.V1ReadableStatusOlapQUEUED
	case sqlcv1.V1EventTypeOlapASSIGNED,
		sqlcv1.V1EventTypeOlapACKNOWLEDGED,
		sqlcv1.V1EventTypeOlapSENTTOWORKER,
		sqlcv1.V1EventTypeOlapSLOTRELEASED,
		sqlcv1.V1EventTypeOlapSTARTED,
		sqlcv1.V1EventTypeOlapTIMEOUTREFRESHED,
		sqlcv1.V1EventTypeOlapDURABLERESTORING,
		// running until the individual tasks are completed
		sqlcv1.V1EventTypeOlapBATCHFLUSHED:
		return sqlcv1.V1ReadableStatusOlapRUNNING
	case sqlcv1.V1EventTypeOlapFINISHED,
		sqlcv1.V1EventTypeOlapSKIPPED:
		return sqlcv1.V1ReadableStatusOlapCOMPLETED
	case sqlcv1.V1EventTypeOlapSCHEDULINGTIMEDOUT,
		sqlcv1.V1EventTypeOlapFAILED,
		sqlcv1.V1EventTypeOlapTIMEDOUT,
		sqlcv1.V1EventTypeOlapRATELIMITERROR,
		sqlcv1.V1EventTypeOlapCOULDNOTSENDTOWORKER:
		return sqlcv1.V1ReadableStatusOlapFAILED
	case sqlcv1.V1EventTypeOlapCANCELLED:
		return sqlcv1.V1ReadableStatusOlapCANCELLED
	case sqlcv1.V1EventTypeOlapDURABLEEVICTED:
		return sqlcv1.V1ReadableStatusOlapEVICTED
	default:
		// treat unknown or informational events as queued
		return sqlcv1.V1ReadableStatusOlapQUEUED
	}
}

// WriteMonitoringEventsBestEffort writes monitoring events straight to the OLAP
// tables, bypassing the outbox and message queue. See the interface doc for the
// best-effort semantics.
func (r *OLAPRepositoryImpl) WriteMonitoringEventsBestEffort(ctx context.Context, tenantId uuid.UUID, payloads ...CreateMonitoringEventPayload) error {
	if len(payloads) == 0 {
		return nil
	}

	taskIds := make([]int64, len(payloads))

	for i, p := range payloads {
		taskIds[i] = p.TaskId
	}

	metas, err := r.queries.ListTaskMetas(ctx, r.pool, sqlcv1.ListTaskMetasParams{
		TenantID: tenantId,
		Ids:      taskIds,
	})

	if err != nil {
		return fmt.Errorf("could not list task metas: %w", err)
	}

	taskIdToMeta := make(map[int64]*sqlcv1.ListTaskMetasRow, len(metas))

	for _, meta := range metas {
		taskIdToMeta[meta.ID] = meta
	}

	events := make([]sqlcv1.CreateTaskEventsOLAPParams, 0, len(payloads))
	eventExternalIdToWorkflowRunId := make(map[uuid.UUID]uuid.UUID, len(payloads))

	for _, p := range payloads {
		meta, ok := taskIdToMeta[p.TaskId]

		if !ok {
			r.l.Warn().Ctx(ctx).Msgf("could not find task meta for task id %d, dropping monitoring event", p.TaskId)
			continue
		}

		externalId := uuid.New()

		event := sqlcv1.CreateTaskEventsOLAPParams{
			TenantID:               tenantId,
			TaskID:                 p.TaskId,
			TaskInsertedAt:         meta.InsertedAt,
			WorkflowID:             meta.WorkflowID,
			EventType:              p.EventType,
			EventTimestamp:         sqlchelpers.TimestamptzFromTime(p.EventTimestamp),
			ReadableStatus:         ReadableStatusForOlapEvent(p.EventType),
			RetryCount:             p.RetryCount,
			DurableInvocationCount: p.DurableInvocationCount,
			WorkerID:               p.WorkerId,
			AdditionalEventMessage: sqlchelpers.TextFromStr(p.EventMessage),
			ExternalID:             externalId,
		}

		// parity with the queue consumer's payload handling
		switch p.EventType {
		case sqlcv1.V1EventTypeOlapFINISHED:
			if p.EventPayload != "" {
				event.Output = []byte(p.EventPayload)
			}
		case sqlcv1.V1EventTypeOlapFAILED:
			event.ErrorMessage = sqlchelpers.TextFromStr(strings.ReplaceAll(p.EventPayload, "\x00", ""))
		}

		events = append(events, event)
		eventExternalIdToWorkflowRunId[externalId] = meta.WorkflowRunID
	}

	if len(events) == 0 {
		return nil
	}

	_, locksNotAcquired, err := r.CreateTaskEvents(ctx, tenantId, events, eventExternalIdToWorkflowRunId)

	if err != nil {
		return err
	}

	if len(locksNotAcquired) > 0 {
		r.l.Debug().Ctx(ctx).Msgf("dropped best-effort monitoring events for %d runs with contended locks", len(locksNotAcquired))
	}

	return nil
}
