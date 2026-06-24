//go:build integration

package v1_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/testutils"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	repo "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	schedv1 "github.com/hatchet-dev/hatchet/pkg/scheduling/v1"
)

type snapshotEvent struct {
	tenantId uuid.UUID
	input    *schedv1.SnapshotInput
}

type captureSnapshotsExt struct {
	ch chan snapshotEvent
}

func (c *captureSnapshotsExt) SetTenants(_ []*sqlcv1.Tenant) {}

func (c *captureSnapshotsExt) ReportSnapshot(tenantId uuid.UUID, input *schedv1.SnapshotInput) {
	// non-blocking
	select {
	case c.ch <- snapshotEvent{tenantId: tenantId, input: input}:
	default:
	}
}

func (c *captureSnapshotsExt) PostAssign(_ uuid.UUID, _ *schedv1.PostAssignInput) {}

func (c *captureSnapshotsExt) CleanupTenant(_ uuid.UUID) error { return nil }

func (c *captureSnapshotsExt) Cleanup() error { return nil }

func runWithDatabase(t *testing.T, test func(conf *database.Layer) error) {
	t.Helper()

	// `internal/testutils.Prepare` constructs a server config and requires a RabbitMQ URL.
	t.Setenv("SERVER_MSGQUEUE_RABBITMQ_URL", "amqp://user:password@localhost:5672/")

	testutils.RunTestWithDatabase(t, test)
}

