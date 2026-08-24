package dagoperator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

type dagCancelledError struct {
	taskActionId string
}

func (e *dagCancelledError) Error() string {
	return fmt.Sprintf("task %q was cancelled", e.taskActionId)
}

func isDagCancelledErr(err error) bool {
	var e *dagCancelledError
	return errors.As(err, &e)
}

// dagChildFailedError signals that the DAG reached a terminal outcome because a child task failed.
// It is distinct from operational errors (failing to build the DAG, trigger a step, etc.): a child
// failure is deterministic under replay, so retrying the orchestrator can never change the outcome.
type dagChildFailedError struct {
	taskActionId string
	errorMessage string
}

func (e *dagChildFailedError) Error() string {
	return fmt.Sprintf("child task %q failed: %s", e.taskActionId, e.errorMessage)
}

func isDagChildFailedErr(err error) bool {
	var e *dagChildFailedError
	return errors.As(err, &e)
}

type dag struct {
	requestCh    chan<- *v1contracts.DurableTaskRequest
	responseCh   <-chan *v1contracts.DurableTaskResponse
	evalBoolExpr func(ctx context.Context, expr string, vars map[string]interface{}) (bool, error)
	triggerStep  func(ctx context.Context, actionId, workflowName string, childIndex int32, parentTaskRunIds []uuid.UUID, isSkipped, isCancelled, parentReExecuted bool) (*operator.DAGStepTriggerResult, error)

	tasks []*task

	// onFailureTask has no parents to gate on (see evaluateOnFailure) and is nil if the workflow
	// has none; once resolved it's appended to tasks so isDone/end-of-run scanning cover it too.
	onFailureTask *task

	pendingTasks    []*task
	externalId      uuid.UUID
	invocationCount int32
	input           string
	err             error

	pendingWaitAcks []*pendingWaitAck

	// pendingEntryCompletions buffers EntryCompleted refs that raced ahead of their WaitForAck
	// (its condition is committed to the DB before the ack is sent); rechecked on each ack.
	pendingEntryCompletions []pendingEntryCompletion

	// responses received while blocked sending on requestCh (see dag.send); drained by the
	// main loop before it blocks on responseCh
	queuedResponses []*v1contracts.DurableTaskResponse

	// cache of the result of each parent override condition, evaluated once when the
	// referenced parent completes instead of repeatedly on every readiness check
	conditionMatches map[*sqlcv1.V1StepMatchCondition]bool
}

type conditionKind int

const (
	conditionKindWait conditionKind = iota
	conditionKindSkip
	conditionKindCancel
)

// pendingEntryCompletion is a buffered EntryCompleted that didn't match any known node/branch id
// at the time it arrived.
type pendingEntryCompletion struct {
	nodeId       int64
	branchId     int64
	isFailure    bool
	errorMessage string
	payload      []byte
}

type pendingWaitAck struct {
	task *task
	kind conditionKind
}

type task struct {
	id           uuid.UUID
	actionId     string
	workflowName string
	readableId   string
	index        int32
	parents      []*task
	isCompleted  bool
	isFailed     bool
	isCancelled  bool
	isTriggered  bool
	isSkipped    bool
	errorMessage string
	output       map[string]interface{}

	// reExecuted is true when the task actually ran this invocation rather than being satisfied
	// from the log; children of a re-executed parent must re-run during a replay
	reExecuted bool

	isWaiting       bool
	isWaitSatisfied bool
	waitNodeId      int64
	waitBranchId    int64

	skipWatchRegistered bool
	skipWatchFired      bool
	skipWatchNodeId     int64
	skipWatchBranchId   int64

	cancelWatchRegistered bool
	cancelWatchFired      bool
	cancelWatchNodeId     int64
	cancelWatchBranchId   int64

	stepConditions []*sqlcv1.V1StepMatchCondition

	nodeId                int64
	branchId              int64
	workflowRunExternalId *uuid.UUID
}

