//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"crypto/sha256"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func createDurableEventsRepository(pool *pgxpool.Pool) DurableEventsRepository {
	logger := zerolog.Nop()

	payloadStore := NewPayloadStoreRepository(pool, &logger, sqlcv1.New(), PayloadStoreRepositoryOpts{})

	shared := &sharedRepository{
		pool:         pool,
		l:            &logger,
		queries:      sqlcv1.New(),
		payloadStore: payloadStore,
	}

	return newDurableEventsRepository(shared)
}

func timestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

var durableTaskIdCounter int64

func newDurableTaskId() (int64, pgtype.Timestamptz) {
	id := atomic.AddInt64(&durableTaskIdCounter, 1)
	insertedAt := timestamptz(time.Now().UTC().Truncate(time.Microsecond))
	return id, insertedAt
}

func createDurableEventLogPartitions(t *testing.T, pool *pgxpool.Pool) {
	ctx := context.Background()
	today := time.Now().UTC().Format("20060102")

	tables := []string{
		"v1_durable_event_log_file",
		"v1_durable_event_log_entry",
		"v1_durable_event_log_callback",
	}

	for _, table := range tables {
		partitionName := table + "_" + today
		_, err := pool.Exec(ctx, `
			DO $$
			BEGIN
				IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = '`+partitionName+`') THEN
					EXECUTE format('CREATE TABLE %I PARTITION OF `+table+` FOR VALUES FROM (%L) TO (%L)',
						'`+partitionName+`',
						CURRENT_DATE,
						CURRENT_DATE + INTERVAL '1 day');
				END IF;
			END $$;
		`)
		require.NoError(t, err)
	}
}

func TestCreateEventLogFiles(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()
	createDurableEventLogPartitions(t, pool)

	repo := createDurableEventsRepository(pool)
	ctx := context.Background()

	durableTaskId, durableTaskInsertedAt := newDurableTaskId()
	latestInsertedAt := timestamptz(time.Now().UTC().Truncate(time.Microsecond))

	opts := []CreateEventLogFileOpts{
		{
			DurableTaskId:                 durableTaskId,
			DurableTaskInsertedAt:         durableTaskInsertedAt,
			LatestInsertedAt:              latestInsertedAt,
			LatestNodeId:                  0,
			LatestBranchId:                0,
			LatestBranchFirstParentNodeId: 0,
		},
	}

	files, err := repo.CreateEventLogFiles(ctx, opts)
	require.NoError(t, err)
	require.Len(t, files, 1)

	assert.Equal(t, durableTaskId, files[0].DurableTaskID)
	assert.Equal(t, int64(0), files[0].LatestNodeID)
	assert.Equal(t, int64(0), files[0].LatestBranchID)
}

func TestGetEventLogFileForTask(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()
	createDurableEventLogPartitions(t, pool)

	repo := createDurableEventsRepository(pool)
	ctx := context.Background()

	durableTaskId, durableTaskInsertedAt := newDurableTaskId()
	latestInsertedAt := timestamptz(time.Now().UTC().Truncate(time.Microsecond))

	opts := []CreateEventLogFileOpts{
		{
			DurableTaskId:                 durableTaskId,
			DurableTaskInsertedAt:         durableTaskInsertedAt,
			LatestInsertedAt:              latestInsertedAt,
			LatestNodeId:                  5,
			LatestBranchId:                1,
			LatestBranchFirstParentNodeId: 3,
		},
	}

	_, err := repo.CreateEventLogFiles(ctx, opts)
	require.NoError(t, err)

	file, err := repo.GetEventLogFileForTask(ctx, durableTaskId, durableTaskInsertedAt)
	require.NoError(t, err)

	assert.Equal(t, durableTaskId, file.DurableTaskID)
	assert.Equal(t, int64(5), file.LatestNodeID)
	assert.Equal(t, int64(1), file.LatestBranchID)
	assert.Equal(t, int64(3), file.LatestBranchFirstParentNodeID)
}

