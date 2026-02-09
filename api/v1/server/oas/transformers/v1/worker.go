package transformers

import (
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func ToSlotState(slots []*sqlcv1.ListSemaphoreSlotsWithStateForWorkerRow, remainingSlots int) *[]gen.SemaphoreSlots {
	resp := make([]gen.SemaphoreSlots, len(slots))

	for i := range slots {
		slot := slots[i]

		var stepRunId uuid.UUID
		var workflowRunId uuid.UUID

		if slot.ExternalID != uuid.Nil {
			stepRunId = slot.ExternalID
			workflowRunId = slot.ExternalID
		}

		status := gen.StepRunStatusRUNNING

		resp[i] = gen.SemaphoreSlots{
			StepRunId:     stepRunId,
			Status:        &status,
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

func ToWorkerSqlc(worker *sqlcv1.Worker, remainingSlots *int, webhookUrl *string, actions []string, workflows *[]*sqlcv1.Workflow) *gen.Worker {

	dispatcherId := worker.DispatcherId

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
			Id:        worker.ID.String(),
			CreatedAt: worker.CreatedAt.Time,
			UpdatedAt: worker.UpdatedAt.Time,
		},
		Name:          worker.Name,
		Type:          gen.WorkerType(worker.Type),
		Status:        &status,
		DispatcherId:  dispatcherId,
		MaxRuns:       &maxRuns,
		AvailableRuns: &availableRuns,
		WebhookUrl:    webhookUrl,
		RuntimeInfo:   ToWorkerRuntimeInfo(worker),
		WebhookId:     worker.WebhookId,
	}

	if !worker.LastHeartbeatAt.Time.IsZero() {
		res.LastHeartbeatAt = &worker.LastHeartbeatAt.Time
	}

	res.Actions = &actions

	if workflows != nil {
		registeredWorkflows := make([]gen.RegisteredWorkflow, 0, len(*workflows))
		uniqueWorkflowIds := make(map[uuid.UUID]struct{})

		for _, workflow := range *workflows {
			if _, ok := uniqueWorkflowIds[workflow.ID]; ok {
				continue
			}

			uniqueWorkflowIds[workflow.ID] = struct{}{}
			registeredWorkflows = append(registeredWorkflows, gen.RegisteredWorkflow{
				Id:   workflow.ID,
				Name: workflow.Name,
			})
		}

		res.RegisteredWorkflows = &registeredWorkflows
	}

	return res
}