func dagDurableTask(
	ctx context.Context,
	tasks []*task,
	onFailureTask *task,
	externalId uuid.UUID,
	invocationCount int32,
	input string,
	requestCh chan<- *v1contracts.DurableTaskRequest,
	responseCh <-chan *v1contracts.DurableTaskResponse,
	evalBoolExpr func(ctx context.Context, expr string, vars map[string]interface{}) (bool, error),
	triggerStep func(ctx context.Context, actionId, workflowName string, childIndex int32, parentTaskRunIds []uuid.UUID, isSkipped, isCancelled, parentReExecuted bool) (*operator.DAGStepTriggerResult, error),
) error {
	ctx, span := telemetry.NewSpan(ctx, "dag.dagDurableTask")
	defer span.End()

	span.SetAttributes(
		attribute.String("dag.external_id", externalId.String()),
		attribute.Int("dag.invocation_count", int(invocationCount)),
		attribute.Int("dag.task_count", len(tasks)),
		attribute.Bool("dag.has_on_failure", onFailureTask != nil),
	)

	d := &dag{
		tasks:            tasks,
		onFailureTask:    onFailureTask,
		pendingTasks:     append([]*task{}, tasks...),
		requestCh:        requestCh,
		responseCh:       responseCh,
		evalBoolExpr:     evalBoolExpr,
		externalId:       externalId,
		invocationCount:  invocationCount,
		input:            input,
		triggerStep:      triggerStep,
		conditionMatches: make(map[*sqlcv1.V1StepMatchCondition]bool),
	}

	for !d.isDone() {
		if err := d.taskEmitter(ctx); err != nil {
			return err
		}

		if triggered, err := d.evaluateOnFailure(ctx); err != nil {
			return err
		} else if triggered {
			continue
		}

		if len(d.queuedResponses) > 0 {
			resp := d.queuedResponses[0]
			d.queuedResponses = d.queuedResponses[1:]
			d.taskConsumer(ctx, resp)
			continue
		}

		if d.isDone() {
			break
		}

		resp, err := d.awaitResponse(ctx, responseCh)

		if err != nil {
			return err
		}

		d.taskConsumer(ctx, resp)
	}

	if d.err != nil {
		return d.err
	}

	for _, t := range d.tasks {
		if t.isFailed {
			return &dagChildFailedError{taskActionId: t.actionId, errorMessage: t.errorMessage}
		}
	}

	for _, t := range d.tasks {
		if t.isCancelled {
			return &dagCancelledError{taskActionId: t.actionId}
		}
	}

	return nil
}

// blocks until the child task returns - this is basically here to just reveal bottlenecks, especially on child spawning, in the traces
func (d *dag) awaitResponse(ctx context.Context, responseCh <-chan *v1contracts.DurableTaskResponse) (*v1contracts.DurableTaskResponse, error) {
	_, span := telemetry.NewSpan(ctx, "dag.awaitResponse")
	defer span.End()

	awaitingChildren, awaitingConditions := 0, 0

	for _, t := range d.tasks {
		switch {
		case t.isTriggered && !t.isCompleted:
			awaitingChildren++
		case t.isWaiting && !t.isWaitSatisfied:
			awaitingConditions++
		}
	}

	span.SetAttributes(
		attribute.Int("dag.awaiting_child_count", awaitingChildren),
		attribute.Int("dag.awaiting_condition_count", awaitingConditions),
		attribute.Int("dag.pending_task_count", len(d.pendingTasks)),
	)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp, ok := <-responseCh:
		if !ok {
			return nil, fmt.Errorf("durable task session closed")
		}

		return resp, nil
	}
}

func (d *dag) taskEmitter(ctx context.Context) error {
	ctx, span := telemetry.NewSpan(ctx, "dag.taskEmitter")
	defer span.End()

	if d.err != nil {
		return nil
	}

	for {
		progressed, err := d.emitReadyTasks(ctx)
		if err != nil {
			return err
		}

		if !progressed {
			return nil
		}
	}
}

