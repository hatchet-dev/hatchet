package dagoperator

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	internalcel "github.com/hatchet-dev/hatchet/internal/cel"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// triggerStepFn is the step-trigger callback the dag invokes for each task; matches the
// signature dagDurableTask expects.
type triggerStepFn = func(ctx context.Context, actionId, workflowName string, childIndex int32, parentTaskRunIds []uuid.UUID, isSkipped, isCancelled, parentReExecuted bool) (*operator.DAGStepTriggerResult, error)

func newTestTask(readableId, actionId string, index int32, parents ...*task) *task {
	return &task{
		id:           uuid.New(),
		actionId:     actionId,
		workflowName: "test-workflow",
		readableId:   readableId,
		index:        index,
		parents:      parents,
	}
}

func stubTriggerStep(t *testing.T, async map[string]bool) triggerStepFn {
	var nextId int64 = 1

	return func(ctx context.Context, actionId, workflowName string, childIndex int32, parentTaskRunIds []uuid.UUID, isSkipped, isCancelled, parentReExecuted bool) (*operator.DAGStepTriggerResult, error) {
		nodeId := nextId
		nextId++

		result := &operator.DAGStepTriggerResult{
			NodeId:                nodeId,
			BranchId:              nodeId,
			WorkflowRunExternalId: uuid.New(),
		}

		switch {
		case isCancelled:
			result.IsSatisfied = true
			result.ResultPayload, _ = json.Marshal(map[string]any{"cancelled": true})
		case isSkipped:
			// dag itself marks the task skipped/completed without needing IsSatisfied.
		case async[actionId]:
			// left unsatisfied; test drives completion via responseCh.
		default:
			result.IsSatisfied = true
			result.ResultPayload, _ = json.Marshal(map[string]any{"ok": true})
		}

		return result, nil
	}
}

type fakeDispatcher struct {
	acks chan *v1contracts.DurableEventLogEntryRef
}

func newFakeDispatcher(requestCh chan *v1contracts.DurableTaskRequest, responseCh chan *v1contracts.DurableTaskResponse, stop <-chan struct{}) *fakeDispatcher {
	fd := &fakeDispatcher{acks: make(chan *v1contracts.DurableEventLogEntryRef, 16)}

	go func() {
		var nextId int64 = 1000

		for {
			select {
			case <-stop:
				return
			case req, ok := <-requestCh:
				if !ok {
					return
				}

				waitFor := req.GetWaitFor()
				if waitFor == nil {
					continue
				}

				ref := &v1contracts.DurableEventLogEntryRef{
					DurableTaskExternalId: waitFor.DurableTaskExternalId,
					InvocationCount:       waitFor.InvocationCount,
					NodeId:                nextId,
					BranchId:              nextId,
				}
				nextId++

				select {
				case responseCh <- &v1contracts.DurableTaskResponse{
					Message: &v1contracts.DurableTaskResponse_WaitForAck{
						WaitForAck: &v1contracts.DurableTaskEventWaitForAckResponse{Ref: ref},
					},
				}:
				case <-stop:
					return
				}

				select {
				case fd.acks <- ref:
				case <-stop:
					return
				}
			}
		}
	}()

	return fd
}

func sendEntryCompleted(t *testing.T, responseCh chan *v1contracts.DurableTaskResponse, ref *v1contracts.DurableEventLogEntryRef, payload []byte) {
	t.Helper()

	select {
	case responseCh <- &v1contracts.DurableTaskResponse{
		Message: &v1contracts.DurableTaskResponse_EntryCompleted{
			EntryCompleted: &v1contracts.DurableTaskEventLogEntryCompletedResponse{
				Ref:     ref,
				Payload: payload,
			},
		},
	}:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out sending EntryCompleted response")
	}
}

type dagHarness struct {
	errCh      chan error
	responseCh chan *v1contracts.DurableTaskResponse
	fd         *fakeDispatcher
	cleanup    func()
}

func startDAG(t *testing.T, tasks []*task, triggerStep triggerStepFn) *dagHarness {
	t.Helper()

	return startDAGFull(t, tasks, nil, triggerStep)
}

func startDAGFull(t *testing.T, tasks []*task, onFailureTask *task, triggerStep triggerStepFn) *dagHarness {
	t.Helper()

	evaluator, err := internalcel.NewBoolExprEvaluator()
	require.NoError(t, err)

	requestCh := make(chan *v1contracts.DurableTaskRequest)
	responseCh := make(chan *v1contracts.DurableTaskResponse)

	stop := make(chan struct{})

	fd := newFakeDispatcher(requestCh, responseCh, stop)

	errCh := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	go func() {
		errCh <- dagDurableTask(ctx, tasks, onFailureTask, uuid.New(), 1, "{}", requestCh, responseCh, evaluator.EvalBoolExpr, triggerStep)
	}()

	return &dagHarness{
		errCh:      errCh,
		responseCh: responseCh,
		fd:         fd,
		cleanup: func() {
			close(stop)
			cancel()
		},
	}
}

func (h *dagHarness) waitErr(t *testing.T) error {
	t.Helper()

	select {
	case err := <-h.errCh:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for dag to finish")
		return nil
	}
}

func (h *dagHarness) waitAck(t *testing.T) *v1contracts.DurableEventLogEntryRef {
	t.Helper()

	select {
	case ref := <-h.fd.acks:
		return ref
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for wait condition ack")
		return nil
	}
}

