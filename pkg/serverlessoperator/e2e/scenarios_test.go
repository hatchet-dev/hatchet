//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// linkCase is one of the two hosts a scenario runs against: the in-engine process (the
// in-process host), or an out-of-process instance (the gRPC host) started for the subtest.
// owner is the process id the scenario pins its tenant's lease to.
type linkCase struct {
	e     *testEnv
	owner uuid.UUID
	name  string
}

// forEachLink runs the scenario once per host. The in-engine case runs first with no
// out-of-process instance alive, the grpc case starts one and stops it at the end of the
// subtest, so at most one out-of-process instance is ever live during these scenarios.
func forEachLink(t *testing.T, fn func(t *testing.T, lc linkCase)) {
	t.Run("engine", func(t *testing.T) {
		e := newEnv(t)
		fn(t, linkCase{e: e, owner: e.inEngine, name: "engine"})
	})

	t.Run("grpc", func(t *testing.T) {
		e := newEnv(t)
		p := startProcess(t, e)
		fn(t, linkCase{e: e, owner: p.id, name: "grpc"})
	})
}

// fullReloadWait is how long the scenario waits for a deleted endpoint to leave the routing
// cache: the out-of-process instances reload every fullReload, the in-engine process every
// 60s (the default, which has no environment knob).
func (lc linkCase) fullReloadWait() time.Duration {
	if lc.name == "engine" {
		return 90 * time.Second
	}

	return 15 * time.Second
}

// assertServedBy checks that the tenant's action is advertised by an active worker of the
// expected process with an operator id.
func assertServedBy(t *testing.T, e *testEnv, tenantId uuid.UUID, action string, owner uuid.UUID) {
	t.Helper()

	w, ok := e.activeWorkerWithAction(tenantId, action)
	require.True(t, ok, "no active serverless worker advertises %s: %+v", action, e.serverlessWorkers(tenantId))
	assert.NotNil(t, w.operatorId, "worker %s has no operator id", w.id)
	assert.Equal(t, owner.String(), w.process, "worker %s belongs to another process", w.id)
}

func TestEcho(t *testing.T) {
	forEachLink(t, func(t *testing.T, lc linkCase) {
		e := lc.e
		tn := e.newTenant()
		e.pinLease(tn.id, 0, lc.owner)

		fake := newFakeEndpoint(t, "alpha", workflow("echo", "svc:echo", false, 0))
		ep := e.createEndpoint(tn, fake, endpointOpts{})
		ns := ep.Namespace
		action := namespaced(ns, "svc:echo")

		dumpOnFailure(t, e, tn, fake)

		e.waitRegistered(ep.ID, action)

		input := map[string]any{"message": "hello", "n": float64(3)}
		details := e.runToCompletion(tn, namespaced(ns, "echo"), input, 30*time.Second)

		out := taskOutput(t, details)
		assert.Equal(t, input, out["input"])
		assert.Equal(t, "alpha", out["endpoint"])

		triggers := fake.requestsOfKind("trigger")
		require.Len(t, triggers, 1)
		assert.Equal(t, ns.String(), triggers[0].namespace)
		assert.Equal(t, ep.ID.String(), triggers[0].endpointId)
		assert.Equal(t, action, triggers[0].actionId)

		healthchecks := fake.requestsOfKind("healthcheck")
		require.NotEmpty(t, healthchecks)
		assert.Equal(t, ns.String(), healthchecks[0].namespace)

		assertServedBy(t, e, tn.id, action, lc.owner)

		owner := e.leaseOwner(tn.id, 0)
		require.NotNil(t, owner)
		assert.Equal(t, lc.owner, *owner)
	})
}

