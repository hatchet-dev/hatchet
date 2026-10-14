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

func TestPopulateDAGMetadata_OneRowPerDAGWithLatestChildOutput(t *testing.T) {
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

	const dagId int64 = 1

	locksNotAcquired, err := repo.CreateDAGs(ctx, tenantId, []*DAGWithData{{
		V1Dag: &sqlcv1.V1Dag{
			ID:                dagId,
			InsertedAt:        insertedAt,
			TenantID:          tenantId,
			ExternalID:        dagExternalId,
			DisplayName:       "two-child-dag",
			WorkflowID:        workflowId,
			WorkflowVersionID: workflowVersionId,
		},
		Input:              []byte(`{}`),
		AdditionalMetadata: []byte(`{}`),
	}})
	require.NoError(t, err)
	require.Empty(t, locksNotAcquired)

	childIds := []int64{2, 3}
	children := make([]*V1TaskWithPayload, 0, len(childIds))

	for _, id := range childIds {
		children = append(children, &V1TaskWithPayload{
			V1Task: &sqlcv1.V1Task{
				ID:                 id,
				InsertedAt:         insertedAt,
				TenantID:           tenantId,
				Queue:              "default",
				ActionID:           "test:two-child-dag-child",
				StepID:             uuid.New(),
				WorkflowID:         workflowId,
				WorkflowVersionID:  workflowVersionId,
				WorkflowRunID:      dagExternalId,
				ScheduleTimeout:    "5m",
				Sticky:             sqlcv1.V1StickyStrategyNONE,
				ExternalID:         uuid.New(),
				DisplayName:        "two-child-dag-child",
				Input:              []byte(`{}`),
				AdditionalMetadata: []byte(`{}`),
				DagID:              pgtype.Int8{Int64: dagId, Valid: true},
				DagInsertedAt:      insertedAt,
			},
			Payload: []byte(`{}`),
		})
	}

	_, locksNotAcquired, err = repo.CreateTasks(ctx, tenantId, children)
	require.NoError(t, err)
	require.Empty(t, locksNotAcquired)

	// Each child finishes in its own write so the FINISHED events get distinct inserted_at values.
	var lastFinishedEventId uuid.UUID

	for _, id := range childIds {
		event := sqlcv1.CreateTaskEventsOLAPParams{
			TenantID:       tenantId,
			TaskID:         id,
			TaskInsertedAt: insertedAt,
			EventType:      sqlcv1.V1EventTypeOlapFINISHED,
			WorkflowID:     workflowId,
			EventTimestamp: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
			ReadableStatus: sqlcv1.V1ReadableStatusOlapCOMPLETED,
			Output:         []byte(`{}`),
			ExternalID:     uuid.New(),
		}
		lastFinishedEventId = event.ExternalID

		_, locksNotAcquired, err = repo.CreateTaskEvents(ctx, tenantId, []sqlcv1.CreateTaskEventsOLAPParams{event}, map[uuid.UUID]uuid.UUID{event.ExternalID: dagExternalId}, nil, nil)
		require.NoError(t, err)
		require.Empty(t, locksNotAcquired)
	}

	rows, err := repo.queries.PopulateDAGMetadata(ctx, pool, sqlcv1.PopulateDAGMetadataParams{
		Ids:             []int64{dagId},
		Insertedats:     []pgtype.Timestamptz{insertedAt},
		Tenantid:        tenantId,
		Includepayloads: true,
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.NotNil(t, rows[0].OutputEventExternalID)
	require.Equal(t, lastFinishedEventId, *rows[0].OutputEventExternalID)
}
