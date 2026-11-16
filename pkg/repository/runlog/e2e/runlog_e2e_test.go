//go:build !e2e && !load && !rampup && !integration

package e2e

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	admincontracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/runlog"
	"github.com/hatchet-dev/hatchet/pkg/testing/harness"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// The engine and the test share one in-memory run log store, so what the engine archives the test reads
// and the other way round.
var store = runlog.NewMemoryStore()

func TestMain(m *testing.M) {
	os.Setenv("SERVER_RUN_LOG_ENABLED", "true")
	os.Setenv("SERVER_RUN_LOG_PREFIX", "e2e")
	os.Setenv("SERVER_LIMITS_CORE_PARTITION_RETENTION", "24h")
	// with consolidation on, OLAP keeps no payload copy and every payload read goes to the core store
	os.Setenv("SERVER_PAYLOAD_STORE_ENABLE_PAYLOAD_CONSOLIDATION", "true")

	harness.RunTestWithEngine(m, loader.WithRunLogStore(store))
}

const (
	retention    = 24 * time.Hour
	pollInterval = 200 * time.Millisecond

	engineReadyTimeout = 2 * time.Minute
	runTimeout         = 90 * time.Second
	suiteTimeout       = 10 * time.Minute

	// how long a case keeps its work in flight; it outlasts a maintenance pass by far
	inFlightFor = 15 * time.Second

	// a reader that hangs on an archived run fails the test after this long instead of blocking it
	readerTimeout = 20 * time.Second

	eventKey     = "runlog-e2e:triggered"
	gateEventKey = "runlog-e2e:gate"
)

type numInput struct {
	N int `json:"n"`
}

type numOutput struct {
	V int `json:"v"`
}

type childIds struct {
	First  string `json:"first"`
	Second string `json:"second"`
}

// suite is one engine-wide scenario. Every run lands in the real today's partitions, so each pinned case is
// set up and checked on its own, one after the other, and a single final pass archives and drops the day
// with all of them.
type suite struct {
	ctx      context.Context
	repo     v1.Repository
	pool     *pgxpool.Pool
	tenantID uuid.UUID
	token    string
	client   *hatchet.Client
	admin    admincontracts.AdminServiceClient
	disp     dispatchercontracts.DispatcherClient
	started  time.Time

	single, dag, eventTask, retry, late, child *hatchet.StandaloneTask
	durable, parent                            *hatchet.StandaloneTask
	dagWorkflow, gatedWorkflow                 *hatchet.Workflow

	// release6 lets the parent of case 6 spawn its child a second time
	release6 chan struct{}

	// asleep2 is closed when the durable task of case 2 goes to sleep
	asleep2     chan struct{}
	asleep2Once sync.Once

	mu          sync.Mutex
	gateParentA map[string]int // what the gated DAG's last step read from its first step, by run
}

