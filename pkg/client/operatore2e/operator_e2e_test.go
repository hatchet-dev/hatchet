//go:build e2e

// Package operatore2e exercises the gRPC operator client against an in-process
// engine started by the test harness with SERVER_GRPC_OPERATORS_ENABLED=true.
package operatore2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/operatorclient"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/operator/hostgrpc"
	"github.com/hatchet-dev/hatchet/pkg/operator/operatortest"
	"github.com/hatchet-dev/hatchet/pkg/testing/harness"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

const (
	testTimeout  = 2 * time.Minute
	pollInterval = 200 * time.Millisecond

	// schedulerConvergence is longer than the scheduler's forced slot replenish
	// interval (1 to 1.5 seconds), which is when a removed action stops being
	// offered to the worker.
	schedulerConvergence = 3 * time.Second
)

func TestMain(m *testing.M) {
	os.Setenv("SERVER_GRPC_OPERATORS_ENABLED", "true")
	harness.RunTestWithEngine(m)
}

func newTestContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	t.Cleanup(cancel)
	return ctx
}

// clients builds the operator-side v0 client and the SDK client used to
// trigger runs and inspect them. Both read HATCHET_CLIENT_TOKEN exported by
// the harness.
func clients(t *testing.T) (client.Client, *hatchet.Client) {
	t.Helper()

	v0, err := client.New()
	require.NoError(t, err)

	sdk, err := hatchet.NewClient()
	require.NoError(t, err)

	return v0, sdk
}

func uniqueName(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, uuid.New().String()[:8])
}

// connect registers an operator worker with the given slot config.
func connect(t *testing.T, ctx context.Context, v0 client.Client, name string, slots map[string]int32) operatorclient.Session {
	t.Helper()

	session, err := v0.Operator().Connect(ctx, &operatorclient.ConnectRequest{
		Name:       uniqueName(name),
		SlotConfig: slots,
	})
	require.NoError(t, err)

	return session
}

// putAndAdd puts the workflow, adds its actions to the session and waits for
// the delta to land on the engine.
func putAndAdd(t *testing.T, ctx context.Context, session operatorclient.Session, wf *v1.CreateWorkflowVersionRequest) *v1.CreateWorkflowVersionResponse {
	t.Helper()

	resp, actions, err := session.PutWorkflow(ctx, wf)
	require.NoError(t, err)
	require.NotEmpty(t, actions)

	session.AddActions(actions...)
	flushAndPoll(t, ctx, session, func(linked []string) bool {
		for _, action := range actions {
			if !slices.Contains(linked, action) {
				return false
			}
		}
		return true
	})

	return resp
}

// flushAndPoll flushes the session's deltas, then polls the worker's linked
// actions until done accepts them: deltas are applied by the engine after the
// stream has accepted them, so Flush alone is not a database barrier.
func flushAndPoll(t *testing.T, ctx context.Context, session operatorclient.Session, done func(linked []string) bool) {
	t.Helper()
	require.NoError(t, session.Flush(ctx))

	workerId := session.Registration().WorkerId
	pollUntil(t, ctx, func() (bool, error) {
		return done(workerActions(t, ctx, workerId)), nil
	})
}

// flushAndConverge is flushAndPoll followed by the scheduler's forced
// replenish window, so its in-memory view of the worker's actions matches.
func flushAndConverge(t *testing.T, ctx context.Context, session operatorclient.Session, done func(linked []string) bool) {
	t.Helper()
	flushAndPoll(t, ctx, session, done)
	time.Sleep(schedulerConvergence)
}

// simpleWorkflow is a one-task workflow whose task runs the given action.
func simpleWorkflow(name, action string, durable bool) *v1.CreateWorkflowVersionRequest {
	return &v1.CreateWorkflowVersionRequest{
		Name: name,
		Tasks: []*v1.CreateTaskOpts{{
			ReadableId: "task",
			Action:     action,
			Timeout:    "60s",
			IsDurable:  durable,
		}},
	}
}