func runDAG(t *testing.T, tasks []*task, triggerStep triggerStepFn) (error, chan *v1contracts.DurableTaskResponse, func()) {
	t.Helper()

	h := startDAG(t, tasks, triggerStep)

	return h.waitErr(t), h.responseCh, h.cleanup
}

type asyncTriggered struct {
	actionId string
	ref      *v1contracts.DurableEventLogEntryRef
}

func asyncTrigger(async map[string]bool) (triggerStepFn, chan asyncTriggered) {
	triggered := make(chan asyncTriggered, 16)
	var nextId int64 = 1

	fn := func(ctx context.Context, actionId, workflowName string, childIndex int32, parentTaskRunIds []uuid.UUID, isSkipped, isCancelled, parentReExecuted bool) (*operator.DAGStepTriggerResult, error) {
		nodeId := nextId
		nextId++

		result := &operator.DAGStepTriggerResult{
			NodeId:                nodeId,
			BranchId:              nodeId,
			WorkflowRunExternalId: uuid.New(),
		}

		switch {
		case isCancelled:
			result.IsSatisfied = true
			result.ResultPayload = mapToJson(map[string]interface{}{"cancelled": true})
		case isSkipped:
			// the dag marks the skip itself; no IsSatisfied needed.
		case async[actionId]:
			triggered <- asyncTriggered{
				actionId: actionId,
				ref:      &v1contracts.DurableEventLogEntryRef{NodeId: nodeId, BranchId: nodeId},
			}
		default:
			result.IsSatisfied = true
			result.ResultPayload = mapToJson(map[string]interface{}{"ok": true})
		}

		return result, nil
	}

	return fn, triggered
}

func recvTriggered(t *testing.T, ch chan asyncTriggered) asyncTriggered {
	t.Helper()

	select {
	case at := <-ch:
		return at
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for a step to be triggered")
		return asyncTriggered{}
	}
}

func mapToJson(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}

func sendEntryFailed(t *testing.T, responseCh chan *v1contracts.DurableTaskResponse, ref *v1contracts.DurableEventLogEntryRef, errMsg string) {
	t.Helper()

	select {
	case responseCh <- &v1contracts.DurableTaskResponse{
		Message: &v1contracts.DurableTaskResponse_EntryCompleted{
			EntryCompleted: &v1contracts.DurableTaskEventLogEntryCompletedResponse{
				Ref:          ref,
				IsFailure:    true,
				ErrorMessage: strPtr(errMsg),
			},
		},
	}:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out sending EntryCompleted failure response")
	}
}

func TestDag_LinearNoConditions(t *testing.T) {
	a := newTestTask("a", "action-a", 0)
	b := newTestTask("b", "action-b", 1, a)

	err, _, cleanup := runDAG(t, []*task{a, b}, stubTriggerStep(t, nil))
	defer cleanup()

	require.NoError(t, err)
	require.True(t, a.isCompleted)
	require.True(t, b.isCompleted)
	require.False(t, a.isFailed || b.isFailed)
	require.False(t, a.isCancelled || b.isCancelled)
}

func TestDag_ParentFailurePropagatesToChild(t *testing.T) {
	a := newTestTask("a", "action-a", 0)
	b := newTestTask("b", "action-b", 1, a)

	triggerStep := func(ctx context.Context, actionId, workflowName string, childIndex int32, parentTaskRunIds []uuid.UUID, isSkipped, isCancelled, parentReExecuted bool) (*operator.DAGStepTriggerResult, error) {
		if actionId == "action-a" {
			return &operator.DAGStepTriggerResult{
				NodeId:                1,
				BranchId:              1,
				WorkflowRunExternalId: uuid.New(),
				IsSatisfied:           true,
				IsFailure:             true,
				ErrorMessage:          strPtr("boom"),
			}, nil
		}

		require.True(t, isCancelled, "child of a failed parent should be triggered as cancelled")

		payload, _ := json.Marshal(map[string]interface{}{"cancelled": true})
		return &operator.DAGStepTriggerResult{
			NodeId:                2,
			BranchId:              2,
			WorkflowRunExternalId: uuid.New(),
			IsSatisfied:           true,
			ResultPayload:         payload,
		}, nil
	}

	err, _, cleanup := runDAG(t, []*task{a, b}, triggerStep)
	defer cleanup()

	require.Error(t, err)
	require.True(t, isDagChildFailedErr(err))
	require.True(t, a.isFailed)
}

func TestDag_ParentReExecutedPropagation(t *testing.T) {
	a := newTestTask("a", "action-a", 0)
	b := newTestTask("b", "action-b", 1)
	c := newTestTask("c", "action-c", 2, a, b)
	d := newTestTask("d", "action-d", 3, b)
	e := newTestTask("e", "action-e", 4, c)

	received := make(map[string]bool)
	var nextId int64 = 1

	triggerStep := func(ctx context.Context, actionId, workflowName string, childIndex int32, parentTaskRunIds []uuid.UUID, isSkipped, isCancelled, parentReExecuted bool) (*operator.DAGStepTriggerResult, error) {
		received[actionId] = parentReExecuted

		nodeId := nextId
		nextId++

		payload, _ := json.Marshal(map[string]interface{}{"ok": true})

		return &operator.DAGStepTriggerResult{
			NodeId:                nodeId,
			BranchId:              nodeId,
			WorkflowRunExternalId: uuid.New(),
			IsSatisfied:           true,
			ResultPayload:         payload,
			ReExecuted:            actionId == "action-a" || parentReExecuted,
		}, nil
	}

	err, _, cleanup := runDAG(t, []*task{a, b, c, d, e}, triggerStep)
	defer cleanup()

	require.NoError(t, err)
	require.Equal(t, map[string]bool{
		"action-a": false,
		"action-b": false,
		"action-c": true,
		"action-d": false,
		"action-e": true,
	}, received)
}