func TestRunLogEndToEnd(t *testing.T) {
	today := runlog.Day(time.Now())

	if passNow(today).Add(-retention).Month() != today.Month() {
		t.Skip("the forced pass would also make this month's v1_lookup_table partition a candidate and drop it, " +
			"which takes away the day of every archived run (ResolveDay); run outside the last two days of a month")
	}

	ctx, cancel := context.WithTimeout(context.Background(), suiteTimeout)
	defer cancel()

	s := newSuite(t, ctx)

	// runs that finish before any pass; they are read back after the final pass
	single := s.runAndWait(t, s.single, numInput{N: 7})
	dagRun := s.runWorkflowAndWait(t, s.dagWorkflow, numInput{N: 4})
	eventRun, eventID := s.pushEventAndWait(t, map[string]any{"from": "event", "n": 3})
	// case 7: no reader sees this run before it is archived; only OLAP is watched for its status
	lateRef, err := s.late.RunNoWait(s.ctx, numInput{N: 100})
	require.NoError(t, err)
	late := uuid.MustParse(lateRef.RunId)
	s.waitOLAPCompleted(t, late)

	before := map[uuid.UUID]*runView{}

	for _, id := range []uuid.UUID{single, dagRun, eventRun} {
		before[id] = s.view(t, id)
	}

	eventPayloadBefore := s.eventPayload(t, eventID)
	require.JSONEq(t, `{"from": "event", "n": 3}`, string(eventPayloadBefore))

	t.Run("case2_durable_sleep_pins_the_day", func(t *testing.T) {
		ref, err := s.durable.RunNoWait(s.ctx, numInput{N: 1})
		require.NoError(t, err)
		runID := uuid.MustParse(ref.RunId)

		select {
		case <-s.asleep2:
		case <-time.After(runTimeout):
			t.Fatal("the durable task never started")
		}

		// asleep: the engine holds a wait for it and it is no longer queued
		require.Eventually(t, func() bool {
			live := s.liveRows(t, runID)
			return slicesContains(live, "match") && !slicesContains(live, "queue_item")
		}, runTimeout, pollInterval, "the durable task never registered its sleep")

		s.requirePinnedPass(t, runID, single)

		s.waitCompleted(t, runID)
		before[runID] = s.view(t, runID)

		require.JSONEq(t, `{"v": 42}`, string(onlyTask(t, s.runDetails(t, runID)).Output), "the durable task must wake and complete")
	})

	t.Run("case3_scheduled_retry_pins_the_day", func(t *testing.T) {
		ref, err := s.retry.RunNoWait(s.ctx, numInput{N: 5})
		require.NoError(t, err)
		runID := uuid.MustParse(ref.RunId)

		require.Eventually(t, func() bool { return slicesContains(s.liveRows(t, runID), "retry_queue_item") },
			runTimeout, pollInterval, "the first attempt never failed into a scheduled retry")

		s.requirePinnedPass(t, runID, single)

		s.waitCompleted(t, runID)
		before[runID] = s.view(t, runID)

		details := s.runDetails(t, runID)
		require.Equal(t, admincontracts.RunStatus_COMPLETED, details.Status)
		require.JSONEq(t, `{"v": 6}`, string(onlyTask(t, details).Output), "the retry must succeed")
	})

	t.Run("case4_dag_with_a_step_not_started_pins_the_day", func(t *testing.T) {
		ref, err := s.gatedWorkflow.RunNoWait(s.ctx, numInput{N: 20})
		require.NoError(t, err)
		runID := uuid.MustParse(ref.RunId)

		// the first step finishes while the gate waits for its event, so the last step is not created
		require.Eventually(t, func() bool {
			d := s.runDetailsOrNil(t, runID)
			return d != nil && d.TaskRuns["first"] != nil && d.TaskRuns["first"].Status == admincontracts.RunStatus_COMPLETED
		}, runTimeout, pollInterval, "the first step never finished")

		d := s.runDetails(t, runID)
		require.Nil(t, d.TaskRuns["last"], "the last step must not exist yet")

		s.requireLive(t, runID)
		s.requirePinnedPass(t, runID, single)

		require.NoError(t, s.client.Events().Push(s.ctx, gateEventKey, map[string]any{}))

		s.waitCompleted(t, runID)
		before[runID] = s.view(t, runID)

		s.mu.Lock()
		got := s.gateParentA[runID.String()]
		s.mu.Unlock()

		assert.Equal(t, 21, got, "the last step must read its first parent's output after the drop attempt")
	})

	var parentRun uuid.UUID

	t.Run("case6_live_parent_pins_its_finished_child_and_keeps_the_child_key", func(t *testing.T) {
		ref, err := s.parent.RunNoWait(s.ctx, numInput{N: 9})
		require.NoError(t, err)
		parentRun = uuid.MustParse(ref.RunId)

		// the parent has spawned its child under a key and waited for it, then blocks until released
		require.Eventually(t, func() bool {
			return len(s.childRunsOf(t, parentRun)) == 1 && s.allFinished(t, s.childRunsOf(t, parentRun)...)
		}, runTimeout, pollInterval, "the child never finished")

		s.requirePinnedPass(t, parentRun, single)

		close(s.release6)

		s.waitCompleted(t, parentRun)
		before[parentRun] = s.view(t, parentRun)

		var ids childIds
		require.NoError(t, json.Unmarshal(onlyTask(t, s.runDetails(t, parentRun)).Output, &ids))

		assert.Equal(t, ids.First, ids.Second, "spawning the same child key again must return the existing child, not a new one")
		assert.Len(t, s.childRunsOf(t, parentRun), 1, "no second child may be created")
	})

	t.Run("final_pass_archives_and_drops_the_day", func(t *testing.T) {
		for _, id := range append(keys(before), late) {
			require.Empty(t, s.liveRows(t, id), "run %s still has in-flight rows", id)
		}

		s.pass(t)

		state := s.dayState(t, today)
		require.Equal(t, "DROPPED", state)
		require.False(t, s.taskPartitionExists(t, today), "today's v1_task partition must be gone")
		require.False(t, s.partitionExists(t, "v1_payload", today), "today's v1_payload partition must be gone")

		s.recreatePartitions(t, today)

		// every process rereads v1_run_log_day at most once a second
		time.Sleep(2 * time.Second)

		for _, id := range append(keys(before), late) {
			flat, err := s.repo.Tasks().FlattenExternalIds(s.ctx, s.tenantID, []uuid.UUID{id})
			require.NoError(t, err)
			require.Empty(t, flat, "core must have forgotten run %s", id)
		}

		require.NotEmpty(t, store.Keys("e2e/v1/"+s.tenantID.String()+"/"), "nothing was archived")
	})

	// case 1 and the run log readers on every finished run, pinned cases included
	t.Run("case1_readers_after_the_drop", func(t *testing.T) {
		for id, want := range before {
			t.Run(id.String(), func(t *testing.T) {
				got := s.view(t, id)

				assert.Equal(t, want.olap, got.olap, "OLAP payload reads (dashboard/API)")
				assert.Equal(t, want.details, got.details, "GetRunDetails")
				assert.Equal(t, want.runEvent, got.runEvent, "SubscribeToWorkflowRuns")
				assert.Equal(t, want.terminal, got.terminal, "SubscribeToWorkflowEvents")
			})
		}

		singleDetails := s.runDetails(t, single)
		require.Equal(t, admincontracts.RunStatus_COMPLETED, singleDetails.Status)
		require.True(t, singleDetails.Done)
		require.JSONEq(t, `{"n": 7}`, string(singleDetails.Input))
		require.JSONEq(t, `{"v": 14}`, string(onlyTask(t, singleDetails).Output))

		// the SDK's result fetch goes through the same GetRunDetails RPC
		sdkDetails, err := s.client.Runs().GetDetails(s.ctx, single)
		require.NoError(t, err)
		require.True(t, sdkDetails.Done)
		require.JSONEq(t, `{"n": 7}`, string(sdkDetails.Input))

		dag := s.runDetails(t, dagRun)
		require.Equal(t, admincontracts.RunStatus_COMPLETED, dag.Status)
		require.JSONEq(t, `{"n": 4}`, string(dag.Input))
		require.Len(t, dag.TaskRuns, 3)
		require.JSONEq(t, `{"v": 5}`, string(dag.TaskRuns["a"].Output))
		require.JSONEq(t, `{"v": 10}`, string(dag.TaskRuns["b"].Output))
		require.JSONEq(t, `{"v": 13}`, string(dag.TaskRuns["c"].Output))

		require.JSONEq(t, string(eventPayloadBefore), string(s.eventPayload(t, eventID)), "event data read through OLAP")
	})

	t.Run("case5_replay_of_an_archived_run_is_refused", func(t *testing.T) {
		for _, id := range []uuid.UUID{single, dagRun} {
			_, err := s.admin.ReplayTasks(s.auth(), &admincontracts.ReplayTasksRequest{ExternalIds: []string{id.String()}})
			requireArchivedReplayError(t, err, id)
		}

		_, err := s.admin.ReplayTasks(s.auth(), &admincontracts.ReplayTasksRequest{
			Filter: &admincontracts.TasksFilter{Since: timestamppb.New(s.started)},
		})
		requireArchivedReplayError(t, err, uuid.Nil)
	})

	t.Run("case6_child_key_after_the_drop", func(t *testing.T) {
		// the parent's day is gone with its SIGNAL_CREATED event; the only way to spawn under the key again
		// is a new parent run, which keys its children by its own id
		require.NotEqual(t, uuid.Nil, parentRun)
		require.Empty(t, s.childRunsOf(t, parentRun), "core must have forgotten the parent's children")
	})

	t.Run("case7_subscription_started_after_the_drop_returns_the_result", func(t *testing.T) {
		ev := s.subscribeRun(t, late)
		require.NotNil(t, ev, "SubscribeToWorkflowRuns never answered for a run archived before anyone subscribed")
		require.Equal(t, dispatchercontracts.WorkflowRunEventType_WORKFLOW_RUN_EVENT_TYPE_FINISHED, ev.EventType)
		require.Len(t, ev.Results, 1)
		require.JSONEq(t, `{"v": 200}`, ev.Results[0].GetOutput())
	})
}