func stepEvent(action *dispatchercontracts.AssignedAction, eventType dispatchercontracts.StepActionEventType, payload string) *dispatchercontracts.StepActionEvent {
	retryCount := action.RetryCount
	return &dispatchercontracts.StepActionEvent{
		JobId:             action.JobId,
		JobRunId:          action.JobRunId,
		TaskId:            action.TaskId,
		TaskRunExternalId: action.TaskRunExternalId,
		ActionId:          action.ActionId,
		EventTimestamp:    timestamppb.Now(),
		EventType:         eventType,
		EventPayload:      payload,
		RetryCount:        &retryCount,
	}
}

// actionInput extracts the task input from a START_STEP_RUN payload.
func actionInput(t *testing.T, action *dispatchercontracts.AssignedAction) json.RawMessage {
	t.Helper()
	var payload struct {
		Input json.RawMessage `json:"input"`
	}
	require.NoError(t, json.Unmarshal([]byte(action.ActionPayload), &payload))
	return payload.Input
}

type taskHandler func(ctx context.Context, action *dispatchercontracts.AssignedAction) (output string, err error)

// serve runs the session's action loop in the background and answers every
// START_STEP_RUN with STARTED followed by COMPLETED or FAILED from handle.
// Non-start actions are ignored. The loop ends when the session closes.
func serve(t *testing.T, ctx context.Context, session operatorclient.Session, handle taskHandler) {
	t.Helper()

	actions, errCh, err := session.Actions(ctx)
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for action := range actions {
			if action.ActionType != dispatchercontracts.ActionType_START_STEP_RUN {
				continue
			}
			wg.Add(1)
			go func(action *dispatchercontracts.AssignedAction) {
				defer wg.Done()
				if _, err := session.SendStepActionEvent(ctx, stepEvent(action, dispatchercontracts.StepActionEventType_STEP_EVENT_TYPE_STARTED, "")); err != nil {
					t.Logf("could not send started event: %v", err)
					return
				}
				output, err := handle(ctx, action)
				eventType := dispatchercontracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED
				if err != nil {
					eventType = dispatchercontracts.StepActionEventType_STEP_EVENT_TYPE_FAILED
					output = err.Error()
				}
				if _, err := session.SendStepActionEvent(ctx, stepEvent(action, eventType, output)); err != nil {
					t.Logf("could not send %s event: %v", eventType, err)
				}
			}(action)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for err := range errCh {
			t.Logf("operator listener error: %v", err)
		}
	}()

	t.Cleanup(func() {
		_ = session.Close()
		wg.Wait()
	})
}

func pollUntil(t *testing.T, ctx context.Context, fn func() (bool, error)) {
	t.Helper()
	for {
		done, err := fn()
		if err != nil {
			t.Logf("poll error: %v", err)
		}
		if done {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("context cancelled while polling: %v", ctx.Err())
		case <-time.After(pollInterval):
		}
	}
}

// runToCompletion triggers the workflow and waits until its run finishes,
// failing the test if it does not complete successfully.
func runToCompletion(t *testing.T, ctx context.Context, sdk *hatchet.Client, workflowName string, input map[string]any) *client.RunDetails {
	t.Helper()

	ref, err := sdk.RunNoWait(ctx, workflowName, input)
	require.NoError(t, err)

	return waitForCompletion(t, ctx, sdk, ref.RunId)
}

// waitForCompletion waits until the run finishes, failing the test if it does
// not complete successfully.
func waitForCompletion(t *testing.T, ctx context.Context, sdk *hatchet.Client, runId string) *client.RunDetails {
	t.Helper()

	var details *client.RunDetails
	pollUntil(t, ctx, func() (bool, error) {
		d, err := sdk.Runs().GetDetails(ctx, uuid.MustParse(runId))
		if err != nil {
			return false, err
		}
		switch d.Status {
		case rest.V1TaskStatusCOMPLETED:
			details = d
			return true, nil
		case rest.V1TaskStatusFAILED, rest.V1TaskStatusCANCELLED:
			t.Fatalf("run %s ended with status %s: %+v", runId, d.Status, d.TaskRuns)
		}
		return false, nil
	})

	return details
}

