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
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	admincontracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/testing/harness"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// TestMain runs the tests in this package against an in-process engine backed by a Postgres
// testcontainer, so they only need a Docker daemon.
func TestMain(m *testing.M) {
	harness.RunTestWithEngine(m)
}

const (
	pollInterval = 200 * time.Millisecond

	engineReadyTimeout = 2 * time.Minute

	// the operator claims itself and registers its actions on 5s polls, so the first run of a
	// workflow can sit queued for a while before the orchestrator is assigned
	rootsStartTimeout = 60 * time.Second

	// the operator reports itself blocked every 10s, and the dispatcher evicts the orchestrator
	// on the first report where nothing is satisfied
	evictionTimeout = 90 * time.Second

	// how long the engine has to cancel the orchestrator and its children after the request
	cancelPropagationTimeout = 30 * time.Second

	testTimeout = 5 * time.Minute

	rootA = "cancel-root-a"
	rootB = "cancel-root-b"
	leaf  = "cancel-leaf"

	// A root blocks for this long if nothing cancels it. It outlasts eviction plus the
	// cancel propagation window, so a root that was never cancelled is still running when the
	// test inspects it instead of having finished on its own.
	rootMaxRuntime = 3 * time.Minute

	// how long teardown waits for cancelled roots to return before stopping the worker
	rootsExitTimeout = 30 * time.Second
)

type dagInput struct {
	// CompleteRootA makes the first root return immediately instead of blocking.
	CompleteRootA bool `json:"complete_root_a"`
}

// cancelEnv is a tenant with the DAG operator enabled and a worker serving a DAG whose two
// parallel roots block until cancelled. The leaf depends on both roots, so it is never reached.
type cancelEnv struct {
	db       *pgxpool.Pool
	workflow *hatchet.Workflow
	admin    admincontracts.AdminServiceClient
	token    string
}