func TestSameNamesCoexist(t *testing.T) {
	forEachLink(t, func(t *testing.T, lc linkCase) {
		e := lc.e
		tn := e.newTenant()
		e.pinLease(tn.id, 0, lc.owner)

		alpha := newFakeEndpoint(t, "alpha", workflow("echo", "svc:echo", false, 0))
		beta := newFakeEndpoint(t, "beta", workflow("echo", "svc:echo", false, 0))

		epA := e.createEndpoint(tn, alpha, endpointOpts{})
		epB := e.createEndpoint(tn, beta, endpointOpts{})

		dumpOnFailure(t, e, tn, alpha, beta)

		e.waitRegistered(epA.ID, namespaced(epA.Namespace, "svc:echo"))
		e.waitRegistered(epB.ID, namespaced(epB.Namespace, "svc:echo"))

		outA := taskOutput(t, e.runToCompletion(tn, namespaced(epA.Namespace, "echo"), map[string]any{"to": "alpha"}, 30*time.Second))
		outB := taskOutput(t, e.runToCompletion(tn, namespaced(epB.Namespace, "echo"), map[string]any{"to": "beta"}, 30*time.Second))

		assert.Equal(t, "alpha", outA["endpoint"])
		assert.Equal(t, map[string]any{"to": "alpha"}, outA["input"])
		assert.Equal(t, "beta", outB["endpoint"])
		assert.Equal(t, map[string]any{"to": "beta"}, outB["input"])

		require.Len(t, alpha.requestsOfKind("trigger"), 1)
		require.Len(t, beta.requestsOfKind("trigger"), 1)
		assert.Equal(t, epA.Namespace.String(), alpha.requestsOfKind("trigger")[0].namespace)
		assert.Equal(t, epB.Namespace.String(), beta.requestsOfKind("trigger")[0].namespace)

		// Both endpoints' actions sit on the same registration.
		w, ok := e.activeWorkerWithAction(tn.id, namespaced(epA.Namespace, "svc:echo"))
		require.True(t, ok)
		assert.Contains(t, w.actions, namespaced(epB.Namespace, "svc:echo"))
		assert.Equal(t, lc.owner.String(), w.process)
	})
}

func TestDurableMemoAndSleep(t *testing.T) {
	forEachLink(t, func(t *testing.T, lc linkCase) {
		e := lc.e
		tn := e.newTenant()
		e.pinLease(tn.id, 0, lc.owner)

		fake := newFakeEndpoint(t, "durable", workflow("sleeper", "svc:sleeper", true, 0))
		fake.setScript(durableScript{
			memoKey:   "memo-key",
			memoValue: json.RawMessage(`{"v":"memoized"}`),
			sleep:     2 * time.Second,
		})

		ep := e.createEndpoint(tn, fake, endpointOpts{inlineWaitBudgetMs: 500})
		ns := ep.Namespace

		dumpOnFailure(t, e, tn, fake)

		e.waitRegistered(ep.ID, namespaced(ns, "svc:sleeper"))

		details := e.runToCompletion(tn, namespaced(ns, "sleeper"), map[string]any{}, 60*time.Second)

		out := taskOutput(t, details)
		assert.Equal(t, map[string]any{"v": "memoized"}, out["memo"])

		runs := fake.durableRuns()
		require.Len(t, runs, 2, "expected one evicted invocation and one re-invocation: %+v", runs)

		assert.False(t, runs[0].memoExisted)
		assert.True(t, runs[0].evicted)

		assert.Equal(t, runs[0].invocation+1, runs[1].invocation, "the re-invocation carries the next invocation count")
		assert.True(t, runs[1].memoExisted, "the memo must replay on the re-invocation")
		assert.False(t, runs[1].evicted)

		assert.Equal(t, float64(runs[1].invocation), out["invocation"], "the run must complete on the re-invocation")

		upgrades := fake.requestsOfKind("upgrade")
		require.Len(t, upgrades, 2)
		assert.Equal(t, ns.String(), upgrades[0].namespace)
		assert.Equal(t, namespaced(ns, "svc:sleeper"), upgrades[0].actionId)

		assertServedBy(t, e, tn.id, namespaced(ns, "svc:sleeper"), lc.owner)
	})
}