// assertStaysQueued polls the run for the given window and fails the test as
// soon as it leaves QUEUED. Lookups that fail while the run is still being
// written are retried rather than treated as a status change.
func assertStaysQueued(t *testing.T, ctx context.Context, sdk *hatchet.Client, runId string, window time.Duration) {
	t.Helper()

	deadline := time.Now().Add(window)
	for time.Now().Before(deadline) {
		d, err := sdk.Runs().GetDetails(ctx, uuid.MustParse(runId))
		if err != nil {
			t.Logf("poll error: %v", err)
		} else {
			require.Equal(t, rest.V1TaskStatusQUEUED, d.Status, "run %s must stay queued while no worker has its action: %+v", runId, d.TaskRuns)
		}
		select {
		case <-ctx.Done():
			t.Fatalf("context cancelled while polling: %v", ctx.Err())
		case <-time.After(pollInterval):
		}
	}
}

// workerActive reads the worker's active flag straight from the database: the
// harness runs the engine only, so there is no REST API to ask.
func workerActive(t *testing.T, ctx context.Context, workerId string) (active bool, state string) {
	t.Helper()

	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	require.NoError(t, err)
	defer conn.Close(ctx)

	var listenerEstablished, lastHeartbeat *time.Time
	err = conn.QueryRow(ctx,
		`SELECT "isActive", "lastListenerEstablished", "lastHeartbeatAt" FROM "Worker" WHERE "id" = $1`,
		uuid.MustParse(workerId),
	).Scan(&active, &listenerEstablished, &lastHeartbeat)
	require.NoError(t, err)

	return active, fmt.Sprintf("isActive=%t lastListenerEstablished=%v lastHeartbeatAt=%v", active, listenerEstablished, lastHeartbeat)
}

// workerPaused reads the worker's paused flag straight from the database.
func workerPaused(t *testing.T, ctx context.Context, workerId string) bool {
	t.Helper()

	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	require.NoError(t, err)
	defer conn.Close(ctx)

	var paused bool
	require.NoError(t, conn.QueryRow(ctx, `SELECT "isPaused" FROM "Worker" WHERE "id" = $1`, uuid.MustParse(workerId)).Scan(&paused))

	return paused
}

func pollWorkerPaused(t *testing.T, ctx context.Context, workerId string, want bool) {
	t.Helper()
	pollUntil(t, ctx, func() (bool, error) {
		return workerPaused(t, ctx, workerId) == want, nil
	})
}

// workerActionHash reads the worker's action hash straight from the database.
func workerActionHash(t *testing.T, ctx context.Context, workerId string) []byte {
	t.Helper()

	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	require.NoError(t, err)
	defer conn.Close(ctx)

	var hash []byte
	require.NoError(t, conn.QueryRow(ctx, `SELECT "actionHash" FROM "Worker" WHERE "id" = $1`, uuid.MustParse(workerId)).Scan(&hash))

	return hash
}

// workerActions reads the worker's linked actions straight from the database,
// sorted by action id.
func workerActions(t *testing.T, ctx context.Context, workerId string) []string {
	t.Helper()

	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	require.NoError(t, err)
	defer conn.Close(ctx)

	rows, err := conn.Query(ctx,
		`SELECT a."actionId"
		FROM "_ActionToWorker" atw
		JOIN "Action" a ON a."id" = atw."A"
		WHERE atw."B" = $1
		ORDER BY a."actionId"`,
		uuid.MustParse(workerId),
	)
	require.NoError(t, err)
	defer rows.Close()

	actions := []string{}
	for rows.Next() {
		var action string
		require.NoError(t, rows.Scan(&action))
		actions = append(actions, action)
	}
	require.NoError(t, rows.Err())

	return actions
}

func pollWorkerActive(t *testing.T, ctx context.Context, workerId string, want bool) {
	t.Helper()
	polls := 0
	pollUntil(t, ctx, func() (bool, error) {
		active, state := workerActive(t, ctx, workerId)
		polls++
		if active != want && polls%25 == 1 {
			t.Logf("worker %s still %s", workerId, state)
		}
		return active == want, nil
	})
}

