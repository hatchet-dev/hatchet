//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/config/limits"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func setupRepositoryWithTTL(t *testing.T, pool *pgxpool.Pool, ttl time.Duration, recheckInterval time.Duration) (Repository, func()) {
	t.Helper()

	logger := zerolog.Nop()
	inlineStoreTTL := 24 * time.Hour
	payloadStoreOpts := PayloadStoreRepositoryOpts{
		InlineStoreTTL: &inlineStoreTTL,
	}
	statusUpdateLimits := StatusUpdateBatchSizeLimits{
		Task: 1000,
		DAG:  1000,
	}

	repo, cleanup := NewRepository(
		pool,
		&logger,
		0,
		24*time.Hour,
		24*time.Hour,
		10,
		TaskOperationLimits{},
		payloadStoreOpts,
		statusUpdateLimits,
		limits.LimitConfigFile{},
		false,
		false,
		ttl,
		recheckInterval,
	)

	return repo, func() {
		_ = cleanup()
	}
}

func setupTenant(t *testing.T, repo Repository, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	queries := sqlcv1.New()

	_, err := queries.CreateControllerPartition(ctx, pool, pgtype.Text{String: "test", Valid: true})
	require.NoError(t, err)

	tenantId := uuid.New()
	_, err = repo.Tenant().CreateTenant(ctx, &CreateTenantOpts{
		ID:   &tenantId,
		Name: "test-tenant",
		Slug: "test-tenant-" + tenantId.String(),
	})
	require.NoError(t, err)

	return tenantId
}

type minimalWorkflow struct {
	WorkflowID        uuid.UUID
	WorkflowVersionID uuid.UUID
}

func createMinimalWorkflow(t *testing.T, pool *pgxpool.Pool, tenantId uuid.UUID, workflowName string) minimalWorkflow {
	t.Helper()
	ctx := context.Background()
	queries := sqlcv1.New()

	workflowId := uuid.New()
	_, err := queries.CreateWorkflow(ctx, pool, sqlcv1.CreateWorkflowParams{
		ID:          workflowId,
		CreatedAt:   pgtype.Timestamp{},
		UpdatedAt:   pgtype.Timestamp{},
		Deletedat:   pgtype.Timestamp{},
		Tenantid:    tenantId,
		Name:        workflowName,
		Description: "test workflow",
	})
	require.NoError(t, err)

	workflowVersionId := uuid.New()
	_, err = queries.CreateWorkflowVersion(ctx, pool, sqlcv1.CreateWorkflowVersionParams{
		ID:                        workflowVersionId,
		CreatedAt:                 pgtype.Timestamp{},
		UpdatedAt:                 pgtype.Timestamp{},
		Deletedat:                 pgtype.Timestamp{},
		Checksum:                  "test-checksum",
		Version:                   pgtype.Text{String: "v1", Valid: true},
		Workflowid:                workflowId,
		Sticky:                    sqlcv1.NullStickyStrategy{},
		Kind:                      sqlcv1.NullWorkflowKind{},
		DefaultPriority:           pgtype.Int4{},
		CreateWorkflowVersionOpts: []byte("{}"),
		InputJsonSchema:           []byte("{}"),
	})
	require.NoError(t, err)

	jobId := uuid.New()
	_, err = queries.CreateJob(ctx, pool, sqlcv1.CreateJobParams{
		ID:                jobId,
		CreatedAt:         pgtype.Timestamp{},
		UpdatedAt:         pgtype.Timestamp{},
		Deletedat:         pgtype.Timestamp{},
		Tenantid:          tenantId,
		Workflowversionid: workflowVersionId,
		Name:              "job",
		Description:       "job description",
		Kind:              sqlcv1.NullJobKind{},
	})
	require.NoError(t, err)

	_, err = queries.UpsertAction(ctx, pool, sqlcv1.UpsertActionParams{
		Action:   "test.action",
		Tenantid: tenantId,
	})
	require.NoError(t, err)

	_, err = queries.CreateStep(ctx, pool, sqlcv1.CreateStepParams{
		ID:              uuid.New(),
		CreatedAt:       pgtype.Timestamp{},
		UpdatedAt:       pgtype.Timestamp{},
		Deletedat:       pgtype.Timestamp{},
		Readableid:      "step-1",
		Tenantid:        tenantId,
		Jobid:           jobId,
		Actionid:        "test.action",
		Timeout:         pgtype.Text{},
		CustomUserData:  []byte("{}"),
		Retries:         pgtype.Int4{},
		ScheduleTimeout: pgtype.Text{},
	})
	require.NoError(t, err)

	return minimalWorkflow{
		WorkflowID:        workflowId,
		WorkflowVersionID: workflowVersionId,
	}
}