func TestDurableCrash(t *testing.T) {
	forEachLink(t, func(t *testing.T, lc linkCase) {
		e := lc.e
		tn := e.newTenant()
		e.pinLease(tn.id, 0, lc.owner)

		fake := newFakeEndpoint(t, "crashy", workflow("crash", "svc:crash", true, 1))
		fake.setScript(durableScript{crashFirstAttempt: true})

		ep := e.createEndpoint(tn, fake, endpointOpts{})
		ns := ep.Namespace

		dumpOnFailure(t, e, tn, fake)

		e.waitRegistered(ep.ID, namespaced(ns, "svc:crash"))

		details := e.runToCompletion(tn, namespaced(ns, "crash"), map[string]any{}, 60*time.Second)
		assert.Contains(t, taskOutput(t, details), "invocation")

		runs := fake.durableRuns()
		require.Len(t, runs, 2, "expected a crashed attempt and a retry: %+v", runs)
		assert.True(t, runs[0].crashed)
		assert.Equal(t, int32(0), runs[0].retryCount)
		assert.False(t, runs[1].crashed)
		assert.Equal(t, int32(1), runs[1].retryCount)
	})
}

func TestHealthcheckChangeAddsWorkflow(t *testing.T) {
	forEachLink(t, func(t *testing.T, lc linkCase) {
		e := lc.e
		tn := e.newTenant()
		e.pinLease(tn.id, 0, lc.owner)

		fake := newFakeEndpoint(t, "growing", workflow("first", "svc:first", false, 0))
		ep := e.createEndpoint(tn, fake, endpointOpts{pollIntervalSeconds: 5})
		ns := ep.Namespace

		dumpOnFailure(t, e, tn, fake)

		e.waitRegistered(ep.ID, namespaced(ns, "svc:first"))

		fake.setWorkflows(
			workflow("first", "svc:first", false, 0),
			workflow("second", "svc:second", false, 0),
		)

		e.waitRegistered(ep.ID, namespaced(ns, "svc:first"), namespaced(ns, "svc:second"))

		out := taskOutput(t, e.runToCompletion(tn, namespaced(ns, "second"), map[string]any{"k": "v"}, 30*time.Second))
		assert.Equal(t, map[string]any{"k": "v"}, out["input"])

		assert.ElementsMatch(t, []string{namespaced(ns, "svc:first"), namespaced(ns, "svc:second")}, e.registeredActions(ep.ID))

		w, ok := e.activeWorkerWithAction(tn.id, namespaced(ns, "svc:second"))
		require.True(t, ok)
		assert.Contains(t, w.actions, namespaced(ns, "svc:first"))
	})
}

func TestEndpointDeleteRemovesActions(t *testing.T) {
	forEachLink(t, func(t *testing.T, lc linkCase) {
		e := lc.e
		tn := e.newTenant()
		e.pinLease(tn.id, 0, lc.owner)

		keep := newFakeEndpoint(t, "keep", workflow("keep", "svc:keep", false, 0))
		gone := newFakeEndpoint(t, "gone", workflow("gone", "svc:gone", false, 0))

		epKeep := e.createEndpoint(tn, keep, endpointOpts{})
		epGone := e.createEndpoint(tn, gone, endpointOpts{})

		dumpOnFailure(t, e, tn, keep, gone)

		keepAction := namespaced(epKeep.Namespace, "svc:keep")
		goneAction := namespaced(epGone.Namespace, "svc:gone")

		e.waitRegistered(epKeep.ID, keepAction)
		e.waitRegistered(epGone.ID, goneAction)

		w, ok := e.activeWorkerWithAction(tn.id, goneAction)
		require.True(t, ok)
		assert.Contains(t, w.actions, keepAction)

		_, err := e.repo.Endpoints().Delete(e.ctx, tn.id, epGone.ID)
		require.NoError(t, err)

		e.pollUntil(lc.fullReloadWait(), "deleted endpoint's action to leave every registration", func() (bool, error) {
			for _, w := range e.serverlessWorkers(tn.id) {
				for _, a := range w.actions {
					if a == goneAction {
						return false, fmt.Errorf("worker %s (active=%t) still advertises %s", w.id, w.active, a)
					}
				}
			}

			return true, nil
		})

		w, ok = e.activeWorkerWithAction(tn.id, keepAction)
		require.True(t, ok, "the remaining endpoint's action must stay registered")
		assert.Equal(t, lc.owner.String(), w.process)

		out := taskOutput(t, e.runToCompletion(tn, namespaced(epKeep.Namespace, "keep"), map[string]any{}, 30*time.Second))
		assert.Equal(t, "keep", out["endpoint"])
	})
}