func taskOutput(t *testing.T, details *client.RunDetails) map[string]any {
	t.Helper()
	task, ok := details.TaskRuns["task"]
	require.True(t, ok, "task run missing from %+v", details.TaskRuns)
	var out map[string]any
	require.NoError(t, json.Unmarshal(task.Output, &out))
	return out
}

func TestEcho(t *testing.T) {
	ctx := newTestContext(t)
	v0, sdk := clients(t)

	session := connect(t, ctx, v0, "echo-operator", map[string]int32{"default": 10})

	reg := session.Registration()
	assert.NotEmpty(t, reg.TenantId)
	assert.NotEmpty(t, reg.OperatorId)
	assert.NotEmpty(t, reg.WorkerId)
	assert.False(t, reg.Resumed)
	assert.Empty(t, workerActions(t, ctx, reg.WorkerId), "registration links no actions")

	serve(t, ctx, session, func(ctx context.Context, action *dispatchercontracts.AssignedAction) (string, error) {
		return string(actionInput(t, action)), nil
	})

	name := uniqueName("grpc-op-echo")
	resp := putAndAdd(t, ctx, session, simpleWorkflow(name, "grpcop:echo", false))
	assert.NotEmpty(t, resp.GetId())
	assert.NotEmpty(t, resp.GetWorkflowId())
	assert.Equal(t, []string{"grpcop:echo"}, workerActions(t, ctx, reg.WorkerId))

	input := map[string]any{"message": "hello", "n": float64(3)}
	details := runToCompletion(t, ctx, sdk, name, input)
	assert.Equal(t, input, taskOutput(t, details))
}

func TestDurableMemo(t *testing.T) {
	ctx := newTestContext(t)
	v0, sdk := clients(t)

	session := connect(t, ctx, v0, "durable-operator", map[string]int32{"default": 10, "durable": 10})

	durable := client.NewDurableTaskListener(session.Registration().WorkerId, session.OpenDurableTaskStream, v0.Logger())
	durable.Start(ctx)
	t.Cleanup(durable.Stop)

	memoKey := []byte("memo-key")
	memoPayload := []byte(`{"memo":"value"}`)

	serve(t, ctx, session, func(ctx context.Context, action *dispatchercontracts.AssignedAction) (string, error) {
		invocation := action.GetDurableTaskInvocationCount()

		ack, err := durable.SendMemoRequest(ctx, action.TaskRunExternalId, invocation, memoKey)
		if err != nil {
			return "", fmt.Errorf("memo request: %w", err)
		}
		if ack.MemoAlreadyExisted {
			return "", fmt.Errorf("memo unexpectedly existed on first invocation")
		}
		if ack.Ref == nil {
			return "", fmt.Errorf("memo ack carried no event log ref")
		}
		if err := durable.SendMemoCompleted(ctx, ack.Ref, memoKey, memoPayload); err != nil {
			return "", fmt.Errorf("memo completion: %w", err)
		}

		out, _ := json.Marshal(map[string]any{
			"invocation": invocation,
			"nodeId":     ack.Ref.GetNodeId(),
		})
		return string(out), nil
	})

	name := uniqueName("grpc-op-durable")
	putAndAdd(t, ctx, session, simpleWorkflow(name, "grpcop:durable", true))

	details := runToCompletion(t, ctx, sdk, name, map[string]any{})
	out := taskOutput(t, details)
	assert.Contains(t, out, "invocation")
	assert.Contains(t, out, "nodeId")
	assert.GreaterOrEqual(t, durable.StreamSeq(), 1, "durable task stream was never opened")
}

func TestStreamCloseDeactivates(t *testing.T) {
	ctx := newTestContext(t)
	v0, _ := clients(t)

	session := connect(t, ctx, v0, "close-operator", map[string]int32{"default": 10})
	workerId := session.Registration().WorkerId

	_, _, err := session.Actions(ctx)
	require.NoError(t, err)

	pollWorkerActive(t, ctx, workerId, true)

	require.NoError(t, session.Close())

	pollWorkerActive(t, ctx, workerId, false)
}

