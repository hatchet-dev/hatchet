package transformers

import (
	"encoding/json"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/olap"
)

func jsonToMap(jsonStr string) map[string]interface{} {
	result := make(map[string]interface{})
	json.Unmarshal([]byte(jsonStr), &result)
	return result
}

func ToWorkflowRuns(
	wfs []*olap.WorkflowRun,
) gen.V2WorkflowRuns {
	toReturn := make([]gen.V2WorkflowRun, len(wfs))

	for i, wf := range wfs {
		additionalMetadata := jsonToMap(*wf.AdditionalMetadata)

		toReturn[i] = gen.V2WorkflowRun{
			AdditionalMetadata: additionalMetadata,
			CreatedAt:          wf.CreatedAt,
			DisplayName:        *wf.DisplayName,
			Duration:           int(*wf.Duration),
			ErrorMessage:       wf.ErrorMessage,
			FinishedAt:         *wf.FinishedAt,
			Id:                 wf.Id,
			Metadata: gen.APIResourceMeta{
				Id:        wf.TaskId.String(),
				CreatedAt: wf.CreatedAt,
				UpdatedAt: wf.CreatedAt,
			},
			StartedAt: *wf.StartedAt,
			Status:    gen.V2TaskStatus(wf.Status),
			TaskId:    wf.TaskId,
			TenantId:  *wf.TenantId,
			Timestamp: wf.Timestamp,
		}
	}

	return gen.V2WorkflowRuns{
		Rows:       toReturn,
		Pagination: gen.PaginationResponse{},
	}
}

func ToTaskRunEvent(
	events []*olap.TaskRunEvent,
) gen.V2ListStepRunEventsForWorkflowRun {
	toReturn := make([]gen.V2StepRunEvent, len(events))

	for i, event := range events {
		data := jsonToMap(event.Data)
		taskInput := jsonToMap(event.TaskInput)
		additionalMetadata := jsonToMap(event.AdditionalMetadata)

		eventTypePtr := gen.V2EventType(event.EventType)

		toReturn[i] = gen.V2StepRunEvent{
			Data:               &data,
			ErrorMessage:       &event.ErrorMsg,
			EventType:          &eventTypePtr,
			Id:                 event.Id,
			Message:            event.Message,
			TaskId:             event.TaskId,
			Timestamp:          event.Timestamp,
			WorkerId:           event.WorkerId,
			TaskDisplayName:    &event.TaskDisplayName,
			TaskInput:          &taskInput,
			AdditionalMetadata: &additionalMetadata,
		}
	}

	return gen.V2ListStepRunEventsForWorkflowRun{
		Rows:       &toReturn,
		Pagination: &gen.PaginationResponse{},
	}
}