func TestLeaseFailover(t *testing.T) {
	e := newEnv(t)

	victim := startProcess(t, e)
	survivor := startProcess(t, e)

	tn := e.newTenant()
	e.pinLease(tn.id, 0, victim.id)

	fake := newFakeEndpoint(t, "failover", workflow("echo", "svc:echo", false, 0))
	ep := e.createEndpoint(tn, fake, endpointOpts{})
	dumpOnFailure(t, e, tn, fake)
	ns := ep.Namespace
	action := namespaced(ns, "svc:echo")

	e.waitRegistered(ep.ID, action)
	e.runToCompletion(tn, namespaced(ns, "echo"), map[string]any{"before": true}, 30*time.Second)
	assertServedBy(t, e, tn.id, action, victim.id)

	victim.crashNow()

	// The victim's row expires after the TTL; the next tick of a live process claims its unit.
	e.pollUntil(3*leaseTTL+3*tick, "victim's unit to be claimed by a live process", func() (bool, error) {
		owner := e.leaseOwner(tn.id, 0)

		if owner == nil || *owner == victim.id {
			return false, fmt.Errorf("owner is %v", owner)
		}

		for _, live := range e.liveProcesses() {
			if live == *owner {
				return true, nil
			}
		}

		return false, fmt.Errorf("owner %s is not live", *owner)
	})

	owner := e.leaseOwner(tn.id, 0)
	require.NotNil(t, owner)
	assert.Contains(t, []uuid.UUID{survivor.id, e.inEngine}, *owner)

	out := taskOutput(t, e.runToCompletion(tn, namespaced(ns, "echo"), map[string]any{"after": true}, 30*time.Second))
	assert.Equal(t, map[string]any{"after": true}, out["input"])

	assertServedBy(t, e, tn.id, action, *owner)

	for _, w := range e.workersOfProcess(victim.id) {
		assert.False(t, w.active, "victim worker %s must be inactive", w.id)
	}
}

func TestShedOnScaleOut(t *testing.T) {
	e := newEnv(t)

	require.Equal(t, []uuid.UUID{e.inEngine}, e.liveProcesses(), "no out-of-process instance may be live before scale-out")

	// Two tenants with two endpoints each, all owned by the in-engine process.
	tenants := make([]*tenant, 0, 2)
	workflows := make([]string, 0, 2)

	for i := 0; i < 2; i++ {
		tn := e.newTenant()
		e.pinLease(tn.id, 0, e.inEngine)

		for j := 0; j < 2; j++ {
			fake := newFakeEndpoint(t, fmt.Sprintf("scale-%d-%d", i, j), workflow("echo", "svc:echo", false, 0))
			ep := e.createEndpoint(tn, fake, endpointOpts{})
			e.waitRegistered(ep.ID, namespaced(ep.Namespace, "svc:echo"))

			if j == 0 {
				workflows = append(workflows, namespaced(ep.Namespace, "echo"))
			}
		}

		tenants = append(tenants, tn)
	}

	require.Equal(t, 2, weightedUnits(e.unitsOwnedBy(e.inEngine)))

	joiner := startProcess(t, e)

	e.pollUntil(10*tick, "units to rebalance onto the new process", func() (bool, error) {
		mine := weightedUnits(e.unitsOwnedBy(e.inEngine))
		theirs := weightedUnits(e.unitsOwnedBy(joiner.id))

		if mine >= 1 && theirs >= 1 {
			return true, nil
		}

		return false, fmt.Errorf("in-engine owns %d weighted units, joiner %d", mine, theirs)
	})

	for i, tn := range tenants {
		out := taskOutput(t, e.runToCompletion(tn, workflows[i], map[string]any{"tenant": float64(i)}, 30*time.Second))
		assert.Equal(t, map[string]any{"tenant": float64(i)}, out["input"])
	}
}