func (d *dag) emitReadyTasks(ctx context.Context) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "dag.emitReadyTasks")
	defer span.End()

	span.SetAttributes(attribute.Int("dag.pending_task_count", len(d.pendingTasks)))

	progressed := false

	stillPending := d.pendingTasks[:0]

	for _, t := range d.pendingTasks {
		if t.isTriggered || t.isSkipped {
			continue
		}

		ready := true
		for _, p := range t.parents {
			if !p.isCompleted {
				ready = false
				break
			}
		}

		if !ready {
			stillPending = append(stillPending, t)
			continue
		}

		cancelled := false
		for _, p := range t.parents {
			if p.isCancelled || p.isFailed {
				cancelled = true
				break
			}
		}

		skip := false

		if !cancelled {
			if allParentsSkipped(t) {
				skip = true
			} else {
				skip, cancelled = d.evaluateParentConditions(ctx, t)
			}

			if !skip && !cancelled {
				if d.hasEventOrSleepConditions(t, conditionKindSkip) && !t.skipWatchRegistered {
					if err := d.registerCondition(ctx, t, conditionKindSkip); err != nil {
						d.err = err
						return progressed, d.err
					}
					t.skipWatchRegistered = true
				}

				if d.hasEventOrSleepConditions(t, conditionKindCancel) && !t.cancelWatchRegistered {
					if err := d.registerCondition(ctx, t, conditionKindCancel); err != nil {
						d.err = err
						return progressed, d.err
					}
					t.cancelWatchRegistered = true
				}

				if t.cancelWatchFired {
					cancelled = true
				} else if t.skipWatchFired {
					skip = true
				}

				if !skip && !cancelled && d.hasEventOrSleepConditions(t, conditionKindWait) && !t.isWaitSatisfied {
					if !t.isWaiting {
						satisfiedGroups := d.evaluateWaitParentConditions(ctx, t)

						if d.allWaitGroupsSatisfied(t, satisfiedGroups) {
							t.isWaitSatisfied = true
						} else {
							if err := d.registerCondition(ctx, t, conditionKindWait, satisfiedGroups); err != nil {
								d.err = err
								return progressed, d.err
							}
							t.isWaiting = true
						}
					}
					if !t.isWaitSatisfied {
						stillPending = append(stillPending, t)
						continue
					}
				}
			}
		}

		var parentTaskRunIds []uuid.UUID
		for _, p := range d.tasks {
			if p.isCompleted && !p.isFailed && p.workflowRunExternalId != nil {
				parentTaskRunIds = append(parentTaskRunIds, *p.workflowRunExternalId)
			}
		}

		parentReExecuted := false
		for _, p := range t.parents {
			if p.reExecuted {
				parentReExecuted = true
				break
			}
		}

		result, err := d.triggerStep(ctx, t.actionId, t.workflowName, t.index, parentTaskRunIds, skip, cancelled, parentReExecuted)
		if err != nil {
			d.err = fmt.Errorf("failed to trigger step %q: %w", t.actionId, err)
			return progressed, d.err
		}

		t.reExecuted = result.ReExecuted

		if cancelled {
			t.isCancelled = true
		} else if skip {
			t.isSkipped = true
			t.isCompleted = true
			t.output = map[string]interface{}{"skipped": true}

			if err := d.evaluateConditionsForParent(ctx, t); err != nil {
				d.err = err
				return progressed, d.err
			}
		}

		t.nodeId = result.NodeId
		t.branchId = result.BranchId
		t.workflowRunExternalId = &result.WorkflowRunExternalId
		t.isTriggered = true
		progressed = true

		if result.IsSatisfied {
			errorMessage := ""
			if result.ErrorMessage != nil {
				errorMessage = *result.ErrorMessage
			}
			if err := d.applyCompletion(ctx, t, result.IsFailure, errorMessage, result.ResultPayload); err != nil {
				d.err = err
				return progressed, d.err
			}
		}
	}

	d.pendingTasks = stillPending

	return progressed, nil
}

