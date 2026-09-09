//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/auth/token"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/operator/hostgrpc"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/contract"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

const (
	// Timings shared by the in-engine process (through TestMain's environment) and the
	// out-of-process instances the tests start.
	leaseTTL     = 3 * time.Second
	tick         = time.Second
	drainTimeout = 5 * time.Second

	// fullReload is the routing cache full reload interval of out-of-process instances; the
	// in-engine process has no environment knob for it and runs the 60s default.
	fullReload = 2 * time.Second

	testTimeout = 3 * time.Minute
	pollEvery   = 200 * time.Millisecond

	// registerWait bounds how long a freshly created endpoint may take to be polled and its
	// workflows registered: a lease tick, a routing refresh, the healthcheck and PutWorkflow.
	registerWait = 30 * time.Second

	// processLabel is the worker label the core writes with the owning process id.
	processLabel = "hatchet-serverless-process"
)

// tokens is the token source every out-of-process instance uses: one token per tenant the
// tests create, plus the harness tenant's token.
var tokens = &tokenRegistry{tokens: map[uuid.UUID]string{}}

type tokenRegistry struct {
	tokens map[uuid.UUID]string
	mu     sync.Mutex
}

func (r *tokenRegistry) add(tenantId uuid.UUID, tok string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tokens[tenantId] = tok
}

// Token implements hostgrpc.TokenSource.
func (r *tokenRegistry) Token(_ context.Context, tenantId uuid.UUID) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	tok, ok := r.tokens[tenantId]

	if !ok {
		return "", hostgrpc.ErrNoToken
	}

	return tok, nil
}

var (
	inEngineMu sync.Mutex
	inEngineId uuid.UUID
)

// testEnv is one test's handle on the shared engine: a database pool, the serverless
// repository over it, the data encryption service the engine uses and the in-engine
// process id.
type testEnv struct {
	t        *testing.T
	ctx      context.Context
	pool     *pgxpool.Pool
	repo     repository.ServerlessRepository
	enc      encryption.EncryptionService
	l        zerolog.Logger
	inEngine uuid.UUID
	lastRun  string
}

func newEnv(t *testing.T) *testEnv {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	t.Cleanup(cancel)

	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	l := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).Level(zerolog.InfoLevel).With().Timestamp().Str("test", t.Name()).Logger()

	repo, cleanupRepo := repository.NewServerlessRepositoryFromPool(pool, &l)
	t.Cleanup(func() { _ = cleanupRepo() })

	enc, err := encryption.NewLocalDataEncryption([]byte(os.Getenv("SERVER_ENCRYPTION_MASTER_KEYSET")))
	require.NoError(t, err)

	e := &testEnv{t: t, ctx: ctx, pool: pool, repo: repo, enc: enc, l: l}
	e.inEngine = e.inEngineProcess()

	return e
}

// inEngineProcess resolves the in-engine operator's process id once: before any test starts
// an out-of-process instance, the only live row in v1_serverless_process is the engine's.
func (e *testEnv) inEngineProcess() uuid.UUID {
	e.t.Helper()

	inEngineMu.Lock()
	defer inEngineMu.Unlock()

	if inEngineId != uuid.Nil {
		return inEngineId
	}

	e.pollUntil(60*time.Second, "in-engine serverless process row", func() (bool, error) {
		live := e.liveProcesses()

		if len(live) != 1 {
			return false, fmt.Errorf("expected exactly one live process, got %v", live)
		}

		inEngineId = live[0]

		return true, nil
	})

	return inEngineId
}

func (e *testEnv) pollUntil(timeout time.Duration, what string, fn func() (bool, error)) {
	e.t.Helper()

	deadline := time.Now().Add(timeout)

	var lastErr error

	for {
		done, err := fn()

		if err != nil {
			lastErr = err
		}

		if done {
			return
		}

		if time.Now().After(deadline) {
			e.t.Fatalf("timed out after %s waiting for %s (last error: %v)", timeout, what, lastErr)
		}

		select {
		case <-e.ctx.Done():
			e.t.Fatalf("context ended while waiting for %s: %v", what, e.ctx.Err())
		case <-time.After(pollEvery):
		}
	}
}

// tenant is a tenant the test created, with a token for the out-of-process host and an SDK
// client for triggering and inspecting runs.
type tenant struct {
	id  uuid.UUID
	sdk *hatchet.Client
}

