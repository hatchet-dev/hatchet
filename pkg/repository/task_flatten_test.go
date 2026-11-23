//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type flattenTestFixture struct {
	repo *TaskRepositoryImpl
	pool *pgxpool.Pool

	standaloneTasks []*V1TaskWithPayload

	dagExternalId         uuid.UUID
	dagPayloadReferenceId uuid.UUID
	dagTasks              []*V1TaskWithPayload
}

// newFlattenTestFixture creates two standalone tasks in separate statements, so that they have
// different inserted_at values, and one DAG with two tasks.
func newFlattenTestFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool) *flattenTestFixture {
	t.Helper()

	repo := newBatchTestRepository(pool)
	shared := repo.sharedRepository
	workflows := &workflowRepository{sharedRepository: shared}

	wf, err := workflows.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts("flatten-external-ids", "v1", nil))
	require.NoError(t, err)

	steps, err := repo.queries.ListStepsByWorkflowVersionIds(ctx, pool, sqlcv1.ListStepsByWorkflowVersionIdsParams{
		Ids:      []uuid.UUID{wf.WorkflowVersion.ID},
		Tenantid: internalTenantId,
	})
	require.NoError(t, err)
	require.Len(t, steps, 1)

	stepId := steps[0].ID

	stepIdsToConfig, err := shared.listStepsByIds(ctx, pool, internalTenantId, []uuid.UUID{stepId})
	require.NoError(t, err)

	f := &flattenTestFixture{
		repo:                  repo,
		pool:                  pool,
		dagExternalId:         uuid.New(),
		dagPayloadReferenceId: uuid.New(),
	}

	insertTask := func(dagId *int64, dagInsertedAt pgtype.Timestamptz, workflowRunId uuid.UUID) *V1TaskWithPayload {
		inserted, err := shared.insertTasks(ctx, repo.pool.ForShared(), internalTenantId, []CreateTaskOpts{
			{
				ExternalId:    uuid.New(),
				WorkflowRunId: workflowRunId,
				StepId:        stepId,
				Input:         &TaskInput{Input: map[string]interface{}{"key": "value"}},
				InitialState:  sqlcv1.V1TaskInitialStateQUEUED,
				DagId:         dagId,
				DagInsertedAt: dagInsertedAt,
			},
		}, stepIdsToConfig)
		require.NoError(t, err)
		require.Len(t, inserted, 1)

		return inserted[0]
	}

	for range 2 {
		f.standaloneTasks = append(f.standaloneTasks, insertTask(nil, pgtype.Timestamptz{}, uuid.New()))
	}

	var (
		dagId         int64
		dagInsertedAt pgtype.Timestamptz
	)

	require.NoError(t, pool.QueryRow(ctx, `
		INSERT INTO v1_dag (tenant_id, external_id, display_name, workflow_id, workflow_version_id, payload_reference_id)
		VALUES ($1, $2, 'flatten-external-ids', $3, $4, $5)
		RETURNING id, inserted_at`,
		internalTenantId, f.dagExternalId, wf.WorkflowVersion.WorkflowId, wf.WorkflowVersion.ID, f.dagPayloadReferenceId,
	).Scan(&dagId, &dagInsertedAt))

	for range 2 {
		f.dagTasks = append(f.dagTasks, insertTask(&dagId, dagInsertedAt, f.dagExternalId))
	}

	return f
}

func flattenedByExternalId(t *testing.T, rows []*sqlcv1.FlattenExternalIdsRow) map[uuid.UUID]*sqlcv1.FlattenExternalIdsRow {
	t.Helper()

	res := make(map[uuid.UUID]*sqlcv1.FlattenExternalIdsRow, len(rows))

	for _, row := range rows {
		require.NotContains(t, res, row.ExternalID, "each task must be returned once")
		res[row.ExternalID] = row
	}

	return res
}

