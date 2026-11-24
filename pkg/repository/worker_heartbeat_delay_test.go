//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestListTasksToReassignHonorsWorkerMaxHeartbeatDelay(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()

	taskRepo := createTaskRepository(pool)
	require.NoError(t, taskRepo.UpdateTablePartitions(ctx))

	workerRepo, _ := createMeteredWorkerRepositoryForTest(t, pool)
	shared := createSharedRepositoryForTest(pool)
	tenantId := createLimitTestTenant(t, pool)
	dispatcherId := seedDispatcherForLimits(t, ctx, pool)

	createWorkerWithAssignedTask := func(name string, taskId int64, maxHeartbeatDelay *time.Duration) uuid.UUID {
		worker, err := workerRepo.CreateNewWorker(ctx, tenantId, &CreateWorkerOpts{
			DispatcherId:                        dispatcherId,
			Name:                                name,
			SlotConfig:                          map[string]int32{SlotTypeDefault: 1},
			MaxHeartbeatDelayBeforeDeclaredDead: maxHeartbeatDelay,
		})
		require.NoError(t, err)

		queueItem := insertQueuedTaskForTest(t, ctx, pool, tenantId, taskId, "1h")

		tx, err := pool.Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback(ctx) // nolint: errcheck

		succeeded, _, err := shared.markQueueItemsProcessed(ctx, tenantId, &AssignResults{
			Assigned: []*AssignedItem{{WorkerId: worker.ID, QueueItem: queueItem}},
		}, tx)
		require.NoError(t, err)
		require.Len(t, succeeded, 1)
		require.NoError(t, tx.Commit(ctx))

		return worker.ID
	}

	setLastHeartbeatAgo := func(workerId uuid.UUID, ago time.Duration) {
		require.NoError(t, workerRepo.UpdateWorkerHeartbeat(ctx, tenantId, workerId, time.Now().UTC().Add(-ago)))
	}

	listReassignmentReasonByTaskId := func() map[int64]string {
		rows, err := taskRepo.queries.ListTasksToReassign(ctx, pool, sqlcv1.ListTasksToReassignParams{
			Tenantid: tenantId,
			Limit:    pgtype.Int4{Int32: 1000, Valid: true},
		})
		require.NoError(t, err)

		reasonByTaskId := make(map[int64]string, len(rows))
		for _, row := range rows {
			reasonByTaskId[row.ID] = row.Reason
		}

		return reasonByTaskId
	}

	const (
		defaultWorkerTaskId    int64 = 910001
		configuredWorkerTaskId int64 = 910002
	)

	oneMinute := time.Minute

	defaultWorkerId := createWorkerWithAssignedTask("default-delay-worker", defaultWorkerTaskId, nil)
	configuredWorkerId := createWorkerWithAssignedTask("one-minute-delay-worker", configuredWorkerTaskId, &oneMinute)

	setLastHeartbeatAgo(defaultWorkerId, 45*time.Second)
	setLastHeartbeatAgo(configuredWorkerId, 45*time.Second)

	reasonByTaskId := listReassignmentReasonByTaskId()

	assert.Equal(t, "Worker did not send a heartbeat for 30 seconds", reasonByTaskId[defaultWorkerTaskId], "a worker without a configured delay is declared dead after 30 seconds")
	assert.NotContains(t, reasonByTaskId, configuredWorkerTaskId, "a worker with a one minute delay is not declared dead after 45 seconds")

	setLastHeartbeatAgo(configuredWorkerId, 65*time.Second)

	reasonByTaskId = listReassignmentReasonByTaskId()

	assert.Equal(t, "Worker did not send a heartbeat for 60 seconds", reasonByTaskId[configuredWorkerTaskId], "a worker with a one minute delay is declared dead after a minute")
}
