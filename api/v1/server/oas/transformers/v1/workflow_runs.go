package transformers

import (
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func WorkflowRunDataToV1TaskSummary(task *v1.WorkflowRunData) gen.V1TaskSummary {
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
	}
}

func ToWorkflowRunMany(
	tasks []*v1.WorkflowRunData,
	dagExternalIdToChildren map[uuid.UUID][]gen.V1TaskSummary,
	total int, limit, offset int64,
) gen.V1TaskSummaryList {
	toReturn := make([]gen.V1TaskSummary, len(tasks))

	for i, task := range tasks {
		dagExternalId := uuid.MustParse(sqlchelpers.UUIDToStr(task.ExternalID))
		toReturn[i] = WorkflowRunDataToV1TaskSummary(task)

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

func PopulateTaskRunDataRowToV1TaskSummary(task *sqlcv1.PopulateTaskRunDataRow) gen.V1TaskSummary {
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
		WorkflowVersionId:  nil,
		Children:           nil,
		TaskExternalId:     uuid.MustParse(sqlchelpers.UUIDToStr(task.ExternalID)),
		TaskId:             int(task.ID),
		TaskInsertedAt:     task.InsertedAt.Time,
		Type:               gen.V1WorkflowTypeTASK,
	}
}

func TaskRunDataRowToWorkflowRunsMany(
	tasks []*sqlcv1.PopulateTaskRunDataRow,
	total int, limit, offset int64,
) gen.V1TaskSummaryList {
	toReturn := make([]gen.V1TaskSummary, len(tasks))

	for i, task := range tasks {
		toReturn[i] = PopulateTaskRunDataRowToV1TaskSummary(task)
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