// newTenant creates a tenant through the repository and mints a token for it the way the
// harness mints the default tenant's: with the engine's JWT keysets from the environment.
func (e *testEnv) newTenant() *tenant {
	e.t.Helper()

	dl, err := loader.NewConfigLoader("").InitDataLayer()
	require.NoError(e.t, err)

	defer func() { _ = dl.Disconnect() }()

	scf, err := loader.LoadServerConfigFile()
	require.NoError(e.t, err)

	jwtEnc, err := encryption.NewLocalEncryption(
		[]byte(os.Getenv("SERVER_ENCRYPTION_MASTER_KEYSET")),
		[]byte(os.Getenv("SERVER_ENCRYPTION_JWT_PRIVATE_KEYSET")),
		[]byte(os.Getenv("SERVER_ENCRYPTION_JWT_PUBLIC_KEYSET")),
	)
	require.NoError(e.t, err)

	jwt, err := token.NewJWTManager(jwtEnc, dl.V1.APIToken(), &token.TokenOpts{
		Issuer:               scf.Runtime.ServerURL,
		Audience:             scf.Runtime.ServerURL,
		ServerURL:            scf.Runtime.ServerURL,
		GRPCBroadcastAddress: scf.Runtime.GRPCBroadcastAddress,
	})
	require.NoError(e.t, err)

	slug := uniqueName("sl-e2e")

	row, err := dl.V1.Tenant().CreateTenant(e.ctx, &repository.CreateTenantOpts{Name: slug, Slug: slug})
	require.NoError(e.t, err)

	// A new tenant gets its scheduler partition from a once-a-minute rebalance; run it now so
	// the tenant's queue is scheduled right away (the scheduler refreshes its tenants every
	// second).
	require.NoError(e.t, dl.V1.Tenant().RebalanceInactiveSchedulerPartitions(e.ctx))

	e.pollUntil(10*time.Second, "tenant to be assigned a scheduler partition", func() (bool, error) {
		var partition *string

		err := e.pool.QueryRow(e.ctx, `SELECT "schedulerPartitionId" FROM "Tenant" WHERE "id" = $1`, row.ID).Scan(&partition)

		return err == nil && partition != nil, err
	})

	// A tenant is only scheduled once the partition rebalancer, which runs every 20s in the
	// engine, has assigned it controller, scheduler and worker partitions. Run the rebalance
	// now and wait for the assignment so runs triggered by the scenario are not left unassigned.
	require.NoError(e.t, dl.V1.Tenant().RebalanceAllControllerPartitions(e.ctx))
	require.NoError(e.t, dl.V1.Tenant().RebalanceAllSchedulerPartitions(e.ctx))
	require.NoError(e.t, dl.V1.Tenant().RebalanceAllTenantWorkerPartitions(e.ctx))

	require.Eventually(e.t, func() bool {
		current, err := dl.V1.Tenant().GetTenantByID(e.ctx, row.ID)

		return err == nil && current.ControllerPartitionId.Valid && current.SchedulerPartitionId.Valid && current.WorkerPartitionId.Valid
	}, 30*time.Second, 250*time.Millisecond, "tenant %s was not assigned partitions", row.ID)

	expires := time.Now().Add(24 * time.Hour)

	tok, err := jwt.GenerateTenantToken(e.ctx, row.ID, "e2e", false, &expires)
	require.NoError(e.t, err)

	tokens.add(row.ID, tok.Token)

	sdk, err := hatchet.NewClient(hatchet.WithToken(tok.Token))
	require.NoError(e.t, err)

	return &tenant{id: row.ID, sdk: sdk}
}

func uniqueName(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, uuid.New().String()[:8])
}

// pinLease writes the lease row of (tenant, shard) with process_id = owner. Call it before the
// tenant's endpoints exist: endpoint creation only inserts the row when absent, and the leaser
// only claims unowned or dead-process units, so a row pinned to a live process stays with it.
// The owner discovers the unit on its next rebalance tick (ListOwned) and opens its
// registration. Shedding cannot undo a pin of one weighted unit per process: a unit is shed
// only when its weight fits within the owner's excess over fair share, and with one weighted
// unit the excess is always below that weight. This is how the scenarios decide which host
// serves a tenant.
func (e *testEnv) pinLease(tenantId uuid.UUID, shard int32, owner uuid.UUID) {
	e.t.Helper()

	_, err := e.pool.Exec(e.ctx, `
		INSERT INTO v1_serverless_lease (tenant_id, shard, process_id, claimed_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (tenant_id, shard) DO UPDATE
		SET process_id = EXCLUDED.process_id, claimed_at = now()`,
		tenantId, shard, owner,
	)
	require.NoError(e.t, err)
}

