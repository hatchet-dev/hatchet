package dagoperator

import (
	"context"
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

// replayLogEntry is one row of the fake engine's durable event log.
type replayLogEntry struct {
	nodeId         int64
	kind           string // WAIT_FOR or RUN
	key            string // WAIT_FOR: the registered conditions; RUN: the child index
	satisfied      bool
	satisfiedOrder int64
	payload        []byte
	invocation     int32 // the invocation that created the entry
}

// replayEngine is the engine side of a durable task session with the engine's log semantics,
// enough of them to replay a DAG: a WAIT_FOR registration is matched to its entry by position
// (latest node id plus one) and checked against the stored conditions, a step trigger is
// matched by child index and moves the latest node id up to its entry, a restore resets the
// latest node id to zero, and completions reach the invocation in satisfied_order, each held
// until every lower order was sent and its own ack went out first. Acks and completions share
// one outbound queue so their relative order is exactly the dispatcher's; the run reads them
// through the real in-engine host (operatorsvc), which applies the ack-before-entry rule.
type replayEngine struct {
	t *testing.T

	mu                   sync.Mutex
	entries              map[int64]*replayLogEntry
	runsByChildIndex     map[int32]*replayLogEntry
	latestNodeId         int64
	latestSatisfiedOrder int64
	invocation           int32
	maxSentOrder         int64
	held                 map[int64]*v1contracts.DurableTaskResponse
	out                  chan *v1contracts.DurableTaskResponse
	createdInInvocation  int
	nondeterminism       []string

	externalId uuid.UUID
	acks       chan int64
	triggers   chan string

	// appends is the sequence of log appends the current invocation issued, registrations and
	// triggers alike, as "KIND@node:key".
	appends []string

	// holdAckKey names a WAIT_FOR whose ack is withheld until releaseHeldAck, so completions
	// can be delivered to the run while a registration is un-acked.
	holdAckKey     string
	heldAck        *replayLogEntry
	heldAckArrived chan struct{}

	// staleBefore lists WAIT_FOR keys whose completion from the previous invocation is delivered
	// to the next invocation before anything else, as a callback delayed across an eviction is:
	// the dispatcher sends a replaced invocation's completion without ordering, stamped with the
	// invocation that satisfied it.
	staleBefore []string

	// satisfyBeforeAck lists, per registered WAIT_FOR key, the keys whose matches fire between
	// the registration's commit and its ack: the registered entry itself may be among them
	// (its completion then reaches the host before the ack naming it, which holds it).
	satisfyBeforeAck map[string][]string
}

func newReplayEngine(t *testing.T) *replayEngine {
	return &replayEngine{
		t:                t,
		entries:          make(map[int64]*replayLogEntry),
		runsByChildIndex: make(map[int32]*replayLogEntry),
		held:             make(map[int64]*v1contracts.DurableTaskResponse),
		externalId:       uuid.New(),
		acks:             make(chan int64, 64),
		triggers:         make(chan string, 64),
		heldAckArrived:   make(chan struct{}, 1),
	}
}

// restore starts the next invocation the way RestoreEvictedTasks does: the invocation count
// goes up, the latest node id restarts at zero and nothing has been sent to the new invocation.
func (e *replayEngine) restore() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.invocation++
	e.latestNodeId = 0
	e.maxSentOrder = 0
	e.held = make(map[int64]*v1contracts.DurableTaskResponse)
	e.createdInInvocation = 0
	e.nondeterminism = nil
	e.appends = nil
	e.holdAckKey = ""
	e.heldAck = nil
	e.satisfyBeforeAck = nil
	e.staleBefore = nil
}

func replayWaitForKey(conditions *v1contracts.DurableEventListenerConditions) string {
	var parts []string

	for _, c := range conditions.GetSleepConditions() {
		parts = append(parts, fmt.Sprintf("sleep(%s:%s)", c.GetBase().GetReadableDataKey(), c.GetSleepFor()))
	}

	for _, c := range conditions.GetUserEventConditions() {
		parts = append(parts, fmt.Sprintf("event(%s:%s)", c.GetBase().GetReadableDataKey(), c.GetUserEventKey()))
	}

	sort.Strings(parts)

	return strings.Join(parts, " and ")
}

func (e *replayEngine) ref(nodeId int64) *v1contracts.DurableEventLogEntryRef {
	return &v1contracts.DurableEventLogEntryRef{
		DurableTaskExternalId: e.externalId.String(),
		InvocationCount:       e.invocation,
		NodeId:                nodeId,
		BranchId:              1,
	}
}

