package transformers

import (
	"encoding/json"
	"math"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/olap"
)

func jsonToMap(jsonStr string) map[string]interface{} {
	result := make(map[string]interface{})
	json.Unmarshal([]byte(jsonStr), &result)
	return result
}

func ToWorkflowRun(wf *olap.WorkflowRun) gen.V2WorkflowRun {
	additionalMetadata := jsonToMap(*wf.AdditionalMetadata)
	input := jsonToMap(wf.Input)
	output := jsonToMap(wf.Output)

	return gen.V2WorkflowRun{
		AdditionalMetadata: additionalMetadata,
		CreatedAt:          wf.CreatedAt,
		DisplayName:        *wf.DisplayName,
		Duration:           int(*wf.Duration),
		ErrorMessage:       wf.ErrorMessage,
		FinishedAt:         *wf.FinishedAt,
		Id:                 wf.Id,
		Input:              input,
		Output:             &output,
		Metadata: gen.APIResourceMeta{
			Id:        wf.TaskId.String(),
			CreatedAt: wf.CreatedAt,
			UpdatedAt: wf.CreatedAt,
		},
		StartedAt:  *wf.StartedAt,
		Status:     gen.V2TaskStatus(wf.Status),
		TaskId:     wf.TaskId,
		TenantId:   *wf.TenantId,
		Timestamp:  wf.Timestamp,
		WorkflowId: &wf.WorkflowId,
	}
}

func ToWorkflowRuns(
	wfs []*olap.WorkflowRun,
	total uint64, limit, offset int64,
) gen.V2WorkflowRuns {
	toReturn := make([]gen.V2WorkflowRun, len(wfs))

	for i, wf := range wfs {
		toReturn[i] = ToWorkflowRun(wf)
	}

	currentPage := (offset / limit) + 1
	nextPage := currentPage + 1
	numPages := int64(math.Ceil(float64(total) / float64(limit)))

	return gen.V2WorkflowRuns{
		Rows: toReturn,
		Pagination: gen.PaginationResponse{
			CurrentPage: &currentPage,
			NextPage:    &nextPage,
			NumPages:    &numPages,
		},
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

func ToTaskRunMetrics(metrics *[]olap.TaskRunMetric) gen.V2TaskRunMetrics {
	statuses := []gen.V2TaskStatus{
		gen.V2TaskStatusCANCELLED,
		gen.V2TaskStatusCOMPLETED,
		gen.V2TaskStatusFAILED,
		gen.V2TaskStatusQUEUED,
		gen.V2TaskStatusRUNNING,
	}

	toReturn := make([]gen.V2TaskRunMetric, len(statuses))

	for i, status := range statuses {
		metric := olap.TaskRunMetric{Count: 0}

		for _, m := range *metrics {
			if m.Status == string(status) {
				metric = m
				break
			}
		}

		toReturn[i] = gen.V2TaskRunMetric{
			Count:  int(metric.Count),
			Status: status,
		}
	}

	return toReturn
}
