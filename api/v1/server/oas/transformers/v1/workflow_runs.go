package transformers

import (
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
)

func ToWorkflowRun(task *v1.WorkflowRunData) gen.V1WorkflowRun {
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

	workflowVersionId := uuid.MustParse(sqlchelpers.UUIDToStr(task.WorkflowVersionId))

	return gen.V1WorkflowRun{
		Metadata: gen.APIResourceMeta{
			Id:        sqlchelpers.UUIDToStr(task.ExternalID),
			CreatedAt: task.InsertedAt.Time,
			UpdatedAt: task.InsertedAt.Time,
		},
		CreatedAt:          &task.CreatedAt.Time,
		DisplayName:        task.DisplayName,
		Duration:           durationPtr,
		StartedAt:          startedAt,
		FinishedAt:         finishedAt,
		AdditionalMetadata: &additionalMetadata,
		ErrorMessage:       &task.ErrorMessage,
		Status:             gen.V1TaskStatus(task.ReadableStatus),
		TenantId:           uuid.MustParse(sqlchelpers.UUIDToStr(task.TenantID)),
		WorkflowId:         uuid.MustParse(sqlchelpers.UUIDToStr(task.WorkflowID)),
		WorkflowVersionId:  &workflowVersionId,
	}
}

func ToWorkflowRunMany(
	tasks []*v1.WorkflowRunData,
	total int, limit, offset int64,
) gen.V1WorkflowRunList {
	toReturn := make([]gen.V1WorkflowRun, len(tasks))

	for i, task := range tasks {
		toReturn[i] = ToWorkflowRun(task)
	}

	currentPage := (offset / limit) + 1
	nextPage := currentPage + 1
	numPages := int64(math.Ceil(float64(total) / float64(limit)))

	return gen.V1WorkflowRunList{
		Rows: toReturn,
		Pagination: gen.PaginationResponse{
			CurrentPage: &currentPage,
			NextPage:    &nextPage,
			NumPages:    &numPages,
		},
	}
}