// handleWaitFor is the dispatcher's handleWaitFor over the fake log. Caller holds e.mu.
func (e *replayEngine) handleWaitFor(req *v1contracts.DurableTaskWaitForRequest) {
	key := replayWaitForKey(req.GetWaitForConditions())
	nodeId := e.latestNodeId + 1

	entry, exists := e.entries[nodeId]

	if exists && (entry.kind != "WAIT_FOR" || entry.key != key) {
		msg := fmt.Sprintf("non-determinism error at node %d:1 expected: %s(%s) received: wait_for(%s)", nodeId, strings.ToLower(entry.kind), entry.key, key)
		e.nondeterminism = append(e.nondeterminism, msg)
		e.out <- &v1contracts.DurableTaskResponse{
			Message: &v1contracts.DurableTaskResponse_Error{
				Error: &v1contracts.DurableTaskErrorResponse{
					Ref:          e.ref(nodeId),
					ErrorType:    v1contracts.DurableTaskErrorType_DURABLE_TASK_ERROR_TYPE_NONDETERMINISM,
					ErrorMessage: msg,
				},
			},
		}
		return
	}

	if !exists {
		entry = &replayLogEntry{nodeId: nodeId, kind: "WAIT_FOR", key: key, invocation: e.invocation}
		e.entries[nodeId] = entry
		e.createdInInvocation++
	}

	e.latestNodeId = nodeId
	e.appends = append(e.appends, fmt.Sprintf("WAIT_FOR@%d:%s", nodeId, key))

	if key == e.holdAckKey {
		e.heldAck = entry
		e.heldAckArrived <- struct{}{}
		return
	}

	for _, fired := range e.satisfyBeforeAck[key] {
		if firedEntry := e.waitForEntry(fired); firedEntry != nil && !firedEntry.satisfied {
			e.markSatisfied(firedEntry, mapToJson(map[string]any{"CREATE": fired}))
			e.deliverOrdered(firedEntry)
		}
	}

	e.ack(entry)
}

// ack sends a registration's ack, then its completion if the entry is already satisfied, as
// handleWaitFor does. Caller holds e.mu.
func (e *replayEngine) ack(entry *replayLogEntry) {
	e.out <- &v1contracts.DurableTaskResponse{
		Message: &v1contracts.DurableTaskResponse_WaitForAck{
			WaitForAck: &v1contracts.DurableTaskEventWaitForAckResponse{Ref: e.ref(entry.nodeId)},
		},
	}

	if entry.satisfied {
		e.deliverOrdered(entry)
	}

	e.acks <- entry.nodeId
}

// releaseHeldAck sends the ack withheld by holdAckKey.
func (e *replayEngine) releaseHeldAck() {
	e.mu.Lock()
	defer e.mu.Unlock()

	require.NotNil(e.t, e.heldAck, "no ack is being held")

	entry := e.heldAck
	e.heldAck = nil
	e.holdAckKey = ""
	e.ack(entry)
}

// triggerStep is TriggerDAGStep over the fake log: a step's RUN entry is keyed by child index
// and a replayed trigger returns the existing entry wherever it sits, moving the latest node
// id up to it. A skipped or cancelled step's entry is satisfied at creation; any other step
// completes at once with a result, which keeps every completion's satisfied_order deterministic.
func (e *replayEngine) triggerStep(_ context.Context, actionId, _ string, childIndex int32, _ []uuid.UUID, isSkipped, isCancelled, _ bool) (*operator.DAGStepTriggerResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	entry, exists := e.runsByChildIndex[childIndex]

	if !exists {
		entry = &replayLogEntry{nodeId: e.latestNodeId + 1, kind: "RUN", key: actionId, invocation: e.invocation}
		e.entries[entry.nodeId] = entry
		e.runsByChildIndex[childIndex] = entry
		e.createdInInvocation++

		switch {
		case isCancelled:
			e.markSatisfied(entry, mapToJson(map[string]any{"cancelled": true}))
		case isSkipped:
			e.markSatisfied(entry, mapToJson(map[string]any{"skipped": true}))
		default:
			e.markSatisfied(entry, mapToJson(map[string]any{"step": actionId}))
		}
	}

	if entry.nodeId > e.latestNodeId {
		e.latestNodeId = entry.nodeId
	}

	e.appends = append(e.appends, fmt.Sprintf("RUN@%d:%s", entry.nodeId, actionId))
	e.triggers <- actionId

	if entry.satisfied {
		e.deliverOrdered(entry)
	}

	return &operator.DAGStepTriggerResult{
		NodeId:                entry.nodeId,
		BranchId:              1,
		WorkflowRunExternalId: uuid.New(),
		IsSatisfied:           entry.satisfied,
		ResultPayload:         entry.payload,
		ReExecuted:            exists,
	}, nil
}

// markSatisfied stamps the next satisfied_order on an entry. Caller holds e.mu.
func (e *replayEngine) markSatisfied(entry *replayLogEntry, payload []byte) {
	e.latestSatisfiedOrder++
	entry.satisfied = true
	entry.satisfiedOrder = e.latestSatisfiedOrder
	entry.payload = payload
}

