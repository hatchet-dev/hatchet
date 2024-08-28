package transformers

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func ToSlotState(slots []*dbsqlc.ListSemaphoreSlotsWithStateForWorkerRow) *[]gen.SemaphoreSlots {
	resp := make([]gen.SemaphoreSlots, len(slots))

	for i := range slots {
		slot := slots[i]

		slotId := uuid.MustParse(sqlchelpers.UUIDToStr(slot.Slot))

		var stepRunId uuid.UUID

		if slot.StepRunId.Valid {
			stepRunId = uuid.MustParse(sqlchelpers.UUIDToStr(slot.StepRunId))
		}

		var workflowRunId uuid.UUID

		if slot.WorkflowRunId.Valid {
			workflowRunId = uuid.MustParse(sqlchelpers.UUIDToStr(slot.WorkflowRunId))
		}

		resp[i] = gen.SemaphoreSlots{
			Slot:          slotId,
			StepRunId:     &stepRunId,
			Status:        (*gen.StepRunStatus)(&slot.Status.StepRunStatus),
			ActionId:      &slot.ActionId.String,
			WorkflowRunId: &workflowRunId,
			TimeoutAt:     &slot.TimeoutAt.Time,
			StartedAt:     &slot.StartedAt.Time,
		}
	}

	return &resp
}

func ToWorkerLabels(labels []*dbsqlc.ListWorkerLabelsRow) *[]gen.WorkerLabel {
	resp := make([]gen.WorkerLabel, len(labels))

	for i := range labels {

		var value *string

		switch {
		case labels[i].IntValue.Valid:
			intValue := labels[i].IntValue.Int32
			stringValue := fmt.Sprintf("%d", intValue)
			value = &stringValue
		case labels[i].StrValue.Valid:
			value = &labels[i].StrValue.String
		default:
			value = nil
		}

		id := fmt.Sprintf("%d", labels[i].ID)

		resp[i] = gen.WorkerLabel{
			Metadata: *toAPIMetadata(id, labels[i].CreatedAt.Time, labels[i].UpdatedAt.Time),
			Key:      labels[i].Key,
			Value:    value,
		}
	}

	return &resp
}

func ToWorker(worker *db.WorkerModel) *gen.Worker {

	var dispatcherUuid uuid.UUID

	if id, ok := worker.DispatcherID(); ok {
		dispatcherUuid = uuid.MustParse(id)
	}

	status := gen.ACTIVE

	if worker.IsPaused {
		status = gen.PAUSED
	}

	if lastHeartbeat, ok := worker.LastHeartbeatAt(); ok && lastHeartbeat.Add(4*time.Second).Before(time.Now()) {
		status = gen.INACTIVE
	}

	res := &gen.Worker{
		Metadata:     *toAPIMetadata(worker.ID, worker.CreatedAt, worker.UpdatedAt),
		Name:         worker.Name,
		DispatcherId: &dispatcherUuid,
		Status:       &status,
		MaxRuns:      &worker.MaxRuns,
	}

	if lastHeartbeatAt, ok := worker.LastHeartbeatAt(); ok {
		res.LastHeartbeatAt = &lastHeartbeatAt
	}

	if lastListenerEstablished, ok := worker.LastListenerEstablished(); ok {
		res.LastListenerEstablished = &lastListenerEstablished
	}

	numSlots := 0
	for _, slot := range worker.Slots() {
		if _, ok := slot.StepRunID(); !ok {
			numSlots++
		}
	}
	res.AvailableRuns = &numSlots

	if worker.RelationsWorker.Actions != nil {
		if actions := worker.Actions(); actions != nil {
			apiActions := make([]string, len(actions))

			for i, action := range actions {
				apiActions[i] = action.ActionID
			}

			res.Actions = &apiActions
		}
	}

	return res
}

func ToWorkerSqlc(worker *dbsqlc.Worker, slots *int, webhookUrl *string, actions []pgtype.Text) *gen.Worker {

	dispatcherId := uuid.MustParse(pgUUIDToStr(worker.DispatcherId))

	maxRuns := int(worker.MaxRuns)

	status := gen.ACTIVE

	if worker.IsPaused {
		status = gen.PAUSED
	}

	if worker.LastHeartbeatAt.Time.Add(5 * time.Second).Before(time.Now()) {
		status = gen.INACTIVE
	}

	var availableRuns int

	if slots != nil {
		availableRuns = maxRuns - *slots
	}

	res := &gen.Worker{
		Metadata:      *toAPIMetadata(pgUUIDToStr(worker.ID), worker.CreatedAt.Time, worker.UpdatedAt.Time),
		Name:          worker.Name,
		Type:          gen.WorkerType(worker.Type),
		Status:        &status,
		DispatcherId:  &dispatcherId,
		MaxRuns:       &maxRuns,
		AvailableRuns: &availableRuns,
		WebhookUrl:    webhookUrl,
	}

	if worker.WebhookId.Valid {
		wid := uuid.MustParse(pgUUIDToStr(worker.WebhookId))
		res.WebhookId = &wid
	}

	if !worker.LastHeartbeatAt.Time.IsZero() {
		res.LastHeartbeatAt = &worker.LastHeartbeatAt.Time
	}

	if actions != nil {
		apiActions := make([]string, len(actions))

		for i := range actions {
			apiActions[i] = actions[i].String
		}

		res.Actions = &apiActions
	}

	return res
}
