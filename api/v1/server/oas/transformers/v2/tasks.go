package transformers

import (
	"encoding/json"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/olap"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v2/timescalev2"
	"github.com/oapi-codegen/runtime/types"
)

func jsonToMap(jsonBytes []byte) map[string]interface{} {
	result := make(map[string]interface{})
	json.Unmarshal(jsonBytes, &result)
	return result
}

func ToTaskSummary(task *timescalev2.PopulateTaskRunDataRow) gen.V2TaskSummary {
	additionalMetadata := jsonToMap(task.AdditionalMetadata)

	var finishedAt *time.Time

	if task.FinishedAt.Valid {
		finishedAt = &task.FinishedAt.Time
	}

	var startedAt *time.Time

	if task.StartedAt.Valid {
		startedAt = &task.StartedAt.Time
	}

	var durationPtr *int

	if task.FinishedAt.Valid && task.StartedAt.Valid {
		duration := int(task.FinishedAt.Time.Sub(task.StartedAt.Time).Milliseconds())
		durationPtr = &duration
	}

	return gen.V2TaskSummary{
		Metadata: gen.APIResourceMeta{
			Id:        sqlchelpers.UUIDToStr(task.ExternalID),
			CreatedAt: task.InsertedAt.Time,
			UpdatedAt: task.InsertedAt.Time,
		},
		DisplayName:        task.DisplayName,
		Duration:           durationPtr,
		StartedAt:          startedAt,
		FinishedAt:         finishedAt,
		AdditionalMetadata: &additionalMetadata,
		ErrorMessage:       &task.ErrorMessage.String,
		Status:             gen.V2TaskStatus(task.Status),
		TenantId:           uuid.MustParse(sqlchelpers.UUIDToStr(task.TenantID)),
		WorkflowId:         uuid.MustParse(sqlchelpers.UUIDToStr(task.WorkflowID)),
		TaskId:             int(task.ID),
		TaskInsertedAt:     task.InsertedAt.Time,
	}
}

func ToTaskSummaryRows(
	tasks []*timescalev2.PopulateTaskRunDataRow,
) []gen.V2TaskSummary {
	toReturn := make([]gen.V2TaskSummary, len(tasks))

	for i, task := range tasks {
		toReturn[i] = ToTaskSummary(task)
	}

	return toReturn
}

func ToDagChildren(
	tasks []*timescalev2.PopulateTaskRunDataRow,
) []gen.V2DagChildren {
	dagIdToTasks := make(map[string][]gen.V2TaskSummary)

	for _, task := range tasks {
		dagId := sqlchelpers.UUIDToStr(task.DagExternalID)
		dagIdToTasks[dagId] = append(dagIdToTasks[dagId], ToTaskSummary(task))
	}

	toReturn := make([]gen.V2DagChildren, 0, len(dagIdToTasks))

	for dagId, tasks := range dagIdToTasks {
		parsedUUID := types.UUID(uuid.MustParse(dagId))
		toReturn = append(toReturn, gen.V2DagChildren{
			DagId:    &parsedUUID,
			Children: &tasks,
		})
	}

	return toReturn
}

func ToTaskSummaryMany(
	tasks []*timescalev2.PopulateTaskRunDataRow,
	total int, limit, offset int64,
) gen.V2TaskSummaryList {
	toReturn := ToTaskSummaryRows(tasks)

	currentPage := (offset / limit) + 1
	nextPage := currentPage + 1
	numPages := int64(math.Ceil(float64(total) / float64(limit)))

	return gen.V2TaskSummaryList{
		Rows: toReturn,
		Pagination: gen.PaginationResponse{
			CurrentPage: &currentPage,
			NextPage:    &nextPage,
			NumPages:    &numPages,
		},
	}
}

func ToTaskRunEventMany(
	events []*timescalev2.ListTaskEventsRow,
	taskExternalId string,
) gen.V2TaskEventList {
	toReturn := make([]gen.V2TaskEvent, len(events))

	for i, event := range events {
		// data := jsonToMap(event.Data)
		// taskInput := jsonToMap(event.TaskInput)
		// additionalMetadata := jsonToMap(event.AdditionalMetadata)

		var workerId *types.UUID

		if event.WorkerID.Valid {
			workerUUid := uuid.MustParse(sqlchelpers.UUIDToStr(event.WorkerID))
			workerId = (*types.UUID)(&workerUUid)
		}

		toReturn[i] = gen.V2TaskEvent{
			Id:           int(event.ID),
			ErrorMessage: &event.ErrorMessage.String,
			EventType:    gen.V2TaskEventType(event.EventType),
			Message:      event.AdditionalEventMessage.String,
			Timestamp:    event.EventTimestamp.Time,
			WorkerId:     workerId,
			TaskId:       uuid.MustParse(taskExternalId),
			// TaskInput:    &taskInput,
		}
	}

	return gen.V2TaskEventList{
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

func ToTask(taskWithData *timescalev2.PopulateSingleTaskRunDataRow) gen.V2Task {
	additionalMetadata := jsonToMap(taskWithData.AdditionalMetadata)

	var finishedAt *time.Time

	if taskWithData.FinishedAt.Valid {
		finishedAt = &taskWithData.FinishedAt.Time
	}

	var startedAt *time.Time

	if taskWithData.StartedAt.Valid {
		startedAt = &taskWithData.StartedAt.Time
	}

	var durationPtr *int

	if taskWithData.FinishedAt.Valid && taskWithData.StartedAt.Valid {
		duration := int(taskWithData.FinishedAt.Time.Sub(taskWithData.StartedAt.Time).Milliseconds())
		durationPtr = &duration
	}

	var output *string

	if taskWithData.Output != nil {
		outputStr := string(taskWithData.Output)
		output = &outputStr
	}

	return gen.V2Task{
		Metadata: gen.APIResourceMeta{
			Id:        sqlchelpers.UUIDToStr(taskWithData.ExternalID),
			CreatedAt: taskWithData.InsertedAt.Time,
			UpdatedAt: taskWithData.InsertedAt.Time,
		},
		TaskId:             int(taskWithData.ID),
		TaskInsertedAt:     taskWithData.InsertedAt.Time,
		DisplayName:        taskWithData.DisplayName,
		AdditionalMetadata: &additionalMetadata,
		Duration:           durationPtr,
		StartedAt:          startedAt,
		FinishedAt:         finishedAt,
		Output:             output,
		Status:             gen.V2TaskStatus(taskWithData.Status),
		Input:              string(taskWithData.Input),
		TenantId:           uuid.MustParse(sqlchelpers.UUIDToStr(taskWithData.TenantID)),
		WorkflowId:         uuid.MustParse(sqlchelpers.UUIDToStr(taskWithData.WorkflowID)),
		ErrorMessage:       &taskWithData.ErrorMessage.String,
	}
}
