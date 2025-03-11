package transformers

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func ToSlotState(slots []*sqlcv1.ListSemaphoreSlotsWithStateForWorkerRow, remainingSlots int) *[]gen.SemaphoreSlots {
	resp := make([]gen.SemaphoreSlots, len(slots))

	for i := range slots {
		slot := slots[i]

		var stepRunId uuid.UUID
		var workflowRunId uuid.UUID

		if slot.ExternalID.Valid {
			stepRunId = uuid.MustParse(sqlchelpers.UUIDToStr(slot.ExternalID))
			workflowRunId = uuid.MustParse(sqlchelpers.UUIDToStr(slot.ExternalID))
		}

		resp[i] = gen.SemaphoreSlots{
			StepRunId:     stepRunId,
			Status:        gen.StepRunStatusRUNNING,
			ActionId:      slot.ActionID,
			WorkflowRunId: workflowRunId,
			TimeoutAt:     &slot.TimeoutAt.Time,
			StartedAt:     &slot.InsertedAt.Time,
		}
	}

	for i := len(slots); i < remainingSlots; i++ {
		resp = append(resp, gen.SemaphoreSlots{})
	}

	return &resp
}

func ToWorkerRuntimeInfo(worker *sqlcv1.Worker) *gen.WorkerRuntimeInfo {

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

func ToWorkerSqlc(worker *sqlcv1.Worker, remainingSlots *int, webhookUrl *string, actions []pgtype.Text) *gen.Worker {

	dispatcherId := uuid.MustParse(sqlchelpers.UUIDToStr(worker.DispatcherId))

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
		Metadata: gen.APIResourceMeta{
			Id:        sqlchelpers.UUIDToStr(worker.ID),
			CreatedAt: worker.CreatedAt.Time,
			UpdatedAt: worker.UpdatedAt.Time,
		},
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
		wid := uuid.MustParse(sqlchelpers.UUIDToStr(worker.WebhookId))
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