func TestActionsDelta(t *testing.T) {
	ctx := newTestContext(t)
	v0, sdk := clients(t)

	session := connect(t, ctx, v0, "delta-operator", map[string]int32{"default": 10})
	workerId := session.Registration().WorkerId

	serve(t, ctx, session, func(ctx context.Context, action *dispatchercontracts.AssignedAction) (string, error) {
		return `{"delta":"ok"}`, nil
	})

	const total = 2500
	prefix := uniqueName("bulk")
	ids := make([]string, 0, total)
	for i := 0; i < total; i++ {
		ids = append(ids, fmt.Sprintf("%s:action%04d", prefix, i))
	}

	// one call larger than a single delta is chunked by the client
	session.AddActions(ids...)
	flushAndPoll(t, ctx, session, func(linked []string) bool { return len(linked) == total })

	linked := workerActions(t, ctx, workerId)
	require.Len(t, linked, total)
	assert.Equal(t, strings.ToLower(ids[0]), linked[0])

	// a second worker built from the same set in the opposite order hashes equal
	other := connect(t, ctx, v0, "delta-operator-other", map[string]int32{"default": 10})
	t.Cleanup(func() { _ = other.Close() })
	reversed := make([]string, total)
	for i, id := range ids {
		reversed[total-1-i] = id
	}
	other.AddActions(reversed...)
	flushAndPoll(t, ctx, other, func(linked []string) bool { return len(linked) == total })
	assert.Equal(t, workerActionHash(t, ctx, workerId), workerActionHash(t, ctx, other.Registration().WorkerId), "equal sets must hash equal whatever the order")

	// remove 1,000 and verify the links
	session.RemoveActions(ids[:1000]...)
	flushAndPoll(t, ctx, session, func(linked []string) bool { return len(linked) == total-1000 })
	assert.NotEqual(t, workerActionHash(t, ctx, workerId), workerActionHash(t, ctx, other.Registration().WorkerId))

	// a run for a removed action stays queued until the action is re-added
	name := uniqueName("grpc-op-delta")
	_, actions, err := session.PutWorkflow(ctx, simpleWorkflow(name, "grpcop:delta", false))
	require.NoError(t, err)
	hasDelta := func(linked []string) bool { return slices.Contains(linked, "grpcop:delta") }
	session.AddActions(actions...)
	flushAndConverge(t, ctx, session, hasDelta)

	session.RemoveActions(actions...)
	// The scheduler keeps its in-memory slots for the dropped action until its
	// next forced replenish, so a run triggered inside that window could still
	// be assigned from the stale view.
	flushAndConverge(t, ctx, session, func(linked []string) bool { return !hasDelta(linked) })

	ref, err := sdk.RunNoWait(ctx, name, map[string]any{})
	require.NoError(t, err)
	assertStaysQueued(t, ctx, sdk, ref.RunId, 10*time.Second)

	session.AddActions(actions...)
	flushAndPoll(t, ctx, session, hasDelta)

	details := waitForCompletion(t, ctx, sdk, ref.RunId)
	assert.Equal(t, map[string]any{"delta": "ok"}, taskOutput(t, details))
}

