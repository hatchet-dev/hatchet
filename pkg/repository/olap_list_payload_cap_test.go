//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestListTasksByDAGId_RejectsPayloadsAboveTaskCap(t *testing.T) {
	basePool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	pool := createEnumAwarePool(t, basePool)
	repo := createOLAPRepositoryWithPayloadStore(t, pool)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	require.NoError(t, repo.UpdateTablePartitions(ctx))

	tenantId := uuid.New()
	workflowId := uuid.New()
	workflowVersionId := uuid.New()
	dagExternalId := uuid.New()
	insertedAt := pgtype.Timestamptz{Time: time.Now().UTC().Truncate(time.Microsecond), Valid: true}

	locksNotAcquired, err := repo.CreateDAGs(ctx, tenantId, []*DAGWithData{{
		V1Dag: &sqlcv1.V1Dag{
			ID:                1,
			InsertedAt:        insertedAt,
			TenantID:          tenantId,
			ExternalID:        dagExternalId,
			DisplayName:       "wide-dag",
			WorkflowID:        workflowId,
			WorkflowVersionID: workflowVersionId,
		},
		Input:              []byte(`{}`),
		AdditionalMetadata: []byte(`{}`),
	}})
	require.NoError(t, err)
	require.Empty(t, locksNotAcquired)

	numChildren := MaxTasksWithPayloads + 1
	children := make([]*V1TaskWithPayload, 0, numChildren)

	for i := range numChildren {
		children = append(children, &V1TaskWithPayload{
			V1Task: &sqlcv1.V1Task{
				ID:                 int64(i + 1),
				InsertedAt:         insertedAt,
				TenantID:           tenantId,
				Queue:              "default",
				ActionID:           "test:wide-dag-child",
				StepID:             uuid.New(),
				WorkflowID:         workflowId,
				WorkflowVersionID:  workflowVersionId,
				WorkflowRunID:      dagExternalId,
				ScheduleTimeout:    "5m",
				Sticky:             sqlcv1.V1StickyStrategyNONE,
				ExternalID:         uuid.New(),
				DisplayName:        "wide-dag-child",
				Input:              []byte(`{}`),
				AdditionalMetadata: []byte(`{}`),
				DagID:              pgtype.Int8{Int64: 1, Valid: true},
				DagInsertedAt:      insertedAt,
			},
			Payload: []byte(`{}`),
		})
	}

	_, locksNotAcquired, err = repo.CreateTasks(ctx, tenantId, children)
	require.NoError(t, err)
	require.Empty(t, locksNotAcquired)

	_, _, err = repo.ListTasksByDAGId(ctx, tenantId, []uuid.UUID{dagExternalId}, true)
	require.ErrorIs(t, err, ErrTooManyTasksWithPayloads)

	tasks, _, err := repo.ListTasksByDAGId(ctx, tenantId, []uuid.UUID{dagExternalId}, false)
	require.NoError(t, err)
	require.Len(t, tasks, numChildren)
}
