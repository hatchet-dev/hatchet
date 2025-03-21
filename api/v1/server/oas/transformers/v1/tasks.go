package transformers

import (
	"encoding/json"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/oapi-codegen/runtime/types"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func jsonToMap(jsonBytes []byte) map[string]interface{} {
	result := make(map[string]interface{})
	json.Unmarshal(jsonBytes, &result) // nolint: errcheck
	return result
}

func ToTaskSummary(task *sqlcv1.PopulateTaskRunDataRow) gen.V1TaskSummary {
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

	taskExternalId := uuid.MustParse(sqlchelpers.UUIDToStr(task.ExternalID))
	stepId := uuid.MustParse(sqlchelpers.UUIDToStr(task.StepID))

	return gen.V1TaskSummary{
		Metadata: gen.APIResourceMeta{
			Id:        sqlchelpers.UUIDToStr(task.ExternalID),
			CreatedAt: task.InsertedAt.Time,
			UpdatedAt: task.InsertedAt.Time,
		},
		Input:              jsonToMap(task.Input),
		Output:             jsonToMap(task.Output),
		Type:               gen.V1WorkflowTypeTASK,
		DisplayName:        task.DisplayName,
		Duration:           durationPtr,
		StartedAt:          startedAt,
		FinishedAt:         finishedAt,
		AdditionalMetadata: &additionalMetadata,
		ErrorMessage:       &task.ErrorMessage.String,
		Status:             gen.V1TaskStatus(task.Status),
		TenantId:           uuid.MustParse(sqlchelpers.UUIDToStr(task.TenantID)),
		WorkflowId:         uuid.MustParse(sqlchelpers.UUIDToStr(task.WorkflowID)),
		TaskId:             int(task.ID),
		TaskInsertedAt:     task.InsertedAt.Time,
		TaskExternalId:     taskExternalId,
		StepId:             &stepId,
	}
}

func ToTaskSummaryRows(
	tasks []*sqlcv1.PopulateTaskRunDataRow,
) []gen.V1TaskSummary {
	toReturn := make([]gen.V1TaskSummary, len(tasks))

	for i, task := range tasks {
		toReturn[i] = ToTaskSummary(task)
	}

	return toReturn
}

func ToDagChildren(
	tasks []*sqlcv1.PopulateTaskRunDataRow,
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
	tasks []*sqlcv1.PopulateTaskRunDataRow,
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
	taskExternalId string,
) gen.V1TaskEventList {
	toReturn := make([]gen.V1TaskEvent, len(events))

	for i, event := range events {
		// data := jsonToMap(event.Data)
		// taskInput := jsonToMap(event.TaskInput)
		// additionalMetadata := jsonToMap(event.AdditionalMetadata)

		var workerId *types.UUID

		if event.WorkerID.Valid {
			workerUUid := uuid.MustParse(sqlchelpers.UUIDToStr(event.WorkerID))
			workerId = &workerUUid
		}

		toReturn[i] = gen.V1TaskEvent{
			Id:           int(event.ID),
			ErrorMessage: &event.ErrorMessage.String,
			EventType:    gen.V1TaskEventType(event.EventType),
			Message:      event.AdditionalEventMessage.String,
			Timestamp:    event.EventTimestamp.Time,
			WorkerId:     workerId,
			TaskId:       uuid.MustParse(taskExternalId),
			// TaskInput:    &taskInput,
		}
	}

	return gen.V1TaskEventList{
		Rows:       &toReturn,
		Pagination: &gen.PaginationResponse{},
	}
}