func (e *testEnv) leaseOwner(tenantId uuid.UUID, shard int32) *uuid.UUID {
	e.t.Helper()

	var owner *uuid.UUID

	err := e.pool.QueryRow(e.ctx,
		`SELECT process_id FROM v1_serverless_lease WHERE tenant_id = $1 AND shard = $2`,
		tenantId, shard,
	).Scan(&owner)
	require.NoError(e.t, err)

	return owner
}

// ownedUnit is one lease row as seen by the tests.
type ownedUnit struct {
	tenantId      uuid.UUID
	shard         int32
	endpointCount int32
}

func (e *testEnv) unitsOwnedBy(processId uuid.UUID) []ownedUnit {
	e.t.Helper()

	rows, err := e.pool.Query(e.ctx,
		`SELECT tenant_id, shard, endpoint_count FROM v1_serverless_lease WHERE process_id = $1 ORDER BY tenant_id, shard`,
		processId,
	)
	require.NoError(e.t, err)

	defer rows.Close()

	out := []ownedUnit{}

	for rows.Next() {
		var u ownedUnit
		require.NoError(e.t, rows.Scan(&u.tenantId, &u.shard, &u.endpointCount))
		out = append(out, u)
	}

	require.NoError(e.t, rows.Err())

	return out
}

func weightedUnits(units []ownedUnit) int {
	n := 0

	for _, u := range units {
		if u.endpointCount > 0 {
			n++
		}
	}

	return n
}

func (e *testEnv) liveProcesses() []uuid.UUID {
	e.t.Helper()

	rows, err := e.pool.Query(e.ctx, `SELECT process_id FROM v1_serverless_process WHERE expires_at > now() ORDER BY process_id`)
	require.NoError(e.t, err)

	defer rows.Close()

	out := []uuid.UUID{}

	for rows.Next() {
		var id uuid.UUID
		require.NoError(e.t, rows.Scan(&id))
		out = append(out, id)
	}

	require.NoError(e.t, rows.Err())

	return out
}

func (e *testEnv) processRowExists(processId uuid.UUID) bool {
	e.t.Helper()

	var n int

	err := e.pool.QueryRow(e.ctx, `SELECT COUNT(*) FROM v1_serverless_process WHERE process_id = $1`, processId).Scan(&n)
	require.NoError(e.t, err)

	return n > 0
}

func (e *testEnv) waitLive(processId uuid.UUID) {
	e.t.Helper()

	e.pollUntil(10*time.Second, fmt.Sprintf("process %s to heartbeat", processId), func() (bool, error) {
		for _, id := range e.liveProcesses() {
			if id == processId {
				return true, nil
			}
		}

		return false, nil
	})
}

// endpointOpts are the per-endpoint knobs a scenario sets.
type endpointOpts struct {
	pollIntervalSeconds int32
	inlineWaitBudgetMs  int32
}

// createEndpoint creates an endpoint for the fake through the repository with the fake's
// secret encrypted the way the API does, and deletes it when the test ends so the lease unit's
// weight drops to zero and later scenarios are not rebalanced around it.
func (e *testEnv) createEndpoint(tn *tenant, fake *fakeEndpoint, opts endpointOpts) *sqlcv1.V1ServerlessEndpoint {
	e.t.Helper()

	secretEnc, err := e.enc.EncryptString(fake.secret, contract.SigningSecretEncryptionDataID)
	require.NoError(e.t, err)

	create := repository.CreateServerlessEndpointOpts{
		Name:             uniqueName(fake.name),
		HealthcheckUrl:   fake.healthcheckURL(),
		TriggerUrl:       fake.triggerURL(),
		SigningSecretEnc: secretEnc,
	}

	if opts.pollIntervalSeconds > 0 {
		create.PollIntervalSeconds = &opts.pollIntervalSeconds
	}

	if opts.inlineWaitBudgetMs > 0 {
		create.InlineWaitBudgetMs = &opts.inlineWaitBudgetMs
	}

	ep, err := e.repo.Endpoints().Create(e.ctx, tn.id, create)
	require.NoError(e.t, err)

	fake.attach(ep.ID)

	e.t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, _ = e.repo.Endpoints().Delete(ctx, tn.id, ep.ID)
	})

	return ep
}