func TestCreateEventLogEntries(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()
	createDurableEventLogPartitions(t, pool)

	repo := createDurableEventsRepository(pool)
	ctx := context.Background()

	tenantId := uuid.New()
	durableTaskId, durableTaskInsertedAt := newDurableTaskId()
	insertedAt := timestamptz(time.Now().UTC().Truncate(time.Microsecond))
	data := []byte(`{"key": "value"}`)

	opts := []CreateEventLogEntryOpts{
		{
			TenantId:              tenantId,
			ExternalId:            uuid.New(),
			DurableTaskId:         durableTaskId,
			DurableTaskInsertedAt: durableTaskInsertedAt,
			InsertedAt:            insertedAt,
			Kind:                  "RUN_TRIGGERED",
			NodeId:                1,
			ParentNodeId:          0,
			BranchId:              0,
			Data:                  data,
		},
	}

	entries, err := repo.CreateEventLogEntries(ctx, opts)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	assert.Equal(t, durableTaskId, entries[0].DurableTaskID)
	assert.Equal(t, int64(1), entries[0].NodeID)
	assert.Equal(t, int64(0), entries[0].BranchID)

	expectedHash := sha256.Sum256(data)
	assert.Equal(t, expectedHash[:], entries[0].DataHash)
	assert.Equal(t, "sha256", entries[0].DataHashAlg.String)
}

func TestCreateEventLogEntriesWithoutData(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()
	createDurableEventLogPartitions(t, pool)

	repo := createDurableEventsRepository(pool)
	ctx := context.Background()

	durableTaskId, durableTaskInsertedAt := newDurableTaskId()
	insertedAt := timestamptz(time.Now().UTC().Truncate(time.Microsecond))

	opts := []CreateEventLogEntryOpts{
		{
			TenantId:              uuid.New(),
			ExternalId:            uuid.New(),
			DurableTaskId:         durableTaskId,
			DurableTaskInsertedAt: durableTaskInsertedAt,
			InsertedAt:            insertedAt,
			Kind:                  "RUN_TRIGGERED",
			NodeId:                1,
			ParentNodeId:          0,
			BranchId:              0,
			Data:                  nil,
		},
	}

	entries, err := repo.CreateEventLogEntries(ctx, opts)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	assert.Nil(t, entries[0].DataHash)
}

func TestGetEventLogEntry(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()
	createDurableEventLogPartitions(t, pool)

	repo := createDurableEventsRepository(pool)
	ctx := context.Background()

	durableTaskId, durableTaskInsertedAt := newDurableTaskId()
	insertedAt := timestamptz(time.Now().UTC().Truncate(time.Microsecond))

	opts := []CreateEventLogEntryOpts{
		{
			TenantId:              uuid.New(),
			ExternalId:            uuid.New(),
			DurableTaskId:         durableTaskId,
			DurableTaskInsertedAt: durableTaskInsertedAt,
			InsertedAt:            insertedAt,
			Kind:                  "WAIT_FOR_STARTED",
			NodeId:                42,
			ParentNodeId:          10,
			BranchId:              2,
			Data:                  nil,
		},
	}

	_, err := repo.CreateEventLogEntries(ctx, opts)
	require.NoError(t, err)

	entry, err := repo.GetEventLogEntry(ctx, durableTaskId, durableTaskInsertedAt, 42)
	require.NoError(t, err)

	assert.Equal(t, durableTaskId, entry.DurableTaskID)
	assert.Equal(t, int64(42), entry.NodeID)
	assert.Equal(t, int64(10), entry.ParentNodeID.Int64)
	assert.Equal(t, int64(2), entry.BranchID)
}

