package transformers

import (
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func ToWorkflowRun(task *repository.WorkflowRunData) gen.V2WorkflowRun {
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

	return gen.V2WorkflowRun{
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
		ErrorMessage:       &task.ErrorMessage,
		Status:             gen.V2TaskStatus(task.ReadableStatus),
		TenantId:           uuid.MustParse(sqlchelpers.UUIDToStr(task.TenantID)),
		WorkflowId:         uuid.MustParse(sqlchelpers.UUIDToStr(task.WorkflowID)),
	}
}

func ToWorkflowRunMany(
	tasks []*repository.WorkflowRunData,
	total int, limit, offset int64,
) gen.V2WorkflowRunList {
	toReturn := make([]gen.V2WorkflowRun, len(tasks))

	for i, task := range tasks {
		toReturn[i] = ToWorkflowRun(task)
	}

	currentPage := (offset / limit) + 1
	nextPage := currentPage + 1
	numPages := int64(math.Ceil(float64(total) / float64(limit)))

	return gen.V2WorkflowRunList{
		Rows: toReturn,
		Pagination: gen.PaginationResponse{
			CurrentPage: &currentPage,
			NextPage:    &nextPage,
			NumPages:    &numPages,
		},
	}
}