// satisfy fires the match of a WAIT_FOR entry registered under key, as the engine does when its
// sleep elapses or its event arrives, and delivers the completion to the current invocation.
func (e *replayEngine) satisfy(key string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	entry := e.waitForEntry(key)
	require.NotNil(e.t, entry, "no WAIT_FOR entry registered for %q", key)
	require.False(e.t, entry.satisfied, "WAIT_FOR entry for %q already satisfied", key)

	e.markSatisfied(entry, mapToJson(map[string]any{"CREATE": key}))
	e.deliverOrdered(entry)
}

// waitForEntry returns the lowest-node WAIT_FOR entry registered under key. Caller holds e.mu.
func (e *replayEngine) waitForEntry(key string) *replayLogEntry {
	var found *replayLogEntry

	for _, entry := range e.entries {
		if entry.kind != "WAIT_FOR" || entry.key != key {
			continue
		}
		if found == nil || entry.nodeId < found.nodeId {
			found = entry
		}
	}

	return found
}

// deliverOrdered is the dispatcher's deliverOrdered: a completion goes out when every lower
// satisfied_order has, otherwise it is held. Caller holds e.mu.
func (e *replayEngine) deliverOrdered(entry *replayLogEntry) {
	resp := &v1contracts.DurableTaskResponse{
		Message: &v1contracts.DurableTaskResponse_EntryCompleted{
			EntryCompleted: &v1contracts.DurableTaskEventLogEntryCompletedResponse{
				Ref:     e.ref(entry.nodeId),
				Payload: entry.payload,
			},
		},
	}

	switch {
	case entry.satisfiedOrder <= e.maxSentOrder:
		e.out <- resp
	case entry.satisfiedOrder == e.maxSentOrder+1:
		e.out <- resp
		e.maxSentOrder++

		for {
			next, ok := e.held[e.maxSentOrder+1]
			if !ok {
				break
			}
			delete(e.held, e.maxSentOrder+1)
			e.out <- next
			e.maxSentOrder++
		}
	default:
		e.held[entry.satisfiedOrder] = resp
	}
}

type replayInvocation struct {
	errCh   chan error
	cancel  context.CancelFunc
	cleanup func()
}

