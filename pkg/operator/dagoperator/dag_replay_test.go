package dagoperator

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	internalcel "github.com/hatchet-dev/hatchet/internal/cel"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// replayLog models the parts of the durable event log a DAG replay depends on: node ids restart
// at 1 on every invocation and are matched by position, a step whose child already exists is
// found by its child instead (and moves the position past that entry), and completions are
// released to an invocation in the order they were satisfied, each one as soon as its
// predecessors are out. A request that lands on a position holding a different entry is recorded
// as a mismatch: the engine reports it as nondeterminism, or, for a step trigger, binds the step
// to that entry.
type replayLog struct {
	t          *testing.T
	externalId uuid.UUID

	mu         sync.Mutex
	entries    []*replayLogEntry
	stepNode   map[string]int64
	nextOrder  int64
	mismatches []string

	// per invocation
	invocation int32
	live       bool
	latestNode int64
	maxSent    int64
	held       map[int64]*v1contracts.DurableTaskResponse
	queue      []*v1contracts.DurableTaskResponse
	wake       chan struct{}
	onBlocked  func()
}

type replayLogEntry struct {
	kind      string
	key       string
	runId     uuid.UUID
	satisfied bool
	order     int64
	payload   []byte
}

func newReplayLog(t *testing.T) *replayLog {
	return &replayLog{t: t, externalId: uuid.New(), stepNode: make(map[string]int64)}
}

func (l *replayLog) startInvocation(invocation int32, onBlocked func()) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.invocation = invocation
	l.live = true
	l.latestNode = 0
	l.maxSent = 0
	l.held = make(map[int64]*v1contracts.DurableTaskResponse)
	l.queue = nil
	l.wake = make(chan struct{}, 1)
	l.onBlocked = onBlocked
}

func (l *replayLog) endInvocation() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.live = false
}

// positional takes the next node id of the invocation for a new kind/key entry. Must be called
// with mu held.
func (l *replayLog) positional(kind, key string) (int64, *replayLogEntry, bool) {
	l.latestNode++
	node := l.latestNode

	if int(node) <= len(l.entries) {
		existing := l.entries[node-1]

		if existing.kind != kind || existing.key != key {
			l.mismatches = append(l.mismatches, fmt.Sprintf("invocation %d: %s %q landed on node %d, which holds %s %q", l.invocation, kind, key, node, existing.kind, existing.key))
			return node, existing, false
		}

		return node, existing, true
	}

	entry := &replayLogEntry{kind: kind, key: key}
	l.entries = append(l.entries, entry)

	return node, entry, true
}

func (l *replayLog) completion(node int64, entry *replayLogEntry) *v1contracts.DurableTaskResponse {
	return &v1contracts.DurableTaskResponse{
		Message: &v1contracts.DurableTaskResponse_EntryCompleted{
			EntryCompleted: &v1contracts.DurableTaskEventLogEntryCompletedResponse{
				Ref: &v1contracts.DurableEventLogEntryRef{
					DurableTaskExternalId: l.externalId.String(),
					InvocationCount:       l.invocation,
					NodeId:                node,
					BranchId:              1,
				},
				Payload: entry.payload,
			},
		},
	}
}

// deliver releases a satisfied entry to the live invocation in satisfied order. Must be called
// with mu held.
func (l *replayLog) deliver(node int64, entry *replayLogEntry) {
	resp := l.completion(node, entry)

	if entry.order != l.maxSent+1 {
		l.held[entry.order] = resp
		return
	}

	l.push(resp)
	l.maxSent++

	for {
		next, ok := l.held[l.maxSent+1]
		if !ok {
			return
		}

		delete(l.held, l.maxSent+1)
		l.push(next)
		l.maxSent++
	}
}

func (l *replayLog) push(resp *v1contracts.DurableTaskResponse) {
	l.queue = append(l.queue, resp)

	select {
	case l.wake <- struct{}{}:
	default:
	}
}

// satisfy marks the entry satisfied, as a child completing or a sleep firing does.
func (l *replayLog) satisfy(node int64, payload []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.satisfyLocked(node, payload)
}

func (l *replayLog) satisfyLocked(node int64, payload []byte) {
	entry := l.entries[node-1]
	l.nextOrder++
	entry.satisfied = true
	entry.order = l.nextOrder
	entry.payload = payload

	if l.live {
		l.deliver(node, entry)
	}
}

func (l *replayLog) nodeOf(kind, key string) int64 {
	l.mu.Lock()
	defer l.mu.Unlock()

	for i, e := range l.entries {
		if e.kind == kind && e.key == key {
			return int64(i + 1)
		}
	}

	l.t.Fatalf("no %s %q entry in the log", kind, key)
	return 0
}

func (l *replayLog) evict() {
	l.push(&v1contracts.DurableTaskResponse{
		Message: &v1contracts.DurableTaskResponse_ServerEvict{
			ServerEvict: &v1contracts.DurableTaskServerEvictNotice{
				DurableTaskExternalId: l.externalId.String(),
				InvocationCount:       l.invocation,
				Reason:                "operator run is blocked waiting on durable events",
			},
		},
	})
}