func ToWorkflowRunTaskRunEventsMany(
	events []*sqlcv1.ListTaskEventsForWorkflowRunRow,
) gen.V1TaskEventList {
	toReturn := make([]gen.V1TaskEvent, len(events))

	for i, event := range events {
		workerId := uuid.MustParse(sqlchelpers.UUIDToStr(event.WorkerID))
		output := string(event.Output)
		taskExternalId := uuid.MustParse(sqlchelpers.UUIDToStr(event.TaskExternalID))

		toReturn[i] = gen.V1TaskEvent{
			ErrorMessage:    &event.ErrorMessage.String,
			EventType:       gen.V1TaskEventType(event.EventType),
			Id:              int(event.ID),
			Message:         event.AdditionalEventMessage.String,
			Output:          &output,
			TaskDisplayName: &event.DisplayName,
			TaskId:          taskExternalId,
			Timestamp:       event.EventTimestamp.Time,
			WorkerId:        &workerId,
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

func ToTask(taskWithData *sqlcv1.PopulateSingleTaskRunDataRow, workflowRunExternalId *pgtype.UUID) gen.V1TaskSummary {
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

	if taskWithData.Output != nil {
		output = jsonToMap(taskWithData.Output)
	}

	input := jsonToMap(taskWithData.Input)

	var parsedWorkflowRunUUID *uuid.UUID

	if workflowRunExternalId != nil && workflowRunExternalId.Valid {
		id := uuid.MustParse(sqlchelpers.UUIDToStr(*workflowRunExternalId))
		parsedWorkflowRunUUID = &id
	}

	stepId := uuid.MustParse(sqlchelpers.UUIDToStr(taskWithData.StepID))

	return gen.V1TaskSummary{
		Metadata: gen.APIResourceMeta{
			Id:        sqlchelpers.UUIDToStr(taskWithData.ExternalID),
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
		TenantId:              uuid.MustParse(sqlchelpers.UUIDToStr(taskWithData.TenantID)),
		WorkflowId:            uuid.MustParse(sqlchelpers.UUIDToStr(taskWithData.WorkflowID)),
		ErrorMessage:          &taskWithData.ErrorMessage.String,
		WorkflowRunExternalId: parsedWorkflowRunUUID,
		TaskExternalId:        uuid.MustParse(sqlchelpers.UUIDToStr(taskWithData.ExternalID)),
		Type:                  gen.V1WorkflowTypeTASK,
		NumSpawnedChildren:    int(taskWithData.SpawnedChildren.Int64),
		StepId:                &stepId,
	}
}

func ToWorkflowRunDetails(
	taskRunEvents []*sqlcv1.ListTaskEventsForWorkflowRunRow,
	workflowRun *v1.WorkflowRunData,
	shape []*dbsqlc.GetWorkflowRunShapeRow,
	tasks []*sqlcv1.PopulateTaskRunDataRow,
	stepIdToTaskExternalId map[pgtype.UUID]pgtype.UUID,
) (gen.V1WorkflowRunDetails, error) {
	workflowVersionId := uuid.MustParse(sqlchelpers.UUIDToStr(workflowRun.WorkflowVersionId))
	duration := int(workflowRun.FinishedAt.Time.Sub(workflowRun.StartedAt.Time).Milliseconds())
	input := jsonToMap(workflowRun.Input)

	output := make(map[string]interface{})

	if workflowRun.Output != nil {
		output = jsonToMap(*workflowRun.Output)
	}

	additionalMetadata := jsonToMap(workflowRun.AdditionalMetadata)

	parsedWorkflowRun := gen.V1WorkflowRun{
		AdditionalMetadata: &additionalMetadata,
		CreatedAt:          &workflowRun.CreatedAt.Time,
		DisplayName:        workflowRun.DisplayName,
		Duration:           &duration,
		ErrorMessage:       &workflowRun.ErrorMessage,
		FinishedAt:         &workflowRun.FinishedAt.Time,
		Metadata: gen.APIResourceMeta{
			Id:        sqlchelpers.UUIDToStr(workflowRun.ExternalID),
			CreatedAt: workflowRun.InsertedAt.Time,
			UpdatedAt: workflowRun.InsertedAt.Time,
		},
		StartedAt:         &workflowRun.StartedAt.Time,
		Status:            gen.V1TaskStatus(workflowRun.ReadableStatus),
		TenantId:          uuid.MustParse(sqlchelpers.UUIDToStr(workflowRun.TenantID)),
		WorkflowId:        uuid.MustParse(sqlchelpers.UUIDToStr(workflowRun.WorkflowID)),
		WorkflowVersionId: &workflowVersionId,
		Input:             input,
		Output:            output,
	}

	shapeRows := make([]gen.WorkflowRunShapeItemForWorkflowRunDetails, len(shape))

	for i, shapeRow := range shape {
		parentExternalId := uuid.MustParse(sqlchelpers.UUIDToStr(stepIdToTaskExternalId[shapeRow.Parentstepid]))
		ChildrenStepIds := make([]uuid.UUID, len(shapeRow.Childrenstepids))
		taskName := shapeRow.Stepname.String
		stepId := shapeRow.Parentstepid

		for c, child := range shapeRow.Childrenstepids {
			ChildrenStepIds[c] = uuid.MustParse(sqlchelpers.UUIDToStr(child))
		}

		shapeRows[i] = gen.WorkflowRunShapeItemForWorkflowRunDetails{
			ChildrenStepIds: ChildrenStepIds,
			TaskExternalId:  parentExternalId,
			TaskName:        taskName,
			StepId:          uuid.MustParse(sqlchelpers.UUIDToStr(stepId)),
		}
	}

	parsedTaskEvents := make([]gen.V1TaskEvent, len(taskRunEvents))

	for i, event := range taskRunEvents {
		workerId := uuid.MustParse(sqlchelpers.UUIDToStr(event.WorkerID))
		output := string(event.Output)

		parsedTaskEvents[i] = gen.V1TaskEvent{
			ErrorMessage:    &event.ErrorMessage.String,
			EventType:       gen.V1TaskEventType(event.EventType),
			Id:              int(event.ID),
			Message:         event.AdditionalEventMessage.String,
			Output:          &output,
			TaskDisplayName: &event.DisplayName,
			Timestamp:       event.EventTimestamp.Time,
			WorkerId:        &workerId,
			TaskId:          uuid.MustParse(sqlchelpers.UUIDToStr(event.TaskExternalID)),
		}
	}

	parsedTasks := ToTaskSummaryRows(tasks)

	return gen.V1WorkflowRunDetails{
		Run:        parsedWorkflowRun,
		Shape:      shapeRows,
		TaskEvents: parsedTaskEvents,
		Tasks:      parsedTasks,
	}, nil
}