func newCancelEnv(t *testing.T) *cancelEnv {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), engineReadyTimeout)
	defer cancel()

	require.NoError(t, harness.WaitEngineReady(ctx, engineReadyTimeout))

	db, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	require.NoError(t, err)
	t.Cleanup(db.Close)

	// The entitlement is read when the workflow is registered, so it has to be on before the
	// worker starts.
	enableDagOperator(ctx, t, db)

	client, err := hatchet.NewClient()
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close(context.Background()) })

	runningRoots := new(atomic.Int32)

	workflow := registerBlockingDag(client, runningRoots)

	worker := newWorker(t, client, workflow)

	stopWorker, err := worker.Start()
	require.NoError(t, err)
	t.Cleanup(func() { _ = stopWorker() })

	// Registered after stopWorker so it runs first. The engine delivers a cancel to the worker
	// after it has already recorded the root as cancelled, so stopping the worker straight after
	// the database shows the cancel can leave a root handler blocked past the end of the test.
	t.Cleanup(func() { waitForRootsToExit(t, runningRoots) })

	// The admin service is dialed directly because the client only reaches cancel through the
	// REST API, which the harness doesn't run.
	conn, err := grpc.NewClient(
		os.Getenv("SERVER_GRPC_BROADCAST_ADDRESS"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	return &cancelEnv{
		db:       db,
		workflow: workflow,
		admin:    admincontracts.NewAdminServiceClient(conn),
		token:    os.Getenv("HATCHET_CLIENT_TOKEN"),
	}
}

// The harness engine reports its version as "testing", which the SDK treats as a legacy engine.
// Once the SDK's deprecation window for legacy engines has passed, NewWorker returns a
// DeprecationError on a random 1 in 5 calls, before it does any work, so retrying is safe.
const newWorkerAttempts = 20

func newWorker(t *testing.T, client *hatchet.Client, workflow *hatchet.Workflow) *hatchet.Worker {
	t.Helper()

	var err error

	for range newWorkerAttempts {
		var worker *hatchet.Worker

		worker, err = client.NewWorker("dag-operator-cancel-worker", hatchet.WithWorkflows(workflow))

		var deprecation *hatchet.DeprecationError
		if !errors.As(err, &deprecation) {
			require.NoError(t, err)

			return worker
		}
	}

	require.NoError(t, err, "NewWorker kept returning the legacy-engine deprecation error")

	return nil
}

func enableDagOperator(ctx context.Context, t *testing.T, db *pgxpool.Pool) {
	t.Helper()

	// The engine creates tenants of its own on startup, so the entitlement has to go to the
	// tenant the client's token belongs to rather than whichever tenant comes back first.
	tenantID := tenantIDFromToken(t, os.Getenv("HATCHET_CLIENT_TOKEN"))

	_, err := db.Exec(ctx, `
		INSERT INTO tenant_entitlement (tenant_id, dag_operator) VALUES ($1, TRUE)
		ON CONFLICT (tenant_id) DO UPDATE SET dag_operator = TRUE
	`, tenantID)
	require.NoError(t, err)
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

func waitForRootsToExit(t *testing.T, running *atomic.Int32) {
	t.Helper()

	deadline := time.Now().Add(rootsExitTimeout)

	for running.Load() > 0 {
		if time.Now().After(deadline) {
			t.Logf("%d root handler(s) still running %s after teardown began", running.Load(), rootsExitTimeout)

			return
		}

		time.Sleep(pollInterval)
	}
}

// registerBlockingDag needs more than one task for the engine to run it on the DAG operator.
// running counts the root handlers that are currently executing.
func registerBlockingDag(client *hatchet.Client, running *atomic.Int32) *hatchet.Workflow {
	workflow := client.NewWorkflow("dag-operator-cancel")

	blockUntilCancelled := func(ctx hatchet.Context, input dagInput) (map[string]string, error) {
		running.Add(1)
		defer running.Add(-1)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(rootMaxRuntime):
			return map[string]string{"status": "completed"}, nil
		}
	}

	completeOrBlock := func(ctx hatchet.Context, input dagInput) (map[string]string, error) {
		if input.CompleteRootA {
			return map[string]string{"status": "completed"}, nil
		}

		return blockUntilCancelled(ctx, input)
	}

	timeout := hatchet.WithExecutionTimeout(2 * rootMaxRuntime)

	a := workflow.NewTask(rootA, completeOrBlock, timeout)
	b := workflow.NewTask(rootB, blockUntilCancelled, timeout)

	workflow.NewTask(leaf,
		func(ctx hatchet.Context, input dagInput) (map[string]string, error) {
			return map[string]string{"status": "completed"}, nil
		},
		hatchet.WithParents(a, b),
	)

	return workflow
}

func (e *cancelEnv) cancel(ctx context.Context, ids ...uuid.UUID) error {
	externalIDs := make([]string, len(ids))
	for i, id := range ids {
		externalIDs[i] = id.String()
	}

	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+e.token)

	_, err := e.admin.CancelTasks(ctx, &admincontracts.CancelTasksRequest{ExternalIds: externalIDs})

	return err
}

// These tests read engine state straight from the database: the API doesn't surface whether an
// orchestrator is evicted, and "the child is cancelled" is most directly a statement about the
// child's runtime row and task events.

// taskState is the engine's view of one task, read from the core tables.
type taskState struct {
	externalID uuid.UUID
	name       string
	hasRuntime bool // has a v1_task_runtime row: running on a worker, or evicted
	evicted    bool
	cancelled  bool // has a CANCELLED task event
	completed  bool // has a COMPLETED task event
}

func (s taskState) String() string {
	return fmt.Sprintf(
		"%s (%s): runtime=%t evicted=%t cancelled=%t completed=%t",
		s.name, s.externalID, s.hasRuntime, s.evicted, s.cancelled, s.completed,
	)
}

const taskStatesQuery = `
SELECT
    l.external_id,
    t.step_readable_id,
    rt.task_id IS NOT NULL AS has_runtime,
    rt.evicted_at IS NOT NULL AS evicted,
    EXISTS (
        SELECT 1 FROM v1_task_event ev
        WHERE (ev.task_id, ev.task_inserted_at) = (t.id, t.inserted_at)
          AND ev.event_type = 'CANCELLED'
    ) AS cancelled,
    EXISTS (
        SELECT 1 FROM v1_task_event ev
        WHERE (ev.task_id, ev.task_inserted_at) = (t.id, t.inserted_at)
          AND ev.event_type = 'COMPLETED'
    ) AS completed
FROM v1_lookup_table l
JOIN v1_task t ON (t.id, t.inserted_at) = (l.task_id, l.inserted_at)
LEFT JOIN v1_task_runtime rt ON (rt.task_id, rt.task_inserted_at, rt.retry_count) = (t.id, t.inserted_at, t.retry_count)
WHERE l.external_id = ANY($1::uuid[])
ORDER BY t.step_readable_id
`

