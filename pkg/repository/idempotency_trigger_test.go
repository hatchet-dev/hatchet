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

func TestTriggerWorkflowIdempotency_ReclaimsAfterTerminalStatus(t *testing.T) {
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

func TestTriggerWorkflowIdempotency_MixedKeysDeniesBatch(t *testing.T) {
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
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrIdempotencyKeyAlreadyClaimed)

	claimedTerminal := getClaimedByExternalId(t, pool, tenantId, keyTerminal)
	require.NotNil(t, claimedTerminal)
	assert.Equal(t, terminalFirst.ExternalId, *claimedTerminal)
}

func TestTriggerWorkflowIdempotency_ConcurrentDuplicateKey(t *testing.T) {
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

func TestTriggerWorkflowIdempotency_RecheckUpdatesLastDeniedAtWhenNonTerminal(t *testing.T) {
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