func TestDag_SkipConditionOnParentOutput(t *testing.T) {
	run := func(t *testing.T, shouldSkip bool) *task {
		a := newTestTask("a", "action-a", 0)
		b := newTestTask("b", "action-b", 1, a)

		b.stepConditions = []*sqlcv1.V1StepMatchCondition{
			{
				Kind:             sqlcv1.V1StepMatchConditionKindPARENTOVERRIDE,
				Action:           sqlcv1.V1MatchConditionActionSKIP,
				OrGroupID:        uuid.New(),
				Expression:       sqlchelpers.TextFromStr("output.shouldSkip == true"),
				ParentReadableID: sqlchelpers.TextFromStr("a"),
			},
		}

		triggerStep := func(ctx context.Context, actionId, workflowName string, childIndex int32, parentTaskRunIds []uuid.UUID, isSkipped, isCancelled, parentReExecuted bool) (*operator.DAGStepTriggerResult, error) {
			if actionId == "action-a" {
				payload, _ := json.Marshal(map[string]interface{}{"shouldSkip": shouldSkip})
				return &operator.DAGStepTriggerResult{
					NodeId: 1, BranchId: 1, WorkflowRunExternalId: uuid.New(),
					IsSatisfied: true, ResultPayload: payload,
				}, nil
			}

			require.Equal(t, shouldSkip, isSkipped, "child's skip state should mirror the parent-override condition")
			return &operator.DAGStepTriggerResult{
				NodeId: 2, BranchId: 2, WorkflowRunExternalId: uuid.New(), IsSatisfied: true,
			}, nil
		}

		err, _, cleanup := runDAG(t, []*task{a, b}, triggerStep)
		defer cleanup()

		require.NoError(t, err)
		require.True(t, a.isCompleted)
		require.Nil(t, a.output, "parent output should be dropped once its dependents' conditions have been evaluated so we don't hold it in memory for the duration of the dag (to avoid leaking memory)")
		return b
	}

	t.Run("condition matches -> child skipped", func(t *testing.T) {
		b := run(t, true)
		require.True(t, b.isSkipped)
		require.True(t, b.isCompleted)
	})

	t.Run("condition doesn't match -> child runs", func(t *testing.T) {
		b := run(t, false)
		require.False(t, b.isSkipped)
		require.True(t, b.isCompleted)
	})
}

func TestDag_CancelConditionOnParentOutput(t *testing.T) {
	a := newTestTask("a", "action-a", 0)
	b := newTestTask("b", "action-b", 1, a)

	b.stepConditions = []*sqlcv1.V1StepMatchCondition{
		{
			Kind:             sqlcv1.V1StepMatchConditionKindPARENTOVERRIDE,
			Action:           sqlcv1.V1MatchConditionActionCANCEL,
			OrGroupID:        uuid.New(),
			Expression:       sqlchelpers.TextFromStr("output.shouldCancel == true"),
			ParentReadableID: sqlchelpers.TextFromStr("a"),
		},
	}

	triggerStep := func(ctx context.Context, actionId, workflowName string, childIndex int32, parentTaskRunIds []uuid.UUID, isSkipped, isCancelled, parentReExecuted bool) (*operator.DAGStepTriggerResult, error) {
		if actionId == "action-a" {
			payload, _ := json.Marshal(map[string]interface{}{"shouldCancel": true})
			return &operator.DAGStepTriggerResult{
				NodeId: 1, BranchId: 1, WorkflowRunExternalId: uuid.New(),
				IsSatisfied: true, ResultPayload: payload,
			}, nil
		}

		require.True(t, isCancelled, "child should be triggered as cancelled")
		payload, _ := json.Marshal(map[string]interface{}{"cancelled": true})
		return &operator.DAGStepTriggerResult{
			NodeId: 2, BranchId: 2, WorkflowRunExternalId: uuid.New(),
			IsSatisfied: true, ResultPayload: payload,
		}, nil
	}

	err, _, cleanup := runDAG(t, []*task{a, b}, triggerStep)
	defer cleanup()

	require.Error(t, err)
	require.True(t, isDagCancelledErr(err))
	require.True(t, b.isCancelled)
}

func TestDag_SleepWaitCondition(t *testing.T) {
	a := newTestTask("a", "action-a", 0)
	b := newTestTask("b", "action-b", 1, a)

	b.stepConditions = []*sqlcv1.V1StepMatchCondition{
		{
			Kind:            sqlcv1.V1StepMatchConditionKindSLEEP,
			Action:          sqlcv1.V1MatchConditionActionQUEUE,
			OrGroupID:       uuid.New(),
			ReadableDataKey: "sleep-1",
			SleepDuration:   sqlchelpers.TextFromStr("5s"),
		},
	}

	evaluator, err := internalcel.NewBoolExprEvaluator()
	require.NoError(t, err)

	requestCh := make(chan *v1contracts.DurableTaskRequest)
	responseCh := make(chan *v1contracts.DurableTaskResponse)
	stop := make(chan struct{})
	defer close(stop)

	fd := newFakeDispatcher(requestCh, responseCh, stop)

	errCh := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		errCh <- dagDurableTask(ctx, []*task{a, b}, nil, uuid.New(), 1, "{}", requestCh, responseCh, evaluator.EvalBoolExpr, stubTriggerStep(t, nil))
	}()

	var ref *v1contracts.DurableEventLogEntryRef
	select {
	case ref = <-fd.acks:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for sleep condition to be registered")
	}

	require.True(t, b.isWaiting)
	require.False(t, b.isCompleted)

	sendEntryCompleted(t, responseCh, ref, nil)

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for dag to finish after sleep fired")
	}

	require.True(t, b.isWaitSatisfied)
	require.True(t, b.isCompleted)
}