func requireSchedulerSchema(t *testing.T, ctx context.Context, conf *database.Layer) {
	t.Helper()

	var hasMaxRuns bool
	err := conf.Pool.QueryRow(
		ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_name = 'Worker' AND column_name = 'maxRuns'
		)`,
	).Scan(&hasMaxRuns)
	require.NoError(t, err)

	if !hasMaxRuns {
		t.Skip(`database schema is missing "Worker"."maxRuns"; run migrations (e.g. "task migrate") and re-run integration tests`)
	}
}

func createTenantDispatcherWorker(
	t *testing.T,
	ctx context.Context,
	r repo.Repository,
	tenantName string,
	maxRuns int,
	actions []string,
) (tenantId uuid.UUID, workerId uuid.UUID, tenant *sqlcv1.Tenant) {
	t.Helper()

	tenantId = uuid.New()
	tenant, err := r.Tenant().CreateTenant(ctx, &repo.CreateTenantOpts{
		ID:   &tenantId,
		Name: tenantName,
		Slug: fmt.Sprintf("%s-%s", tenantName, tenantId.String()),
	})
	require.NoError(t, err)

	dispatcherId := uuid.New()
	_, err = r.Dispatcher().CreateNewDispatcher(ctx, &repo.CreateDispatcherOpts{ID: dispatcherId})
	require.NoError(t, err)

	worker, err := r.Workers().CreateNewWorker(ctx, tenantId, &repo.CreateWorkerOpts{
		DispatcherId: dispatcherId,
		Name:         "worker-it",
		Services:     []string{},
		Actions:      actions,
		SlotConfig:   map[string]int32{repo.SlotTypeDefault: int32(maxRuns)},
	})
	require.NoError(t, err)

	now := time.Now().UTC()
	require.NoError(t, r.Workers().UpdateWorkerHeartbeat(ctx, tenantId, worker.ID, now))

	isActive := true
	isPaused := false
	_, err = r.Workers().UpdateWorker(ctx, tenantId, worker.ID, &repo.UpdateWorkerOpts{
		IsActive: &isActive,
		IsPaused: &isPaused,
	})
	require.NoError(t, err)

	return tenantId, worker.ID, tenant
}

func waitForWorkerUtilization(
	t *testing.T,
	ch <-chan snapshotEvent,
	tenantId uuid.UUID,
	workerId uuid.UUID,
	timeout time.Duration,
) *schedv1.SlotUtilization {
	t.Helper()

	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	for {
		select {
		case <-deadline.C:
			t.Fatalf("timed out waiting for snapshot for tenant %s worker %s", tenantId, workerId)
		case ev := <-ch:
			if ev.input == nil || ev.tenantId != tenantId {
				continue
			}

			if ev.input.WorkerSlotUtilization == nil {
				continue
			}

			if util, ok := ev.input.WorkerSlotUtilization[workerId]; ok {
				// Skip snapshots captured before replenish has populated any slots.
				if util.UtilizedSlots+util.NonUtilizedSlots == 0 {
					continue
				}
				return util
			}
		}
	}
}

func requireNoSnapshotsForTenant(
	t *testing.T,
	ch <-chan snapshotEvent,
	tenantId uuid.UUID,
	dur time.Duration,
) {
	t.Helper()

	timer := time.NewTimer(dur)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			return
		case ev := <-ch:
			if ev.tenantId == tenantId {
				t.Fatalf("unexpected snapshot for removed tenant %s", tenantId)
			}
		}
	}
}

func TestScheduler_NotifyQueuesColdStartsTenantManager(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		requireSchedulerSchema(t, ctx, conf)

		action := "test:run"
		tenantId, _, _ := createTenantDispatcherWorker(t, ctx, conf.V1, "scheduler-queue-cold-start", 1, []string{action})

		desc := "queue cold-start workflow"
		wfVersion, err := conf.V1.Workflows().PutWorkflowVersion(ctx, tenantId, &repo.CreateWorkflowVersionOpts{
			Name:        "queue-cold-start-test",
			Description: &desc,
			Tasks: []repo.CreateStepOpts{
				{
					ReadableId: "my-task",
					Action:     action,
				},
			},
		})
		require.NoError(t, err)

		shape, err := conf.V1.Workflows().GetWorkflowShape(ctx, wfVersion.WorkflowVersion.ID)
		require.NoError(t, err)
		require.Len(t, shape, 1)

		taskParams := newCreateTasksParams(1)
		taskParams.Tenantids[0] = tenantId
		taskParams.Queues[0] = "default"
		taskParams.Actionids[0] = action
		taskParams.Stepids[0] = shape[0].Parentstepid
		taskParams.Stepreadableids[0] = "my-task"
		taskParams.Workflowids[0] = wfVersion.WorkflowVersion.WorkflowId
		taskParams.Scheduletimeouts[0] = "5m"
		taskParams.Priorities[0] = 1
		taskParams.Stickies[0] = string(sqlcv1.V1StickyStrategyNONE)
		taskParams.Externalids[0] = uuid.New()
		taskParams.Displaynames[0] = "queue-cold-start-task"
		taskParams.Inputs[0] = []byte(`{}`)
		taskParams.Additionalmetadatas[0] = []byte(`{}`)
		taskParams.InitialStates[0] = string(sqlcv1.V1TaskInitialStateQUEUED)
		taskParams.Concurrencyparentstrategyids[0] = []pgtype.Int8{}
		taskParams.ConcurrencyStrategyIds[0] = []int64{}
		taskParams.ConcurrencyKeys[0] = []string{}
		taskParams.WorkflowVersionIds[0] = wfVersion.WorkflowVersion.ID
		taskParams.WorkflowRunIds[0] = uuid.New()

		queries := sqlcv1.New()

		err = queries.UpsertQueues(ctx, conf.Pool, sqlcv1.UpsertQueuesParams{
			TenantID: tenantId,
			Names:    []string{"default"},
		})
		require.NoError(t, err)

		tasks, err := queries.CreateTasks(ctx, conf.Pool, taskParams)
		require.NoError(t, err)
		require.Len(t, tasks, 1)

		l := zerolog.Nop()
		pool, cleanup, err := schedv1.NewSchedulingPool(
			conf.V1.Scheduler(),
			&l,
			100,
			20,
			5*time.Millisecond,
			6*time.Millisecond,
			50*time.Millisecond,
			100*time.Millisecond,
			5*time.Millisecond,
			false,
			1,
			nil,
		)
		require.NoError(t, err)
		defer func() { _ = cleanup() }()

		resultsChan := pool.GetResultsCh()
		pool.NotifyQueues(ctx, tenantId, []string{"default"})

		deadline := time.NewTimer(2 * time.Second)
		defer deadline.Stop()

		for {
			select {
			case <-deadline.C:
				t.Fatal("timed out waiting for queue assignment after cold-start notification")
			case res := <-resultsChan:
				if res.TenantId != tenantId || len(res.Assigned) == 0 {
					continue
				}

				require.Len(t, res.Assigned, 1)
				return nil
			}
		}
	})
}

func TestScheduler_ReplenishIntegration_SingleActionUtilizationEqualsMaxRuns(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		requireSchedulerSchema(t, ctx, conf)

		actionA := "test:run"
		maxRuns := 3

		tenantId, workerId, tenant := createTenantDispatcherWorker(t, ctx, conf.V1, "scheduler-it", maxRuns, []string{actionA})

		l := zerolog.Nop()
		pool, cleanup, err := schedv1.NewSchedulingPool(
			conf.V1.Scheduler(),
			&l,
			100,                  // singleQueueLimit
			20,                   // schedulerConcurrencyRateLimit
			10*time.Millisecond,  // schedulerConcurrencyPollingMinInterval
			50*time.Millisecond,  // schedulerConcurrencyPollingMaxInterval
			50*time.Millisecond,  // schedulerCheckActiveMinInterval
			100*time.Millisecond, // schedulerCheckActiveMaxInterval
			5*time.Millisecond,   // schedulerAdvisoryLockTimeout
			false,                // optimisticSchedulingEnabled
			1,                    // optimisticSlots
			nil,                  // promGate
		)
		require.NoError(t, err)
		defer func() { _ = cleanup() }()

		ext := &captureSnapshotsExt{ch: make(chan snapshotEvent, 100)}
		pool.Extensions.Add(ext)

		pool.SetTenants([]*sqlcv1.Tenant{tenant})

		util := waitForWorkerUtilization(t, ext.ch, tenantId, workerId, 5*time.Second)
		require.Equal(t, 0, util.UtilizedSlots)
		require.Equal(t, maxRuns, util.NonUtilizedSlots)

		return nil
	})
}

func TestScheduler_ReplenishIntegration_MultipleActionsDoesNotMultiplySlots(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		requireSchedulerSchema(t, ctx, conf)

		// If replenish created distinct slots per action (instead of sharing per worker capacity),
		// snapshots would report NonUtilizedSlots == maxRuns * len(actions). We expect == maxRuns.
		actionA := "test:run"
		actionB := "test:other"
		maxRuns := 2

		tenantId, workerId, tenant := createTenantDispatcherWorker(t, ctx, conf.V1, "scheduler-it2", maxRuns, []string{actionA, actionB})

		l := zerolog.Nop()
		pool, cleanup, err := schedv1.NewSchedulingPool(
			conf.V1.Scheduler(),
			&l,
			100,
			20,
			10*time.Millisecond,
			50*time.Millisecond,
			50*time.Millisecond,
			100*time.Millisecond,
			5*time.Millisecond,
			false,
			1,
			nil,
		)
		require.NoError(t, err)
		defer func() { _ = cleanup() }()

		ext := &captureSnapshotsExt{ch: make(chan snapshotEvent, 100)}
		pool.Extensions.Add(ext)

		pool.SetTenants([]*sqlcv1.Tenant{tenant})

		util := waitForWorkerUtilization(t, ext.ch, tenantId, workerId, 5*time.Second)
		require.Equal(t, 0, util.UtilizedSlots)
		require.Equal(t, maxRuns, util.NonUtilizedSlots)

		return nil
	})
}

func TestScheduler_ReplenishIntegration_IsSafeUnderConcurrentSnapshots(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		requireSchedulerSchema(t, ctx, conf)

		actionA := "test:run"
		maxRuns := 2
		_, _, tenant := createTenantDispatcherWorker(t, ctx, conf.V1, "scheduler-it3", maxRuns, []string{actionA})

		l := zerolog.Nop()
		pool, cleanup, err := schedv1.NewSchedulingPool(
			conf.V1.Scheduler(),
			&l,
			100,
			20,
			10*time.Millisecond,
			50*time.Millisecond,
			50*time.Millisecond,
			100*time.Millisecond,
			5*time.Millisecond,
			false,
			1,
			nil,
		)
		require.NoError(t, err)
		defer func() { _ = cleanup() }()

		ext := &captureSnapshotsExt{ch: make(chan snapshotEvent, 1000)}
		pool.Extensions.Add(ext)
		pool.SetTenants([]*sqlcv1.Tenant{tenant})

		// Drain a few snapshots concurrently to smoke-test races/panics around snapshotting + replenish.
		wg := sync.WaitGroup{}
		wg.Add(1)

		go func() {
			defer wg.Done()
			timeout := time.NewTimer(2 * time.Second)
			defer timeout.Stop()
			for {
				select {
				case <-timeout.C:
					return
				case <-ext.ch:
				}
			}
		}()

		wg.Wait()
		return nil
	})
}

func TestScheduler_PoolIntegration_RemovingTenantStopsSnapshots(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		requireSchedulerSchema(t, ctx, conf)

		actionA := "test:run"
		maxRuns := 2

		tenantId, workerId, tenant := createTenantDispatcherWorker(t, ctx, conf.V1, "scheduler-it-remove", maxRuns, []string{actionA})

		l := zerolog.Nop()
		pool, cleanup, err := schedv1.NewSchedulingPool(
			conf.V1.Scheduler(),
			&l,
			100,
			20,
			10*time.Millisecond,
			50*time.Millisecond,
			50*time.Millisecond,
			100*time.Millisecond,
			5*time.Millisecond,
			false,
			1,
			nil,
		)
		require.NoError(t, err)
		defer func() { _ = cleanup() }()

		ext := &captureSnapshotsExt{ch: make(chan snapshotEvent, 1000)}
		pool.Extensions.Add(ext)

		// Start the tenant and confirm we see snapshots for it.
		pool.SetTenants([]*sqlcv1.Tenant{tenant})
		_ = waitForWorkerUtilization(t, ext.ch, tenantId, workerId, 5*time.Second)

		// Remove tenant from pool and ensure snapshots stop.
		pool.SetTenants([]*sqlcv1.Tenant{})

		// Give cleanup a short moment to cancel loops, then assert no new snapshots arrive.
		time.Sleep(50 * time.Millisecond)
		requireNoSnapshotsForTenant(t, ext.ch, tenantId, 350*time.Millisecond)

		return nil
	})
}
