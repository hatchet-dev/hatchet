package v1

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type WorkflowRepository interface {
	ListWorkflowNamesByIds(ctx context.Context, tenantId string, workflowIds []pgtype.UUID) (map[pgtype.UUID]string, error)
}

type workflowRepository struct {
	*sharedRepository
}

func newWorkflowRepository(shared *sharedRepository) WorkflowRepository {
	return &workflowRepository{
		sharedRepository: shared,
	}
}

func (w *workflowRepository) ListWorkflowNamesByIds(ctx context.Context, tenantId string, workflowIds []pgtype.UUID) (map[pgtype.UUID]string, error) {
	workflowNames, err := w.queries.ListWorkflowNamesByIds(ctx, w.pool, workflowIds)

	if err != nil {
		return nil, err
	}

	workflowIdToNameMap := make(map[pgtype.UUID]string)

	for _, row := range workflowNames {
		workflowIdToNameMap[row.ID] = row.Name
	}

	return workflowIdToNameMap, nil
}
