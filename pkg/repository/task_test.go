//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/cel"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

func newBatchTestRepository(pool *pgxpool.Pool) *TaskRepositoryImpl {
	logger := zerolog.Nop()
	queries := sqlcv1.New()
	payloadStore := NewPayloadStoreRepository(pool, &logger, queries, PayloadStoreRepositoryOpts{
		ExternalCutoverProcessInterval: time.Second,
		ExternalCutoverBatchSize:       1,
	})

	queueCache := cache.New(5 * time.Minute)
	taskLookupCache, _ := lru.New[taskExternalIdTenantIdTuple, *sqlcv1.FlattenExternalIdsRow](1024)

	shared := &sharedRepository{
		pool:              pool,
		ddlPool:           pool,
		l:                 &logger,
		v:                 validator.NewDefaultValidator(),
		queries:           queries,
		queueCache:        queueCache,
		celParser:         cel.NewCELParser(),
		taskLookupCache:   taskLookupCache,
		payloadStore:      payloadStore,
		stepIdConfigCache: expirable.NewLRU(10000, func(key uuid.UUID, value *sqlcv1.ListStepsByIdsRow) {}, 5*time.Minute),
	}

	return &TaskRepositoryImpl{
		sharedRepository:      shared,
		taskRetentionPeriod:   24 * time.Hour,
		maxInternalRetryCount: 3,
	}
}

func TestInsertTasksPersistsBatchKeys(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo := newBatchTestRepository(pool)

	ctx := context.Background()

	_, err := pool.Exec(ctx, `ALTER TABLE v1_task ADD COLUMN IF NOT EXISTS batch_key TEXT`)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `ALTER TABLE v1_queue_item ADD COLUMN IF NOT EXISTS batch_key TEXT`)
	require.NoError(t, err)

	tenantID := uuid.New()
	stepID := uuid.New()
	workflowID := uuid.New()
	workflowVersionID := uuid.New()
	jobID := uuid.New()
	actionID := "batch-action"

	stepConfig := &sqlcv1.ListStepsByIdsRow{
		ID: stepID,
		ReadableId: pgtype.Text{
			String: "batch-step",
			Valid:  true,
		},
		TenantId:           tenantID,
		JobId:              jobID,
		ActionId:           actionID,
		Timeout:            sqlchelpers.TextFromStr("PT1M"),
		RetryBackoffFactor: pgtype.Float8{},
		RetryMaxBackoff:    pgtype.Int4{},
		ScheduleTimeout:    "PT1M",
		BatchGroupKey:      sqlchelpers.TextFromStr("input.batchKey"),
		WorkflowVersionId:  workflowVersionID,
		WorkflowId:         workflowID,
		DefaultPriority:    1,
	}

	stepIdsToConfig := map[uuid.UUID]*sqlcv1.ListStepsByIdsRow{
		stepID: stepConfig,
	}

	workflowRunID := uuid.New()

	makeTask := func(batchKey string) CreateTaskOpts {
		return CreateTaskOpts{
			ExternalId:         uuid.New(),
			WorkflowRunId:      workflowRunID,
			StepId:             stepID,
			Input:              &TaskInput{Input: map[string]interface{}{"batchKey": batchKey}},
			StepIndex:          0,
			InitialState:       sqlcv1.V1TaskInitialStateQUEUED,
			AdditionalMetadata: nil,
		}
	}

	tasks := []CreateTaskOpts{
		makeTask("1"),
		makeTask("2"),
	}

	_, err = repo.sharedRepository.insertTasks(ctx, repo.pool, tenantID, tasks, stepIdsToConfig)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		UPDATE v1_queue_item qi
		SET batch_key = vt.batch_key
		FROM v1_task vt
		WHERE qi.task_id = vt.id
		  AND qi.task_inserted_at = vt.inserted_at
		  AND qi.retry_count = vt.retry_count
		  AND qi.batch_key IS DISTINCT FROM vt.batch_key`)
	require.NoError(t, err)

	taskRows, err := pool.Query(ctx, `
		SELECT batch_key
		FROM v1_task
		WHERE tenant_id = $1
		ORDER BY id`,
		tenantID,
	)
	require.NoError(t, err)
	defer taskRows.Close()

	var taskKeys []string
	for taskRows.Next() {
		var key pgtype.Text
		require.NoError(t, taskRows.Scan(&key))
		require.True(t, key.Valid)
		taskKeys = append(taskKeys, key.String)
	}
	require.NoError(t, taskRows.Err())
	require.Equal(t, []string{"1", "2"}, taskKeys)

	queueRows, err := pool.Query(ctx, `
		SELECT batch_key
		FROM v1_queue_item
		WHERE tenant_id = $1
		ORDER BY task_id`,
		tenantID,
	)
	require.NoError(t, err)
	defer queueRows.Close()

	var queueKeys []string
	for queueRows.Next() {
		var key pgtype.Text
		require.NoError(t, queueRows.Scan(&key))
		require.True(t, key.Valid)
		queueKeys = append(queueKeys, key.String)
	}
	require.NoError(t, queueRows.Err())
	require.Equal(t, []string{"1", "2"}, queueKeys)
}

func TestV1TaskRuntimeDeleteTrigger_CleansUpBatchRuntime(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()

	tenantID := uuid.New()
	batchID := uuid.New()
	stepID := uuid.New()

	_, err := pool.Exec(ctx, `
		INSERT INTO v1_batch_runtime (tenant_id, step_id, action_id, batch_key, batch_id)
		VALUES ($1, $2, $3, $4, $5)
	`, tenantID, stepID, "test-action", "test-key", batchID)
	require.NoError(t, err)

	now := time.Now().UTC()
	timeoutAt := now.Add(time.Hour)

	for _, taskID := range []int64{1, 2} {
		_, err := pool.Exec(ctx, `
			INSERT INTO v1_task_runtime (task_id, task_inserted_at, retry_count, tenant_id, batch_id, timeout_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, taskID, now, int32(0), tenantID, batchID, timeoutAt)
		require.NoError(t, err)
	}

	// Delete one runtime: reservation should still exist.
	_, err = pool.Exec(ctx, `
		DELETE FROM v1_task_runtime
		WHERE tenant_id = $1 AND batch_id = $2 AND task_id = $3
	`, tenantID, batchID, int64(1))
	require.NoError(t, err)

	var cnt int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM v1_batch_runtime
		WHERE tenant_id = $1 AND batch_id = $2
	`, tenantID, batchID).Scan(&cnt)
	require.NoError(t, err)
	require.Equal(t, 1, cnt)

	// Delete the final runtime: trigger should delete the reservation.
	_, err = pool.Exec(ctx, `
		DELETE FROM v1_task_runtime
		WHERE tenant_id = $1 AND batch_id = $2
	`, tenantID, batchID)
	require.NoError(t, err)

	err = pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM v1_batch_runtime
		WHERE tenant_id = $1 AND batch_id = $2
	`, tenantID, batchID).Scan(&cnt)
	require.NoError(t, err)
	require.Equal(t, 0, cnt)
}

