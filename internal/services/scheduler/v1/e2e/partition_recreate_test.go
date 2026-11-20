//go:build !e2e && !load && !rampup && !integration

package e2e

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/testing/harness"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// TestMain runs the tests in this package against an in-process engine backed by a Postgres
// testcontainer, so they only need a Docker daemon.
func TestMain(m *testing.M) {
	// Optimistic scheduling assigns tasks inside the trigger request, without a scheduler
	// wake-up, which would hide a lost wake-up.
	if err := os.Setenv("TESTING_MATRIX_OPTIMISTIC_SCHEDULING", "false"); err != nil {
		panic(err)
	}

	harness.RunTestWithEngine(m)
}

const (
	engineReadyTimeout = 2 * time.Minute

	// A task started within this time was assigned on a wake-up: an idle queue's poll has
	// backed off to 30-45s by the time the test probes it.
	wakeupTimeout = 5 * time.Second

	// covers a heartbeat (5s) and an inactive rebalance (10s), both jittered
	reassignTimeout = time.Minute

	// The queue poll interval starts at 1s and doubles after three empty polls up to 30s, each
	// with up to 50% jitter, so a queue reaches the 30s cap at most 51s after its loop starts.
	// The loop restarts when the scheduler picks the tenant up again, up to 6s after the
	// reassignment (1s tenant refresh and 5s queue lease).
	idleTimeout = 60 * time.Second

	// Each probe task has its own queue with its own jittered poll timer. A lost wake-up fails
	// the probe unless every one of these polls happens to land within wakeupTimeout.
	probeTasks = 3

	pollInterval = 200 * time.Millisecond
)

// A scheduler subscribes to wake-ups on the topic of the partition id it started with. After a
// database outage longer than the partition staleness threshold, a rebalance deletes the
// partition rows and unassigns their tenants, and the heartbeat recreates the partition. Tasks
// queued after that must still be assigned on a wake-up, not on the backed-off queue poll.
func TestSchedulerWakeupsSurvivePartitionRecreate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	require.NoError(t, harness.WaitEngineReady(ctx, engineReadyTimeout))

	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	tenantID := tenantIDFromToken(t, os.Getenv("HATCHET_CLIENT_TOKEN"))

	client, err := hatchet.NewClient()
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close(context.Background()) })

	started := make(chan string, 2*probeTasks)
	tasks := make([]*hatchet.StandaloneTask, probeTasks)
	workflows := make([]hatchet.WorkflowBase, probeTasks)

	for i := range tasks {
		name := fmt.Sprintf("scheduler-partition-recreate-probe-%d", i)

		tasks[i] = client.NewStandaloneTask(name, func(ctx hatchet.Context, input struct{}) (struct{}, error) {
			started <- name
			return struct{}{}, nil
		})
		workflows[i] = tasks[i]
	}

	worker := newWorker(t, client, workflows...)

	stopWorker, err := worker.Start()
	require.NoError(t, err)
	t.Cleanup(func() { _ = stopWorker() })

	// shows the probe passes while the scheduler keeps its partition
	probe(ctx, t, tasks, started)

	partitionID := tenantSchedulerPartitionID(ctx, t, pool, tenantID)
	require.True(t, partitionID.Valid, "the tenant should have a scheduler partition")

	// what the inactive rebalance does once the database is back: the FK unassigns the tenants
	_, err = pool.Exec(ctx, `DELETE FROM "SchedulerPartition"`)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return tenantSchedulerPartitionID(ctx, t, pool, tenantID).Valid
	}, reassignTimeout, pollInterval, "the tenant was not assigned to a scheduler partition again")

	t.Logf("tenant reassigned from scheduler partition %s to %s", partitionID.String, tenantSchedulerPartitionID(ctx, t, pool, tenantID).String)

	recreatedAt := schedulerPartitionCreatedAt(ctx, t, pool)

	time.Sleep(idleTimeout)

	probe(ctx, t, tasks, started)

	// The idle wait spans several staleness windows. Heartbeats on the existing row must have kept
	// it alive: a row that went stale would have been deleted and recreated with a newer createdAt.
	require.Equal(t, recreatedAt, schedulerPartitionCreatedAt(ctx, t, pool), "heartbeats did not keep the scheduler partition alive")
}

// schedulerPartitionCreatedAt returns when the only scheduler partition was created.
func schedulerPartitionCreatedAt(ctx context.Context, t *testing.T, pool *pgxpool.Pool) time.Time {
	t.Helper()

	var (
		partitions int
		createdAt  pgtype.Timestamp
	)
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*), MIN("createdAt") FROM "SchedulerPartition"`).Scan(&partitions, &createdAt))
	require.Equal(t, 1, partitions, "the engine should have exactly one scheduler partition")

	return createdAt.Time
}

// probe runs every task and requires each to start within wakeupTimeout.
func probe(ctx context.Context, t *testing.T, tasks []*hatchet.StandaloneTask, started <-chan string) {
	t.Helper()

	start := time.Now()

	for _, task := range tasks {
		_, err := task.RunNoWait(ctx, struct{}{})
		require.NoError(t, err)
	}

	timeout := time.After(wakeupTimeout)

	for i := range tasks {
		select {
		case name := <-started:
			t.Logf("%s started after %s", name, time.Since(start).Round(time.Millisecond))
		case <-timeout:
			require.FailNowf(t, "task not started on a wake-up", "%d of %d tasks started within %s of being run", i, len(tasks), wakeupTimeout)
		}
	}
}

func tenantSchedulerPartitionID(ctx context.Context, t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) pgtype.Text {
	t.Helper()

	var partitionID pgtype.Text
	require.NoError(t, pool.QueryRow(ctx, `SELECT "schedulerPartitionId" FROM "Tenant" WHERE "id" = $1`, tenantID).Scan(&partitionID))

	return partitionID
}

// The harness engine reports its version as "testing", which the SDK treats as a legacy engine.
// Once the SDK's deprecation window for legacy engines has passed, NewWorker returns a
// DeprecationError on a random 1 in 5 calls, before it does any work, so retrying is safe.
const newWorkerAttempts = 20

func newWorker(t *testing.T, client *hatchet.Client, workflows ...hatchet.WorkflowBase) *hatchet.Worker {
	t.Helper()

	var err error

	for range newWorkerAttempts {
		var worker *hatchet.Worker

		worker, err = client.NewWorker("scheduler-partition-recreate-worker", hatchet.WithWorkflows(workflows...))

		var deprecation *hatchet.DeprecationError
		if !errors.As(err, &deprecation) {
			require.NoError(t, err)

			return worker
		}
	}

	require.NoError(t, err, "NewWorker kept returning the legacy-engine deprecation error")

	return nil
}

// tenantIDFromToken reads the subject claim, which is the tenant id for a tenant API token. The
// signature is not checked because the token comes from the harness.
func tenantIDFromToken(t *testing.T, token string) uuid.UUID {
	t.Helper()

	parts := strings.Split(token, ".")
	require.Len(t, parts, 3, "HATCHET_CLIENT_TOKEN is not a JWT")

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)

	var claims struct {
		Sub string `json:"sub"`
	}
	require.NoError(t, json.Unmarshal(payload, &claims))

	tenantID, err := uuid.Parse(claims.Sub)
	require.NoError(t, err)

	return tenantID
}