func TestFlattenExternalIds(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	f := newFlattenTestFixture(t, ctx, pool)

	t.Run("task external ids resolve to the tasks themselves", func(t *testing.T) {
		rows, err := f.repo.FlattenExternalIds(ctx, internalTenantId, []uuid.UUID{
			f.standaloneTasks[0].ExternalID,
			f.standaloneTasks[1].ExternalID,
		})
		require.NoError(t, err)

		byExternalId := flattenedByExternalId(t, rows)
		require.Len(t, byExternalId, 2)

		for _, task := range f.standaloneTasks {
			row, ok := byExternalId[task.ExternalID]
			require.True(t, ok)
			require.Equal(t, task.ID, row.ID)
			require.True(t, task.InsertedAt.Time.Equal(row.InsertedAt.Time))
			require.Equal(t, task.ExternalID, row.WorkflowRunExternalID)
			require.Equal(t, task.StepID, row.StepID)
			require.Nil(t, row.DagPayloadReferenceID)
		}
	})

	t.Run("a dag external id resolves to every task in the dag", func(t *testing.T) {
		rows, err := f.repo.FlattenExternalIds(ctx, internalTenantId, []uuid.UUID{f.dagExternalId})
		require.NoError(t, err)

		byExternalId := flattenedByExternalId(t, rows)
		require.Len(t, byExternalId, 2)

		for _, task := range f.dagTasks {
			row, ok := byExternalId[task.ExternalID]
			require.True(t, ok)
			require.Equal(t, task.ID, row.ID)
			require.Equal(t, f.dagExternalId, row.WorkflowRunExternalID)
			require.NotNil(t, row.DagPayloadReferenceID)
			require.Equal(t, f.dagPayloadReferenceId, *row.DagPayloadReferenceID)
		}
	})

	t.Run("task and dag external ids can be mixed", func(t *testing.T) {
		rows, err := f.repo.FlattenExternalIds(ctx, internalTenantId, []uuid.UUID{
			f.standaloneTasks[0].ExternalID,
			f.dagExternalId,
			f.dagTasks[0].ExternalID,
		})
		require.NoError(t, err)

		// the dag task looked up by its own external id is returned again by the dag's external id
		require.Len(t, rows, 4)

		var standalone, viaTask, viaDag int

		for _, row := range rows {
			switch row.WorkflowRunExternalID {
			case f.standaloneTasks[0].ExternalID:
				standalone++
			case f.dagTasks[0].ExternalID:
				viaTask++
			case f.dagExternalId:
				viaDag++
			}
		}

		require.Equal(t, 1, standalone)
		require.Equal(t, 1, viaTask)
		require.Equal(t, 2, viaDag)
	})

	t.Run("unknown external ids and other tenants resolve to nothing", func(t *testing.T) {
		rows, err := f.repo.FlattenExternalIds(ctx, internalTenantId, []uuid.UUID{uuid.New()})
		require.NoError(t, err)
		require.Empty(t, rows)

		rows, err = f.repo.FlattenExternalIds(ctx, uuid.New(), []uuid.UUID{f.standaloneTasks[0].ExternalID, f.dagExternalId})
		require.NoError(t, err)
		require.Empty(t, rows)

		rows, err = f.repo.FlattenExternalIds(ctx, internalTenantId, nil)
		require.NoError(t, err)
		require.Empty(t, rows)
	})
}