func insertTaskWithRetries(t *testing.T, ctx context.Context, repo *TaskRepositoryImpl, workflowName string, retries int) *V1TaskWithPayload {
	t.Helper()

	workflows := &workflowRepository{sharedRepository: repo.sharedRepository}

	opts := minimalWorkflowOpts(workflowName, "v1", nil)
	opts.Tasks[0].Retries = &retries

	wf, err := workflows.PutWorkflowVersion(ctx, internalTenantId, opts)
	require.NoError(t, err)

	steps, err := repo.queries.ListStepsByWorkflowVersionIds(ctx, repo.pool, sqlcv1.ListStepsByWorkflowVersionIdsParams{
		Ids:      []uuid.UUID{wf.WorkflowVersion.ID},
		Tenantid: internalTenantId,
	})
	require.NoError(t, err)
	require.Len(t, steps, 1)

	stepID := steps[0].ID

	stepIdsToConfig, err := repo.sharedRepository.listStepsByIds(ctx, repo.pool, internalTenantId, []uuid.UUID{stepID})
	require.NoError(t, err)
	require.Contains(t, stepIdsToConfig, stepID)

	insertedTasks, err := repo.sharedRepository.insertTasks(ctx, repo.pool, internalTenantId, []CreateTaskOpts{
		{
			ExternalId:    uuid.New(),
			WorkflowRunId: uuid.New(),
			StepId:        stepID,
			Input:         &TaskInput{Input: map[string]any{"key": "value"}},
			StepIndex:     0,
			InitialState:  sqlcv1.V1TaskInitialStateQUEUED,
		},
	}, stepIdsToConfig)
	require.NoError(t, err)
	require.Len(t, insertedTasks, 1)

	return insertedTasks[0]
}

func TestFailTasksRetryDecisionIsPerTask(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo := newBatchTestRepository(pool)

	ctx := context.Background()

	cases := []struct {
		name           string
		retries        int
		hasRuntime     bool
		isAppError     bool
		isNonRetryable bool
		wantRetry      bool
	}{
		{name: "app error, step allows no retries", retries: 0, hasRuntime: true, isAppError: true},
		{name: "app error, step allows retries", retries: 2, hasRuntime: true, isAppError: true, wantRetry: true},
		{name: "app error, non-retryable", retries: 2, hasRuntime: true, isAppError: true, isNonRetryable: true},
		{name: "app error, no runtime", retries: 2, isAppError: true},
		{name: "internal error", hasRuntime: true, wantRetry: true},
		{name: "internal error, no runtime"},
	}

	names := make(map[int64]string, len(cases))
	opts := make([]FailTaskOpts, 0, len(cases))
	runtimes := sqlcv1.InsertBufferedTaskRuntimesParams{Tenantid: internalTenantId}
	want := make([]string, 0, len(cases))

	for i, c := range cases {
		task := insertTaskWithRetries(t, ctx, repo, fmt.Sprintf("fail-batch-%d", i), c.retries)
		names[task.ID] = c.name

		opts = append(opts, FailTaskOpts{
			TaskIdInsertedAtRetryCount: &TaskIdInsertedAtRetryCount{
				Id:         task.ID,
				InsertedAt: task.InsertedAt,
				RetryCount: task.RetryCount,
			},
			IsAppError:     c.isAppError,
			IsNonRetryable: c.isNonRetryable,
			ErrorMessage:   "task failed",
		})

		if c.hasRuntime {
			runtimes.Taskids = append(runtimes.Taskids, task.ID)
			runtimes.Taskinsertedats = append(runtimes.Taskinsertedats, task.InsertedAt)
			runtimes.Taskretrycounts = append(runtimes.Taskretrycounts, task.RetryCount)
		}

		if c.wantRetry {
			want = append(want, c.name)
		}
	}

	require.NoError(t, repo.queries.InsertBufferedTaskRuntimes(ctx, repo.pool, runtimes))

	res, err := repo.FailTasks(ctx, internalTenantId, opts)
	require.NoError(t, err)

	got := make([]string, 0, len(res.RetriedTasks))
	for _, r := range res.RetriedTasks {
		got = append(got, names[r.Id])
	}

	require.ElementsMatch(t, want, got)
}
