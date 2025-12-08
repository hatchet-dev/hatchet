//go:build !e2e && !load && !rampup && !integration

package v1

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/cel"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func newBatchTestRepository(pool *pgxpool.Pool) *TaskRepositoryImpl {
	logger := zerolog.Nop()
	queries := sqlcv1.New()
	payloadStore := NewPayloadStoreRepository(pool, &logger, queries, PayloadStoreRepositoryOpts{
		WALEnabled:                     true,
		WALPollLimit:                   1,
		WALProcessInterval:             time.Second,
		ExternalCutoverProcessInterval: time.Second,
	})

	queueCache := cache.New(5 * time.Minute)
	stepExpressionCache := cache.New(5 * time.Minute)
	tenantIdWorkflowNameCache := cache.New(5 * time.Minute)

	taskLookupCache, _ := lru.New[taskExternalIdTenantIdTuple, *sqlcv1.FlattenExternalIdsRow](1024)

	shared := &sharedRepository{
		pool:                      pool,
		l:                         &logger,
		queries:                   queries,
		queueCache:                queueCache,
		stepExpressionCache:       stepExpressionCache,
		tenantIdWorkflowNameCache: tenantIdWorkflowNameCache,
		celParser:                 cel.NewCELParser(),
		taskLookupCache:           taskLookupCache,
		payloadStore:              payloadStore,
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

	tenantID := uuid.NewString()
	stepID := uuid.NewString()
	workflowID := uuid.NewString()
	workflowVersionID := uuid.NewString()
	jobID := uuid.NewString()
	actionID := "batch-action"

	stepConfig := &sqlcv1.ListStepsByIdsRow{
		ID:                   sqlchelpers.UUIDFromStr(stepID),
		ReadableId:           sqlchelpers.TextFromStr("batch-step"),
		TenantId:             sqlchelpers.UUIDFromStr(tenantID),
		JobId:                sqlchelpers.UUIDFromStr(jobID),
		ActionId:             actionID,
		Timeout:              sqlchelpers.TextFromStr("PT1M"),
		RetryBackoffFactor:   pgtype.Float8{},
		RetryMaxBackoff:      pgtype.Int4{},
		ScheduleTimeout:      "PT1M",
		BatchSize:            pgtype.Int4{Int32: 10, Valid: true},
		BatchFlushIntervalMs: pgtype.Int4{},
		BatchKeyExpression:   sqlchelpers.TextFromStr("input.batchKey"),
		BatchMaxRuns:         pgtype.Int4{Int32: 2, Valid: true},
		WorkflowVersionId:    sqlchelpers.UUIDFromStr(workflowVersionID),
		WorkflowId:           sqlchelpers.UUIDFromStr(workflowID),
		DefaultPriority:      1,
	}

	stepIdsToConfig := map[string]*sqlcv1.ListStepsByIdsRow{
		stepID: stepConfig,
	}

	workflowRunID := uuid.NewString()

	makeTask := func(batchKey string) CreateTaskOpts {
		return CreateTaskOpts{
			ExternalId:         uuid.NewString(),
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