func (d *dag) taskConsumer(ctx context.Context, resp *v1contracts.DurableTaskResponse) {
	ctx, span := telemetry.NewSpan(ctx, "dag.taskConsumer")
	defer span.End()

	if resp == nil || resp.Message == nil {
		return
	}

	span.SetAttributes(attribute.String("dag.response_type", fmt.Sprintf("%T", resp.Message)))

	switch m := resp.Message.(type) {
	case *v1contracts.DurableTaskResponse_WaitForAck:
		ref := m.WaitForAck.GetRef()
		if ref == nil || len(d.pendingWaitAcks) == 0 {
			return
		}
		// Correlate in FIFO order: the dispatcher processes requestCh sequentially,
		// so acks arrive in the same order we sent the WAITFOR requests.
		ack := d.pendingWaitAcks[0]
		d.pendingWaitAcks = d.pendingWaitAcks[1:]
		switch ack.kind {
		case conditionKindSkip:
			ack.task.skipWatchNodeId = ref.GetNodeId()
			ack.task.skipWatchBranchId = ref.GetBranchId()
		case conditionKindCancel:
			ack.task.cancelWatchNodeId = ref.GetNodeId()
			ack.task.cancelWatchBranchId = ref.GetBranchId()
		default:
			ack.task.waitNodeId = ref.GetNodeId()
			ack.task.waitBranchId = ref.GetBranchId()
		}

		// Its completion may already be buffered as unmatched (see pendingEntryCompletions).
		d.drainPendingEntryCompletions(ctx)

	case *v1contracts.DurableTaskResponse_EntryCompleted:
		ref := m.EntryCompleted.GetRef()
		if ref == nil {
			return
		}

		entry := pendingEntryCompletion{
			nodeId:       ref.GetNodeId(),
			branchId:     ref.GetBranchId(),
			isFailure:    m.EntryCompleted.GetIsFailure(),
			errorMessage: m.EntryCompleted.GetErrorMessage(),
			payload:      m.EntryCompleted.GetPayload(),
		}

		if !d.tryApplyEntryCompletion(ctx, entry) {
			// Raced ahead of its own WaitForAck; retry once that ack lands.
			d.pendingEntryCompletions = append(d.pendingEntryCompletions, entry)
		}
	}
}

// tryApplyEntryCompletion attempts to match a completion against a registered skip/cancel/wait
// watch or a triggered task's node/branch id, applying it if found. Returns false if nothing
// currently known matches.
func (d *dag) tryApplyEntryCompletion(ctx context.Context, entry pendingEntryCompletion) bool {
	for _, t := range d.tasks {
		if t.skipWatchRegistered && !t.skipWatchFired && t.skipWatchNodeId == entry.nodeId && t.skipWatchBranchId == entry.branchId {
			t.skipWatchFired = true
			return true
		}

		if t.cancelWatchRegistered && !t.cancelWatchFired && t.cancelWatchNodeId == entry.nodeId && t.cancelWatchBranchId == entry.branchId {
			t.cancelWatchFired = true
			return true
		}

		if t.isWaiting && !t.isWaitSatisfied && t.waitNodeId == entry.nodeId && t.waitBranchId == entry.branchId {
			t.isWaitSatisfied = true
			return true
		}

		if t.nodeId != entry.nodeId || t.branchId != entry.branchId {
			continue
		}

		if err := d.applyCompletion(ctx, t, entry.isFailure, entry.errorMessage, entry.payload); err != nil {
			d.err = err
		}

		return true
	}

	return false
}

// drainPendingEntryCompletions retries buffered completions that couldn't be matched when they
// first arrived, keeping any that still don't match.
func (d *dag) drainPendingEntryCompletions(ctx context.Context) {
	if len(d.pendingEntryCompletions) == 0 {
		return
	}

	remaining := d.pendingEntryCompletions[:0]

	for _, entry := range d.pendingEntryCompletions {
		if !d.tryApplyEntryCompletion(ctx, entry) {
			remaining = append(remaining, entry)
		}
	}

	d.pendingEntryCompletions = remaining
}