func TestGracefulShutdownDeactivates(t *testing.T) {
	e := newEnv(t)

	p := startProcess(t, e)

	tn := e.newTenant()
	e.pinLease(tn.id, 0, p.id)

	fake := newFakeEndpoint(t, "graceful", workflow("echo", "svc:echo", false, 0))
	ep := e.createEndpoint(tn, fake, endpointOpts{})
	dumpOnFailure(t, e, tn, fake)
	ns := ep.Namespace
	action := namespaced(ns, "svc:echo")

	e.waitRegistered(ep.ID, action)
	e.runToCompletion(tn, namespaced(ns, "echo"), map[string]any{}, 30*time.Second)

	workers := e.workersOfProcess(p.id)
	require.NotEmpty(t, workers)

	for _, w := range workers {
		assert.True(t, w.active, "worker %s must be active before shutdown", w.id)
		assert.False(t, w.paused, "worker %s must not be paused before shutdown", w.id)
	}

	// A delivery is held at the endpoint while the process stops: the worker must be paused
	// (nothing new assigned to it) before the drain waits on that delivery, and still active
	// while it does, since deactivation is the close that follows the drain.
	release := fake.holdTriggers()

	ref, err := tn.sdk.RunNoWait(e.ctx, namespaced(ns, "echo"), map[string]any{"held": true})
	require.NoError(t, err)

	e.pollUntil(30*time.Second, "the held run to reach the endpoint", func() (bool, error) {
		return len(fake.requestsOfKind("trigger")) >= 2, nil
	})

	// The process serves other tenants too, whose registrations have nothing in flight and
	// finish their teardown at once; the ordering shows on this tenant's worker.
	tenantWorker := func() (workerRow, bool) {
		for _, w := range e.serverlessWorkers(tn.id) {
			if w.process == p.id.String() {
				return w, true
			}
		}

		return workerRow{}, false
	}

	stopped := make(chan error, 1)

	go func() { stopped <- p.stop() }()

	e.pollUntil(10*time.Second, "the worker to be paused while its delivery is held", func() (bool, error) {
		w, ok := tenantWorker()

		if !ok {
			return false, fmt.Errorf("no worker of process %s for the tenant", p.id)
		}

		if !w.active {
			return false, fmt.Errorf("worker %s was deactivated before its delivery drained", w.id)
		}

		if !w.paused {
			return false, fmt.Errorf("worker %s is not paused", w.id)
		}

		return true, nil
	})

	select {
	case err := <-stopped:
		t.Fatalf("stop returned while a delivery was held: %v", err)
	default:
	}

	release()

	select {
	case err := <-stopped:
		require.NoError(t, err)
	case <-time.After(drainTimeout + 30*time.Second):
		t.Fatal("stop did not return once the held delivery was released")
	}

	held := e.waitForCompletion(tn, ref.RunId, 30*time.Second)
	assert.Equal(t, map[string]any{"held": true}, taskOutput(t, held)["input"], "the held delivery completed on the paused worker")

	// The close that follows the drain deactivates the worker on the engine side; the flip
	// lands shortly after stop returns.
	e.pollUntil(10*time.Second, "workers of the stopped process to be inactive", func() (bool, error) {
		for _, w := range e.workersOfProcess(p.id) {
			if w.active {
				return false, fmt.Errorf("worker %s is still active", w.id)
			}
		}

		return true, nil
	})

	assert.False(t, e.processRowExists(p.id), "process row must be deleted")

	owner := e.leaseOwner(tn.id, 0)
	assert.True(t, owner == nil || *owner != p.id, "lease must be released, owner is %v", owner)

	// The released unit is picked up by the in-engine process and the workflow still runs.
	out := taskOutput(t, e.runToCompletion(tn, namespaced(ns, "echo"), map[string]any{"after": "shutdown"}, 30*time.Second))
	assert.Equal(t, map[string]any{"after": "shutdown"}, out["input"])
	assertServedBy(t, e, tn.id, action, e.inEngine)
}