// run starts an invocation of the DAG against the engine through the real in-engine host
// (operatorsvc over the dispatcher double): requests are handled on the engine's goroutine and
// responses go through the host's channel, so its ack-before-entry rule and its ordering are
// what the run sees.
func (e *replayEngine) run(t *testing.T, tasks []*task) *replayInvocation {
	t.Helper()

	evaluator, err := internalcel.NewBoolExprEvaluator()
	require.NoError(t, err)

	e.mu.Lock()
	e.out = make(chan *v1contracts.DurableTaskResponse, 1024)
	out := e.out
	invocation := e.invocation

	for _, key := range e.staleBefore {
		entry := e.waitForEntry(key)
		require.NotNil(t, entry, "no WAIT_FOR entry registered for %q", key)

		out <- &v1contracts.DurableTaskResponse{
			Message: &v1contracts.DurableTaskResponse_EntryCompleted{
				EntryCompleted: &v1contracts.DurableTaskEventLogEntryCompletedResponse{
					Ref: &v1contracts.DurableEventLogEntryRef{
						DurableTaskExternalId: e.externalId.String(),
						InvocationCount:       invocation - 1,
						NodeId:                entry.nodeId,
						BranchId:              1,
					},
					Payload: entry.payload,
				},
			},
		}
	}
	e.mu.Unlock()

	ch, inv, _, workerId := engineChannel(t, engineInvocation{externalId: e.externalId, invocation: invocation})

	stop := make(chan struct{})
	forwarderDone := make(chan struct{})

	// the dispatcher double closes inv.Responses when the channel closes, so the forwarder must
	// be gone before cleanup closes the channel
	go func() {
		defer close(forwarderDone)

		for {
			select {
			case <-stop:
				return
			case resp := <-out:
				select {
				case inv.Responses <- resp:
				case <-stop:
					return
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case <-stop:
				return
			case req, ok := <-inv.Requests:
				if !ok {
					return
				}

				if waitFor := req.GetWaitFor(); waitFor != nil {
					e.mu.Lock()
					e.handleWaitFor(waitFor)
					e.mu.Unlock()
				}
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	errCh := make(chan error, 1)

	go func() {
		errCh <- dagDurableTask(ctx, tasks, nil, e.externalId, workerId, invocation, "{}", ch, evaluator.EvalBoolExpr, e.triggerStep)
	}()

	return &replayInvocation{
		errCh:  errCh,
		cancel: cancel,
		cleanup: func() {
			cancel()
			close(stop)
			<-forwarderDone
			_ = ch.Close()
		},
	}
}

func (e *replayEngine) waitAcks(t *testing.T, n int) {
	t.Helper()

	for range n {
		select {
		case <-e.acks:
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for a wait registration ack")
		}
	}
}

func (e *replayEngine) waitTrigger(t *testing.T, actionId string) {
	t.Helper()

	select {
	case got := <-e.triggers:
		require.Equal(t, actionId, got, "unexpected step triggered")
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s to be triggered", actionId)
	}
}

func (e *replayEngine) drainTriggers() {
	for {
		select {
		case <-e.triggers:
		default:
			return
		}
	}
}

func (e *replayEngine) appendsSnapshot() []string {
	e.mu.Lock()
	defer e.mu.Unlock()

	return append([]string{}, e.appends...)
}

func (e *replayEngine) snapshot() map[int64]string {
	e.mu.Lock()
	defer e.mu.Unlock()

	out := make(map[int64]string, len(e.entries))
	for nodeId, entry := range e.entries {
		out[nodeId] = entry.kind + ":" + entry.key
	}

	return out
}

// requireReplayComplete asserts that a replay neither hit a nondeterminism error nor moved or
// re-created an entry the first invocation wrote (a new entry may only be the RUN of a step the
// first invocation never reached), and that every satisfied completion reached the invocation
// (an orphaned entry's order would leave every higher order held forever).
func (e *replayEngine) requireReplayComplete(t *testing.T, before map[int64]string) {
	t.Helper()

	e.mu.Lock()
	defer e.mu.Unlock()

	require.Empty(t, e.nondeterminism, "the replay reported nondeterminism")

	for nodeId, entry := range e.entries {
		if want, ok := before[nodeId]; ok {
			require.Equal(t, want, entry.kind+":"+entry.key, "entry at node %d changed on replay", nodeId)
			require.NotEqual(t, e.invocation, entry.invocation, "entry at node %d was re-created on replay", nodeId)
			continue
		}

		require.Equal(t, "RUN", entry.kind, "the replay created a new %s entry at node %d (%s) instead of reusing the original one", entry.kind, nodeId, entry.key)
	}

	for nodeId := range before {
		require.Contains(t, e.entries, nodeId)
	}

	require.Empty(t, e.held, "completions were held waiting for a satisfied_order that never arrived")
	require.Equal(t, e.latestSatisfiedOrder, e.maxSentOrder, "not every satisfied completion was delivered to the replay")
}

const (
	replaySleepKey      = "wait_for_sleep.sleep"
	replaySkipKey       = "skip_on_event.skip"
	replaySkipWaitKey   = "skip_on_event.sleep"
	replayOrGroupsFirst = "wait_for_or_groups.first"
	replayOrGroupsSecnd = "wait_for_or_groups.second"
	replayOrGroupsEvent = "wait_for_or_groups.event"
	replayEventSleepKey = "wait_for_event.sleep"
	replayEventKey      = "wait_for_event.event"
)

// conditionStepNames are the four steps of the conditions example that register watches when
// start completes; a workflow version lists them in an arbitrary but fixed order.
var conditionStepNames = []string{"wait_for_sleep", "skip_on_event", "wait_for_or_groups", "wait_for_event"}

// newReplayDAG builds the conditions example's TaskConditionWorkflow shape with its
// condition-bearing steps in stepOrder: start, then the four steps, then sum.
func newReplayDAG(stepOrder []string) (tasks []*task, byName map[string]*task) {
	byName = make(map[string]*task)

	start := newTestTask("start", "action-start", 0)
	tasks = append(tasks, start)
	byName["start"] = start

	for i, name := range stepOrder {
		t := newTestTask(name, "action-"+name, int32(i+1), start) // nolint:gosec // small test index
		byName[name] = t
		tasks = append(tasks, t)

		switch name {
		case "wait_for_sleep":
			t.stepConditions = []*sqlcv1.V1StepMatchCondition{
				{Kind: sqlcv1.V1StepMatchConditionKindSLEEP, Action: sqlcv1.V1MatchConditionActionQUEUE, OrGroupID: uuid.New(), ReadableDataKey: replaySleepKey, SleepDuration: sqlchelpers.TextFromStr("10s")},
			}
		case "skip_on_event":
			t.stepConditions = []*sqlcv1.V1StepMatchCondition{
				{Kind: sqlcv1.V1StepMatchConditionKindUSEREVENT, Action: sqlcv1.V1MatchConditionActionSKIP, OrGroupID: uuid.New(), ReadableDataKey: replaySkipKey, EventKey: sqlchelpers.TextFromStr("skip_on_event:skip")},
				{Kind: sqlcv1.V1StepMatchConditionKindSLEEP, Action: sqlcv1.V1MatchConditionActionQUEUE, OrGroupID: uuid.New(), ReadableDataKey: replaySkipWaitKey, SleepDuration: sqlchelpers.TextFromStr("30s")},
			}
		case "wait_for_or_groups":
			second := uuid.New()
			t.stepConditions = []*sqlcv1.V1StepMatchCondition{
				{Kind: sqlcv1.V1StepMatchConditionKindSLEEP, Action: sqlcv1.V1MatchConditionActionQUEUE, OrGroupID: uuid.New(), ReadableDataKey: replayOrGroupsFirst, SleepDuration: sqlchelpers.TextFromStr("30s")},
				{Kind: sqlcv1.V1StepMatchConditionKindSLEEP, Action: sqlcv1.V1MatchConditionActionQUEUE, OrGroupID: second, ReadableDataKey: replayOrGroupsSecnd, SleepDuration: sqlchelpers.TextFromStr("30s")},
				{Kind: sqlcv1.V1StepMatchConditionKindUSEREVENT, Action: sqlcv1.V1MatchConditionActionQUEUE, OrGroupID: second, ReadableDataKey: replayOrGroupsEvent, EventKey: sqlchelpers.TextFromStr("payment:processed")},
			}
		case "wait_for_event":
			group := uuid.New()
			t.stepConditions = []*sqlcv1.V1StepMatchCondition{
				{Kind: sqlcv1.V1StepMatchConditionKindSLEEP, Action: sqlcv1.V1MatchConditionActionQUEUE, OrGroupID: group, ReadableDataKey: replayEventSleepKey, SleepDuration: sqlchelpers.TextFromStr("1m")},
				{Kind: sqlcv1.V1StepMatchConditionKindUSEREVENT, Action: sqlcv1.V1MatchConditionActionQUEUE, OrGroupID: group, ReadableDataKey: replayEventKey, EventKey: sqlchelpers.TextFromStr("wait_for_event:start")},
			}
		}
	}

	parents := make([]*task, 0, len(stepOrder)+1)
	parents = append(parents, start)
	for _, name := range stepOrder {
		parents = append(parents, byName[name])
	}

	sum := newTestTask("sum", "action-sum", int32(len(tasks)), parents...) // nolint:gosec // small test index
	byName["sum"] = sum
	tasks = append(tasks, sum)

	return tasks, byName
}

// registeredKey is what the DAG registers for a step's watch, as the engine's log stores it.
func registeredKey(t *task, action sqlcv1.V1MatchConditionAction) string {
	conditions := &v1contracts.DurableEventListenerConditions{}

	for _, c := range t.stepConditions {
		if c.Action != action {
			continue
		}
		switch c.Kind {
		case sqlcv1.V1StepMatchConditionKindSLEEP:
			conditions.SleepConditions = append(conditions.SleepConditions, &v1contracts.SleepMatchCondition{
				Base:     &v1contracts.BaseMatchCondition{ReadableDataKey: c.ReadableDataKey},
				SleepFor: c.SleepDuration.String,
			})
		case sqlcv1.V1StepMatchConditionKindUSEREVENT:
			conditions.UserEventConditions = append(conditions.UserEventConditions, &v1contracts.UserEventMatchCondition{
				Base:         &v1contracts.BaseMatchCondition{ReadableDataKey: c.ReadableDataKey},
				UserEventKey: c.EventKey.String,
			})
		case sqlcv1.V1StepMatchConditionKindPARENTOVERRIDE:
		}
	}

	return replayWaitForKey(conditions)
}

func permutations(items []string) [][]string {
	if len(items) <= 1 {
		return [][]string{append([]string{}, items...)}
	}

	var out [][]string

	for i, item := range items {
		rest := make([]string, 0, len(items)-1)
		rest = append(rest, items[:i]...)
		rest = append(rest, items[i+1:]...)

		for _, p := range permutations(rest) {
			out = append(out, append([]string{item}, p...))
		}
	}

	return out
}

// TestDag_ReplayReusesEveryEntryRegardlessOfStepOrder is the conditions example's test_waits
// under the DAG operator: start completes, the four condition-bearing steps register their
// watches, the skip event fires first, the other watches fire later. The run is then replayed
// into a new invocation, for every order the workflow version can list the steps in and for
// an eviction at each point of the run. A replay must find every registration at the node the
// first invocation wrote it to, reuse every step entry and receive every satisfied completion.
//
// Before the fix, a replay honoured the skip completion as soon as the engine re-delivered it,
// which is right after the skip watch's own registration is acked, so when the workflow version
// listed skip_on_event before another waiting step the skipped step's trigger moved the log's
// latest node id past the original registrations and the next WAIT_FOR landed on a node that
// held a RUN (a nondeterminism error) or on an empty node (a fresh entry, orphaning the
// original one and holding every later completion forever).
func TestDag_ReplayReusesEveryEntryRegardlessOfStepOrder(t *testing.T) {
	// evictAfter names the step whose trigger the first invocation is evicted after; "" runs the
	// first invocation to completion before the replay. "stale" additionally delivers the
	// previous invocation's completion of skip_on_event's wait to the replay before anything
	// else, as a callback delayed across the eviction is delivered.
	for _, variant := range []string{"", "skip_on_event", "wait_for_event", "stale"} {
		evictAfter := variant
		if variant == "stale" {
			evictAfter = "wait_for_event"
		}

		for _, order := range permutations(conditionStepNames) {
			name := strings.Join(order, ",")
			if variant != "" {
				name += "/evicted-after-" + evictAfter
			}
			if variant == "stale" {
				name += "/stale-completion"
			}

			t.Run(name, func(t *testing.T) {
				engine := newReplayEngine(t)
				engine.restore()

				tasks, byName := newReplayDAG(order)

				first := engine.run(t, tasks)
				defer first.cleanup()

				engine.waitTrigger(t, "action-start")
				engine.waitAcks(t, 5)

				// The skip event arrives first, then wait_for_event's event, then the sleeps
				// elapse: wait_for_sleep's (10s), then wait_for_or_groups' and skip_on_event's
				// (both 30s), the same order test_waits produces.
				fire := []struct {
					key     string
					trigger string
				}{
					{registeredKey(byName["skip_on_event"], sqlcv1.V1MatchConditionActionSKIP), "action-skip_on_event"},
					{registeredKey(byName["wait_for_event"], sqlcv1.V1MatchConditionActionQUEUE), "action-wait_for_event"},
					{registeredKey(byName["wait_for_sleep"], sqlcv1.V1MatchConditionActionQUEUE), "action-wait_for_sleep"},
					{registeredKey(byName["wait_for_or_groups"], sqlcv1.V1MatchConditionActionQUEUE), "action-wait_for_or_groups"},
					{registeredKey(byName["skip_on_event"], sqlcv1.V1MatchConditionActionQUEUE), ""},
				}

				evicted := false

				for _, f := range fire {
					engine.satisfy(f.key)
					if f.trigger != "" {
						engine.waitTrigger(t, f.trigger)
					}

					if f.trigger == "action-"+evictAfter {
						if variant == "stale" {
							// the sleep of skip_on_event's wait elapses just before the eviction
							engine.satisfy(registeredKey(byName["skip_on_event"], sqlcv1.V1MatchConditionActionQUEUE))
						}

						first.cancel()
						require.Error(t, waitInvocation(t, first), "the evicted invocation should end with the context error")
						evicted = true
						break
					}
				}

				if !evicted {
					engine.waitTrigger(t, "action-sum")
					require.NoError(t, waitInvocation(t, first))
					require.True(t, byName["skip_on_event"].isSkipped)
					require.True(t, byName["sum"].isCompleted)
				}

				before := engine.snapshot()
				firstAppends := engine.appendsSnapshot()
				engine.drainTriggers()

				// The registration wave is not delayed by held completions: every watch of the
				// four ready steps is registered right after start completes, before any step is
				// triggered, as it was before the fix.
				require.Len(t, firstAppends, len(firstAppends))
				require.True(t, strings.HasPrefix(firstAppends[0], "RUN@1:action-start"))
				for i := 1; i <= 5; i++ {
					require.True(t, strings.HasPrefix(firstAppends[i], fmt.Sprintf("WAIT_FOR@%d:", i+1)), "append %d of the first invocation is %q, expected the registration wave", i, firstAppends[i])
				}

				engine.restore()

				if variant == "stale" {
					// skip_on_event's wait was satisfied in the first invocation; its completion
					// arrives late, addressed to that invocation
					engine.mu.Lock()
					engine.staleBefore = []string{registeredKey(byName["skip_on_event"], sqlcv1.V1MatchConditionActionQUEUE)}
					engine.mu.Unlock()
				}

				replayTasks, replayByName := newReplayDAG(order)

				replay := engine.run(t, replayTasks)
				defer replay.cleanup()

				if evicted {
					// The replay re-registers every watch at its original node, then the
					// remaining matches fire as they did not get to in the first invocation.
					engine.waitAcks(t, 5)

					engine.mu.Lock()
					require.Empty(t, engine.nondeterminism)
					require.Equal(t, 0, engine.createdInInvocation, "the replay re-registered a watch at a new node")
					engine.mu.Unlock()

					for _, f := range fire {
						engine.mu.Lock()
						alreadySatisfied := engine.waitForEntry(f.key).satisfied
						engine.mu.Unlock()

						if !alreadySatisfied {
							engine.satisfy(f.key)
						}
					}
				}

				require.NoError(t, waitInvocation(t, replay))
				engine.requireReplayComplete(t, before)

				replayAppends := engine.appendsSnapshot()
				if evicted {
					require.GreaterOrEqual(t, len(replayAppends), len(firstAppends))
					require.Equal(t, firstAppends, replayAppends[:len(firstAppends)], "the replay appended the log in a different order than the first invocation")
				} else {
					require.Equal(t, firstAppends, replayAppends, "the replay appended the log in a different order than the first invocation")
				}

				require.True(t, replayByName["skip_on_event"].isSkipped)
				require.True(t, replayByName["skip_on_event"].isWaiting, "skip_on_event's wait watch is registered on replay as it was originally")
				require.True(t, replayByName["sum"].isCompleted)

				for _, name := range conditionStepNames {
					require.True(t, replayByName[name].isCompleted, "%s did not complete on replay", name)
				}
			})
		}
	}
}

func waitInvocation(t *testing.T, inv *replayInvocation) error {
	t.Helper()

	select {
	case err := <-inv.errCh:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the invocation to finish")
		return nil
	}
}

// TestDag_ReplayAppliesOneCompletionPerPass pins the rule that a completion's consequences,
// including the registrations of the steps it makes ready, are appended before the next
// completion is applied: two completions queued behind one un-acked registration must not be
// applied together. Applying them together would trigger both steps before the first step's
// child registered its watch, a layout that depends on how many completions happened to be
// queued, and that a replay (which delivers them one by one as the entries are re-registered)
// would not reproduce.
func TestDag_ReplayAppliesOneCompletionPerPass(t *testing.T) {
	newDAG := func() (tasks []*task, byName map[string]*task) {
		byName = make(map[string]*task)
		start := newTestTask("start", "action-start", 0)
		// a waits on a sleep and is skipped by its skip watch, which completes it at its
		// trigger; c has start as a second parent so it is not skipped along with a and
		// registers its own wait
		a := newTestTask("a", "action-a", 1, start)
		b := newTestTask("b", "action-b", 2, start)
		x := newTestTask("x", "action-x", 3, start)
		c := newTestTask("c", "action-c", 4, start, a)

		a.stepConditions = []*sqlcv1.V1StepMatchCondition{
			{Kind: sqlcv1.V1StepMatchConditionKindSLEEP, Action: sqlcv1.V1MatchConditionActionSKIP, OrGroupID: uuid.New(), ReadableDataKey: "a.skip", SleepDuration: sqlchelpers.TextFromStr("10s")},
		}

		for name, task := range map[string]*task{"a": a, "b": b, "x": x, "c": c} {
			task.stepConditions = append(task.stepConditions, &sqlcv1.V1StepMatchCondition{
				Kind: sqlcv1.V1StepMatchConditionKindSLEEP, Action: sqlcv1.V1MatchConditionActionQUEUE, OrGroupID: uuid.New(), ReadableDataKey: name + ".sleep", SleepDuration: sqlchelpers.TextFromStr("10s"),
			})
		}

		sum := newTestTask("sum", "action-sum", 5, a, b, c, x)

		for _, task := range []*task{start, a, b, x, c, sum} {
			byName[task.readableId] = task
		}

		return []*task{start, a, b, x, c, sum}, byName
	}

	engine := newReplayEngine(t)
	engine.restore()

	tasks, byName := newDAG()
	waitKey := func(name string) string { return registeredKey(byName[name], sqlcv1.V1MatchConditionActionQUEUE) }
	skipKey := registeredKey(byName["a"], sqlcv1.V1MatchConditionActionSKIP)

	engine.mu.Lock()
	engine.holdAckKey = waitKey("x")
	engine.mu.Unlock()

	first := engine.run(t, tasks)
	defer first.cleanup()

	engine.waitTrigger(t, "action-start")
	engine.waitAcks(t, 3) // a's skip watch, a's wait, b's wait

	select {
	case <-engine.heldAckArrived:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for x's registration")
	}

	// a's skip and b's wait fire while x's registration is un-acked, so both completions
	// reach the run before it can apply either
	engine.satisfy(skipKey)
	engine.satisfy(waitKey("b"))
	engine.releaseHeldAck()
	engine.waitAcks(t, 1) // x

	engine.waitTrigger(t, "action-a")
	engine.waitAcks(t, 1) // c, made ready by a's skip and registered before b is applied
	engine.waitTrigger(t, "action-b")

	engine.satisfy(waitKey("c"))
	engine.waitTrigger(t, "action-c")
	engine.satisfy(waitKey("x"))
	engine.waitTrigger(t, "action-x")
	engine.waitTrigger(t, "action-sum")
	require.NoError(t, waitInvocation(t, first))
	require.True(t, byName["a"].isSkipped)
	require.True(t, byName["c"].isCompleted)
	require.False(t, byName["c"].isSkipped)

	firstAppends := engine.appendsSnapshot()
	require.Equal(t, []string{
		"RUN@1:action-start",
		"WAIT_FOR@2:" + skipKey,
		"WAIT_FOR@3:" + waitKey("a"),
		"WAIT_FOR@4:" + waitKey("b"),
		"WAIT_FOR@5:" + waitKey("x"),
		"RUN@6:action-a",
		"WAIT_FOR@7:" + waitKey("c"),
		"RUN@8:action-b",
		"RUN@9:action-c",
		"RUN@10:action-x",
		"RUN@11:action-sum",
	}, firstAppends)

	before := engine.snapshot()
	engine.drainTriggers()
	engine.restore()

	replayTasks, _ := newDAG()
	replay := engine.run(t, replayTasks)
	defer replay.cleanup()

	require.NoError(t, waitInvocation(t, replay))
	engine.requireReplayComplete(t, before)
	require.Equal(t, firstAppends, engine.appendsSnapshot())
}

// TestDag_ReplayFollowsDeliveryOrderAcrossHeldCompletion covers the host's part of the
// contract: a completion the host holds for its ack (its match fired between the registration's
// commit and the ack) must not be overtaken by a later completion whose entry is already
// acknowledged, because the first invocation applies them in the host's order and the replay,
// where both entries are acknowledged before either completion is re-delivered, applies them in
// the dispatcher's order. A host that let the later one through made the first invocation skip
// `s` and register `c`'s wait before triggering `o`, while the replay triggered `o` first.
func TestDag_ReplayFollowsDeliveryOrderAcrossHeldCompletion(t *testing.T) {
	newDAG := func() (tasks []*task, byName map[string]*task) {
		byName = make(map[string]*task)
		start := newTestTask("start", "action-start", 0)
		s := newTestTask("s", "action-s", 1, start)
		o := newTestTask("o", "action-o", 2, start)
		c := newTestTask("c", "action-c", 3, start, s)

		s.stepConditions = []*sqlcv1.V1StepMatchCondition{
			{Kind: sqlcv1.V1StepMatchConditionKindUSEREVENT, Action: sqlcv1.V1MatchConditionActionSKIP, OrGroupID: uuid.New(), ReadableDataKey: "s.skip", EventKey: sqlchelpers.TextFromStr("s:skip")},
			{Kind: sqlcv1.V1StepMatchConditionKindSLEEP, Action: sqlcv1.V1MatchConditionActionQUEUE, OrGroupID: uuid.New(), ReadableDataKey: "s.sleep", SleepDuration: sqlchelpers.TextFromStr("10s")},
		}
		o.stepConditions = []*sqlcv1.V1StepMatchCondition{
			{Kind: sqlcv1.V1StepMatchConditionKindSLEEP, Action: sqlcv1.V1MatchConditionActionQUEUE, OrGroupID: uuid.New(), ReadableDataKey: "o.sleep", SleepDuration: sqlchelpers.TextFromStr("1s")},
		}
		c.stepConditions = []*sqlcv1.V1StepMatchCondition{
			{Kind: sqlcv1.V1StepMatchConditionKindSLEEP, Action: sqlcv1.V1MatchConditionActionQUEUE, OrGroupID: uuid.New(), ReadableDataKey: "c.sleep", SleepDuration: sqlchelpers.TextFromStr("10s")},
		}

		sum := newTestTask("sum", "action-sum", 4, s, o, c)

		for _, task := range []*task{start, s, o, c, sum} {
			byName[task.readableId] = task
		}

		return []*task{start, s, o, c, sum}, byName
	}

	engine := newReplayEngine(t)
	engine.restore()

	tasks, byName := newDAG()
	waitKey := func(name string) string { return registeredKey(byName[name], sqlcv1.V1MatchConditionActionQUEUE) }
	skipKey := registeredKey(byName["s"], sqlcv1.V1MatchConditionActionSKIP)

	// o's sleep elapses and s's skip event arrives while o's registration is between its commit
	// and its ack: the dispatcher delivers o's completion (order 2) then s's skip (order 3)
	engine.mu.Lock()
	engine.satisfyBeforeAck = map[string][]string{waitKey("o"): {waitKey("o"), skipKey}}
	engine.mu.Unlock()

	first := engine.run(t, tasks)
	defer first.cleanup()

	engine.waitTrigger(t, "action-start")
	engine.waitAcks(t, 3) // s's skip watch, s's wait, o's wait
	engine.waitTrigger(t, "action-o")
	engine.waitTrigger(t, "action-s")
	engine.waitAcks(t, 1) // c's wait
	engine.satisfy(waitKey("c"))
	engine.waitTrigger(t, "action-c")
	engine.satisfy(waitKey("s"))
	engine.waitTrigger(t, "action-sum")
	require.NoError(t, waitInvocation(t, first))
	require.True(t, byName["s"].isSkipped)

	firstAppends := engine.appendsSnapshot()
	require.Equal(t, []string{
		"RUN@1:action-start",
		"WAIT_FOR@2:" + skipKey,
		"WAIT_FOR@3:" + waitKey("s"),
		"WAIT_FOR@4:" + waitKey("o"),
		"RUN@5:action-o",
		"RUN@6:action-s",
		"WAIT_FOR@7:" + waitKey("c"),
		"RUN@8:action-c",
		"RUN@9:action-sum",
	}, firstAppends)

	before := engine.snapshot()
	engine.drainTriggers()
	engine.restore()

	replayTasks, _ := newDAG()
	replay := engine.run(t, replayTasks)
	defer replay.cleanup()

	require.NoError(t, waitInvocation(t, replay))
	engine.requireReplayComplete(t, before)
	require.Equal(t, firstAppends, engine.appendsSnapshot())
}