func TestDag_SkipOrGroupAcrossParents(t *testing.T) {
	a := newTestTask("a", "action-a", 0)
	b := newTestTask("b", "action-b", 1)
	c := newTestTask("c", "action-c", 2, a, b)
	d := newTestTask("d", "action-d", 3, c)
	e := newTestTask("e", "action-e", 4, d)
	f := newTestTask("f", "action-f", 5, c, d)
	g := newTestTask("g", "action-g", 6, a, f)

	orGroup := uuid.New()
	c.stepConditions = []*sqlcv1.V1StepMatchCondition{
		{
			Kind:             sqlcv1.V1StepMatchConditionKindPARENTOVERRIDE,
			Action:           sqlcv1.V1MatchConditionActionSKIP,
			OrGroupID:        orGroup,
			Expression:       sqlchelpers.TextFromStr("output.shouldSkip == true"),
			ParentReadableID: sqlchelpers.TextFromStr("a"),
		},
		{
			Kind:             sqlcv1.V1StepMatchConditionKindPARENTOVERRIDE,
			Action:           sqlcv1.V1MatchConditionActionSKIP,
			OrGroupID:        orGroup,
			Expression:       sqlchelpers.TextFromStr("output.shouldSkip == true"),
			ParentReadableID: sqlchelpers.TextFromStr("b"),
		},
	}

	triggerStep := func(ctx context.Context, actionId, workflowName string, childIndex int32, parentTaskRunIds []uuid.UUID, isSkipped, isCancelled, parentReExecuted bool) (*operator.DAGStepTriggerResult, error) {
		switch actionId {
		case "action-a":
			payload, _ := json.Marshal(map[string]interface{}{"shouldSkip": false})
			return &operator.DAGStepTriggerResult{
				NodeId: 1, BranchId: 1, WorkflowRunExternalId: uuid.New(),
				IsSatisfied: true, ResultPayload: payload,
			}, nil
		case "action-b":
			payload, _ := json.Marshal(map[string]interface{}{"shouldSkip": true})
			return &operator.DAGStepTriggerResult{
				NodeId: 2, BranchId: 2, WorkflowRunExternalId: uuid.New(),
				IsSatisfied: true, ResultPayload: payload,
			}, nil
		case "action-c":
			payload, _ := json.Marshal(map[string]interface{}{"ok": true})
			return &operator.DAGStepTriggerResult{
				NodeId: 3, BranchId: 3, WorkflowRunExternalId: uuid.New(),
				IsSatisfied: true, ResultPayload: payload,
			}, nil
		case "action-g":
			require.False(t, isSkipped, "g should not be skipped because one of its parents (a) is not")
			return &operator.DAGStepTriggerResult{
				NodeId: 4, BranchId: 4, WorkflowRunExternalId: uuid.New(),
				IsSatisfied: true,
			}, nil
		default:
			require.True(t, isSkipped, "c (and downstream) should be skipped once the shared or-group is satisfied by b")
			return &operator.DAGStepTriggerResult{NodeId: 3, BranchId: 3, WorkflowRunExternalId: uuid.New()}, nil
		}
	}

	err, _, cleanup := runDAG(t, []*task{a, b, c, d, e, f, g}, triggerStep)
	defer cleanup()

	require.NoError(t, err)
	require.True(t, c.isSkipped)
	require.True(t, c.isCompleted)
	require.True(t, d.isSkipped)
	require.True(t, d.isCompleted)
	require.True(t, e.isSkipped)
	require.True(t, e.isCompleted)
	require.True(t, f.isSkipped)
	require.True(t, f.isCompleted)
	require.False(t, g.isSkipped, "g should not be skipped because one of its parents (a) is not")
	require.True(t, g.isCompleted)
}

func TestDag_AsyncCompletionViaEntryCompleted(t *testing.T) {
	a := newTestTask("a", "action-a", 0)
	b := newTestTask("b", "action-b", 1, a)

	trigger, triggered := asyncTrigger(map[string]bool{"action-a": true, "action-b": true})

	h := startDAG(t, []*task{a, b}, trigger)
	defer h.cleanup()

	first := recvTriggered(t, triggered)
	require.Equal(t, "action-a", first.actionId)

	// b depends on a, so it must not be triggered until a's completion lands.
	select {
	case unexpected := <-triggered:
		t.Fatalf("%q triggered before its parent completed", unexpected.actionId)
	case <-time.After(100 * time.Millisecond):
	}

	sendEntryCompleted(t, h.responseCh, first.ref, mapToJson(map[string]interface{}{"ok": true}))

	second := recvTriggered(t, triggered)
	require.Equal(t, "action-b", second.actionId)

	sendEntryCompleted(t, h.responseCh, second.ref, mapToJson(map[string]interface{}{"ok": true}))

	require.NoError(t, h.waitErr(t))
	require.True(t, a.isCompleted && b.isCompleted)
	require.False(t, a.isFailed || b.isFailed)
	require.False(t, a.isCancelled || b.isCancelled)
}