func (d *dag) applyCompletion(ctx context.Context, t *task, isFailure bool, errorMessage string, payload []byte) error {
	t.isCompleted = true

	if isFailure && !t.isCancelled {
		if errorMessage == repository.TaskCancelledErrorMessage {
			t.isCancelled = true
		} else {
			t.isFailed = true
			t.errorMessage = errorMessage
		}
	} else if len(payload) > 0 {
		outputData := make(map[string]interface{})
		if err := json.Unmarshal(payload, &outputData); err == nil {
			t.output = outputData
			if skipped, ok := outputData["skipped"].(bool); ok && skipped {
				t.isSkipped = true
			}
			if cancelled, ok := outputData["cancelled"].(bool); ok && cancelled {
				t.isCancelled = true
			}
		}
	}

	return d.evaluateConditionsForParent(ctx, t)
}

func (d *dag) evaluateConditionsForParent(ctx context.Context, parent *task) error {
	if parent.output == nil {
		return nil
	}

	ctx, span := telemetry.NewSpan(ctx, "dag.evaluateConditionsForParent")
	defer span.End()

	span.SetAttributes(attribute.String("dag.parent_readable_id", parent.readableId))

	for _, t := range d.tasks {
		for _, cond := range t.stepConditions {
			if cond.Kind != sqlcv1.V1StepMatchConditionKindPARENTOVERRIDE {
				continue
			}
			if cond.ParentReadableID.String != parent.readableId {
				continue
			}
			if _, ok := d.conditionMatches[cond]; ok {
				continue
			}

			expr := cond.Expression.String
			if expr == "" {
				expr = "true"
			}

			matched, err := d.evalBoolExpr(ctx, expr, map[string]interface{}{"output": parent.output})
			if err != nil {
				return fmt.Errorf("CEL eval error for task %q condition %q: %w", t.actionId, expr, err)
			}

			d.conditionMatches[cond] = matched
		}
	}

	// getting rid of memory here so we don't hold onto the output for the lifetime of the DAG
	parent.output = nil

	return nil
}

func allParentsSkipped(t *task) bool {
	if len(t.parents) == 0 {
		return false
	}

	for _, p := range t.parents {
		if !p.isSkipped {
			return false
		}
	}

	return true
}

func (d *dag) isDone() bool {
	if d.err != nil {
		return true
	}

	for _, t := range d.tasks {
		if !t.isCompleted {
			return false
		}
	}

	// Not yet in d.tasks until evaluateOnFailure resolves it.
	if d.onFailureTask != nil && !d.onFailureTask.isCompleted {
		return false
	}

	return true
}

func (d *dag) evaluateOnFailure(ctx context.Context) (bool, error) {
	if d.onFailureTask == nil || d.onFailureTask.isTriggered {
		return false, nil
	}

	anyFailed := false
	allOthersDone := true

	for _, t := range d.tasks {
		if t.isFailed {
			anyFailed = true
		}
		if !t.isCompleted {
			allOthersDone = false
		}
	}

	if !anyFailed && !allOthersDone {
		return false, nil
	}

	ctx, span := telemetry.NewSpan(ctx, "dag.evaluateOnFailure")
	defer span.End()

	skip := !anyFailed

	span.SetAttributes(
		attribute.String("dag.on_failure_action_id", d.onFailureTask.actionId),
		attribute.Bool("dag.on_failure_skipped", skip),
	)

	var parentTaskRunIds []uuid.UUID
	for _, p := range d.tasks {
		if p.isCompleted && !p.isFailed && p.workflowRunExternalId != nil {
			parentTaskRunIds = append(parentTaskRunIds, *p.workflowRunExternalId)
		}
	}

	result, err := d.triggerStep(ctx, d.onFailureTask.actionId, d.onFailureTask.workflowName, d.onFailureTask.index, parentTaskRunIds, skip, false, false)
	if err != nil {
		d.err = fmt.Errorf("failed to trigger on-failure step %q: %w", d.onFailureTask.actionId, err)
		return false, d.err
	}

	d.onFailureTask.reExecuted = result.ReExecuted
	d.onFailureTask.nodeId = result.NodeId
	d.onFailureTask.branchId = result.BranchId
	d.onFailureTask.workflowRunExternalId = &result.WorkflowRunExternalId
	d.onFailureTask.isTriggered = true

	if skip {
		d.onFailureTask.isSkipped = true
		d.onFailureTask.isCompleted = true
		d.onFailureTask.output = map[string]interface{}{"skipped": true}
	}

	d.tasks = append(d.tasks, d.onFailureTask)

	if result.IsSatisfied {
		errorMessage := ""
		if result.ErrorMessage != nil {
			errorMessage = *result.ErrorMessage
		}
		if err := d.applyCompletion(ctx, d.onFailureTask, result.IsFailure, errorMessage, result.ResultPayload); err != nil {
			d.err = err
			return true, d.err
		}
	}

	return true, nil
}