func newSuite(t *testing.T, ctx context.Context) *suite {
	t.Helper()

	readyCtx, cancel := context.WithTimeout(ctx, engineReadyTimeout)
	defer cancel()

	require.NoError(t, harness.WaitEngineReady(readyCtx, engineReadyTimeout))

	dl, err := loader.NewConfigLoader("", loader.WithRunLogStore(store)).InitDataLayer()
	require.NoError(t, err)
	t.Cleanup(func() { _ = dl.Disconnect() })

	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	token := os.Getenv("HATCHET_CLIENT_TOKEN")

	client, err := hatchet.NewClient()
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close(context.Background()) })

	conn, err := grpc.NewClient(os.Getenv("SERVER_GRPC_BROADCAST_ADDRESS"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	s := &suite{
		ctx:         ctx,
		repo:        dl.V1,
		pool:        pool,
		tenantID:    tenantIDFromToken(t, token),
		token:       token,
		client:      client,
		admin:       admincontracts.NewAdminServiceClient(conn),
		disp:        dispatchercontracts.NewDispatcherClient(conn),
		started:     time.Now().Add(-time.Minute),
		release6:    make(chan struct{}),
		asleep2:     make(chan struct{}),
		gateParentA: map[string]int{},
	}

	s.registerWorkflows()

	worker := newWorker(t, client, s.single, s.eventTask, s.retry, s.late, s.child, s.durable, s.parent, s.dagWorkflow, s.gatedWorkflow)

	stop, err := worker.Start()
	require.NoError(t, err)
	t.Cleanup(func() { _ = stop() })

	return s
}

func (s *suite) registerWorkflows() {
	c := s.client

	s.single = c.NewStandaloneTask("runlog-e2e-single", func(ctx hatchet.Context, in numInput) (numOutput, error) {
		return numOutput{V: in.N * 2}, nil
	})

	s.late = c.NewStandaloneTask("runlog-e2e-late", func(ctx hatchet.Context, in numInput) (numOutput, error) {
		return numOutput{V: in.N * 2}, nil
	})

	s.eventTask = c.NewStandaloneTask("runlog-e2e-event", func(ctx hatchet.Context, in map[string]any) (map[string]any, error) {
		return map[string]any{"seen": in["from"]}, nil
	}, hatchet.WithWorkflowEvents(eventKey))

	s.retry = c.NewStandaloneTask("runlog-e2e-retry", func(ctx hatchet.Context, in numInput) (numOutput, error) {
		if ctx.RetryCount() == 0 {
			return numOutput{}, errors.New("first attempt fails on purpose")
		}

		return numOutput{V: in.N + 1}, nil
	}, hatchet.WithRetries(1), hatchet.WithRetryBackoff(float32(inFlightFor.Seconds()), int(inFlightFor.Seconds())))

	s.durable = c.NewStandaloneDurableTask("runlog-e2e-durable", func(ctx hatchet.DurableContext, in numInput) (numOutput, error) {
		s.asleep2Once.Do(func() { close(s.asleep2) })

		if _, err := ctx.SleepFor(inFlightFor); err != nil {
			return numOutput{}, err
		}

		return numOutput{V: in.N + 41}, nil
	})

	s.child = c.NewStandaloneTask("runlog-e2e-child", func(ctx hatchet.Context, in numInput) (numOutput, error) {
		return numOutput{V: in.N * 3}, nil
	})

	s.parent = c.NewStandaloneTask("runlog-e2e-parent", func(ctx hatchet.Context, in numInput) (childIds, error) {
		first, err := s.child.RunNoWait(ctx, in, hatchet.WithRunKey("the-child"))
		if err != nil {
			return childIds{}, err
		}

		if _, err := first.ResultWithContext(ctx); err != nil {
			return childIds{}, err
		}

		select {
		case <-s.release6:
		case <-ctx.Done():
			return childIds{}, ctx.Err()
		}

		second, err := s.child.RunNoWait(ctx, in, hatchet.WithRunKey("the-child"))
		if err != nil {
			return childIds{}, err
		}

		return childIds{First: first.RunId, Second: second.RunId}, nil
	}, hatchet.WithExecutionTimeout(5*time.Minute))

	// a 3-step chain; more than one task puts the workflow on the DAG operator
	s.dagWorkflow = c.NewWorkflow("runlog-e2e-dag")

	a := s.dagWorkflow.NewTask("a", func(ctx hatchet.Context, in numInput) (numOutput, error) {
		return numOutput{V: in.N + 1}, nil
	})

	b := s.dagWorkflow.NewTask("b", func(ctx hatchet.Context, in numInput) (numOutput, error) {
		var p numOutput
		if err := ctx.ParentOutput(a, &p); err != nil {
			return numOutput{}, err
		}

		return numOutput{V: p.V * 2}, nil
	}, hatchet.WithParents(a))

	s.dagWorkflow.NewTask("c", func(ctx hatchet.Context, in numInput) (numOutput, error) {
		var p numOutput
		if err := ctx.ParentOutput(b, &p); err != nil {
			return numOutput{}, err
		}

		return numOutput{V: p.V + 3}, nil
	}, hatchet.WithParents(b))

	// first finishes at once, gate waits for an event, last needs both
	s.gatedWorkflow = c.NewWorkflow("runlog-e2e-gated")

	first := s.gatedWorkflow.NewTask("first", func(ctx hatchet.Context, in numInput) (numOutput, error) {
		return numOutput{V: in.N + 1}, nil
	})

	gate := s.gatedWorkflow.NewTask("gate", func(ctx hatchet.Context, in numInput) (numOutput, error) {
		return numOutput{V: 0}, nil
	}, hatchet.WithWaitFor(hatchet.UserEventCondition(gateEventKey, "")))

	s.gatedWorkflow.NewTask("last", func(ctx hatchet.Context, in numInput) (numOutput, error) {
		var p numOutput
		if err := ctx.ParentOutput(first, &p); err != nil {
			return numOutput{}, err
		}

		s.mu.Lock()
		s.gateParentA[ctx.WorkflowRunId()] = p.V
		s.mu.Unlock()

		return numOutput{V: p.V}, nil
	}, hatchet.WithParents(first, gate))
}

// passNow is the clock of a forced maintenance pass: today and tomorrow are past retention, so both are
// candidates, and today's v1_payload can go because tomorrow can too.
func passNow(today time.Time) time.Time {
	return today.AddDate(0, 0, 3).Add(time.Hour)
}

// pass runs one maintenance pass with the clock moved forward. The pass drops the real tomorrow's
// partitions (an empty day), so callers recreate partitions afterwards for the engine to keep working.
func (s *suite) pass(t *testing.T) {
	t.Helper()

	today := runlog.Day(time.Now())

	require.Eventually(t, func() bool {
		err := s.repo.Tasks().UpdateTablePartitionsAt(s.ctx, passNow(today), retention)

		if errors.Is(err, v1.ErrPartitionLeaseHeld) {
			return false
		}

		require.NoError(t, err)

		return true
	}, time.Minute, pollInterval, "the partition maintenance lease stayed taken")
}

func (s *suite) recreatePartitions(t *testing.T, today time.Time) {
	t.Helper()

	require.Eventually(t, func() bool {
		require.NoError(t, s.repo.Tasks().CreateTablePartitions(s.ctx))

		return s.taskPartitionExists(t, today) && s.taskPartitionExists(t, today.AddDate(0, 0, 1))
	}, time.Minute, pollInterval, "today's partitions were not recreated")
}

// requirePinnedPass runs a pass while runID is in flight and checks that the day stayed whole: no archive
// record, today's partition is still there, and a finished run of the day is still in core.
func (s *suite) requirePinnedPass(t *testing.T, runID, finished uuid.UUID) {
	t.Helper()

	live := s.liveRows(t, runID)
	require.NotEmpty(t, live, "run %s must be in flight at the drop attempt", runID)
	t.Logf("run %s is in flight through %v at the drop attempt", runID, live)

	today := runlog.Day(time.Now())

	s.pass(t)

	assert.Equal(t, "", s.dayState(t, today), "a day with in-flight work must not be archived")
	require.True(t, s.taskPartitionExists(t, today))

	s.recreatePartitions(t, today)

	flat, err := s.repo.Tasks().FlattenExternalIds(s.ctx, s.tenantID, []uuid.UUID{finished})
	require.NoError(t, err)
	require.NotEmpty(t, flat, "a finished run of the pinned day must still be in core")
}

// liveRows names the in-flight rows that point at a run's tasks: hot queue and runtime tables and matches.
func (s *suite) liveRows(t *testing.T, runID uuid.UUID) []string {
	t.Helper()

	rows, err := s.pool.Query(s.ctx, `
		WITH t AS (
			SELECT id, inserted_at FROM v1_task WHERE tenant_id = $1 AND (workflow_run_id = $2 OR external_id = $2)
		)
		SELECT DISTINCT kind FROM (
			SELECT 'queue_item' AS kind FROM v1_queue_item x JOIN t ON x.task_id = t.id AND x.task_inserted_at = t.inserted_at
			UNION ALL SELECT 'retry_queue_item' FROM v1_retry_queue_item x JOIN t ON x.task_id = t.id AND x.task_inserted_at = t.inserted_at
			UNION ALL SELECT 'runtime' FROM v1_task_runtime x JOIN t ON x.task_id = t.id AND x.task_inserted_at = t.inserted_at
			UNION ALL SELECT 'concurrency_slot' FROM v1_concurrency_slot x JOIN t ON x.task_id = t.id AND x.task_inserted_at = t.inserted_at
			UNION ALL SELECT 'match' FROM v1_match m JOIN t ON
				(m.signal_task_id = t.id AND m.signal_task_inserted_at = t.inserted_at)
				OR (m.trigger_parent_task_id = t.id AND m.trigger_parent_task_inserted_at = t.inserted_at)
				OR (m.trigger_existing_task_id = t.id AND m.trigger_existing_task_inserted_at = t.inserted_at)
		) k
		ORDER BY 1
	`, s.tenantID, runID)
	require.NoError(t, err)
	defer rows.Close()

	var kinds []string

	for rows.Next() {
		var k string
		require.NoError(t, rows.Scan(&k))
		kinds = append(kinds, k)
	}

	require.NoError(t, rows.Err())

	return kinds
}

func (s *suite) requireLive(t *testing.T, runID uuid.UUID) {
	t.Helper()

	require.Eventually(t, func() bool { return len(s.liveRows(t, runID)) > 0 }, runTimeout, pollInterval)
}

func (s *suite) childRunsOf(t *testing.T, parentRun uuid.UUID) []uuid.UUID {
	t.Helper()

	rows, err := s.pool.Query(s.ctx, `
		SELECT c.external_id
		FROM v1_task c
		JOIN v1_task p ON p.id = c.parent_task_id AND p.inserted_at = c.parent_task_inserted_at
		WHERE p.tenant_id = $1 AND p.workflow_run_id = $2
	`, s.tenantID, parentRun)
	require.NoError(t, err)
	defer rows.Close()

	var ids []uuid.UUID

	for rows.Next() {
		var id uuid.UUID
		require.NoError(t, rows.Scan(&id))
		ids = append(ids, id)
	}

	require.NoError(t, rows.Err())

	return ids
}

func (s *suite) allFinished(t *testing.T, ids ...uuid.UUID) bool {
	t.Helper()

	for _, id := range ids {
		d := s.runDetailsOrNil(t, id)

		if d == nil || !d.Done {
			return false
		}
	}

	return true
}

func (s *suite) dayState(t *testing.T, day time.Time) string {
	t.Helper()

	var state string

	err := s.pool.QueryRow(s.ctx, `SELECT coalesce((SELECT state FROM v1_run_log_day WHERE day = $1), '')`,
		pgtype.Date{Time: day, Valid: true}).Scan(&state)
	require.NoError(t, err)

	return state
}

func (s *suite) taskPartitionExists(t *testing.T, day time.Time) bool {
	t.Helper()

	return s.partitionExists(t, "v1_task", day)
}

func (s *suite) partitionExists(t *testing.T, table string, day time.Time) bool {
	t.Helper()

	var exists bool

	err := s.pool.QueryRow(s.ctx, `SELECT to_regclass($1) IS NOT NULL`, fmt.Sprintf("%s_%s", table, day.Format("20060102"))).Scan(&exists)
	require.NoError(t, err)

	return exists
}

func (s *suite) runAndWait(t *testing.T, task *hatchet.StandaloneTask, input any) uuid.UUID {
	t.Helper()

	ref, err := task.RunNoWait(s.ctx, input)
	require.NoError(t, err)

	runID := uuid.MustParse(ref.RunId)
	s.waitCompleted(t, runID)

	return runID
}

func (s *suite) runWorkflowAndWait(t *testing.T, w *hatchet.Workflow, input any) uuid.UUID {
	t.Helper()

	ref, err := w.RunNoWait(s.ctx, input)
	require.NoError(t, err)

	runID := uuid.MustParse(ref.RunId)
	s.waitCompleted(t, runID)

	return runID
}

func (s *suite) pushEventAndWait(t *testing.T, payload map[string]any) (runID, eventID uuid.UUID) {
	t.Helper()

	require.NoError(t, s.client.Events().Push(s.ctx, eventKey, payload))

	require.Eventually(t, func() bool {
		err := s.pool.QueryRow(s.ctx, `
			SELECT e.external_id, r.external_id
			FROM v1_events_olap e
			JOIN v1_event_to_run_olap er ON er.event_id = e.id AND er.event_seen_at = e.seen_at
			JOIN v1_runs_olap r ON r.id = er.run_id AND r.inserted_at = er.run_inserted_at
			WHERE e.tenant_id = $1 AND e.key = $2
		`, s.tenantID, eventKey).Scan(&eventID, &runID)

		return err == nil
	}, runTimeout, pollInterval, "the event never triggered a run")

	s.waitCompleted(t, runID)

	return runID, eventID
}

// waitCompleted waits until the run is finished in core and its final status has reached OLAP, which
// ingests status from core rows and must do so before they are dropped.
func (s *suite) waitCompleted(t *testing.T, runID uuid.UUID) {
	t.Helper()

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		d := s.runDetailsOrNil(t, runID)
		require.NotNil(c, d)
		require.True(c, d.Done, "run %s is not done", runID)
		require.Equal(c, admincontracts.RunStatus_COMPLETED, d.Status)
	}, runTimeout, pollInterval)

	s.waitOLAPCompleted(t, runID)
}