// Each lookup must only lock the partitions that hold the rows it reads. Locking every daily
// partition per call exhausts the backend's fast-path lock slots and turns a point lookup into
// contention on the shared lock table.
func TestFlattenExternalIdsLocksOnlyMatchingPartitions(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	f := newFlattenTestFixture(t, ctx, pool)

	for _, daysAhead := range []int{2, 3, 4} {
		for _, table := range []string{"v1_task", "v1_dag", "v1_dag_to_task"} {
			_, err := pool.Exec(ctx, `SELECT create_v1_range_partition($1, (NOW() + make_interval(days => $2))::date)`, table, daysAhead)
			require.NoError(t, err)
		}
	}

	lockedPartitions := func(tx sqlcv1.DBTX, table string) (locked, total int) {
		require.NoError(t, tx.QueryRow(ctx, `
			SELECT
				COUNT(*) FILTER (WHERE EXISTS (
					SELECT 1
					FROM pg_locks l
					WHERE l.pid = pg_backend_pid() AND l.locktype = 'relation' AND l.relation = i.inhrelid
				)),
				COUNT(*)
			FROM pg_inherits i
			WHERE i.inhparent = $1::regclass`,
			table,
		).Scan(&locked, &total))

		return locked, total
	}

	for _, tc := range []struct {
		name        string
		externalIds []uuid.UUID
		rows        int
		locked      map[string]int
	}{
		{
			name:        "tasks",
			externalIds: []uuid.UUID{f.standaloneTasks[0].ExternalID, f.standaloneTasks[1].ExternalID},
			rows:        2,
			locked:      map[string]int{"v1_task": 1, "v1_dag": 0, "v1_dag_to_task": 0},
		},
		{
			name:        "dag",
			externalIds: []uuid.UUID{f.dagExternalId},
			rows:        2,
			// a dag's tasks can be inserted on any day from the dag's own day onwards
			locked: map[string]int{"v1_dag": 1, "v1_dag_to_task": 1},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conn, err := pool.Acquire(ctx)
			require.NoError(t, err)
			defer conn.Release()

			// run on one connection enough times for a cached statement to switch to a generic plan
			for range 8 {
				tx, err := conn.Begin(ctx)
				require.NoError(t, err)

				rows, err := f.repo.queries.FlattenExternalIds(ctx, tx, sqlcv1.FlattenExternalIdsParams{
					Tenantid:    internalTenantId,
					Externalids: tc.externalIds,
				})
				require.NoError(t, err)
				require.Len(t, rows, tc.rows)

				for table, expected := range tc.locked {
					locked, total := lockedPartitions(tx, table)
					require.Greater(t, total, 3, "the test needs several partitions of %s to be meaningful", table)
					require.Equal(t, expected, locked, "locked partitions of %s", table)
				}

				require.NoError(t, tx.Rollback(ctx))
			}
		})
	}
}

func TestResolveTaskExternalIds(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	f := newFlattenTestFixture(t, ctx, pool)

	task := f.standaloneTasks[0]
	dagTask := f.dagTasks[0]
	unknownExternalId := uuid.New()

	resolved, err := f.repo.ResolveTaskExternalIds(ctx, internalTenantId, []uuid.UUID{
		task.ExternalID,
		task.ExternalID,
		dagTask.ExternalID,
		f.dagExternalId,
		unknownExternalId,
	})
	require.NoError(t, err)
	require.Len(t, resolved, 2, "only external ids of tasks are resolved")

	for _, expected := range []*V1TaskWithPayload{task, dagTask} {
		key, ok := resolved[expected.ExternalID]
		require.True(t, ok)
		require.Equal(t, expected.ID, key.ID)
		require.Equal(t, expected.InsertedAt.Time.UnixMicro(), key.InsertedAtUnixMicros)
	}

	otherTenant, err := f.repo.ResolveTaskExternalIds(ctx, uuid.New(), []uuid.UUID{task.ExternalID})
	require.NoError(t, err)
	require.Empty(t, otherTenant)

	// with the lookup rows gone, a second resolution can only be answered from the cache
	_, err = pool.Exec(ctx, `DELETE FROM v1_lookup_table WHERE external_id = ANY($1::uuid[])`, []uuid.UUID{task.ExternalID, dagTask.ExternalID})
	require.NoError(t, err)

	cached, err := f.repo.ResolveTaskExternalIds(ctx, internalTenantId, []uuid.UUID{task.ExternalID, dagTask.ExternalID, unknownExternalId})
	require.NoError(t, err)
	require.Equal(t, resolved, cached)

	otherTenant, err = f.repo.ResolveTaskExternalIds(ctx, uuid.New(), []uuid.UUID{task.ExternalID})
	require.NoError(t, err)
	require.Empty(t, otherTenant, "a cached key must not be served to another tenant")
}