func (d *dag) evaluateParentConditions(ctx context.Context, t *task) (skip bool, cancel bool) {
	_, span := telemetry.NewSpan(ctx, "dag.evaluateParentConditions")
	defer span.End()

	type groupKey struct {
		action    sqlcv1.V1MatchConditionAction
		orGroupId uuid.UUID
	}
	groupResults := make(map[groupKey]bool)
	groupActions := make(map[groupKey]sqlcv1.V1MatchConditionAction)

	for _, cond := range t.stepConditions {
		if cond.Kind != sqlcv1.V1StepMatchConditionKindPARENTOVERRIDE {
			continue
		}
		if cond.Action != sqlcv1.V1MatchConditionActionSKIP && cond.Action != sqlcv1.V1MatchConditionActionCANCEL {
			continue
		}

		matched, ok := d.conditionMatches[cond]
		if !ok {
			continue
		}

		key := groupKey{action: cond.Action, orGroupId: cond.OrGroupID}
		groupActions[key] = cond.Action
		if matched {
			groupResults[key] = true
		} else if _, seen := groupResults[key]; !seen {
			groupResults[key] = false
		}
	}

	skipGroups := make(map[uuid.UUID]bool)
	cancelGroups := make(map[uuid.UUID]bool)
	skipTotal, cancelTotal := 0, 0

	for key, satisfied := range groupResults {
		switch groupActions[key] {
		case sqlcv1.V1MatchConditionActionSKIP:
			skipTotal++
			if satisfied {
				skipGroups[key.orGroupId] = true
			}
		case sqlcv1.V1MatchConditionActionCANCEL:
			cancelTotal++
			if satisfied {
				cancelGroups[key.orGroupId] = true
			}
		}
	}

	if cancelTotal > 0 && len(cancelGroups) == cancelTotal {
		return false, true
	}
	if skipTotal > 0 && len(skipGroups) == skipTotal {
		return true, false
	}

	return false, false
}

func getMatchConditionActionForWatchKind(kind conditionKind) sqlcv1.V1MatchConditionAction {
	switch kind {
	case conditionKindSkip:
		return sqlcv1.V1MatchConditionActionSKIP
	case conditionKindCancel:
		return sqlcv1.V1MatchConditionActionCANCEL
	default:
		return sqlcv1.V1MatchConditionActionQUEUE
	}
}

func (d *dag) hasEventOrSleepConditions(t *task, kind conditionKind) bool {
	action := getMatchConditionActionForWatchKind(kind)

	for _, c := range t.stepConditions {
		if c.Action != action {
			continue
		}

		if c.Kind == sqlcv1.V1StepMatchConditionKindSLEEP || c.Kind == sqlcv1.V1StepMatchConditionKindUSEREVENT {
			return true
		}
	}

	return false
}