func TestReconnectResumesWorker(t *testing.T) {
	ctx := newTestContext(t)
	v0, sdk := clients(t)

	session := connect(t, ctx, v0, "reconnect-operator", map[string]int32{"default": 10})
	workerId := session.Registration().WorkerId

	serve(t, ctx, session, func(ctx context.Context, action *dispatchercontracts.AssignedAction) (string, error) {
		return `{"worker":"` + session.Registration().WorkerId + `"}`, nil
	})

	name := uniqueName("grpc-op-reconnect")
	putAndAdd(t, ctx, session, simpleWorkflow(name, "grpcop:reconnect", false))
	hash := workerActionHash(t, ctx, workerId)

	runToCompletion(t, ctx, sdk, name, map[string]any{})

	closer, ok := session.(interface{ CloseListenStream() error })
	require.True(t, ok, "session does not expose CloseListenStream")
	require.NoError(t, closer.CloseListenStream())

	// The worker is deactivated when the stream ends, so it is only active
	// once the automatic reconnect has resumed it.
	pollWorkerActive(t, ctx, workerId, true)

	reg := session.Registration()
	assert.Equal(t, workerId, reg.WorkerId, "reconnect must resume the same worker")
	assert.True(t, reg.Resumed)
	assert.Equal(t, hash, workerActionHash(t, ctx, workerId), "a resumed worker keeps its set: nothing is replayed")
	assert.Equal(t, []string{"grpcop:reconnect"}, workerActions(t, ctx, workerId))

	details := runToCompletion(t, ctx, sdk, name, map[string]any{})
	assert.Equal(t, map[string]any{"worker": workerId}, taskOutput(t, details))
}

func TestReconnectWithoutResumeReplaysActions(t *testing.T) {
	ctx := newTestContext(t)
	v0, sdk := clients(t)

	session := connect(t, ctx, v0, "replay-operator", map[string]int32{"default": 10})
	firstWorkerId := session.Registration().WorkerId

	serve(t, ctx, session, func(ctx context.Context, action *dispatchercontracts.AssignedAction) (string, error) {
		return `{"worker":"` + session.Registration().WorkerId + `"}`, nil
	})

	name := uniqueName("grpc-op-replay")
	putAndAdd(t, ctx, session, simpleWorkflow(name, "grpcop:replay", false))
	session.AddActions("grpcop:replay-extra")
	flushAndPoll(t, ctx, session, func(linked []string) bool { return len(linked) == 2 })

	runToCompletion(t, ctx, sdk, name, map[string]any{})

	// forget the worker so Register creates a new one, then drop the stream
	forgetter, ok := session.(interface{ ForgetWorker() })
	require.True(t, ok, "session does not expose ForgetWorker")
	forgetter.ForgetWorker()

	closer, ok := session.(interface{ CloseListenStream() error })
	require.True(t, ok, "session does not expose CloseListenStream")
	require.NoError(t, closer.CloseListenStream())

	pollUntil(t, ctx, func() (bool, error) {
		reg := session.Registration()
		return reg.WorkerId != "" && reg.WorkerId != firstWorkerId, nil
	})

	reg := session.Registration()
	assert.False(t, reg.Resumed)
	pollWorkerActive(t, ctx, reg.WorkerId, true)
	pollWorkerActive(t, ctx, firstWorkerId, false)

	// the new worker carries the whole desired set
	pollUntil(t, ctx, func() (bool, error) {
		return len(workerActions(t, ctx, reg.WorkerId)) == 2, nil
	})
	assert.Equal(t, []string{"grpcop:replay", "grpcop:replay-extra"}, workerActions(t, ctx, reg.WorkerId))

	details := runToCompletion(t, ctx, sdk, name, map[string]any{})
	assert.Equal(t, map[string]any{"worker": reg.WorkerId}, taskOutput(t, details))
}