func (e *testEnv) registeredActions(endpointId uuid.UUID) []string {
	e.t.Helper()

	var actions []string

	err := e.pool.QueryRow(e.ctx, `SELECT registered_actions FROM v1_serverless_endpoint WHERE id = $1`, endpointId).Scan(&actions)
	require.NoError(e.t, err)

	return actions
}

// waitRegistered waits until the endpoint's registered_actions carry every given action,
// which the owner writes after PutWorkflow succeeded.
func (e *testEnv) waitRegistered(endpointId uuid.UUID, actions ...string) {
	e.t.Helper()

	e.pollUntil(registerWait, fmt.Sprintf("endpoint %s to register %v", endpointId, actions), func() (bool, error) {
		have := map[string]struct{}{}

		for _, a := range e.registeredActions(endpointId) {
			have[a] = struct{}{}
		}

		for _, a := range actions {
			if _, ok := have[a]; !ok {
				return false, fmt.Errorf("registered_actions is %v", e.registeredActions(endpointId))
			}
		}

		return true, nil
	})
}

// workerRow is a serverless worker of a tenant with the process it belongs to.
type workerRow struct {
	id         uuid.UUID
	operatorId *uuid.UUID
	process    string
	active     bool
	paused     bool
	actions    []string
}

// serverlessWorkers lists the tenant's workers that carry the process label, with their
// linked actions.
func (e *testEnv) serverlessWorkers(tenantId uuid.UUID) []workerRow {
	e.t.Helper()

	rows, err := e.pool.Query(e.ctx, `
		SELECT w."id", w."operatorId", w."isActive", w."isPaused", l."strValue",
			COALESCE((
				SELECT array_agg(a."actionId" ORDER BY a."actionId")
				FROM "_ActionToWorker" atw
				JOIN "Action" a ON a."id" = atw."A"
				WHERE atw."B" = w."id"
			), '{}'::text[])
		FROM "Worker" w
		JOIN "WorkerLabel" l ON l."workerId" = w."id" AND l."key" = $2
		WHERE w."tenantId" = $1
		ORDER BY w."createdAt"`,
		tenantId, processLabel,
	)
	require.NoError(e.t, err)

	defer rows.Close()

	out := []workerRow{}

	for rows.Next() {
		var w workerRow
		var process *string

		require.NoError(e.t, rows.Scan(&w.id, &w.operatorId, &w.active, &w.paused, &process, &w.actions))

		if process != nil {
			w.process = *process
		}

		out = append(out, w)
	}

	require.NoError(e.t, rows.Err())

	return out
}

// activeWorkerWithAction returns the tenant's active serverless worker advertising the action.
func (e *testEnv) activeWorkerWithAction(tenantId uuid.UUID, action string) (workerRow, bool) {
	e.t.Helper()

	for _, w := range e.serverlessWorkers(tenantId) {
		if !w.active {
			continue
		}

		for _, a := range w.actions {
			if a == action {
				return w, true
			}
		}
	}

	return workerRow{}, false
}

func (e *testEnv) workersOfProcess(processId uuid.UUID) []workerRow {
	e.t.Helper()

	rows, err := e.pool.Query(e.ctx, `
		SELECT w."id", w."isActive", w."isPaused"
		FROM "Worker" w
		JOIN "WorkerLabel" l ON l."workerId" = w."id" AND l."key" = $1 AND l."strValue" = $2`,
		processLabel, processId.String(),
	)
	require.NoError(e.t, err)

	defer rows.Close()

	out := []workerRow{}

	for rows.Next() {
		var w workerRow
		require.NoError(e.t, rows.Scan(&w.id, &w.active, &w.paused))
		w.process = processId.String()
		out = append(out, w)
	}

	require.NoError(e.t, rows.Err())

	return out
}

// runToCompletion triggers the namespaced workflow and waits for the run to complete. The
// trigger is retried while the engine has not seen the workflow yet.
func (e *testEnv) runToCompletion(tn *tenant, workflow string, input map[string]any, timeout time.Duration) *client.RunDetails {
	e.t.Helper()

	var runId string

	e.pollUntil(registerWait, fmt.Sprintf("workflow %s to be triggerable", workflow), func() (bool, error) {
		ref, err := tn.sdk.RunNoWait(e.ctx, workflow, input)

		if err != nil {
			return false, err
		}

		runId = ref.RunId
		e.lastRun = runId

		return true, nil
	})

	return e.waitForCompletion(tn, runId, timeout)
}

