package transformers

import (
	"encoding/json"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func jsonToMap(jsonBytes []byte) map[string]interface{} {
	result := make(map[string]interface{})
	json.Unmarshal(jsonBytes, &result) // nolint: errcheck
	return result
}

func ToTaskSummary(task *v1.TaskWithPayloads) gen.V1TaskSummary {
	workflowVersionID := task.WorkflowVersionID
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

	taskExternalId := task.ExternalID
	stepId := task.StepID

	retryCount := int(task.RetryCount)
	attempt := retryCount + 1
	return gen.V1TaskSummary{
		Metadata: gen.APIResourceMeta{
			Id:        task.ExternalID.String(),
			CreatedAt: task.InsertedAt.Time,
			UpdatedAt: task.InsertedAt.Time,
		},
		Input:                 jsonToMap(task.InputPayload),
		Output:                jsonToMap(task.OutputPayload),
		Type:                  gen.V1WorkflowTypeTASK,
		DisplayName:           task.DisplayName,
		Duration:              durationPtr,
		StartedAt:             startedAt,
		FinishedAt:            finishedAt,
		AdditionalMetadata:    &additionalMetadata,
		ErrorMessage:          &task.ErrorMessage.String,
		Status:                gen.V1TaskStatus(task.Status),
		TenantId:              task.TenantID,
		WorkflowId:            task.WorkflowID,
		TaskId:                int(task.ID),
		TaskInsertedAt:        task.InsertedAt.Time,
		TaskExternalId:        taskExternalId,
		StepId:                &stepId,
		ActionId:              &task.ActionID,
		WorkflowRunExternalId: task.WorkflowRunID,
		WorkflowVersionId:     &workflowVersionID,
		RetryCount:            &retryCount,
		Attempt:               &attempt,
		ParentTaskExternalId:  task.ParentTaskExternalID,
	}
}

func ToTaskSummaryRows(
	tasks []*v1.TaskWithPayloads,
) []gen.V1TaskSummary {
	toReturn := make([]gen.V1TaskSummary, len(tasks))

	for i, task := range tasks {
		toReturn[i] = ToTaskSummary(task)
	}

	return toReturn
}

func ToDagChildren(
	tasks []*v1.TaskWithPayloads,
	taskIdToDagExternalId map[int64]uuid.UUID,
) []gen.V1DagChildren {
	dagIdToTasks := make(map[uuid.UUID][]gen.V1TaskSummary)

	for _, task := range tasks {
		dagId := taskIdToDagExternalId[task.ID]

		dagIdToTasks[dagId] = append(dagIdToTasks[dagId], ToTaskSummary(task))
	}

	toReturn := make([]gen.V1DagChildren, 0, len(dagIdToTasks))

	for dagId, tasks := range dagIdToTasks {
		dagIdCp := dagId
		tasksCp := tasks

		toReturn = append(toReturn, gen.V1DagChildren{
			DagId:    &dagIdCp,
			Children: &tasksCp,
		})
	}

	return toReturn
}

func ToTaskSummaryMany(
	tasks []*v1.TaskWithPayloads,
	total int, limit, offset int64,
) gen.V1TaskSummaryList {
	toReturn := ToTaskSummaryRows(tasks)

	currentPage := (offset / limit) + 1
	nextPage := currentPage + 1
	numPages := int64(math.Ceil(float64(total) / float64(limit)))

	return gen.V1TaskSummaryList{
		Rows: toReturn,
		Pagination: gen.PaginationResponse{
			CurrentPage: &currentPage,
			NextPage:    &nextPage,
			NumPages:    &numPages,
		},
	}
}

func ToTaskRunEventMany(
	events []*sqlcv1.ListTaskEventsRow,
	taskExternalId uuid.UUID,
) gen.V1TaskEventList {
	toReturn := make([]gen.V1TaskEvent, len(events))

	for i, event := range events {
		retryCount := int(event.RetryCount)
		attempt := retryCount + 1

		toReturn[i] = gen.V1TaskEvent{
			Id:           int(event.ID),
			ErrorMessage: &event.ErrorMessage.String,
			EventType:    gen.V1TaskEventType(event.EventType),
			Message:      event.AdditionalEventMessage.String,
			Timestamp:    event.EventTimestamp.Time,
			WorkerId:     event.WorkerID,
			TaskId:       taskExternalId,
			RetryCount:   &retryCount,
			Attempt:      &attempt,
		}
	}

	return gen.V1TaskEventList{
		Rows:       &toReturn,
		Pagination: &gen.PaginationResponse{},
	}
}

func ToWorkflowRunTaskRunEventsMany(
	events []*v1.TaskEventWithPayloads,
) gen.V1TaskEventList {
	toReturn := make([]gen.V1TaskEvent, len(events))

	for i, event := range events {
		output := string(event.OutputPayload)
		retryCount := int(event.RetryCount)
		attempt := retryCount + 1

		toReturn[i] = gen.V1TaskEvent{
			ErrorMessage:    &event.ErrorMessage.String,
			EventType:       gen.V1TaskEventType(event.EventType),
			Id:              int(event.ID),
			Message:         event.AdditionalEventMessage.String,
			Output:          &output,
			TaskDisplayName: &event.DisplayName,
			TaskId:          event.TaskExternalID,
			Timestamp:       event.EventTimestamp.Time,
			WorkerId:        event.WorkerID,
			RetryCount:      &retryCount,
			Attempt:         &attempt,
		}
	}

	return gen.V1TaskEventList{
		Rows:       &toReturn,
		Pagination: &gen.PaginationResponse{},
	}
}

func ToTaskRunMetrics(metrics *[]v1.TaskRunMetric) gen.V1TaskRunMetrics {
	statuses := []gen.V1TaskStatus{
		gen.V1TaskStatusCANCELLED,
		gen.V1TaskStatusCOMPLETED,
		gen.V1TaskStatusFAILED,
		gen.V1TaskStatusQUEUED,
		gen.V1TaskStatusRUNNING,
	}

	toReturn := make([]gen.V1TaskRunMetric, len(statuses))

	for i, status := range statuses {
		metric := v1.TaskRunMetric{Count: 0}

		for _, m := range *metrics {
			if m.Status == string(status) {
				metric = m
				break
			}
		}

		toReturn[i] = gen.V1TaskRunMetric{
			Count:  int(metric.Count), // nolint: gosec
			Status: status,
		}
	}

	return toReturn
}

func ToTask(taskWithData *v1.TaskWithPayloads, workflowRunExternalId uuid.UUID, workflowVersion *sqlcv1.GetWorkflowVersionByIdRow) gen.V1TaskSummary {
	workflowVersionID := taskWithData.WorkflowVersionID
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

	output := make(map[string]interface{})

	if len(taskWithData.OutputPayload) > 0 {
		output = jsonToMap(taskWithData.OutputPayload)
	}

	input := jsonToMap(taskWithData.InputPayload)

	stepId := taskWithData.StepID

	retryCount := int(taskWithData.RetryCount)
	attempt := retryCount + 1

	workflowConfig := make(map[string]interface{})

	if workflowVersion.WorkflowVersion.CreateWorkflowVersionOpts != nil {
		workflowConfig = jsonToMap(workflowVersion.WorkflowVersion.CreateWorkflowVersionOpts)
	}

	var parentTaskExternalId *uuid.UUID

	if taskWithData.ParentTaskExternalID != nil {
		parentTaskUUID, err := uuid.Parse(taskWithData.ParentTaskExternalID.String())

		if err == nil {
			parentTaskExternalId = &parentTaskUUID
		}
	}

	return gen.V1TaskSummary{
		Metadata: gen.APIResourceMeta{
			Id:        taskWithData.ExternalID.String(),
			CreatedAt: taskWithData.InsertedAt.Time,
			UpdatedAt: taskWithData.InsertedAt.Time,
		},
		TaskId:                int(taskWithData.ID),
		TaskInsertedAt:        taskWithData.InsertedAt.Time,
		DisplayName:           taskWithData.DisplayName,
		AdditionalMetadata:    &additionalMetadata,
		Duration:              durationPtr,
		StartedAt:             startedAt,
		FinishedAt:            finishedAt,
		Output:                output,
		Status:                gen.V1TaskStatus(taskWithData.Status),
		Input:                 input,
		TenantId:              taskWithData.TenantID,
		WorkflowId:            taskWithData.WorkflowID,
		ErrorMessage:          &taskWithData.ErrorMessage.String,
		WorkflowRunExternalId: workflowRunExternalId,
		TaskExternalId:        taskWithData.ExternalID,
		Type:                  gen.V1WorkflowTypeTASK,
		NumSpawnedChildren:    int(taskWithData.NumSpawnedChildren),
		StepId:                &stepId,
		ActionId:              &taskWithData.ActionID,
		WorkflowVersionId:     &workflowVersionID,
		RetryCount:            &retryCount,
		Attempt:               &attempt,
		WorkflowConfig:        &workflowConfig,
		ParentTaskExternalId:  parentTaskExternalId,
	}
}

func ToWorkflowRunDetails(
	taskRunEvents []*v1.TaskEventWithPayloads,
	workflowRun *v1.WorkflowRunData,
	shape []*sqlcv1.GetWorkflowShapeRow,
	tasks []*v1.TaskWithPayloads,
	stepIdToTaskExternalId map[uuid.UUID]uuid.UUID,
	workflowVersion *sqlcv1.GetWorkflowVersionByIdRow,
) (gen.V1WorkflowRunDetails, error) {
	workflowVersionId := workflowRun.WorkflowVersionId
	duration := int(workflowRun.FinishedAt.Time.Sub(workflowRun.StartedAt.Time).Milliseconds())
	input := jsonToMap(workflowRun.Input)

	output := make(map[string]interface{})

	if len(workflowRun.Output) > 0 {
		output = jsonToMap(workflowRun.Output)
	}

	additionalMetadata := jsonToMap(workflowRun.AdditionalMetadata)

	parsedWorkflowRun := gen.V1WorkflowRun{
		AdditionalMetadata:   &additionalMetadata,
		CreatedAt:            &workflowRun.CreatedAt.Time,
		DisplayName:          workflowRun.DisplayName,
		Duration:             &duration,
		ErrorMessage:         &workflowRun.ErrorMessage,
		FinishedAt:           &workflowRun.FinishedAt.Time,
		ParentTaskExternalId: workflowRun.ParentTaskExternalId,
		Metadata: gen.APIResourceMeta{
			Id:        workflowRun.ExternalID.String(),
			CreatedAt: workflowRun.InsertedAt.Time,
			UpdatedAt: workflowRun.InsertedAt.Time,
		},
		StartedAt:         &workflowRun.StartedAt.Time,
		Status:            gen.V1TaskStatus(workflowRun.ReadableStatus),
		TenantId:          workflowRun.TenantID,
		WorkflowId:        workflowRun.WorkflowID,
		WorkflowVersionId: &workflowVersionId,
		Input:             input,
		Output:            output,
	}

	shapeRows := make([]gen.WorkflowRunShapeItemForWorkflowRunDetails, len(shape))

	for i, shapeRow := range shape {
		parentExternalId := stepIdToTaskExternalId[shapeRow.Parentstepid]
		taskName := shapeRow.Stepname.String
		stepId := shapeRow.Parentstepid

		shapeRows[i] = gen.WorkflowRunShapeItemForWorkflowRunDetails{
			ChildrenStepIds: shapeRow.Childrenstepids,
			TaskExternalId:  parentExternalId,
			TaskName:        taskName,
			StepId:          stepId,
		}
	}

	parsedTaskEvents := make([]gen.V1TaskEvent, len(taskRunEvents))

	for i, event := range taskRunEvents {
		output := string(event.OutputPayload)
		retryCount := int(event.RetryCount)
		attempt := retryCount + 1

		parsedTaskEvents[i] = gen.V1TaskEvent{
			ErrorMessage:    &event.ErrorMessage.String,
			EventType:       gen.V1TaskEventType(event.EventType),
			Id:              int(event.ID),
			Message:         event.AdditionalEventMessage.String,
			Output:          &output,
			TaskDisplayName: &event.DisplayName,
			Timestamp:       event.EventTimestamp.Time,
			WorkerId:        event.WorkerID,
			TaskId:          event.TaskExternalID,
			RetryCount:      &retryCount,
			Attempt:         &attempt,
		}
	}

	parsedTasks := ToTaskSummaryRows(tasks)

	workflowConfig := make(map[string]interface{})

	if workflowVersion.WorkflowVersion.CreateWorkflowVersionOpts != nil {
		workflowConfig = jsonToMap(workflowVersion.WorkflowVersion.CreateWorkflowVersionOpts)
	}

	return gen.V1WorkflowRunDetails{
		Run:            parsedWorkflowRun,
		Shape:          shapeRows,
		TaskEvents:     parsedTaskEvents,
		Tasks:          parsedTasks,
		WorkflowConfig: &workflowConfig,
	}, nil
}

func ToTaskTimings(
	timings []*sqlcv1.PopulateTaskRunDataRow,
	idsToDepth map[uuid.UUID]int32,
) []gen.V1TaskTiming {
	toReturn := make([]gen.V1TaskTiming, len(timings))

	for i, timing := range timings {
		depth := idsToDepth[timing.ExternalID]

		workflowRunId := timing.WorkflowRunID
		retryCount := int(timing.RetryCount)
		attempt := retryCount + 1

		toReturn[i] = gen.V1TaskTiming{
			Metadata: gen.APIResourceMeta{
				Id:        timing.ExternalID.String(),
				CreatedAt: timing.InsertedAt.Time,
				UpdatedAt: timing.InsertedAt.Time,
			},
			Status:               gen.V1TaskStatus(timing.Status),
			TaskDisplayName:      timing.DisplayName,
			TaskId:               int(timing.ID),
			TaskInsertedAt:       timing.InsertedAt.Time,
			TaskExternalId:       timing.ExternalID,
			TenantId:             timing.TenantID,
			Depth:                int(depth),
			WorkflowRunId:        &workflowRunId,
			RetryCount:           &retryCount,
			Attempt:              &attempt,
			ParentTaskExternalId: timing.ParentTaskExternalID,
		}

		if timing.QueuedAt.Valid {
			toReturn[i].QueuedAt = &timing.QueuedAt.Time
		}

		if timing.StartedAt.Valid {
			toReturn[i].StartedAt = &timing.StartedAt.Time
		}

		if timing.FinishedAt.Valid {
			toReturn[i].FinishedAt = &timing.FinishedAt.Time
		}
	}

	return toReturn
}

func ToCancelledOrReplayedTaskResponse(ids []string) gen.V1ReplayedTasks {
	idUuids := make([]types.UUID, len(ids))

	for i, id := range ids {
		idUuids[i] = uuid.MustParse(id)
	}

	return gen.V1ReplayedTasks{
		Ids: &idUuids,
	}
}