// triggerStep is the engine's DAG step trigger; every child it creates completes at once.
func (l *replayLog) triggerStep(_ context.Context, actionId, _ string, _ int32, _ []uuid.UUID, isSkipped, isCancelled, _ bool) (*operator.DAGStepTriggerResult, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if node, ok := l.stepNode[actionId]; ok {
		l.latestNode = max(l.latestNode, node)
		entry := l.entries[node-1]

		if entry.satisfied {
			l.deliver(node, entry)
		}

		return &operator.DAGStepTriggerResult{NodeId: node, BranchId: 1, WorkflowRunExternalId: entry.runId, IsSatisfied: entry.satisfied}, nil
	}

	node, entry, ok := l.positional("RUN", actionId)
	if !ok {
		return nil, errors.New("step trigger landed on another entry")
	}

	entry.runId = uuid.New()
	l.stepNode[actionId] = node

	if isSkipped || isCancelled {
		l.satisfyLocked(node, mapToJson(map[string]any{"skipped": isSkipped, "cancelled": isCancelled}))
	} else {
		l.satisfyLocked(node, mapToJson(map[string]any{"random_number": 1}))
	}

	return &operator.DAGStepTriggerResult{NodeId: node, BranchId: 1, WorkflowRunExternalId: entry.runId}, nil
}

type replayChannel struct{ l *replayLog }

func (c *replayChannel) Send(_ context.Context, req *v1contracts.DurableTaskRequest) error {
	l := c.l

	l.mu.Lock()
	defer l.mu.Unlock()

	switch m := req.GetMessage().(type) {
	case *v1contracts.DurableTaskRequest_WaitFor:
		var parts []string
		for _, s := range m.WaitFor.GetWaitForConditions().GetSleepConditions() {
			parts = append(parts, "sleep "+s.GetSleepFor())
		}
		for _, e := range m.WaitFor.GetWaitForConditions().GetUserEventConditions() {
			parts = append(parts, "event "+e.GetUserEventKey())
		}
		sort.Strings(parts)

		node, entry, ok := l.positional("WAITFOR", strings.Join(parts, ","))
		if !ok {
			l.push(&v1contracts.DurableTaskResponse{
				Message: &v1contracts.DurableTaskResponse_Error{
					Error: &v1contracts.DurableTaskErrorResponse{
						ErrorType:    v1contracts.DurableTaskErrorType_DURABLE_TASK_ERROR_TYPE_NONDETERMINISM,
						ErrorMessage: "wait_for landed on another entry",
					},
				},
			})
			return nil
		}

		l.push(&v1contracts.DurableTaskResponse{
			Message: &v1contracts.DurableTaskResponse_WaitForAck{
				WaitForAck: &v1contracts.DurableTaskEventWaitForAckResponse{
					Ref: &v1contracts.DurableEventLogEntryRef{
						DurableTaskExternalId: l.externalId.String(),
						InvocationCount:       l.invocation,
						NodeId:                node,
						BranchId:              1,
					},
				},
			},
		})

		if entry.satisfied {
			l.deliver(node, entry)
		}
	case *v1contracts.DurableTaskRequest_WorkerStatus:
		if l.onBlocked != nil {
			onBlocked := l.onBlocked
			l.onBlocked = nil
			onBlocked()
		}
	}

	return nil
}