func (e *testEnv) waitForCompletion(tn *tenant, runId string, timeout time.Duration) *client.RunDetails {
	e.t.Helper()

	var details *client.RunDetails

	e.pollUntil(timeout, fmt.Sprintf("run %s to complete", runId), func() (bool, error) {
		d, err := tn.sdk.Runs().GetDetails(e.ctx, uuid.MustParse(runId))

		if err != nil {
			return false, err
		}

		switch d.Status {
		case rest.V1TaskStatusCOMPLETED:
			details = d
			return true, nil
		case rest.V1TaskStatusFAILED, rest.V1TaskStatusCANCELLED:
			e.t.Fatalf("run %s ended with status %s: %s", runId, d.Status, describeTasks(d))
		}

		return false, nil
	})

	return details
}

func describeTasks(d *client.RunDetails) string {
	parts := make([]string, 0, len(d.TaskRuns))

	for name, tr := range d.TaskRuns {
		errMsg := ""

		if tr.Error != nil {
			errMsg = *tr.Error
		}

		parts = append(parts, fmt.Sprintf("%s: %s %q", name, tr.Status, errMsg))
	}

	return fmt.Sprint(parts)
}

func taskOutput(t *testing.T, details *client.RunDetails) map[string]any {
	t.Helper()

	task, ok := details.TaskRuns["task"]
	require.True(t, ok, "task run missing from %+v", details.TaskRuns)

	var out map[string]any
	require.NoError(t, json.Unmarshal(task.Output, &out))

	return out
}

// workflow is a one-task workflow definition as an endpoint advertises it: un-prefixed.
func workflow(name, action string, durable bool, retries int32) *v1.CreateWorkflowVersionRequest {
	return &v1.CreateWorkflowVersionRequest{
		Name: name,
		Tasks: []*v1.CreateTaskOpts{{
			ReadableId: "task",
			Action:     action,
			Timeout:    "60s",
			IsDurable:  durable,
			Retries:    retries,
		}},
	}
}

func namespaced(ns uuid.UUID, name string) string {
	return ns.String() + "_" + name
}

// dumpOnFailure logs what the fakes saw and the tenant's serverless workers when the test
// fails, since a run that never completes is otherwise silent.
func dumpOnFailure(t *testing.T, e *testEnv, tn *tenant, fakes ...*fakeEndpoint) {
	t.Helper()

	t.Cleanup(func() {
		if !t.Failed() {
			return
		}

		for _, f := range fakes {
			f.mu.Lock()
			t.Logf("fake %s requests: %+v runs: %+v", f.name, f.requests, f.runs)
			f.mu.Unlock()
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		e.ctx = ctx

		t.Logf("workers: %+v", e.serverlessWorkers(tn.id))
		t.Logf("lease owner: %v live: %v", e.leaseOwner(tn.id, 0), e.liveProcesses())

		rows, err := e.pool.Query(ctx, `SELECT w."id", w."dispatcherId", w."lastHeartbeatAt", w."lastListenerEstablished", w."isActive", w."isPaused", w."maxRuns" FROM "Worker" w WHERE w."tenantId" = $1`, tn.id)

		if err == nil {
			for rows.Next() {
				var id uuid.UUID
				var dispatcherId *uuid.UUID
				var hb, le *time.Time
				var active, paused bool
				var maxRuns *int32
				_ = rows.Scan(&id, &dispatcherId, &hb, &le, &active, &paused, &maxRuns)
				t.Logf("worker %s dispatcher=%v heartbeat=%v listener=%v active=%t paused=%t maxRuns=%v now=%v", id, dispatcherId, hb, le, active, paused, maxRuns, time.Now().UTC())
			}

			rows.Close()
		}

		rt, err := e.pool.Query(ctx, `SELECT row_to_json(r)::text FROM v1_task_runtime r WHERE r.tenant_id = $1`, tn.id)

		if err == nil {
			for rt.Next() {
				var row string
				_ = rt.Scan(&row)
				t.Logf("task runtime: %s", row)
			}

			rt.Close()
		}

		if e.lastRun != "" {
			d, err := tn.sdk.Runs().GetDetails(ctx, uuid.MustParse(e.lastRun))

			if err == nil {
				t.Logf("run %s status=%s tasks=%s", e.lastRun, d.Status, describeTasks(d))
			} else {
				t.Logf("run %s details error: %v", e.lastRun, err)
			}
		}
	})
}
