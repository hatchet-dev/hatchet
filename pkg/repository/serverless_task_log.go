package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// ServerlessTaskLogOpts is one log line the serverless operator writes on a task it delivered.
type ServerlessTaskLogOpts struct {
	Message        string
	Metadata       []byte
	TaskExternalId uuid.UUID
	RetryCount     int32
}

// ServerlessTaskLogRepository writes the operator's own lines into tasks' logs, such as which
// endpoint a task was delivered to. They are written directly rather than through log
// ingestion, so they do not count as the tenant's logs.
type ServerlessTaskLogRepository interface {
	// Insert writes lines on one tenant's tasks in a single statement. Lines whose task is
	// not found are skipped.
	Insert(ctx context.Context, tenantId uuid.UUID, lines []ServerlessTaskLogOpts) error
}

type serverlessTaskLogRepository struct {
	*sharedRepository
}

func (r *serverlessTaskLogRepository) Insert(ctx context.Context, tenantId uuid.UUID, lines []ServerlessTaskLogOpts) error {
	if len(lines) == 0 {
		return nil
	}

	externalIds := make([]uuid.UUID, 0, len(lines))

	for _, line := range lines {
		externalIds = append(externalIds, line.TaskExternalId)
	}

	tasks, err := r.lookupExternalIds(ctx, r.pool, tenantId, externalIds)

	if err != nil {
		return err
	}

	byExternalId := make(map[uuid.UUID]*sqlcv1.FlattenExternalIdsRow, len(tasks))

	for _, task := range tasks {
		byExternalId[task.ExternalID] = task
	}

	params := make([]sqlcv1.InsertLogLineParams, 0, len(lines))

	for _, line := range lines {
		task, ok := byExternalId[line.TaskExternalId]

		if !ok {
			continue
		}

		params = append(params, sqlcv1.InsertLogLineParams{
			TenantID:       tenantId,
			TaskID:         task.ID,
			TaskInsertedAt: task.InsertedAt,
			Message:        line.Message,
			Metadata:       line.Metadata,
			RetryCount:     line.RetryCount,
			Level:          sqlcv1.V1LogLineLevelINFO,
			WorkflowID:     &task.WorkflowID,
			StepID:         &task.StepID,
		})
	}

	if len(params) == 0 {
		return nil
	}

	_, err = r.queries.InsertLogLine(ctx, r.pool, params)

	return err
}
