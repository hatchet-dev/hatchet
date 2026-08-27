//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/jackc/pgx/v5/pgtype"
)

func insertSignalCreatedEvent(t *testing.T, ctx context.Context, repo *TaskRepositoryImpl, tenantId uuid.UUID, taskId int64, taskInsertedAt time.Time, eventKey string) {
	t.Helper()

	_, err := repo.pool.Exec(
		ctx,
		`INSERT INTO v1_task_event (tenant_id, task_id, task_inserted_at, retry_count, event_type, event_key, data)
		 VALUES ($1, $2, $3, -1, 'SIGNAL_CREATED', $4, $5)`,
		tenantId,
		taskId,
		taskInsertedAt,
		eventKey,
		[]byte(fmt.Sprintf(`{"key": %q}`, eventKey)),
	)
	require.NoError(t, err)
}

func TestLockSignalCreatedEvents_MultiParentBatch(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo := createTaskRepository(pool)

	// v1_task_event is range-partitioned; inserts require the partitions to exist
	require.NoError(t, repo.UpdateTablePartitions(ctx))

	tenantId := uuid.New()
	otherTenantId := uuid.New()
	now := time.Now().UTC().Truncate(time.Millisecond)

	parentA := int64(101)
	parentAInsertedAt := now
	parentB := int64(202)
	parentBInsertedAt := now.Add(-1 * time.Minute)

	// parent A has three signal events, one of which is not requested
	insertSignalCreatedEvent(t, ctx, repo, tenantId, parentA, parentAInsertedAt, "a.key.0")
	insertSignalCreatedEvent(t, ctx, repo, tenantId, parentA, parentAInsertedAt, "a.key.1")
	insertSignalCreatedEvent(t, ctx, repo, tenantId, parentA, parentAInsertedAt, "a.key.2")

	insertSignalCreatedEvent(t, ctx, repo, tenantId, parentB, parentBInsertedAt, "b.key.0")
	insertSignalCreatedEvent(t, ctx, repo, tenantId, parentB, parentBInsertedAt, "b.key.1")

	// a different tenant has an event with the same task id and key as parent A
	insertSignalCreatedEvent(t, ctx, repo, otherTenantId, parentA, parentAInsertedAt, "a.key.0")

	toTimestamptz := func(ts time.Time) pgtype.Timestamptz {
		return sqlchelpers.TimestamptzFromTime(ts)
	}

	// a single batch spanning both parents, in interleaved order
	events, err := repo.lockSignalCreatedEvents(
		ctx,
		pool,
		tenantId,
		[]int64{parentA, parentB, parentA},
		[]pgtype.Timestamptz{toTimestamptz(parentAInsertedAt), toTimestamptz(parentBInsertedAt), toTimestamptz(parentAInsertedAt)},
		[]string{"a.key.0", "b.key.1", "a.key.2"},
	)
	require.NoError(t, err)

	gotKeys := make([]string, 0, len(events))

	for _, event := range events {
		gotKeys = append(gotKeys, event.EventKey.String)
	}

	assert.ElementsMatch(t, []string{"a.key.0", "b.key.1", "a.key.2"}, gotKeys)

	// keys only match events of the parent task they are requested under
	events, err = repo.lockSignalCreatedEvents(
		ctx,
		pool,
		tenantId,
		[]int64{parentB},
		[]pgtype.Timestamptz{toTimestamptz(parentBInsertedAt)},
		[]string{"a.key.0"},
	)
	require.NoError(t, err)
	assert.Empty(t, events)
}