func (s *suite) waitOLAPCompleted(t *testing.T, runID uuid.UUID) {
	t.Helper()

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		run, err := s.repo.OLAP().ReadWorkflowRun(s.ctx, runID)
		require.NoError(c, err)
		require.Equal(c, "COMPLETED", string(run.WorkflowRun.ReadableStatus))
	}, runTimeout, pollInterval)
}

func (s *suite) auth() context.Context {
	return metadata.AppendToOutgoingContext(s.ctx, "authorization", "Bearer "+s.token)
}

func (s *suite) runDetailsOrNil(t *testing.T, runID uuid.UUID) *admincontracts.GetRunDetailsResponse {
	t.Helper()

	d, err := s.admin.GetRunDetails(s.auth(), &admincontracts.GetRunDetailsRequest{ExternalId: runID.String()})

	if status.Code(err) == codes.NotFound {
		return nil
	}

	require.NoError(t, err)

	return d
}

func (s *suite) runDetails(t *testing.T, runID uuid.UUID) *admincontracts.GetRunDetailsResponse {
	t.Helper()

	d := s.runDetailsOrNil(t, runID)
	require.NotNil(t, d, "GetRunDetails: run %s not found", runID)

	return d
}

func onlyTask(t *testing.T, d *admincontracts.GetRunDetailsResponse) *admincontracts.TaskRunDetail {
	t.Helper()

	require.Len(t, d.TaskRuns, 1)

	for _, tr := range d.TaskRuns {
		return tr
	}

	return nil
}