func TestListEventLogEntries(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()
	createDurableEventLogPartitions(t, pool)

	repo := createDurableEventsRepository(pool)
	ctx := context.Background()

	durableTaskId, durableTaskInsertedAt := newDurableTaskId()
	insertedAt := timestamptz(time.Now().UTC().Truncate(time.Microsecond))

	opts := []CreateEventLogEntryOpts{
		{
			TenantId:              uuid.New(),
			ExternalId:            uuid.New(),
			DurableTaskId:         durableTaskId,
			DurableTaskInsertedAt: durableTaskInsertedAt,
			InsertedAt:            insertedAt,
			Kind:                  "RUN_TRIGGERED",
			NodeId:                1,
			ParentNodeId:          0,
			BranchId:              0,
		},
		{
			TenantId:              uuid.New(),
			ExternalId:            uuid.New(),
			DurableTaskId:         durableTaskId,
			DurableTaskInsertedAt: durableTaskInsertedAt,
			InsertedAt:            insertedAt,
			Kind:                  "WAIT_FOR_STARTED",
			NodeId:                2,
			ParentNodeId:          1,
			BranchId:              0,
		},
		{
			TenantId:              uuid.New(),
			ExternalId:            uuid.New(),
			DurableTaskId:         durableTaskId,
			DurableTaskInsertedAt: durableTaskInsertedAt,
			InsertedAt:            insertedAt,
			Kind:                  "MEMO_STARTED",
			NodeId:                3,
			ParentNodeId:          2,
			BranchId:              0,
		},
	}

	_, err := repo.CreateEventLogEntries(ctx, opts)
	require.NoError(t, err)

	entries, err := repo.ListEventLogEntries(ctx, durableTaskId, durableTaskInsertedAt)
	require.NoError(t, err)
	require.Len(t, entries, 3)

	nodeIds := make([]int64, len(entries))
	for i, e := range entries {
		nodeIds[i] = e.NodeID
	}
	assert.ElementsMatch(t, []int64{1, 2, 3}, nodeIds)
}

func TestCreateEventLogCallbacks(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()
	createDurableEventLogPartitions(t, pool)

	repo := createDurableEventsRepository(pool)
	ctx := context.Background()

	durableTaskId, durableTaskInsertedAt := newDurableTaskId()
	insertedAt := timestamptz(time.Now().UTC().Truncate(time.Microsecond))

	opts := []CreateEventLogCallbackOpts{
		{
			DurableTaskId:         durableTaskId,
			DurableTaskInsertedAt: durableTaskInsertedAt,
			InsertedAt:            insertedAt,
			Kind:                  "RUN_COMPLETED",
			Key:                   "run:abc123",
			NodeId:                1,
			IsSatisfied:           false,
		},
	}

	callbacks, err := repo.CreateEventLogCallbacks(ctx, opts)
	require.NoError(t, err)
	require.Len(t, callbacks, 1)

	assert.Equal(t, durableTaskId, callbacks[0].DurableTaskID)
	assert.Equal(t, "run:abc123", callbacks[0].Key)
	assert.Equal(t, int64(1), callbacks[0].NodeID)
	assert.False(t, callbacks[0].IsSatisfied)
}

func TestGetEventLogCallback(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()
	createDurableEventLogPartitions(t, pool)

	repo := createDurableEventsRepository(pool)
	ctx := context.Background()

	durableTaskId, durableTaskInsertedAt := newDurableTaskId()
	insertedAt := timestamptz(time.Now().UTC().Truncate(time.Microsecond))

	opts := []CreateEventLogCallbackOpts{
		{
			DurableTaskId:         durableTaskId,
			DurableTaskInsertedAt: durableTaskInsertedAt,
			InsertedAt:            insertedAt,
			Kind:                  "WAIT_FOR_COMPLETED",
			Key:                   "wait:sleep:5s",
			NodeId:                5,
			IsSatisfied:           true,
		},
	}

	_, err := repo.CreateEventLogCallbacks(ctx, opts)
	require.NoError(t, err)

	callback, err := repo.GetEventLogCallback(ctx, durableTaskId, durableTaskInsertedAt, "wait:sleep:5s")
	require.NoError(t, err)

	assert.Equal(t, durableTaskId, callback.DurableTaskID)
	assert.Equal(t, "wait:sleep:5s", callback.Key)
	assert.Equal(t, int64(5), callback.NodeID)
	assert.True(t, callback.IsSatisfied)
}

