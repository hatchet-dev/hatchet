//go:build !e2e && !load && !rampup && !integration

package e2e

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	admincontracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/statusutils"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
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
	repo     v1.Repository
	tenantID uuid.UUID
	workflow *hatchet.Workflow
	admin    admincontracts.AdminServiceClient
	token    string
}

func newCancelEnv(t *testing.T) *cancelEnv {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), engineReadyTimeout)
	defer cancel()

	require.NoError(t, harness.WaitEngineReady(ctx, engineReadyTimeout))

	// Mirrors how the harness engine itself builds a repository (pkg/testing/harness/engine.go):
	// a second connection to the same database the engine under test is using.
	cf := loader.NewConfigLoader("")
	dl, err := cf.InitDataLayer()
	require.NoError(t, err)
	t.Cleanup(func() { _ = dl.Disconnect() })

	token := os.Getenv("HATCHET_CLIENT_TOKEN")
	tenantID := tenantIDFromToken(t, token)

	// The entitlement is read when the workflow is registered, so it has to be on before the
	// worker starts.
	require.NoError(t, dl.V1.TenantEntitlement().SetEntitlements(ctx, tenantID, v1.TenantEntitlements{
		DAGOperator: true,
	}))

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
		repo:     dl.V1,
		tenantID: tenantID,
		workflow: workflow,
		admin:    admincontracts.NewAdminServiceClient(conn),
		token:    token,
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

// runDetails is the same repository call the admin service's GetRunDetails RPC uses
// (internal/services/admin/v1/server.go): for a DAG-operator orchestrator's own external id it
// reports the orchestrator's own status separately (OrchestratorStatus) from each child it has
// spawned (ReadableIdToDetails, keyed by step readable id). It returns (nil, nil) if runID hasn't
// shown up in the lookup table yet.
func (e *cancelEnv) runDetails(ctx context.Context, runID uuid.UUID) (*v1.WorkflowRunDetails, error) {
	return e.repo.Tasks().GetWorkflowRunResultDetails(ctx, e.tenantID, runID)
}

func (e *cancelEnv) requireOperatorRun(ctx context.Context, t *testing.T, runID uuid.UUID) {
	t.Helper()

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		details, err := e.runDetails(ctx, runID)
		require.NoError(c, err)
		require.NotNil(c, details, "run %s never showed up", runID)
		assert.NotNil(c, details.OrchestratorStatus, "run %s was not created as a DAG operator orchestrator", runID)
	}, 15*time.Second, pollInterval)
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
		details, err := e.runDetails(ctx, runID)
		require.NoError(c, err)
		require.NotNil(c, details)
		require.Len(c, details.ReadableIdToDetails, 2, "the orchestrator should have spawned both roots")

		for readableID, d := range details.ReadableIdToDetails {
			started := d.Status == statusutils.V1RunStatusRunning || d.Status == statusutils.V1RunStatusCompleted
			assert.True(c, started, "%s is not running yet: %s", readableID, d.Status)
		}
	}, rootsStartTimeout, pollInterval)

	details, err := e.runDetails(ctx, runID)
	require.NoError(t, err)
	require.NotNil(t, details)

	for _, d := range details.ReadableIdToDetails {
		rootIDs = append(rootIDs, d.ExternalId)
	}

	return runID, rootIDs
}

func (e *cancelEnv) requireOrchestratorEvicted(ctx context.Context, t *testing.T, runID uuid.UUID) {
	t.Helper()

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		details, err := e.runDetails(ctx, runID)
		require.NoError(c, err)
		require.NotNil(c, details)

		if assert.NotNil(c, details.OrchestratorStatus) {
			assert.Equal(c, statusutils.V1RunStatusEvicted, *details.OrchestratorStatus, "orchestrator is not evicted yet")
		}
	}, evictionTimeout, pollInterval)
}

// requireRunAndChildrenCancelled waits for the orchestrator and every child in rootIDs to reach
// CANCELLED status.
func (e *cancelEnv) requireRunAndChildrenCancelled(ctx context.Context, t *testing.T, runID uuid.UUID, rootIDs []uuid.UUID) {
	t.Helper()

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		details, err := e.runDetails(ctx, runID)
		require.NoError(c, err)
		require.NotNil(c, details)

		if assert.NotNil(c, details.OrchestratorStatus) {
			assert.Equal(c, statusutils.V1RunStatusCancelled, *details.OrchestratorStatus, "orchestrator not cancelled")
		}

		statusByExternalID := make(map[uuid.UUID]statusutils.V1RunStatus, len(details.ReadableIdToDetails))
		for _, d := range details.ReadableIdToDetails {
			statusByExternalID[d.ExternalId] = d.Status
		}

		for _, id := range rootIDs {
			status, found := statusByExternalID[id]
			assert.True(c, found && status == statusutils.V1RunStatusCancelled,
				"child %s not cancelled (found=%t, status=%s)", id, found, status)
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

		details, err := env.runDetails(ctx, runID)
		require.NoError(t, err)
		require.NotNil(t, details)
		require.NotNil(t, details.OrchestratorStatus)

		if *details.OrchestratorStatus == statusutils.V1RunStatusEvicted {
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

		runID, _ := env.startBlockingDag(ctx, t, dagInput{CompleteRootA: true})

		var blocked uuid.UUID

		require.EventuallyWithT(t, func(c *assert.CollectT) {
			details, err := env.runDetails(ctx, runID)
			require.NoError(c, err)
			require.NotNil(c, details)

			if a, ok := details.ReadableIdToDetails[rootA]; assert.True(c, ok, "%s has not spawned yet", rootA) {
				assert.Equal(c, statusutils.V1RunStatusCompleted, a.Status, "%s has not finished yet", rootA)
			}

			if b, ok := details.ReadableIdToDetails[rootB]; assert.True(c, ok, "%s has not spawned yet", rootB) {
				assert.Equal(c, statusutils.V1RunStatusRunning, b.Status, "%s is not running yet", rootB)
				blocked = b.ExternalId
			}
		}, rootsStartTimeout, pollInterval)

		env.requireOrchestratorEvicted(ctx, t, runID)

		require.NoError(t, env.cancel(ctx, runID))

		env.requireRunAndChildrenCancelled(ctx, t, runID, []uuid.UUID{blocked})

		details, err := env.runDetails(ctx, runID)
		require.NoError(t, err)
		require.NotNil(t, details)

		finished, ok := details.ReadableIdToDetails[rootA]
		require.True(t, ok)
		assert.Equal(t, statusutils.V1RunStatusCompleted, finished.Status,
			"a child that finished before the cancel must keep its result")
	})
}