func TestDag_AsyncFailureViaEntryCompleted(t *testing.T) {
	a := newTestTask("a", "action-a", 0)
	b := newTestTask("b", "action-b", 1, a)

	trigger, triggered := asyncTrigger(map[string]bool{"action-a": true})

	h := startDAG(t, []*task{a, b}, trigger)
	defer h.cleanup()

	first := recvTriggered(t, triggered)
	require.Equal(t, "action-a", first.actionId)

	sendEntryFailed(t, h.responseCh, first.ref, "boom")

	err := h.waitErr(t)
	require.Error(t, err)
	require.True(t, isDagChildFailedErr(err))
	require.True(t, a.isFailed)
	require.True(t, b.isCancelled)
}

func TestDag_AsyncCancelViaErrorMessage(t *testing.T) {
	a := newTestTask("a", "action-a", 0)
	b := newTestTask("b", "action-b", 1, a)

	trigger, triggered := asyncTrigger(map[string]bool{"action-a": true})

	h := startDAG(t, []*task{a, b}, trigger)
	defer h.cleanup()

	first := recvTriggered(t, triggered)

	sendEntryFailed(t, h.responseCh, first.ref, repository.TaskCancelledErrorMessage)

	err := h.waitErr(t)
	require.Error(t, err)
	require.True(t, isDagCancelledErr(err))
	require.True(t, a.isCancelled)
	require.False(t, a.isFailed)
	require.True(t, b.isCancelled, "child of a cancelled parent should be triggered as cancelled")
	require.False(t, b.isFailed)
}

func TestDag_SleepSkipWatch(t *testing.T) {
	a := newTestTask("a", "action-a", 0)
	b := newTestTask("b", "action-b", 1, a)

	b.stepConditions = []*sqlcv1.V1StepMatchCondition{
		{
			Kind:            sqlcv1.V1StepMatchConditionKindUSEREVENT,
			Action:          sqlcv1.V1MatchConditionActionQUEUE,
			OrGroupID:       uuid.New(),
			ReadableDataKey: "wait-1",
			EventKey:        sqlchelpers.TextFromStr("go"),
		},
		{
			Kind:            sqlcv1.V1StepMatchConditionKindSLEEP,
			Action:          sqlcv1.V1MatchConditionActionSKIP,
			OrGroupID:       uuid.New(),
			ReadableDataKey: "skip-after",
			SleepDuration:   sqlchelpers.TextFromStr("5s"),
		},
	}

	h := startDAG(t, []*task{a, b}, stubTriggerStep(t, nil))
	defer h.cleanup()

	skipRef := h.waitAck(t)
	h.waitAck(t)

	sendEntryCompleted(t, h.responseCh, skipRef, nil)

	require.NoError(t, h.waitErr(t))
	require.True(t, b.isSkipped)
	require.True(t, b.isCompleted)
	require.False(t, b.isCancelled)
}

func TestDag_UserEventCancelWatch(t *testing.T) {
	a := newTestTask("a", "action-a", 0)
	b := newTestTask("b", "action-b", 1, a)

	b.stepConditions = []*sqlcv1.V1StepMatchCondition{
		{
			Kind:            sqlcv1.V1StepMatchConditionKindSLEEP,
			Action:          sqlcv1.V1MatchConditionActionQUEUE,
			OrGroupID:       uuid.New(),
			ReadableDataKey: "wait-1",
			SleepDuration:   sqlchelpers.TextFromStr("5s"),
		},
		{
			Kind:            sqlcv1.V1StepMatchConditionKindUSEREVENT,
			Action:          sqlcv1.V1MatchConditionActionCANCEL,
			OrGroupID:       uuid.New(),
			ReadableDataKey: "cancel-on",
			EventKey:        sqlchelpers.TextFromStr("cancel"),
		},
	}

	h := startDAG(t, []*task{a, b}, stubTriggerStep(t, nil))
	defer h.cleanup()

	cancelRef := h.waitAck(t)
	h.waitAck(t)

	sendEntryCompleted(t, h.responseCh, cancelRef, nil)

	err := h.waitErr(t)
	require.Error(t, err)
	require.True(t, isDagCancelledErr(err))
	require.True(t, b.isCancelled)
}

func TestDag_CancelWatchWithSkipWatchRegistered(t *testing.T) {
	a := newTestTask("a", "action-a", 0)
	b := newTestTask("b", "action-b", 1, a)

	b.stepConditions = []*sqlcv1.V1StepMatchCondition{
		{
			Kind:            sqlcv1.V1StepMatchConditionKindUSEREVENT,
			Action:          sqlcv1.V1MatchConditionActionQUEUE,
			OrGroupID:       uuid.New(),
			ReadableDataKey: "wait-1",
			EventKey:        sqlchelpers.TextFromStr("go"),
		},
		{
			Kind:            sqlcv1.V1StepMatchConditionKindSLEEP,
			Action:          sqlcv1.V1MatchConditionActionSKIP,
			OrGroupID:       uuid.New(),
			ReadableDataKey: "skip-after",
			SleepDuration:   sqlchelpers.TextFromStr("5s"),
		},
		{
			Kind:            sqlcv1.V1StepMatchConditionKindSLEEP,
			Action:          sqlcv1.V1MatchConditionActionCANCEL,
			OrGroupID:       uuid.New(),
			ReadableDataKey: "cancel-after",
			SleepDuration:   sqlchelpers.TextFromStr("5s"),
		},
	}

	h := startDAG(t, []*task{a, b}, stubTriggerStep(t, nil))
	defer h.cleanup()

	h.waitAck(t)
	cancelRef := h.waitAck(t)
	h.waitAck(t)

	sendEntryCompleted(t, h.responseCh, cancelRef, nil)

	err := h.waitErr(t)
	require.Error(t, err)
	require.True(t, isDagCancelledErr(err))
	require.True(t, b.isCancelled)
	require.False(t, b.isSkipped)
}