func TestListEventLogCallbacks(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()
	createDurableEventLogPartitions(t, pool)

	repo := createDurableEventsRepository(pool)
	ctx := context.Background()

	durableTaskId, durableTaskInsertedAt := newDurableTaskId()
	insertedAt := timestamptz(time.Now().UTC().Truncate(time.Microsecond))

	opts := []CreateEventLogCallbackOpts{
		{
			DurableTaskId:         durableTaskId,
			DurableTaskInsertedAt: durableTaskInsertedAt,
			InsertedAt:            insertedAt,
			Kind:                  "RUN_COMPLETED",
			Key:                   "run:task1",
			NodeId:                1,
			IsSatisfied:           false,
		},
		{
			DurableTaskId:         durableTaskId,
			DurableTaskInsertedAt: durableTaskInsertedAt,
			InsertedAt:            insertedAt,
			Kind:                  "WAIT_FOR_COMPLETED",
			Key:                   "wait:event:user_input",
			NodeId:                2,
			IsSatisfied:           false,
		},
		{
			DurableTaskId:         durableTaskId,
			DurableTaskInsertedAt: durableTaskInsertedAt,
			InsertedAt:            insertedAt,
			Kind:                  "MEMO_COMPLETED",
			Key:                   "memo:cache_key",
			NodeId:                3,
			IsSatisfied:           true,
		},
	}

	_, err := repo.CreateEventLogCallbacks(ctx, opts)
	require.NoError(t, err)

	callbacks, err := repo.ListEventLogCallbacks(ctx, durableTaskId, durableTaskInsertedAt)
	require.NoError(t, err)
	require.Len(t, callbacks, 3)

	keys := make([]string, len(callbacks))
	for i, c := range callbacks {
		keys[i] = c.Key
	}
	assert.ElementsMatch(t, []string{"run:task1", "wait:event:user_input", "memo:cache_key"}, keys)
}

func TestUpdateEventLogCallbackSatisfied(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()
	createDurableEventLogPartitions(t, pool)

	repo := createDurableEventsRepository(pool)
	ctx := context.Background()

	durableTaskId, durableTaskInsertedAt := newDurableTaskId()
	insertedAt := timestamptz(time.Now().UTC().Truncate(time.Microsecond))

	opts := []CreateEventLogCallbackOpts{
		{
			DurableTaskId:         durableTaskId,
			DurableTaskInsertedAt: durableTaskInsertedAt,
			InsertedAt:            insertedAt,
			Kind:                  "WAIT_FOR_COMPLETED",
			Key:                   "wait:child_workflow",
			NodeId:                10,
			IsSatisfied:           false,
		},
	}

	_, err := repo.CreateEventLogCallbacks(ctx, opts)
	require.NoError(t, err)

	callback, err := repo.GetEventLogCallback(ctx, durableTaskId, durableTaskInsertedAt, "wait:child_workflow")
	require.NoError(t, err)
	assert.False(t, callback.IsSatisfied)

	updated, err := repo.UpdateEventLogCallbackSatisfied(ctx, durableTaskId, durableTaskInsertedAt, "wait:child_workflow", true)
	require.NoError(t, err)
	assert.True(t, updated.IsSatisfied)

	callback, err = repo.GetEventLogCallback(ctx, durableTaskId, durableTaskInsertedAt, "wait:child_workflow")
	require.NoError(t, err)
	assert.True(t, callback.IsSatisfied)
}

// Callback satisfaction can toggle back to false per schema documentation:
// "is_satisfied _may_ change multiple times through the lifecycle of a callback"
func TestUpdateEventLogCallbackSatisfiedToggle(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()
	createDurableEventLogPartitions(t, pool)

	repo := createDurableEventsRepository(pool)
	ctx := context.Background()

	durableTaskId, durableTaskInsertedAt := newDurableTaskId()
	insertedAt := timestamptz(time.Now().UTC().Truncate(time.Microsecond))

	opts := []CreateEventLogCallbackOpts{
		{
			DurableTaskId:         durableTaskId,
			DurableTaskInsertedAt: durableTaskInsertedAt,
			InsertedAt:            insertedAt,
			Kind:                  "WAIT_FOR_COMPLETED",
			Key:                   "wait:toggleable",
			NodeId:                1,
			IsSatisfied:           false,
		},
	}

	_, err := repo.CreateEventLogCallbacks(ctx, opts)
	require.NoError(t, err)

	_, err = repo.UpdateEventLogCallbackSatisfied(ctx, durableTaskId, durableTaskInsertedAt, "wait:toggleable", true)
	require.NoError(t, err)

	callback, err := repo.GetEventLogCallback(ctx, durableTaskId, durableTaskInsertedAt, "wait:toggleable")
	require.NoError(t, err)
	assert.True(t, callback.IsSatisfied)

	_, err = repo.UpdateEventLogCallbackSatisfied(ctx, durableTaskId, durableTaskInsertedAt, "wait:toggleable", false)
	require.NoError(t, err)

	callback, err = repo.GetEventLogCallback(ctx, durableTaskId, durableTaskInsertedAt, "wait:toggleable")
	require.NoError(t, err)
	assert.False(t, callback.IsSatisfied)
}