// subscribeRun opens a new SubscribeToWorkflowRuns stream for one run and returns its first event, or nil
// when none arrives within readerTimeout.
func (s *suite) subscribeRun(t *testing.T, runID uuid.UUID) *dispatchercontracts.WorkflowRunEvent {
	t.Helper()

	ctx, cancel := context.WithTimeout(s.auth(), readerTimeout)
	defer cancel()

	stream, err := s.disp.SubscribeToWorkflowRuns(ctx)
	require.NoError(t, err)
	require.NoError(t, stream.Send(&dispatchercontracts.SubscribeToWorkflowRunsRequest{WorkflowRunId: runID.String()}))

	ev, err := stream.Recv()

	if status.Code(err) == codes.DeadlineExceeded {
		return nil
	}

	require.NoError(t, err)

	return ev
}

// subscribeEvents opens SubscribeToWorkflowEvents for a finished run and returns the terminal event it
// hangs up with.
func (s *suite) subscribeEvents(t *testing.T, runID uuid.UUID) *dispatchercontracts.WorkflowEvent {
	t.Helper()

	ctx, cancel := context.WithTimeout(s.auth(), readerTimeout)
	defer cancel()

	id := runID.String()

	stream, err := s.disp.SubscribeToWorkflowEvents(ctx, &dispatchercontracts.SubscribeToWorkflowEventsRequest{WorkflowRunId: &id})
	require.NoError(t, err)

	for {
		ev, err := stream.Recv()
		require.NoError(t, err, "SubscribeToWorkflowEvents for %s", runID)

		if ev.Hangup {
			return ev
		}
	}
}