func (e *cancelEnv) taskStates(ctx context.Context, externalIDs []uuid.UUID) ([]taskState, error) {
	rows, err := e.db.Query(ctx, taskStatesQuery, externalIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []taskState
	for rows.Next() {
		var s taskState
		if err := rows.Scan(&s.externalID, &s.name, &s.hasRuntime, &s.evicted, &s.cancelled, &s.completed); err != nil {
			return nil, err
		}
		states = append(states, s)
	}

	return states, rows.Err()
}

// For an operator run the run's external id belongs to the orchestrator task. For a regular DAG
// it belongs to a DAG row instead, so there is no task to join to.
const runIsOperatorQuery = `
SELECT COALESCE(t.is_dag_orchestrator, FALSE)
FROM v1_lookup_table l
LEFT JOIN v1_task t ON (t.id, t.inserted_at) = (l.task_id, l.inserted_at)
WHERE l.external_id = $1
`

// The children an operator DAG has spawned are the RUN entries in the orchestrator's durable
// event log.
const orchestratorChildrenQuery = `
SELECT DISTINCT e.child_task_external_id
FROM v1_lookup_table l
JOIN v1_task orch ON (orch.id, orch.inserted_at, orch.is_dag_orchestrator) = (l.task_id, l.inserted_at, TRUE)
JOIN v1_durable_event_log_entry e ON (e.durable_task_id, e.durable_task_inserted_at) = (orch.id, orch.inserted_at)
WHERE l.external_id = $1
  AND e.kind = 'RUN'
  AND e.child_task_external_id IS NOT NULL
`

func (e *cancelEnv) orchestratorChildren(ctx context.Context, runID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := e.db.Query(ctx, orchestratorChildrenQuery, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

func (e *cancelEnv) requireOperatorRun(ctx context.Context, t *testing.T, runID uuid.UUID) {
	t.Helper()

	deadline := time.Now().Add(15 * time.Second)

	for {
		var isOperator bool
		err := e.db.QueryRow(ctx, runIsOperatorQuery, runID).Scan(&isOperator)

		switch {
		case err == nil:
			require.True(t, isOperator, "run %s was not created as a DAG operator orchestrator", runID)
			return
		case !errors.Is(err, pgx.ErrNoRows):
			require.NoError(t, err)
		}

		if time.Now().After(deadline) {
			t.Fatalf("run %s never showed up in v1_lookup_table", runID)
		}

		time.Sleep(pollInterval)
	}
}

// startBlockingDag runs the DAG and returns once the operator has spawned both roots and each is
// either running on the worker or already finished. It registers a cleanup that cancels
// everything, so a failing test doesn't leave blocked roots holding worker slots.
func (e *cancelEnv) startBlockingDag(ctx context.Context, t *testing.T, input dagInput) (runID uuid.UUID, rootIDs []uuid.UUID) {
	t.Helper()

	ref, err := e.workflow.RunNoWait(ctx, input)
	require.NoError(t, err)

	runID = uuid.MustParse(ref.RunId)

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := e.cancel(cleanupCtx, append([]uuid.UUID{runID}, rootIDs...)...); err != nil {
			t.Logf("cleanup: could not cancel dag run %s: %v", runID, err)
		}
	})

	e.requireOperatorRun(ctx, t, runID)

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		ids, err := e.orchestratorChildren(ctx, runID)
		require.NoError(c, err)
		require.Len(c, ids, 2, "the orchestrator should have spawned both roots")

		states, err := e.taskStates(ctx, ids)
		require.NoError(c, err)
		require.Len(c, states, 2)

		for _, s := range states {
			assert.True(c, s.hasRuntime || s.completed, "root is not running yet: %s", s)
		}
	}, rootsStartTimeout, pollInterval)

	rootIDs, err = e.orchestratorChildren(ctx, runID)
	require.NoError(t, err)

	return runID, rootIDs
}