func TestCreateMultipleEventLogFiles(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()
	createDurableEventLogPartitions(t, pool)

	repo := createDurableEventsRepository(pool)
	ctx := context.Background()

	task1Id, task1InsertedAt := newDurableTaskId()
	task2Id, task2InsertedAt := newDurableTaskId()

	latestInsertedAt := timestamptz(time.Now().UTC().Truncate(time.Microsecond))

	opts := []CreateEventLogFileOpts{
		{
			DurableTaskId:                 task1Id,
			DurableTaskInsertedAt:         task1InsertedAt,
			LatestInsertedAt:              latestInsertedAt,
			LatestNodeId:                  10,
			LatestBranchId:                1,
			LatestBranchFirstParentNodeId: 5,
		},
		{
			DurableTaskId:                 task2Id,
			DurableTaskInsertedAt:         task2InsertedAt,
			LatestInsertedAt:              latestInsertedAt,
			LatestNodeId:                  20,
			LatestBranchId:                2,
			LatestBranchFirstParentNodeId: 15,
		},
	}

	files, err := repo.CreateEventLogFiles(ctx, opts)
	require.NoError(t, err)
	require.Len(t, files, 2)

	file1, err := repo.GetEventLogFileForTask(ctx, task1Id, task1InsertedAt)
	require.NoError(t, err)
	assert.Equal(t, int64(10), file1.LatestNodeID)

	file2, err := repo.GetEventLogFileForTask(ctx, task2Id, task2InsertedAt)
	require.NoError(t, err)
	assert.Equal(t, int64(20), file2.LatestNodeID)
}

// Tree structure test: entries on different branches share the same durable task
// but have different branch_ids, allowing for replay from specific points.
func TestEventLogEntriesBranching(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()
	createDurableEventLogPartitions(t, pool)

	repo := createDurableEventsRepository(pool)
	ctx := context.Background()

	durableTaskId, durableTaskInsertedAt := newDurableTaskId()
	insertedAt := timestamptz(time.Now().UTC().Truncate(time.Microsecond))

	opts := []CreateEventLogEntryOpts{
		{
			TenantId:              uuid.New(),
			ExternalId:            uuid.New(),
			DurableTaskId:         durableTaskId,
			DurableTaskInsertedAt: durableTaskInsertedAt,
			InsertedAt:            insertedAt,
			Kind:                  "RUN_TRIGGERED",
			NodeId:                1,
			ParentNodeId:          0,
			BranchId:              0,
		},
		{
			TenantId:              uuid.New(),
			ExternalId:            uuid.New(),
			DurableTaskId:         durableTaskId,
			DurableTaskInsertedAt: durableTaskInsertedAt,
			InsertedAt:            insertedAt,
			Kind:                  "WAIT_FOR_STARTED",
			NodeId:                2,
			ParentNodeId:          1,
			BranchId:              0,
		},
		{
			TenantId:              uuid.New(),
			ExternalId:            uuid.New(),
			DurableTaskId:         durableTaskId,
			DurableTaskInsertedAt: durableTaskInsertedAt,
			InsertedAt:            insertedAt,
			Kind:                  "RUN_TRIGGERED",
			NodeId:                3,
			ParentNodeId:          1,
			BranchId:              1,
		},
	}

	_, err := repo.CreateEventLogEntries(ctx, opts)
	require.NoError(t, err)

	entries, err := repo.ListEventLogEntries(ctx, durableTaskId, durableTaskInsertedAt)
	require.NoError(t, err)
	require.Len(t, entries, 3)

	branch0Count := 0
	branch1Count := 0
	for _, e := range entries {
		if e.BranchID == 0 {
			branch0Count++
		} else if e.BranchID == 1 {
			branch1Count++
		}
	}

	assert.Equal(t, 2, branch0Count)
	assert.Equal(t, 1, branch1Count)
}