func (s *suite) eventPayload(t *testing.T, eventID uuid.UUID) []byte {
	t.Helper()

	ev, err := s.repo.OLAP().GetEventWithPayload(s.ctx, eventID, s.tenantID)
	require.NoError(t, err)

	return ev.Payload
}

// runView is what each reader returns for a run, without the fields that are the time of the read.
type runView struct {
	olap     olapView
	details  string
	runEvent string
	terminal string
}

type olapView struct {
	RunInput  string
	RunOutput string
	// task external id -> input and output, as the run detail page reads them
	TaskInputs  map[string]string
	TaskOutputs map[string]string
	// task external id -> output payloads of its events, in order
	EventOutputs map[string][]string
}

func (s *suite) view(t *testing.T, runID uuid.UUID) *runView {
	t.Helper()

	run, err := s.repo.OLAP().ReadWorkflowRun(s.ctx, runID)
	require.NoError(t, err)

	v := &runView{olap: olapView{
		RunInput:     compactJSON(run.WorkflowRun.Input),
		RunOutput:    compactJSON(run.WorkflowRun.Output),
		TaskInputs:   map[string]string{},
		TaskOutputs:  map[string]string{},
		EventOutputs: map[string][]string{},
	}}

	tasks, err := s.repo.OLAP().ListTasksByIdAndInsertedAt(s.ctx, s.tenantID, run.TaskMetadata, true)
	require.NoError(t, err)

	for _, task := range tasks {
		v.olap.TaskInputs[task.ExternalID.String()] = compactJSON(task.InputPayload)
		v.olap.TaskOutputs[task.ExternalID.String()] = compactJSON(task.OutputPayload)
	}

	events, err := s.repo.OLAP().ListTaskRunEventsByWorkflowRunId(s.ctx, s.tenantID, runID, false)
	require.NoError(t, err)

	for _, e := range events {
		if len(e.OutputPayload) > 0 {
			k := e.TaskExternalID.String()
			v.olap.EventOutputs[k] = append(v.olap.EventOutputs[k], compactJSON(e.OutputPayload))
		}
	}

	details := s.runDetails(t, runID)
	details.Input = []byte(compactJSON(details.Input))
	details.AdditionalMetadata = []byte(compactJSON(details.AdditionalMetadata))

	for _, tr := range details.TaskRuns {
		tr.Output = []byte(compactJSON(tr.Output))
	}

	v.details = mustJSON(t, details)

	ev := s.subscribeRun(t, runID)
	require.NotNil(t, ev, "SubscribeToWorkflowRuns did not answer for %s", runID)
	ev.EventTimestamp = nil

	for _, r := range ev.Results {
		if r.Output != nil {
			out := compactJSON([]byte(*r.Output))
			r.Output = &out
		}
	}

	// the dispatcher builds an operator DAG's results from a map, so their order is random
	sort.Slice(ev.Results, func(i, j int) bool { return ev.Results[i].TaskName < ev.Results[j].TaskName })

	v.runEvent = mustJSON(t, ev)

	terminal := s.subscribeEvents(t, runID)
	terminal.EventTimestamp = nil
	v.terminal = mustJSON(t, terminal)

	return v
}

