package transformers

import (
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func WorkflowRunDataToV1TaskSummary(task *v1.WorkflowRunData, workflowIdsToNames map[pgtype.UUID]string, actionId string) gen.V1TaskSummary {
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

	input := jsonToMap(task.Input)

	var output map[string]interface{}

	if task.Output != nil {
		output = jsonToMap(*task.Output)
	}

	workflowVersionId := uuid.MustParse(sqlchelpers.UUIDToStr(task.WorkflowVersionId))

	var taskId int
	if task.TaskId != nil {
		taskId = int(*task.TaskId)
	}

	var stepId uuid.UUID
	if task.StepId != nil {
		stepId = uuid.MustParse(sqlchelpers.UUIDToStr(*task.StepId))
	} else {
		stepId = uuid.Nil
	}

	var workflowName *string

	if name, ok := workflowIdsToNames[task.WorkflowID]; ok {
		workflowName = &name
	}

	return gen.V1TaskSummary{
		Metadata: gen.APIResourceMeta{
			Id:        sqlchelpers.UUIDToStr(task.ExternalID),
			CreatedAt: task.InsertedAt.Time,
			UpdatedAt: task.InsertedAt.Time,
		},
		CreatedAt:          task.CreatedAt.Time,
		DisplayName:        task.DisplayName,
		Duration:           durationPtr,
		StartedAt:          startedAt,
		FinishedAt:         finishedAt,
		Input:              input,
		Output:             output,
		AdditionalMetadata: &additionalMetadata,
		ErrorMessage:       &task.ErrorMessage,
		Status:             gen.V1TaskStatus(task.ReadableStatus),
		TenantId:           uuid.MustParse(sqlchelpers.UUIDToStr(task.TenantID)),
		WorkflowId:         uuid.MustParse(sqlchelpers.UUIDToStr(task.WorkflowID)),
		WorkflowVersionId:  &workflowVersionId,
		TaskExternalId:     uuid.MustParse(sqlchelpers.UUIDToStr(task.ExternalID)),
		TaskId:             taskId,
		TaskInsertedAt:     task.InsertedAt.Time,
		Type:               gen.V1WorkflowTypeDAG,
		WorkflowName:       workflowName,
		StepId:             &stepId,
		ActionId:           &actionId,
	}
}

func ToWorkflowRunMany(
	tasks []*v1.WorkflowRunData,
	dagExternalIdToChildren map[uuid.UUID][]gen.V1TaskSummary,
	taskIdToActionId map[int64]string,
	workflowIdsToNames map[pgtype.UUID]string,
	total int, limit, offset int64,
) gen.V1TaskSummaryList {
	toReturn := make([]gen.V1TaskSummary, len(tasks))

	for i, task := range tasks {
		dagExternalId := uuid.MustParse(sqlchelpers.UUIDToStr(task.ExternalID))

		actionId := ""

		if task.TaskId != nil {
			actionIdFromMap, ok := taskIdToActionId[*task.TaskId]

			if ok {
				actionId = actionIdFromMap
			}
		}

		toReturn[i] = WorkflowRunDataToV1TaskSummary(task, workflowIdsToNames, actionId)

		children, ok := dagExternalIdToChildren[dagExternalId]

		if ok {
			toReturn[i].Children = &children
		}
	}

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

func PopulateTaskRunDataRowToV1TaskSummary(task *sqlcv1.PopulateTaskRunDataRow, workflowName *string) gen.V1TaskSummary {
	workflowVersionId := uuid.MustParse(sqlchelpers.UUIDToStr(task.WorkflowVersionID))
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

	input := jsonToMap(task.Input)
	output := jsonToMap(task.Output)
	stepId := uuid.MustParse(sqlchelpers.UUIDToStr(task.StepID))

	return gen.V1TaskSummary{
		Metadata: gen.APIResourceMeta{
			Id:        sqlchelpers.UUIDToStr(task.ExternalID),
			CreatedAt: task.InsertedAt.Time,
			UpdatedAt: task.InsertedAt.Time,
		},
		CreatedAt:          task.InsertedAt.Time,
		DisplayName:        task.DisplayName,
		Duration:           durationPtr,
		StartedAt:          startedAt,
		FinishedAt:         finishedAt,
		Input:              input,
		Output:             output,
		AdditionalMetadata: &additionalMetadata,
		ErrorMessage:       &task.ErrorMessage.String,
		Status:             gen.V1TaskStatus(task.Status),
		TenantId:           uuid.MustParse(sqlchelpers.UUIDToStr(task.TenantID)),
		WorkflowId:         uuid.MustParse(sqlchelpers.UUIDToStr(task.WorkflowID)),
		WorkflowVersionId:  &workflowVersionId,
		Children:           nil,
		TaskExternalId:     uuid.MustParse(sqlchelpers.UUIDToStr(task.ExternalID)),
		TaskId:             int(task.ID),
		TaskInsertedAt:     task.InsertedAt.Time,
		Type:               gen.V1WorkflowTypeTASK,
		WorkflowName:       workflowName,
		StepId:             &stepId,
		ActionId:           &task.ActionID,
	}
}

func TaskRunDataRowToWorkflowRunsMany(
	tasks []*sqlcv1.PopulateTaskRunDataRow,
	taskIdToWorkflowName map[int64]string,
	total int, limit, offset int64,
) gen.V1TaskSummaryList {
	toReturn := make([]gen.V1TaskSummary, len(tasks))

	for i, task := range tasks {
		workflowName := taskIdToWorkflowName[task.ID]
		toReturn[i] = PopulateTaskRunDataRowToV1TaskSummary(task, &workflowName)
	}

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

func ToWorkflowRunDisplayNamesList(
	displayNames []*sqlcv1.ListWorkflowRunDisplayNamesRow,
) gen.V1WorkflowRunDisplayNameList {
	result := make([]gen.V1WorkflowRunDisplayName, len(displayNames))

	for i, record := range displayNames {
		result[i] = gen.V1WorkflowRunDisplayName{
			DisplayName: record.DisplayName,
			Metadata: gen.APIResourceMeta{
				Id:        sqlchelpers.UUIDToStr(record.ExternalID),
				CreatedAt: record.InsertedAt.Time,
				UpdatedAt: record.InsertedAt.Time,
			},
		}
	}

	page := int64(1)

	return gen.V1WorkflowRunDisplayNameList{
		Rows: result,
		Pagination: gen.PaginationResponse{
			CurrentPage: &page,
			NextPage:    nil,
			NumPages:    &page,
		},
	}
}