func TestEventLogEntriesIsolation(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()
	createDurableEventLogPartitions(t, pool)

	repo := createDurableEventsRepository(pool)
	ctx := context.Background()

	task1Id, task1InsertedAt := newDurableTaskId()
	task2Id, task2InsertedAt := newDurableTaskId()

	insertedAt := timestamptz(time.Now().UTC().Truncate(time.Microsecond))

	opts := []CreateEventLogEntryOpts{
		{
			TenantId:              uuid.New(),
			ExternalId:            uuid.New(),
			DurableTaskId:         task1Id,
			DurableTaskInsertedAt: task1InsertedAt,
			InsertedAt:            insertedAt,
			Kind:                  "RUN_TRIGGERED",
			NodeId:                1,
			ParentNodeId:          0,
			BranchId:              0,
		},
		{
			TenantId:              uuid.New(),
			ExternalId:            uuid.New(),
			DurableTaskId:         task1Id,
			DurableTaskInsertedAt: task1InsertedAt,
			InsertedAt:            insertedAt,
			Kind:                  "WAIT_FOR_STARTED",
			NodeId:                2,
			ParentNodeId:          1,
			BranchId:              0,
		},
	}
	_, err := repo.CreateEventLogEntries(ctx, opts)
	require.NoError(t, err)

	opts2 := []CreateEventLogEntryOpts{
		{
			TenantId:              uuid.New(),
			ExternalId:            uuid.New(),
			DurableTaskId:         task2Id,
			DurableTaskInsertedAt: task2InsertedAt,
			InsertedAt:            insertedAt,
			Kind:                  "MEMO_STARTED",
			NodeId:                1,
			ParentNodeId:          0,
			BranchId:              0,
		},
	}
	_, err = repo.CreateEventLogEntries(ctx, opts2)
	require.NoError(t, err)

	entries1, err := repo.ListEventLogEntries(ctx, task1Id, task1InsertedAt)
	require.NoError(t, err)
	assert.Len(t, entries1, 2)

	entries2, err := repo.ListEventLogEntries(ctx, task2Id, task2InsertedAt)
	require.NoError(t, err)
	assert.Len(t, entries2, 1)
}

func TestEventLogCallbacksIsolation(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()
	createDurableEventLogPartitions(t, pool)

	repo := createDurableEventsRepository(pool)
	ctx := context.Background()

	task1Id, task1InsertedAt := newDurableTaskId()
	task2Id, task2InsertedAt := newDurableTaskId()

	insertedAt := timestamptz(time.Now().UTC().Truncate(time.Microsecond))

	_, err := repo.CreateEventLogCallbacks(ctx, []CreateEventLogCallbackOpts{
		{
			DurableTaskId:         task1Id,
			DurableTaskInsertedAt: task1InsertedAt,
			InsertedAt:            insertedAt,
			Kind:                  "RUN_COMPLETED",
			Key:                   "shared_key_name",
			NodeId:                1,
			IsSatisfied:           false,
		},
	})
	require.NoError(t, err)

	_, err = repo.CreateEventLogCallbacks(ctx, []CreateEventLogCallbackOpts{
		{
			DurableTaskId:         task2Id,
			DurableTaskInsertedAt: task2InsertedAt,
			InsertedAt:            insertedAt,
			Kind:                  "RUN_COMPLETED",
			Key:                   "shared_key_name",
			NodeId:                1,
			IsSatisfied:           true,
		},
	})
	require.NoError(t, err)

	cb1, err := repo.GetEventLogCallback(ctx, task1Id, task1InsertedAt, "shared_key_name")
	require.NoError(t, err)
	assert.False(t, cb1.IsSatisfied)

	cb2, err := repo.GetEventLogCallback(ctx, task2Id, task2InsertedAt, "shared_key_name")
	require.NoError(t, err)
	assert.True(t, cb2.IsSatisfied)
}
