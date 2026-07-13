//go:build integration

package v1_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/pgoutbox"
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
	ch        chan snapshotEvent
	cleanupCh chan uuid.UUID
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

func (c *captureSnapshotsExt) CleanupTenant(tenantId uuid.UUID) error {
	select {
	case c.cleanupCh <- tenantId:
	default:
	}
	return nil
}

func (c *captureSnapshotsExt) Cleanup() error { return nil }

func runWithDatabase(t *testing.T, test func(conf *database.Layer) error) {
	t.Helper()

	// `internal/testutils.Prepare` constructs a server config and requires a RabbitMQ URL.
	t.Setenv("SERVER_MSGQUEUE_RABBITMQ_URL", "amqp://user:password@localhost:5672/")

	testutils.RunTestWithDatabase(t, test)
}

// newTestOutbox builds a pgoutbox.Outbox for tests against the migrated test database. The
// outbox table and triggers are created by migrations, so auto-migration is disabled.
func newTestOutbox(t *testing.T, conf *database.Layer) pgoutbox.Outbox {
	t.Helper()

	o, err := pgoutbox.NewOutbox(t.Context(), conf.Pool, pgoutbox.WithAutoMigrate(false))
	require.NoError(t, err)

	return o
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

func waitForTenantCleanup(
	t *testing.T,
	cleanupCh <-chan uuid.UUID,
	tenantId uuid.UUID,
	timeout time.Duration,
) {
	t.Helper()

	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	for {
		select {
		case <-deadline.C:
			t.Fatalf("timed out waiting for cleanup of tenant %s", tenantId)
		case id := <-cleanupCh:
			if id == tenantId {
				return
			}
		}
	}
}

func requireSnapshotsEventuallyStopForTenant(
	t *testing.T,
	ch <-chan snapshotEvent,
	tenantId uuid.UUID,
	quietFor time.Duration,
	timeout time.Duration,
) {
	t.Helper()

	overallDeadline := time.NewTimer(timeout)
	defer overallDeadline.Stop()

	quietTimer := time.NewTimer(quietFor)
	defer quietTimer.Stop()

	for {
		select {
		case <-overallDeadline.C:
			t.Fatalf("timed out waiting for snapshots to stop for tenant %s after %s", tenantId, timeout)
		case <-quietTimer.C:
			if drainSnapshotsForTenant(ch, tenantId) {
				quietTimer.Reset(quietFor)
				continue
			}
			return
		case ev := <-ch:
			if ev.tenantId != tenantId {
				continue
			}
			if !quietTimer.Stop() {
				select {
				case <-quietTimer.C:
				default:
				}
			}
			quietTimer.Reset(quietFor)
		}
	}
}

func drainSnapshotsForTenant(ch <-chan snapshotEvent, tenantId uuid.UUID) bool {
	for {
		select {
		case ev := <-ch:
			if ev.tenantId == tenantId {
				return true
			}
		default:
			return false
		}
	}
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
			newTestOutbox(t, conf),
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
			true,
			nil, // promGate
		)
		require.NoError(t, err)
		defer func() { _ = cleanup() }()

		ext := &captureSnapshotsExt{
			ch:        make(chan snapshotEvent, 100),
			cleanupCh: make(chan uuid.UUID, 10),
		}
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
			newTestOutbox(t, conf),
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
			true,
			nil,
		)
		require.NoError(t, err)
		defer func() { _ = cleanup() }()

		ext := &captureSnapshotsExt{
			ch:        make(chan snapshotEvent, 100),
			cleanupCh: make(chan uuid.UUID, 10),
		}
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
			newTestOutbox(t, conf),
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
			true,
			nil,
		)
		require.NoError(t, err)
		defer func() { _ = cleanup() }()

		ext := &captureSnapshotsExt{
			ch:        make(chan snapshotEvent, 1000),
			cleanupCh: make(chan uuid.UUID, 10),
		}
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
			newTestOutbox(t, conf),
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
			true,
			nil,
		)
		require.NoError(t, err)
		defer func() { _ = cleanup() }()

		ext := &captureSnapshotsExt{
			ch:        make(chan snapshotEvent, 1000),
			cleanupCh: make(chan uuid.UUID, 10),
		}
		pool.Extensions.Add(ext)

		// Start the tenant and confirm we see snapshots for it.
		pool.SetTenants([]*sqlcv1.Tenant{tenant})
		_ = waitForWorkerUtilization(t, ext.ch, tenantId, workerId, 5*time.Second)

		// Remove tenant from pool and ensure snapshots stop.
		pool.SetTenants([]*sqlcv1.Tenant{})

		waitForTenantCleanup(t, ext.cleanupCh, tenantId, 5*time.Second)
		// In-flight snapshot callbacks can arrive after removal; wait for sustained quiescence.
		requireSnapshotsEventuallyStopForTenant(t, ext.ch, tenantId, 2*time.Second, 5*time.Second)

		return nil
	})
}