func TestDag_CancelBeatsSkipParentOverride(t *testing.T) {
	a := newTestTask("a", "action-a", 0)
	b := newTestTask("b", "action-b", 1, a)

	b.stepConditions = []*sqlcv1.V1StepMatchCondition{
		{
			Kind:             sqlcv1.V1StepMatchConditionKindPARENTOVERRIDE,
			Action:           sqlcv1.V1MatchConditionActionSKIP,
			OrGroupID:        uuid.New(),
			Expression:       sqlchelpers.TextFromStr("output.flag == true"),
			ParentReadableID: sqlchelpers.TextFromStr("a"),
		},
		{
			Kind:             sqlcv1.V1StepMatchConditionKindPARENTOVERRIDE,
			Action:           sqlcv1.V1MatchConditionActionCANCEL,
			OrGroupID:        uuid.New(),
			Expression:       sqlchelpers.TextFromStr("output.flag == true"),
			ParentReadableID: sqlchelpers.TextFromStr("a"),
		},
	}

	triggerStep := func(ctx context.Context, actionId, workflowName string, childIndex int32, parentTaskRunIds []uuid.UUID, isSkipped, isCancelled, parentReExecuted bool) (*operator.DAGStepTriggerResult, error) {
		if actionId == "action-a" {
			return &operator.DAGStepTriggerResult{
				NodeId: 1, BranchId: 1, WorkflowRunExternalId: uuid.New(),
				IsSatisfied: true, ResultPayload: mapToJson(map[string]interface{}{"flag": true}),
			}, nil
		}

		require.True(t, isCancelled, "cancel must take precedence over skip")
		require.False(t, isSkipped)
		return &operator.DAGStepTriggerResult{
			NodeId: 2, BranchId: 2, WorkflowRunExternalId: uuid.New(),
			IsSatisfied: true, ResultPayload: mapToJson(map[string]interface{}{"cancelled": true}),
		}, nil
	}

	err, _, cleanup := runDAG(t, []*task{a, b}, triggerStep)
	defer cleanup()

	require.Error(t, err)
	require.True(t, isDagCancelledErr(err))
	require.True(t, b.isCancelled)
	require.False(t, b.isSkipped)
}

func TestDag_MultipleSkipGroupsAllRequired(t *testing.T) {
	run := func(t *testing.T, aFlag, bFlag bool) *task {
		a := newTestTask("a", "action-a", 0)
		b := newTestTask("b", "action-b", 1)
		c := newTestTask("c", "action-c", 2, a, b)

		c.stepConditions = []*sqlcv1.V1StepMatchCondition{
			{
				Kind:             sqlcv1.V1StepMatchConditionKindPARENTOVERRIDE,
				Action:           sqlcv1.V1MatchConditionActionSKIP,
				OrGroupID:        uuid.New(),
				Expression:       sqlchelpers.TextFromStr("output.skip == true"),
				ParentReadableID: sqlchelpers.TextFromStr("a"),
			},
			{
				Kind:             sqlcv1.V1StepMatchConditionKindPARENTOVERRIDE,
				Action:           sqlcv1.V1MatchConditionActionSKIP,
				OrGroupID:        uuid.New(),
				Expression:       sqlchelpers.TextFromStr("output.skip == true"),
				ParentReadableID: sqlchelpers.TextFromStr("b"),
			},
		}

		triggerStep := func(ctx context.Context, actionId, workflowName string, childIndex int32, parentTaskRunIds []uuid.UUID, isSkipped, isCancelled, parentReExecuted bool) (*operator.DAGStepTriggerResult, error) {
			switch actionId {
			case "action-a":
				return &operator.DAGStepTriggerResult{
					NodeId: 1, BranchId: 1, WorkflowRunExternalId: uuid.New(),
					IsSatisfied: true, ResultPayload: mapToJson(map[string]interface{}{"skip": aFlag}),
				}, nil
			case "action-b":
				return &operator.DAGStepTriggerResult{
					NodeId: 2, BranchId: 2, WorkflowRunExternalId: uuid.New(),
					IsSatisfied: true, ResultPayload: mapToJson(map[string]interface{}{"skip": bFlag}),
				}, nil
			default:
				return &operator.DAGStepTriggerResult{
					NodeId: 3, BranchId: 3, WorkflowRunExternalId: uuid.New(),
					IsSatisfied: true, ResultPayload: mapToJson(map[string]interface{}{"ok": true}),
				}, nil
			}
		}

		err, _, cleanup := runDAG(t, []*task{a, b, c}, triggerStep)
		defer cleanup()

		require.NoError(t, err)
		return c
	}

	t.Run("both groups match -> skipped", func(t *testing.T) {
		c := run(t, true, true)
		require.True(t, c.isSkipped)
	})

	t.Run("one group matches -> runs", func(t *testing.T) {
		c := run(t, true, false)
		require.False(t, c.isSkipped)
		require.True(t, c.isCompleted)
	})

	t.Run("neither group matches -> runs", func(t *testing.T) {
		c := run(t, false, false)
		require.False(t, c.isSkipped)
		require.True(t, c.isCompleted)
	})
}