func (e *cancelEnv) requireOrchestratorEvicted(ctx context.Context, t *testing.T, runID uuid.UUID) {
	t.Helper()

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		states, err := e.taskStates(ctx, []uuid.UUID{runID})
		require.NoError(c, err)
		require.Len(c, states, 1)

		assert.True(c, states[0].evicted, "orchestrator is not evicted yet: %s", states[0])
	}, evictionTimeout, pollInterval)
}

// requireRunAndChildrenCancelled waits for the orchestrator and every child it had spawned to be
// cancelled: no runtime row left, a CANCELLED event recorded, and never completed.
func (e *cancelEnv) requireRunAndChildrenCancelled(ctx context.Context, t *testing.T, runID uuid.UUID, rootIDs []uuid.UUID) {
	t.Helper()

	all := append([]uuid.UUID{runID}, rootIDs...)

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		states, err := e.taskStates(ctx, all)
		require.NoError(c, err)
		require.Len(c, states, len(all))

		for _, s := range states {
			assert.True(c, s.cancelled && !s.hasRuntime && !s.completed, "not cancelled: %s", s)
		}
	}, cancelPropagationTimeout, pollInterval,
		"cancelling a dag operator run must cancel the orchestrator and every child it had already spawned")
}

func TestDagOperatorCancel(t *testing.T) {
	env := newCancelEnv(t)

	// Baseline: the orchestrator is still on the operator worker when the cancel arrives, so the
	// operator sees the cancel and cancels the children it triggered.
	t.Run("not_evicted", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		runID, rootIDs := env.startBlockingDag(ctx, t, dagInput{})

		orchestrator, err := env.taskStates(ctx, []uuid.UUID{runID})
		require.NoError(t, err)
		require.Len(t, orchestrator, 1)

		if orchestrator[0].evicted {
			t.Skip("orchestrator was evicted before the cancel was sent, so this run doesn't exercise the not-evicted path")
		}

		require.NoError(t, env.cancel(ctx, runID))

		env.requireRunAndChildrenCancelled(ctx, t, runID, rootIDs)
	})

	// With both roots blocked the orchestrator is evicted, which releases its worker and slots.
	// The cancel then has no operator worker to be delivered to, but the children that were
	// already spawned must still be cancelled.
	t.Run("evicted", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		runID, rootIDs := env.startBlockingDag(ctx, t, dagInput{})

		env.requireOrchestratorEvicted(ctx, t, runID)

		require.NoError(t, env.cancel(ctx, runID))

		env.requireRunAndChildrenCancelled(ctx, t, runID, rootIDs)
	})

	// Cancelling releases a task and records a CANCELLED event for it whether or not it had
	// already finished, so the children that were cancelled must be only the unfinished ones.
	t.Run("evicted_leaves_finished_child_alone", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		runID, rootIDs := env.startBlockingDag(ctx, t, dagInput{CompleteRootA: true})

		// the roots come back ordered by name, so the finished one is first
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			states, err := env.taskStates(ctx, rootIDs)
			require.NoError(c, err)
			require.Len(c, states, 2)

			assert.True(c, states[0].completed, "root a has not finished yet: %s", states[0])
			assert.True(c, states[1].hasRuntime, "root b is not running yet: %s", states[1])
		}, rootsStartTimeout, pollInterval)

		roots, err := env.taskStates(ctx, rootIDs)
		require.NoError(t, err)
		require.Len(t, roots, 2)

		finished, blocked := roots[0].externalID, roots[1].externalID

		env.requireOrchestratorEvicted(ctx, t, runID)

		require.NoError(t, env.cancel(ctx, runID))

		env.requireRunAndChildrenCancelled(ctx, t, runID, []uuid.UUID{blocked})

		states, err := env.taskStates(ctx, []uuid.UUID{finished})
		require.NoError(t, err)
		require.Len(t, states, 1)
		assert.True(t, states[0].completed && !states[0].cancelled,
			"a child that finished before the cancel must keep its result: %s", states[0])
	})
}