// compactJSON drops insignificant whitespace: core returns payloads as Postgres prints jsonb, the run log
// as encoding/json compacts them.
func compactJSON(raw []byte) string {
	var buf bytes.Buffer

	if json.Compact(&buf, raw) != nil {
		return string(raw)
	}

	return buf.String()
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()

	b, err := json.Marshal(v)
	require.NoError(t, err)

	return string(b)
}

func requireArchivedReplayError(t *testing.T, err error, id uuid.UUID) {
	t.Helper()

	require.Error(t, err, "replay of an archived run must fail")
	require.Equal(t, codes.FailedPrecondition, status.Code(err), err.Error())
	assert.Contains(t, err.Error(), "archived")

	if id != uuid.Nil {
		assert.Contains(t, err.Error(), id.String())
	}
}

func keys[K comparable, V any](m map[K]V) []K {
	res := make([]K, 0, len(m))

	for k := range m {
		res = append(res, k)
	}

	return res
}

func slicesContains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}

	return false
}

// The harness engine reports its version as "testing", which the SDK treats as a legacy engine. Once the
// SDK's deprecation window for legacy engines has passed, NewWorker returns a DeprecationError on a random
// 1 in 5 calls, before it does any work, so retrying is safe.
func newWorker(t *testing.T, client *hatchet.Client, workflows ...hatchet.WorkflowBase) *hatchet.Worker {
	t.Helper()

	var err error

	for range 20 {
		var worker *hatchet.Worker

		worker, err = client.NewWorker("runlog-e2e-worker", hatchet.WithWorkflows(workflows...), hatchet.WithSlots(20))

		var deprecation *hatchet.DeprecationError
		if !errors.As(err, &deprecation) {
			require.NoError(t, err)

			return worker
		}
	}

	require.NoError(t, err, "NewWorker kept returning the legacy-engine deprecation error")

	return nil
}

// tenantIDFromToken reads the subject claim, which is the tenant id for a tenant API token.
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

	return uuid.MustParse(claims.Sub)
}