func TestDag_Diamond(t *testing.T) {
	a := newTestTask("a", "action-a", 0)
	b := newTestTask("b", "action-b", 1, a)
	c := newTestTask("c", "action-c", 2, a)
	d := newTestTask("d", "action-d", 3, b, c)

	trigger, triggered := asyncTrigger(map[string]bool{
		"action-a": true, "action-b": true, "action-c": true, "action-d": true,
	})

	h := startDAG(t, []*task{a, b, c, d}, trigger)
	defer h.cleanup()

	first := recvTriggered(t, triggered)
	require.Equal(t, "action-a", first.actionId)
	sendEntryCompleted(t, h.responseCh, first.ref, mapToJson(map[string]interface{}{"ok": true}))

	second := recvTriggered(t, triggered)
	third := recvTriggered(t, triggered)
	require.ElementsMatch(t, []string{"action-b", "action-c"}, []string{second.actionId, third.actionId})

	sendEntryCompleted(t, h.responseCh, second.ref, mapToJson(map[string]interface{}{"ok": true}))

	select {
	case unexpected := <-triggered:
		t.Fatalf("%q triggered before both fan-in parents completed", unexpected.actionId)
	case <-time.After(100 * time.Millisecond):
	}

	sendEntryCompleted(t, h.responseCh, third.ref, mapToJson(map[string]interface{}{"ok": true}))

	fourth := recvTriggered(t, triggered)
	require.Equal(t, "action-d", fourth.actionId)
	sendEntryCompleted(t, h.responseCh, fourth.ref, mapToJson(map[string]interface{}{"ok": true}))

	require.NoError(t, h.waitErr(t))
	for _, task := range []*task{a, b, c, d} {
		require.True(t, task.isCompleted && !task.isFailed && !task.isCancelled)
	}
}

func TestDag_MultipleIndependentRoots(t *testing.T) {
	a := newTestTask("a", "action-a", 0)
	b := newTestTask("b", "action-b", 1)
	c := newTestTask("c", "action-c", 2, a, b)
	d := newTestTask("d", "action-d", 3)

	trigger, triggered := asyncTrigger(map[string]bool{"action-a": true, "action-b": true, "action-d": true})

	h := startDAG(t, []*task{a, b, c, d}, trigger)
	defer h.cleanup()

	first := recvTriggered(t, triggered)
	second := recvTriggered(t, triggered)
	third := recvTriggered(t, triggered)
	require.ElementsMatch(t, []string{"action-a", "action-b", "action-d"}, []string{first.actionId, second.actionId, third.actionId})

	sendEntryCompleted(t, h.responseCh, first.ref, mapToJson(map[string]interface{}{"ok": true}))

	select {
	case unexpected := <-triggered:
		t.Fatalf("%q triggered before all roots completed", unexpected.actionId)
	case <-time.After(100 * time.Millisecond):
	}

	sendEntryCompleted(t, h.responseCh, second.ref, mapToJson(map[string]interface{}{"ok": true}))
	sendEntryCompleted(t, h.responseCh, third.ref, mapToJson(map[string]interface{}{"ok": true}))

	require.NoError(t, h.waitErr(t))
	require.True(t, a.isCompleted && b.isCompleted && c.isCompleted && d.isCompleted)
}

func TestDag_SkippedAndFailedParentChildCancelled(t *testing.T) {
	a := newTestTask("a", "action-a", 0)
	b := newTestTask("b", "action-b", 1)
	c := newTestTask("c", "action-c", 2, a, b)

	triggerStep := func(ctx context.Context, actionId, workflowName string, childIndex int32, parentTaskRunIds []uuid.UUID, isSkipped, isCancelled, parentReExecuted bool) (*operator.DAGStepTriggerResult, error) {
		switch actionId {
		case "action-a":
			return &operator.DAGStepTriggerResult{
				NodeId: 1, BranchId: 1, WorkflowRunExternalId: uuid.New(),
				IsSatisfied: true, ResultPayload: mapToJson(map[string]interface{}{"skipped": true}),
			}, nil
		case "action-b":
			return &operator.DAGStepTriggerResult{
				NodeId: 2, BranchId: 2, WorkflowRunExternalId: uuid.New(),
				IsSatisfied: true, IsFailure: true, ErrorMessage: strPtr("boom"),
			}, nil
		default:
			require.True(t, isCancelled, "c should be cancelled because a parent failed")
			return &operator.DAGStepTriggerResult{
				NodeId: 3, BranchId: 3, WorkflowRunExternalId: uuid.New(),
				IsSatisfied: true, ResultPayload: mapToJson(map[string]interface{}{"cancelled": true}),
			}, nil
		}
	}

	err, _, cleanup := runDAG(t, []*task{a, b, c}, triggerStep)
	defer cleanup()

	require.Error(t, err)
	require.True(t, isDagChildFailedErr(err))
	require.True(t, a.isSkipped)
	require.True(t, b.isFailed)
	require.True(t, c.isCancelled)
	require.False(t, c.isFailed)
	require.False(t, c.isSkipped)
}

func strPtr(s string) *string { return new(s) }

func TestDag_OnFailureTriggersOnSiblingFailure(t *testing.T) {
	a := newTestTask("a", "action-a", 0)
	onFailure := newTestTask("on-failure", "action-on-failure", 1)

	trigger, triggered := asyncTrigger(map[string]bool{"action-a": true, "action-on-failure": true})

	h := startDAGFull(t, []*task{a}, onFailure, trigger)
	defer h.cleanup()

	first := recvTriggered(t, triggered)
	require.Equal(t, "action-a", first.actionId)

	sendEntryFailed(t, h.responseCh, first.ref, "boom")

	second := recvTriggered(t, triggered)
	require.Equal(t, "action-on-failure", second.actionId)

	sendEntryCompleted(t, h.responseCh, second.ref, mapToJson(map[string]interface{}{"ok": true}))

	err := h.waitErr(t)
	require.Error(t, err)
	require.True(t, isDagChildFailedErr(err))
	require.True(t, a.isFailed)
	require.True(t, onFailure.isTriggered)
	require.False(t, onFailure.isSkipped)
	require.True(t, onFailure.isCompleted)
}