func (c *replayChannel) Recv(ctx context.Context) (*v1contracts.DurableTaskResponse, error) {
	l := c.l

	for {
		l.mu.Lock()
		if len(l.queue) > 0 {
			resp := l.queue[0]
			l.queue = l.queue[1:]
			l.mu.Unlock()

			return resp, nil
		}
		wake := l.wake
		l.mu.Unlock()

		select {
		case <-wake:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (c *replayChannel) ExpectEntry(int64, int64) error { return nil }

func (c *replayChannel) Close() error { return nil }

func sleepCondition(action sqlcv1.V1MatchConditionAction, duration string) *sqlcv1.V1StepMatchCondition {
	return &sqlcv1.V1StepMatchCondition{
		Kind:            sqlcv1.V1StepMatchConditionKindSLEEP,
		Action:          action,
		OrGroupID:       uuid.New(),
		ReadableDataKey: "sleep:" + duration,
		SleepDuration:   sqlchelpers.TextFromStr(duration),
	}
}

func userEventCondition(action sqlcv1.V1MatchConditionAction, key string) *sqlcv1.V1StepMatchCondition {
	return &sqlcv1.V1StepMatchCondition{
		Kind:            sqlcv1.V1StepMatchConditionKindUSEREVENT,
		Action:          action,
		OrGroupID:       uuid.New(),
		ReadableDataKey: key,
		EventKey:        sqlchelpers.TextFromStr(key),
	}
}

// conditionsWorkflow builds the Ruby SDK's TaskConditionWorkflow example in the given task order;
// the operator gets its steps ordered by step id, which is random per workflow version.
func conditionsWorkflow(order []string) []*task {
	byId := map[string]*task{}
	for i, id := range order {
		byId[id] = newTestTask(id, "wf:"+id, int32(i))
	}

	parents := map[string][]string{
		"wait_for_sleep":             {"start"},
		"skip_with_multiple_parents": {"start", "wait_for_sleep"},
		"skip_on_event":              {"start"},
		"left_branch":                {"wait_for_sleep"},
		"right_branch":               {"wait_for_sleep"},
		"wait_for_event":             {"start"},
		"sum":                        {"start", "wait_for_sleep", "wait_for_event", "skip_on_event", "left_branch", "right_branch"},
	}

	for id, ps := range parents {
		for _, p := range ps {
			byId[id].parents = append(byId[id].parents, byId[p])
		}
	}

	byId["wait_for_sleep"].stepConditions = []*sqlcv1.V1StepMatchCondition{
		sleepCondition(sqlcv1.V1MatchConditionActionQUEUE, "10s"),
	}
	byId["skip_on_event"].stepConditions = []*sqlcv1.V1StepMatchCondition{
		sleepCondition(sqlcv1.V1MatchConditionActionQUEUE, "30s"),
		userEventCondition(sqlcv1.V1MatchConditionActionSKIP, "skip_on_event:skip"),
	}

	tasks := make([]*task, 0, len(order))
	for _, id := range order {
		tasks = append(tasks, byId[id])
	}

	return tasks
}

// TestDag_ReplayAfterEvictionKeepsLogPositions runs the conditions workflow the way it runs when
// the dispatcher evicts it between its sleeps: evicted while waiting on the 10s sleep, replayed
// once it fires, evicted again while waiting on the 30s sleep and replayed to completion. Every
// replay has to re-issue the log's entries in their original positions. The orders are the step
// orders of eight CI jobs; only the first one failed there.
func TestDag_ReplayAfterEvictionKeepsLogPositions(t *testing.T) {
	orders := [][]string{
		{"start", "wait_for_sleep", "sum", "skip_with_multiple_parents", "wait_for_event", "right_branch", "left_branch", "skip_on_event"},
		{"start", "sum", "skip_on_event", "wait_for_sleep", "skip_with_multiple_parents", "wait_for_event", "right_branch", "left_branch"},
		{"sum", "skip_with_multiple_parents", "skip_on_event", "start", "wait_for_event", "wait_for_sleep", "left_branch", "right_branch"},
		{"skip_on_event", "start", "wait_for_sleep", "skip_with_multiple_parents", "sum", "left_branch", "right_branch", "wait_for_event"},
		{"left_branch", "wait_for_sleep", "skip_on_event", "start", "skip_with_multiple_parents", "wait_for_event", "right_branch", "sum"},
		{"left_branch", "skip_with_multiple_parents", "right_branch", "start", "skip_on_event", "sum", "wait_for_sleep", "wait_for_event"},
		{"skip_on_event", "right_branch", "left_branch", "start", "wait_for_sleep", "skip_with_multiple_parents", "wait_for_event", "sum"},
		{"sum", "wait_for_event", "skip_on_event", "start", "left_branch", "skip_with_multiple_parents", "right_branch", "wait_for_sleep"},
	}

	for _, order := range orders {
		t.Run(strings.Join(order, ","), func(t *testing.T) {
			shortenBlockedStatusReportInterval(t, 20*time.Millisecond)

			evaluator, err := internalcel.NewBoolExprEvaluator()
			require.NoError(t, err)

			l := newReplayLog(t)

			invoke := func(invocation int32, onBlocked func()) ([]*task, error) {
				t.Helper()

				l.startInvocation(invocation, onBlocked)
				defer l.endInvocation()

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				tasks := conditionsWorkflow(order)

				return tasks, dagDurableTask(ctx, tasks, nil, l.externalId, uuid.New(), invocation, "{}", &replayChannel{l: l}, evaluator.EvalBoolExpr, l.triggerStep)
			}

			_, err = invoke(1, l.evict)
			require.True(t, isDagEvictedErr(err), "invocation 1 should be evicted while blocked, got %v", err)

			l.satisfy(l.nodeOf("WAITFOR", "sleep 10s"), mapToJson(map[string]any{}))

			_, err = invoke(2, l.evict)
			require.Empty(t, l.mismatches)
			require.True(t, isDagEvictedErr(err), "invocation 2 should be evicted while blocked, got %v", err)

			l.satisfy(l.nodeOf("WAITFOR", "sleep 30s"), mapToJson(map[string]any{}))

			tasks, err := invoke(3, nil)
			require.Empty(t, l.mismatches)
			require.NoError(t, err)

			for _, tk := range tasks {
				require.True(t, tk.isCompleted, "%s did not complete", tk.readableId)
				require.False(t, tk.isSkipped, "%s was skipped", tk.readableId)
			}
		})
	}
}