func TestOperatorIdRequired(t *testing.T) {
	ctx := newTestContext(t)
	v0, _ := clients(t)

	session := connect(t, ctx, v0, "authz-operator", map[string]int32{"default": 10})
	t.Cleanup(func() { _ = session.Close() })

	conn, err := grpc.NewClient(os.Getenv("SERVER_GRPC_BROADCAST_ADDRESS"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	raw := v1.NewOperatorServiceClient(conn)
	authed := metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+os.Getenv("HATCHET_CLIENT_TOKEN"))
	event := &dispatchercontracts.StepActionEvent{
		WorkerId:       session.Registration().WorkerId,
		ActionId:       "grpcop:noop",
		EventTimestamp: timestamppb.Now(),
		EventType:      dispatchercontracts.StepActionEventType_STEP_EVENT_TYPE_STARTED,
	}

	_, err = raw.SendStepActionEvent(authed, event)
	assert.Equal(t, codes.InvalidArgument, status.Code(err), "missing operator id: %v", err)

	listen, err := raw.Listen(authed)
	require.NoError(t, err)
	require.NoError(t, listen.Send(&v1.OperatorListenRequest{Message: &v1.OperatorListenRequest_Start{
		Start: &v1.OperatorListenStart{WorkerId: session.Registration().WorkerId},
	}}))
	_, err = listen.Recv()
	assert.Equal(t, codes.InvalidArgument, status.Code(err), "missing operator id on Listen: %v", err)

	unknown := metadata.AppendToOutgoingContext(authed, "hatchet-operator-id", uuid.New().String())
	_, err = raw.SendStepActionEvent(unknown, event)
	assert.Equal(t, codes.PermissionDenied, status.Code(err), "unknown operator id: %v", err)
}

func TestDurableTaskRejectsForeignWorker(t *testing.T) {
	ctx := newTestContext(t)
	v0, _ := clients(t)

	mine := connect(t, ctx, v0, "durable-authz-operator", map[string]int32{"default": 10, "durable": 10})
	t.Cleanup(func() { _ = mine.Close() })
	theirs := connect(t, ctx, v0, "durable-authz-other", map[string]int32{"default": 10, "durable": 10})
	t.Cleanup(func() { _ = theirs.Close() })

	conn, err := grpc.NewClient(os.Getenv("SERVER_GRPC_BROADCAST_ADDRESS"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	raw := v1.NewOperatorServiceClient(conn)
	authed := metadata.AppendToOutgoingContext(ctx,
		"authorization", "Bearer "+os.Getenv("HATCHET_CLIENT_TOKEN"),
		"hatchet-operator-id", mine.Registration().OperatorId,
	)

	stream, err := raw.DurableTask(authed)
	require.NoError(t, err)
	require.NoError(t, stream.Send(&v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_RegisterWorker{
		RegisterWorker: &v1.DurableTaskRequestRegisterWorker{WorkerId: theirs.Registration().WorkerId},
	}}))
	_, err = stream.Recv()
	assert.Equal(t, codes.PermissionDenied, status.Code(err), "another operator's worker: %v", err)

	own, err := raw.DurableTask(authed)
	require.NoError(t, err)
	require.NoError(t, own.Send(&v1.DurableTaskRequest{Message: &v1.DurableTaskRequest_RegisterWorker{
		RegisterWorker: &v1.DurableTaskRequestRegisterWorker{WorkerId: mine.Registration().WorkerId},
	}}))
	resp, err := own.Recv()
	require.NoError(t, err)
	assert.NotNil(t, resp.GetRegisterWorker(), "own worker registers on the durable stream")
}

// A paused worker keeps its actions and its session but gets nothing delivered: the pause is a
// message on the Listen stream, and once its ack is back the engine stops assigning to the
// worker and returns to the queue anything it had assigned in the meantime. That is what lets
// an operator drain before it hangs up. Resuming delivers the queued run once.
func TestPauseStopsAssignment(t *testing.T) {
	ctx := newTestContext(t)
	v0, sdk := clients(t)

	session := connect(t, ctx, v0, "pause-operator", map[string]int32{"default": 10})
	workerId := session.Registration().WorkerId

	var handled atomic.Int32

	serve(t, ctx, session, func(ctx context.Context, action *dispatchercontracts.AssignedAction) (string, error) {
		handled.Add(1)
		return `{"paused":"ok"}`, nil
	})

	name := uniqueName("grpc-op-pause")
	putAndAdd(t, ctx, session, simpleWorkflow(name, "grpcop:pause", false))
	time.Sleep(schedulerConvergence)

	// the ack has been received when Pause returns; the row is written before the ack
	require.NoError(t, session.Pause(ctx))
	assert.True(t, workerPaused(t, ctx, workerId), "the pause is committed before it is acknowledged")

	// a run triggered right after the ack is either never assigned or assigned and returned to
	// the queue; either way it does not reach the operator
	ref, err := sdk.RunNoWait(ctx, name, map[string]any{})
	require.NoError(t, err)

	assertStaysQueued(t, ctx, sdk, ref.RunId, schedulerConvergence)
	assert.Zero(t, handled.Load(), "nothing is delivered after the pause ack")

	active, state := workerActive(t, ctx, workerId)
	assert.True(t, active, "a paused worker keeps its session: %s", state)

	require.NoError(t, session.Resume(ctx))
	assert.False(t, workerPaused(t, ctx, workerId), "the resume is committed before it is acknowledged")

	waitForCompletion(t, ctx, sdk, ref.RunId)
	assert.EqualValues(t, 1, handled.Load(), "the run held back by the pause runs once after the resume")
}

// Close is pause then drain: it pauses the worker before it hangs up, so nothing new is
// assigned while the operator finishes what it holds.
func TestCloseDrainsBeforeDeactivating(t *testing.T) {
	ctx := newTestContext(t)
	v0, _ := clients(t)

	session := connect(t, ctx, v0, "drain-operator", map[string]int32{"default": 10})
	workerId := session.Registration().WorkerId

	_, _, err := session.Actions(ctx)
	require.NoError(t, err)

	pollWorkerActive(t, ctx, workerId, true)

	require.NoError(t, session.Close())

	assert.True(t, workerPaused(t, ctx, workerId), "the worker is paused before the session hangs up")
	pollWorkerActive(t, ctx, workerId, false)
}

// The same echo operator the in-process host runs in its unit tests is hosted here over
// OperatorService through pkg/operator/hostgrpc: opened with the token's tenant as its
// identity, given a workflow through the session, driven to a completed run, and torn down in
// the host's order (pause, drain, close).
func TestHostGRPCEcho(t *testing.T) {
	ctx := newTestContext(t)
	_, sdk := clients(t)

	source, err := hostgrpc.NewStaticExchange(os.Getenv("HATCHET_CLIENT_TOKEN"))
	require.NoError(t, err)

	host, err := hostgrpc.New(hostgrpc.WithTokenSource(source))
	require.NoError(t, err)
	t.Cleanup(host.Close)

	echo := &operatortest.Echo{}

	session, err := host.Open(ctx, operator.Identity{TenantId: source.TenantId(), Name: uniqueName("host-echo")}, operator.OpenOpts{
		Handler:    echo,
		SlotConfig: map[string]int32{"default": 10},
	})
	require.NoError(t, err)
	require.NoError(t, echo.Start(ctx, session))

	reg := session.Registration()
	assert.Equal(t, source.TenantId(), reg.TenantId)
	assert.NotEqual(t, uuid.Nil, reg.OperatorId)
	assert.NotEqual(t, uuid.Nil, reg.WorkerId)
	assert.False(t, reg.Resumed)

	workerId := reg.WorkerId.String()
	pollWorkerActive(t, ctx, workerId, true)
	assert.Empty(t, workerActions(t, ctx, workerId), "opening links no actions")

	name := uniqueName("host-echo-wf")
	actions, err := session.PutWorkflow(ctx, simpleWorkflow(name, "hostecho:run", false))
	require.NoError(t, err)
	assert.Equal(t, []string{"hostecho:run"}, actions)

	require.NoError(t, session.AddActions(ctx, actions))
	require.NoError(t, session.Flush(ctx))
	pollUntil(t, ctx, func() (bool, error) {
		return slices.Contains(workerActions(t, ctx, workerId), "hostecho:run"), nil
	})

	input := map[string]any{"message": "hello", "n": float64(3)}
	details := runToCompletion(t, ctx, sdk, name, input)
	assert.Equal(t, input, taskOutput(t, details))
	assert.Equal(t, 1, echo.Handled())

	// the host's teardown order: pause, the operator's drain, close
	require.NoError(t, session.Pause(ctx))
	pollWorkerPaused(t, ctx, workerId, true)

	echo.Drain(ctx)

	require.NoError(t, session.Close(ctx))
	pollWorkerActive(t, ctx, workerId, false)

	assert.ErrorIs(t, session.Flush(ctx), operator.ErrSessionClosed)
}