func TestDag_OnFailureSkippedWhenAllTasksSucceed(t *testing.T) {
	a := newTestTask("a", "action-a", 0)
	b := newTestTask("b", "action-b", 1)
	onFailure := newTestTask("on-failure", "action-on-failure", 2)

	h := startDAGFull(t, []*task{a, b}, onFailure, stubTriggerStep(t, nil))
	defer h.cleanup()

	err := h.waitErr(t)
	require.NoError(t, err)
	require.True(t, a.isCompleted)
	require.True(t, b.isCompleted)
	require.True(t, onFailure.isTriggered)
	require.True(t, onFailure.isSkipped)
	require.True(t, onFailure.isCompleted)
}

func TestDag_OnFailureNotTriggeredByCancellation(t *testing.T) {
	a := newTestTask("a", "action-a", 0)
	onFailure := newTestTask("on-failure", "action-on-failure", 1)

	triggerStep := func(ctx context.Context, actionId, workflowName string, childIndex int32, parentTaskRunIds []uuid.UUID, isSkipped, isCancelled, parentReExecuted bool) (*operator.DAGStepTriggerResult, error) {
		switch actionId {
		case "action-a":
			return &operator.DAGStepTriggerResult{
				NodeId: 1, BranchId: 1, WorkflowRunExternalId: uuid.New(),
				IsSatisfied: true, ResultPayload: mapToJson(map[string]interface{}{"cancelled": true}),
			}, nil
		default:
			require.True(t, isSkipped, "on-failure should be skipped, not run, for a cancelled (not failed) sibling")
			return &operator.DAGStepTriggerResult{
				NodeId: 2, BranchId: 2, WorkflowRunExternalId: uuid.New(),
			}, nil
		}
	}

	h := startDAGFull(t, []*task{a}, onFailure, triggerStep)
	defer h.cleanup()

	err := h.waitErr(t)
	require.Error(t, err)
	require.True(t, isDagCancelledErr(err))
	require.True(t, a.isCancelled)
	require.True(t, onFailure.isTriggered)
	require.True(t, onFailure.isSkipped)
}

func TestDag_EntryCompletedRacesAheadOfWaitForAck(t *testing.T) {
	a := newTestTask("a", "action-a", 0)
	b := newTestTask("b", "action-b", 1, a)

	b.stepConditions = []*sqlcv1.V1StepMatchCondition{
		{
			Kind:            sqlcv1.V1StepMatchConditionKindUSEREVENT,
			Action:          sqlcv1.V1MatchConditionActionQUEUE,
			OrGroupID:       uuid.New(),
			ReadableDataKey: "wait-1",
			EventKey:        sqlchelpers.TextFromStr("go"),
		},
		{
			Kind:            sqlcv1.V1StepMatchConditionKindSLEEP,
			Action:          sqlcv1.V1MatchConditionActionSKIP,
			OrGroupID:       uuid.New(),
			ReadableDataKey: "skip-after",
			SleepDuration:   sqlchelpers.TextFromStr("5s"),
		},
	}

	evaluator, err := internalcel.NewBoolExprEvaluator()
	require.NoError(t, err)

	requestCh := make(chan *v1contracts.DurableTaskRequest)
	responseCh := make(chan *v1contracts.DurableTaskResponse)
	stop := make(chan struct{})
	defer close(stop)

	errCh := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		errCh <- dagDurableTask(ctx, []*task{a, b}, nil, uuid.New(), 1, "{}", requestCh, responseCh, evaluator.EvalBoolExpr, stubTriggerStep(t, nil))
	}()

	// Drive the dispatcher side ourselves (instead of using newFakeDispatcher) so we can choose
	// to deliver the skip watch's EntryCompleted before the WaitForAck that would normally
	// precede it. b registers its skip watch first, then its wait watch (dag.go:300-314).
	recvWaitFor := func() *v1contracts.DurableTaskWaitForRequest {
		t.Helper()
		select {
		case req := <-requestCh:
			waitFor := req.GetWaitFor()
			require.NotNil(t, waitFor)
			return waitFor
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for a condition registration request")
			return nil
		}
	}

	skipWaitFor := recvWaitFor()
	_ = recvWaitFor()

	ref := &v1contracts.DurableEventLogEntryRef{
		DurableTaskExternalId: skipWaitFor.DurableTaskExternalId,
		InvocationCount:       skipWaitFor.InvocationCount,
		NodeId:                4242,
		BranchId:              4242,
	}

	sendEntryCompleted(t, responseCh, ref, nil)

	select {
	case responseCh <- &v1contracts.DurableTaskResponse{
		Message: &v1contracts.DurableTaskResponse_WaitForAck{
			WaitForAck: &v1contracts.DurableTaskEventWaitForAckResponse{Ref: ref},
		},
	}:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out sending WaitForAck")
	}

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for dag to finish; the raced EntryCompleted was likely dropped instead of buffered")
	}

	require.True(t, b.isSkipped)
	require.True(t, b.isCompleted)
	require.False(t, b.isCancelled)
}