func (d *dag) evaluateWaitParentConditions(ctx context.Context, t *task) map[uuid.UUID]bool {
	_, span := telemetry.NewSpan(ctx, "dag.evaluateWaitParentConditions")
	defer span.End()

	satisfied := make(map[uuid.UUID]bool)

	for _, c := range t.stepConditions {
		if c.Action != sqlcv1.V1MatchConditionActionQUEUE {
			continue
		}
		if c.Kind != sqlcv1.V1StepMatchConditionKindPARENTOVERRIDE {
			continue
		}
		if satisfied[c.OrGroupID] {
			continue
		}

		matched, ok := d.conditionMatches[c]
		if !ok {
			continue
		}

		if matched {
			satisfied[c.OrGroupID] = true
		}
	}

	return satisfied
}

func (d *dag) allWaitGroupsSatisfied(t *task, satisfiedGroups map[uuid.UUID]bool) bool {
	for _, c := range t.stepConditions {
		if c.Action != sqlcv1.V1MatchConditionActionQUEUE {
			continue
		}
		if !satisfiedGroups[c.OrGroupID] {
			return false
		}
	}
	return true
}

// send writes req to the durable-task session without deadlocking: the dispatcher side is a
// single goroutine that blocks delivering each response on responseCh before reading the next
// request, so a plain channel send here while responses are undelivered would leave both sides
// blocked sending to each other. Responses received while waiting are queued for the main loop.
func (d *dag) send(ctx context.Context, req *v1contracts.DurableTaskRequest) error {
	for {
		select {
		case d.requestCh <- req:
			return nil
		case resp, ok := <-d.responseCh:
			if !ok {
				return fmt.Errorf("durable task session closed")
			}
			d.queuedResponses = append(d.queuedResponses, resp)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (d *dag) registerCondition(ctx context.Context, t *task, kind conditionKind, satisfiedGroups ...map[uuid.UUID]bool) error {
	_, span := telemetry.NewSpan(ctx, "dag.registerCondition")
	defer span.End()

	span.SetAttributes(
		attribute.String("dag.task_action_id", t.actionId),
		attribute.Int("dag.condition_kind", int(kind)),
	)

	action := getMatchConditionActionForWatchKind(kind)
	conditions := &v1contracts.DurableEventListenerConditions{}

	var skip map[uuid.UUID]bool
	if len(satisfiedGroups) > 0 {
		skip = satisfiedGroups[0]
	}

	for _, c := range t.stepConditions {
		if c.Action != action {
			continue
		}
		if skip[c.OrGroupID] {
			continue
		}
		switch c.Kind {
		case sqlcv1.V1StepMatchConditionKindSLEEP:
			conditions.SleepConditions = append(conditions.SleepConditions, &v1contracts.SleepMatchCondition{
				Base: &v1contracts.BaseMatchCondition{
					ReadableDataKey: c.ReadableDataKey,
					OrGroupId:       c.OrGroupID.String(),
				},
				SleepFor: c.SleepDuration.String,
			})
		case sqlcv1.V1StepMatchConditionKindUSEREVENT:
			conditions.UserEventConditions = append(conditions.UserEventConditions, &v1contracts.UserEventMatchCondition{
				Base: &v1contracts.BaseMatchCondition{
					ReadableDataKey: c.ReadableDataKey,
					OrGroupId:       c.OrGroupID.String(),
					Expression:      c.Expression.String,
				},
				UserEventKey: c.EventKey.String,
			})
		}
	}

	if err := d.send(ctx, &v1contracts.DurableTaskRequest{
		Message: &v1contracts.DurableTaskRequest_WaitFor{
			WaitFor: &v1contracts.DurableTaskWaitForRequest{
				DurableTaskExternalId: d.externalId.String(),
				InvocationCount:       d.invocationCount,
				WaitForConditions:     conditions,
			},
		},
	}); err != nil {
		return err
	}

	d.pendingWaitAcks = append(d.pendingWaitAcks, &pendingWaitAck{task: t, kind: kind})

	return nil
}