func insertRunStatus(t *testing.T, pool *pgxpool.Pool, tenantId uuid.UUID, wf minimalWorkflow, externalId uuid.UUID, status sqlcv1.V1ReadableStatusOlap) {
	t.Helper()
	ctx := context.Background()

	insertedAt := time.Now().UTC()
	runId := time.Now().UnixNano()

	_, err := pool.Exec(ctx, `
        INSERT INTO v1_runs_olap (
            tenant_id,
            id,
            inserted_at,
            external_id,
            readable_status,
            kind,
            workflow_id,
            workflow_version_id,
            additional_metadata,
            parent_task_external_id
        ) VALUES ($1, $2, $3, $4, $5, 'TASK', $6, $7, NULL, NULL)
    `, tenantId, runId, insertedAt, externalId, status, wf.WorkflowID, wf.WorkflowVersionID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
        INSERT INTO v1_lookup_table_olap (
            tenant_id,
            external_id,
            task_id,
            dag_id,
            inserted_at
        ) VALUES ($1, $2, $3, NULL, $4)
    `, tenantId, externalId, runId, insertedAt)
	require.NoError(t, err)
}

func insertTerminalTaskEventsForRun(t *testing.T, pool *pgxpool.Pool, tenantId uuid.UUID, externalId uuid.UUID) {
	t.Helper()
	ctx := context.Background()

	rows, err := pool.Query(ctx, `
		WITH lookup AS (
			SELECT external_id, dag_id, task_id, inserted_at
			FROM v1_lookup_table
			WHERE tenant_id = $1 AND external_id = $2
		), dag_tasks AS (
			SELECT t.id, t.inserted_at, t.retry_count
			FROM lookup l
			JOIN v1_dag_to_task dt ON dt.dag_id = l.dag_id AND dt.dag_inserted_at = l.inserted_at
			JOIN v1_task t ON t.id = dt.task_id AND t.inserted_at = dt.task_inserted_at
			WHERE l.dag_id IS NOT NULL
		), task_only AS (
			SELECT t.id, t.inserted_at, t.retry_count
			FROM lookup l
			JOIN v1_task t ON t.id = l.task_id AND t.inserted_at = l.inserted_at
			WHERE l.task_id IS NOT NULL
		)
		SELECT id, inserted_at, retry_count FROM dag_tasks
		UNION ALL
		SELECT id, inserted_at, retry_count FROM task_only
	`, tenantId, externalId)
	require.NoError(t, err)
	defer rows.Close()

	taskCount := 0
	for rows.Next() {
		var (
			taskId       int64
			taskInserted time.Time
			retryCount   int32
		)
		require.NoError(t, rows.Scan(&taskId, &taskInserted, &retryCount))
		taskCount++

		_, err = pool.Exec(ctx, `
			INSERT INTO v1_task_event (tenant_id, task_id, task_inserted_at, retry_count, event_type)
			VALUES ($1, $2, $3, $4, 'COMPLETED')
		`, tenantId, taskId, taskInserted, retryCount)
		require.NoError(t, err)
	}
	require.NoError(t, rows.Err())
	require.Greater(t, taskCount, 0)
}

func getLastDeniedAt(t *testing.T, pool *pgxpool.Pool, tenantId uuid.UUID, key IdempotencyKey) pgtype.Timestamptz {
	t.Helper()
	ctx := context.Background()

	var lastDeniedAt pgtype.Timestamptz
	err := pool.QueryRow(ctx, `
        SELECT last_denied_at
        FROM v1_idempotency_key
        WHERE tenant_id = $1 AND key = $2
    `, tenantId, string(key)).Scan(&lastDeniedAt)
	require.NoError(t, err)
	return lastDeniedAt
}

func getClaimedByExternalId(t *testing.T, pool *pgxpool.Pool, tenantId uuid.UUID, key IdempotencyKey) *uuid.UUID {
	t.Helper()
	ctx := context.Background()

	var claimedByExternalId *uuid.UUID
	err := pool.QueryRow(ctx, `
        SELECT claimed_by_external_id
        FROM v1_idempotency_key
        WHERE tenant_id = $1 AND key = $2
    `, tenantId, string(key)).Scan(&claimedByExternalId)
	require.NoError(t, err)
	return claimedByExternalId
}

func TestTriggerWorkflowIdempotency_DedupesActiveRun(t *testing.T) {
	// Ensures a duplicate idempotency key is rejected while an active run holds the claim.
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo, cleanupRepo := setupRepositoryWithTTL(t, pool, 2*time.Hour, 5*time.Minute)
	defer cleanupRepo()

	tenantId := setupTenant(t, repo, pool)
	_ = createMinimalWorkflow(t, pool, tenantId, "test-workflow")

	ctx := context.Background()
	key := IdempotencyKey("idem-key")

	opts1 := &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{
			WorkflowName: "test-workflow",
		},
		ExternalId:     uuid.New(),
		IdempotencyKey: &key,
	}

	_, _, err := repo.Triggers().TriggerFromWorkflowNames(ctx, tenantId, []*WorkflowNameTriggerOpts{opts1})
	require.NoError(t, err)

	opts2 := &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{
			WorkflowName: "test-workflow",
		},
		ExternalId:     uuid.New(),
		IdempotencyKey: &key,
	}

	_, _, err = repo.Triggers().TriggerFromWorkflowNames(ctx, tenantId, []*WorkflowNameTriggerOpts{opts2})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrIdempotencyKeyAlreadyClaimed)
}

func TestTriggerWorkflowIdempotency_AllowsAfterRelease(t *testing.T) {
	// Shows that once a key is released (deleted), the same key can be used again.
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo, cleanupRepo := setupRepositoryWithTTL(t, pool, 30*time.Minute, 5*time.Minute)
	defer cleanupRepo()

	tenantId := setupTenant(t, repo, pool)
	_ = createMinimalWorkflow(t, pool, tenantId, "test-workflow")

	ctx := context.Background()
	key := IdempotencyKey("idem-key-release")

	first := &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{
			WorkflowName: "test-workflow",
		},
		ExternalId:     uuid.New(),
		IdempotencyKey: &key,
	}

	_, _, err := repo.Triggers().TriggerFromWorkflowNames(ctx, tenantId, []*WorkflowNameTriggerOpts{first})
	require.NoError(t, err)

	err = repo.Idempotency().DeleteIdempotencyKeysByExternalId(ctx, tenantId, first.ExternalId)
	require.NoError(t, err)

	second := &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{
			WorkflowName: "test-workflow",
		},
		ExternalId:     uuid.New(),
		IdempotencyKey: &key,
	}

	_, _, err = repo.Triggers().TriggerFromWorkflowNames(ctx, tenantId, []*WorkflowNameTriggerOpts{second})
	require.NoError(t, err)
}

func TestTriggerWorkflowIdempotency_UsesConfiguredTTL(t *testing.T) {
	// Validates that the configured TTL is applied to the stored idempotency key.
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ttl := 90 * time.Minute
	repo, cleanupRepo := setupRepositoryWithTTL(t, pool, ttl, 5*time.Minute)
	defer cleanupRepo()

	tenantId := setupTenant(t, repo, pool)
	_ = createMinimalWorkflow(t, pool, tenantId, "test-workflow")

	ctx := context.Background()
	key := IdempotencyKey("idem-key-ttl")

	opts := &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{
			WorkflowName: "test-workflow",
		},
		ExternalId:     uuid.New(),
		IdempotencyKey: &key,
	}

	_, _, err := repo.Triggers().TriggerFromWorkflowNames(ctx, tenantId, []*WorkflowNameTriggerOpts{opts})
	require.NoError(t, err)

	var expiresAt time.Time
	err = pool.QueryRow(ctx,
		`SELECT expires_at FROM v1_idempotency_key WHERE tenant_id = $1 AND key = $2`,
		tenantId,
		string(key),
	).Scan(&expiresAt)
	require.NoError(t, err)

	expected := time.Now().Add(ttl)
	assert.WithinDuration(t, expected, expiresAt, 2*time.Minute)
}

func TestTriggerWorkflowIdempotency_ExpiredUnclaimedKeyIsReusable(t *testing.T) {
	// An expired, unclaimed key should be refreshed and claimed instead of blocking.
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo, cleanupRepo := setupRepositoryWithTTL(t, pool, 30*time.Minute, 1*time.Minute)
	defer cleanupRepo()

	tenantId := setupTenant(t, repo, pool)
	_ = createMinimalWorkflow(t, pool, tenantId, "test-workflow")

	ctx := context.Background()
	key := IdempotencyKey("idem-key-expired")

	_, err := pool.Exec(ctx,
		`INSERT INTO v1_idempotency_key (tenant_id, key, expires_at) VALUES ($1, $2, $3)`,
		tenantId,
		string(key),
		time.Now().Add(-1*time.Hour),
	)
	require.NoError(t, err)

	opts := &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{
			WorkflowName: "test-workflow",
		},
		ExternalId:     uuid.New(),
		IdempotencyKey: &key,
	}

	_, _, err = repo.Triggers().TriggerFromWorkflowNames(ctx, tenantId, []*WorkflowNameTriggerOpts{opts})
	require.NoError(t, err)

	claimed := getClaimedByExternalId(t, pool, tenantId, key)
	require.NotNil(t, claimed)
	assert.Equal(t, opts.ExternalId, *claimed)

	var expiresAt time.Time
	err = pool.QueryRow(ctx,
		`SELECT expires_at FROM v1_idempotency_key WHERE tenant_id = $1 AND key = $2`,
		tenantId,
		string(key),
	).Scan(&expiresAt)
	require.NoError(t, err)
	assert.True(t, expiresAt.After(time.Now()))
}

func TestTriggerWorkflowIdempotency_ReclaimsAfterTerminalStatus(t *testing.T) {
	// When OLAP reports terminal status, a previously claimed key is reclaimed and a new run is allowed.
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo, cleanupRepo := setupRepositoryWithTTL(t, pool, 30*time.Minute, 1*time.Minute)
	defer cleanupRepo()

	tenantId := setupTenant(t, repo, pool)
	wf := createMinimalWorkflow(t, pool, tenantId, "test-workflow")

	ctx := context.Background()
	key := IdempotencyKey("idem-key-reclaim")

	first := &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{
			WorkflowName: "test-workflow",
		},
		ExternalId:     uuid.New(),
		IdempotencyKey: &key,
	}

	_, _, err := repo.Triggers().TriggerFromWorkflowNames(ctx, tenantId, []*WorkflowNameTriggerOpts{first})
	require.NoError(t, err)

	insertRunStatus(t, pool, tenantId, wf, first.ExternalId, sqlcv1.V1ReadableStatusOlapFAILED)

	second := &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{
			WorkflowName: "test-workflow",
		},
		ExternalId:     uuid.New(),
		IdempotencyKey: &key,
	}

	_, _, err = repo.Triggers().TriggerFromWorkflowNames(ctx, tenantId, []*WorkflowNameTriggerOpts{second})
	require.NoError(t, err)
}

func TestTriggerWorkflowIdempotency_ReclaimDoesNotUpdateLastDeniedAt(t *testing.T) {
	// Reclaimed keys should not have last_denied_at set since the enqueue succeeds.
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo, cleanupRepo := setupRepositoryWithTTL(t, pool, 30*time.Minute, 1*time.Minute)
	defer cleanupRepo()

	tenantId := setupTenant(t, repo, pool)
	wf := createMinimalWorkflow(t, pool, tenantId, "test-workflow")

	ctx := context.Background()
	key := IdempotencyKey("idem-key-reclaim-denied")

	first := &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{
			WorkflowName: "test-workflow",
		},
		ExternalId:     uuid.New(),
		IdempotencyKey: &key,
	}

	_, _, err := repo.Triggers().TriggerFromWorkflowNames(ctx, tenantId, []*WorkflowNameTriggerOpts{first})
	require.NoError(t, err)

	insertRunStatus(t, pool, tenantId, wf, first.ExternalId, sqlcv1.V1ReadableStatusOlapFAILED)

	second := &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{
			WorkflowName: "test-workflow",
		},
		ExternalId:     uuid.New(),
		IdempotencyKey: &key,
	}

	_, _, err = repo.Triggers().TriggerFromWorkflowNames(ctx, tenantId, []*WorkflowNameTriggerOpts{second})
	require.NoError(t, err)

	lastDeniedAt := getLastDeniedAt(t, pool, tenantId, key)
	assert.False(t, lastDeniedAt.Valid)
}

func TestTriggerWorkflowIdempotency_MixedKeysAllowsPartial(t *testing.T) {
	// Mixed batch: one reclaimable, one running. We should allow the reclaimable run and skip the duplicate.
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo, cleanupRepo := setupRepositoryWithTTL(t, pool, 30*time.Minute, 1*time.Minute)
	defer cleanupRepo()

	tenantId := setupTenant(t, repo, pool)
	wf := createMinimalWorkflow(t, pool, tenantId, "test-workflow")

	ctx := context.Background()
	keyTerminal := IdempotencyKey("idem-key-terminal")
	keyRunning := IdempotencyKey("idem-key-running")

	terminalFirst := &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{
			WorkflowName: "test-workflow",
		},
		ExternalId:     uuid.New(),
		IdempotencyKey: &keyTerminal,
	}
	_, _, err := repo.Triggers().TriggerFromWorkflowNames(ctx, tenantId, []*WorkflowNameTriggerOpts{terminalFirst})
	require.NoError(t, err)
	insertRunStatus(t, pool, tenantId, wf, terminalFirst.ExternalId, sqlcv1.V1ReadableStatusOlapFAILED)

	runningFirst := &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{
			WorkflowName: "test-workflow",
		},
		ExternalId:     uuid.New(),
		IdempotencyKey: &keyRunning,
	}
	_, _, err = repo.Triggers().TriggerFromWorkflowNames(ctx, tenantId, []*WorkflowNameTriggerOpts{runningFirst})
	require.NoError(t, err)
	insertRunStatus(t, pool, tenantId, wf, runningFirst.ExternalId, sqlcv1.V1ReadableStatusOlapRUNNING)

	terminalSecond := &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{
			WorkflowName: "test-workflow",
		},
		ExternalId:     uuid.New(),
		IdempotencyKey: &keyTerminal,
	}
	runningSecond := &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{
			WorkflowName: "test-workflow",
		},
		ExternalId:     uuid.New(),
		IdempotencyKey: &keyRunning,
	}

	_, _, err = repo.Triggers().TriggerFromWorkflowNames(ctx, tenantId, []*WorkflowNameTriggerOpts{terminalSecond, runningSecond})
	require.NoError(t, err)

	claimedTerminal := getClaimedByExternalId(t, pool, tenantId, keyTerminal)
	require.NotNil(t, claimedTerminal)
	assert.Equal(t, terminalSecond.ExternalId, *claimedTerminal)

	claimedRunning := getClaimedByExternalId(t, pool, tenantId, keyRunning)
	require.NotNil(t, claimedRunning)
	assert.Equal(t, runningFirst.ExternalId, *claimedRunning)

	lastDeniedAt := getLastDeniedAt(t, pool, tenantId, keyRunning)
	assert.True(t, lastDeniedAt.Valid)
	assert.WithinDuration(t, time.Now(), lastDeniedAt.Time, 2*time.Minute)
}

func TestTriggerWorkflowIdempotency_ConcurrentDuplicateKey(t *testing.T) {
	// Two concurrent triggers with the same key should yield one success and one duplicate error.
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo, cleanupRepo := setupRepositoryWithTTL(t, pool, 30*time.Minute, 1*time.Minute)
	defer cleanupRepo()

	tenantId := setupTenant(t, repo, pool)
	_ = createMinimalWorkflow(t, pool, tenantId, "test-workflow")

	key := IdempotencyKey("idem-key-concurrent")

	start := make(chan struct{})
	errs := make(chan error, 2)

	run := func(externalId uuid.UUID) {
		<-start
		opts := &WorkflowNameTriggerOpts{
			TriggerTaskData: &TriggerTaskData{
				WorkflowName: "test-workflow",
			},
			ExternalId:     externalId,
			IdempotencyKey: &key,
		}
		_, _, err := repo.Triggers().TriggerFromWorkflowNames(context.Background(), tenantId, []*WorkflowNameTriggerOpts{opts})
		errs <- err
	}

	go run(uuid.New())
	go run(uuid.New())

	close(start)

	var (
		successCount int
		dupCount     int
	)

	for i := 0; i < 2; i++ {
		err := <-errs
		if err == nil {
			successCount++
			continue
		}
		if errors.Is(err, ErrIdempotencyKeyAlreadyClaimed) {
			dupCount++
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	assert.Equal(t, 1, successCount)
	assert.Equal(t, 1, dupCount)
}

func TestTriggerWorkflowIdempotency_RecheckThrottled(t *testing.T) {
	// If last_denied_at is recent, recheck is skipped and the duplicate is denied.
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	recheckInterval := 2 * time.Hour
	repo, cleanupRepo := setupRepositoryWithTTL(t, pool, 30*time.Minute, recheckInterval)
	defer cleanupRepo()

	tenantId := setupTenant(t, repo, pool)
	wf := createMinimalWorkflow(t, pool, tenantId, "test-workflow")

	ctx := context.Background()
	key := IdempotencyKey("idem-key-throttle")

	first := &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{
			WorkflowName: "test-workflow",
		},
		ExternalId:     uuid.New(),
		IdempotencyKey: &key,
	}

	_, _, err := repo.Triggers().TriggerFromWorkflowNames(ctx, tenantId, []*WorkflowNameTriggerOpts{first})
	require.NoError(t, err)

	insertRunStatus(t, pool, tenantId, wf, first.ExternalId, sqlcv1.V1ReadableStatusOlapFAILED)

	_, err = pool.Exec(ctx, `UPDATE v1_idempotency_key SET last_denied_at = NOW() WHERE tenant_id = $1 AND key = $2`, tenantId, string(key))
	require.NoError(t, err)

	second := &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{
			WorkflowName: "test-workflow",
		},
		ExternalId:     uuid.New(),
		IdempotencyKey: &key,
	}

	_, _, err = repo.Triggers().TriggerFromWorkflowNames(ctx, tenantId, []*WorkflowNameTriggerOpts{second})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrIdempotencyKeyAlreadyClaimed)
}

func TestTriggerWorkflowIdempotency_RecheckDisabled(t *testing.T) {
	// With recheck disabled, duplicates are denied and last_denied_at remains unset.
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo, cleanupRepo := setupRepositoryWithTTL(t, pool, 30*time.Minute, 0)
	defer cleanupRepo()

	tenantId := setupTenant(t, repo, pool)
	wf := createMinimalWorkflow(t, pool, tenantId, "test-workflow")

	ctx := context.Background()
	key := IdempotencyKey("idem-key-no-recheck")

	first := &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{
			WorkflowName: "test-workflow",
		},
		ExternalId:     uuid.New(),
		IdempotencyKey: &key,
	}

	_, _, err := repo.Triggers().TriggerFromWorkflowNames(ctx, tenantId, []*WorkflowNameTriggerOpts{first})
	require.NoError(t, err)

	insertRunStatus(t, pool, tenantId, wf, first.ExternalId, sqlcv1.V1ReadableStatusOlapFAILED)

	second := &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{
			WorkflowName: "test-workflow",
		},
		ExternalId:     uuid.New(),
		IdempotencyKey: &key,
	}

	_, _, err = repo.Triggers().TriggerFromWorkflowNames(ctx, tenantId, []*WorkflowNameTriggerOpts{second})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrIdempotencyKeyAlreadyClaimed)

	lastDeniedAt := getLastDeniedAt(t, pool, tenantId, key)
	assert.False(t, lastDeniedAt.Valid)
}

func TestTriggerWorkflowIdempotency_RecheckMissingOlapUpdatesLastDeniedAt(t *testing.T) {
	// If OLAP has no status row, duplicates should be denied and last_denied_at updated.
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo, cleanupRepo := setupRepositoryWithTTL(t, pool, 30*time.Minute, 1*time.Minute)
	defer cleanupRepo()

	tenantId := setupTenant(t, repo, pool)
	_ = createMinimalWorkflow(t, pool, tenantId, "test-workflow")

	ctx := context.Background()
	key := IdempotencyKey("idem-key-missing-olap")

	first := &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{
			WorkflowName: "test-workflow",
		},
		ExternalId:     uuid.New(),
		IdempotencyKey: &key,
	}

	_, _, err := repo.Triggers().TriggerFromWorkflowNames(ctx, tenantId, []*WorkflowNameTriggerOpts{first})
	require.NoError(t, err)

	second := &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{
			WorkflowName: "test-workflow",
		},
		ExternalId:     uuid.New(),
		IdempotencyKey: &key,
	}

	_, _, err = repo.Triggers().TriggerFromWorkflowNames(ctx, tenantId, []*WorkflowNameTriggerOpts{second})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrIdempotencyKeyAlreadyClaimed)

	lastDeniedAt := getLastDeniedAt(t, pool, tenantId, key)
	assert.True(t, lastDeniedAt.Valid)
	assert.WithinDuration(t, time.Now(), lastDeniedAt.Time, 2*time.Minute)
}

func TestTriggerWorkflowIdempotency_RecheckUsesCoreWhenOlapMissing(t *testing.T) {
	// If OLAP is missing but core task events are terminal, we should reclaim and allow a new run.
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo, cleanupRepo := setupRepositoryWithTTL(t, pool, 30*time.Minute, 1*time.Minute)
	defer cleanupRepo()

	tenantId := setupTenant(t, repo, pool)
	_ = createMinimalWorkflow(t, pool, tenantId, "test-workflow")

	ctx := context.Background()
	key := IdempotencyKey("idem-key-core-fallback")

	first := &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{
			WorkflowName: "test-workflow",
		},
		ExternalId:     uuid.New(),
		IdempotencyKey: &key,
	}

	_, _, err := repo.Triggers().TriggerFromWorkflowNames(ctx, tenantId, []*WorkflowNameTriggerOpts{first})
	require.NoError(t, err)

	insertTerminalTaskEventsForRun(t, pool, tenantId, first.ExternalId)

	second := &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{
			WorkflowName: "test-workflow",
		},
		ExternalId:     uuid.New(),
		IdempotencyKey: &key,
	}

	_, _, err = repo.Triggers().TriggerFromWorkflowNames(ctx, tenantId, []*WorkflowNameTriggerOpts{second})
	require.NoError(t, err)

	claimed := getClaimedByExternalId(t, pool, tenantId, key)
	require.NotNil(t, claimed)
	assert.Equal(t, second.ExternalId, *claimed)
}

func TestTriggerWorkflowIdempotency_RecheckUpdatesLastDeniedAtWhenNonTerminal(t *testing.T) {
	// If the run is non-terminal, we deny and update last_denied_at for throttling.
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	repo, cleanupRepo := setupRepositoryWithTTL(t, pool, 30*time.Minute, 1*time.Minute)
	defer cleanupRepo()

	tenantId := setupTenant(t, repo, pool)
	wf := createMinimalWorkflow(t, pool, tenantId, "test-workflow")

	ctx := context.Background()
	key := IdempotencyKey("idem-key-non-terminal")

	first := &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{
			WorkflowName: "test-workflow",
		},
		ExternalId:     uuid.New(),
		IdempotencyKey: &key,
	}

	_, _, err := repo.Triggers().TriggerFromWorkflowNames(ctx, tenantId, []*WorkflowNameTriggerOpts{first})
	require.NoError(t, err)

	insertRunStatus(t, pool, tenantId, wf, first.ExternalId, sqlcv1.V1ReadableStatusOlapRUNNING)

	second := &WorkflowNameTriggerOpts{
		TriggerTaskData: &TriggerTaskData{
			WorkflowName: "test-workflow",
		},
		ExternalId:     uuid.New(),
		IdempotencyKey: &key,
	}

	_, _, err = repo.Triggers().TriggerFromWorkflowNames(ctx, tenantId, []*WorkflowNameTriggerOpts{second})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrIdempotencyKeyAlreadyClaimed)

	lastDeniedAt := getLastDeniedAt(t, pool, tenantId, key)
	assert.True(t, lastDeniedAt.Valid)
	assert.WithinDuration(t, time.Now(), lastDeniedAt.Time, 2*time.Minute)
}
