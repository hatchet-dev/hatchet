package transformers

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func ToSlotState(slots []*dbsqlc.ListSemaphoreSlotsWithStateForWorkerRow, remainingSlots int) *[]gen.SemaphoreSlots {
	resp := make([]gen.SemaphoreSlots, len(slots))

	for i := range slots {
		slot := slots[i]

		var stepRunId uuid.UUID

		if slot.StepRunId.Valid {
			stepRunId = uuid.MustParse(sqlchelpers.UUIDToStr(slot.StepRunId))
		}

		var workflowRunId uuid.UUID

		if slot.WorkflowRunId.Valid {
			workflowRunId = uuid.MustParse(sqlchelpers.UUIDToStr(slot.WorkflowRunId))
		}

		resp[i] = gen.SemaphoreSlots{
			StepRunId:     stepRunId,
			Status:        gen.StepRunStatus(slot.Status),
			ActionId:      slot.ActionId,
			WorkflowRunId: workflowRunId,
			TimeoutAt:     &slot.TimeoutAt.Time,
			StartedAt:     &slot.StartedAt.Time,
		}
	}

	for i := len(slots); i < remainingSlots; i++ {
		resp = append(resp, gen.SemaphoreSlots{})
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

func ToWorkerRuntimeInfo(worker *dbsqlc.Worker) *gen.WorkerRuntimeInfo {

	runtime := &gen.WorkerRuntimeInfo{
		SdkVersion:      &worker.SdkVersion.String,
		LanguageVersion: &worker.LanguageVersion.String,
		Os:              &worker.Os.String,
		RuntimeExtra:    &worker.RuntimeExtra.String,
	}

	if worker.Language.Valid {
		lang := gen.WorkerRuntimeSDKs(worker.Language.WorkerSDKS)
		runtime.Language = &lang
	}

	return runtime
}

func ToWorkerSqlc(worker *dbsqlc.Worker, remainingSlots *int, webhookUrl *string, actions []pgtype.Text) *gen.Worker {

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

	if remainingSlots != nil {
		availableRuns = *remainingSlots
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
		RuntimeInfo:   ToWorkerRuntimeInfo(worker),
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
