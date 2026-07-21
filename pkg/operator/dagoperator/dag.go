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
	evalBoolExpr func(ctx context.Context, expr string, vars map[string]interface{}) (bool, error)
	triggerStep  func(ctx context.Context, actionId, workflowName string, childIndex int32, parentTaskRunIds []uuid.UUID, isSkipped, isCancelled bool) (*operator.DAGStepTriggerResult, error)

	tasks []*task

	pendingTasks    []*task
	externalId      uuid.UUID
	invocationCount int32
	input           string
	err             error

	pendingWaitAcks []*pendingWaitAck

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
	externalId uuid.UUID,
	invocationCount int32,
	input string,
	requestCh chan<- *v1contracts.DurableTaskRequest,
	responseCh <-chan *v1contracts.DurableTaskResponse,
	evalBoolExpr func(ctx context.Context, expr string, vars map[string]interface{}) (bool, error),
	triggerStep func(ctx context.Context, actionId, workflowName string, childIndex int32, parentTaskRunIds []uuid.UUID, isSkipped, isCancelled bool) (*operator.DAGStepTriggerResult, error),
) error {
	ctx, span := telemetry.NewSpan(ctx, "dag.dagDurableTask")
	defer span.End()

	span.SetAttributes(
		attribute.String("dag.external_id", externalId.String()),
		attribute.Int("dag.invocation_count", int(invocationCount)),
		attribute.Int("dag.task_count", len(tasks)),
	)

	d := &dag{
		tasks:            tasks,
		pendingTasks:     append([]*task{}, tasks...),
		requestCh:        requestCh,
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

		if d.isDone() {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case resp, ok := <-responseCh:
			if !ok {
				return fmt.Errorf("durable task session closed")
			}
			d.taskConsumer(ctx, resp)
		}
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

		var skip bool

		if !cancelled {
			skip, cancelled = d.evaluateParentConditions(ctx, t)

			if !cancelled {
				if d.hasEventOrSleepConditions(t, conditionKindSkip) && !t.skipWatchRegistered {
					d.registerCondition(ctx, t, conditionKindSkip)
					t.skipWatchRegistered = true
				}

				if d.hasEventOrSleepConditions(t, conditionKindCancel) && !t.cancelWatchRegistered {
					d.registerCondition(ctx, t, conditionKindCancel)
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
							d.registerCondition(ctx, t, conditionKindWait, satisfiedGroups)
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

		result, err := d.triggerStep(ctx, t.actionId, t.workflowName, t.index, parentTaskRunIds, skip, cancelled)
		if err != nil {
			d.err = fmt.Errorf("failed to trigger step %q: %w", t.actionId, err)
			return progressed, d.err
		}

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

	case *v1contracts.DurableTaskResponse_EntryCompleted:
		ref := m.EntryCompleted.GetRef()
		if ref == nil {
			return
		}

		nodeId := ref.GetNodeId()
		branchId := ref.GetBranchId()

		for _, t := range d.tasks {
			if t.skipWatchRegistered && !t.skipWatchFired && t.skipWatchNodeId == nodeId && t.skipWatchBranchId == branchId {
				t.skipWatchFired = true
				return
			}

			if t.cancelWatchRegistered && !t.cancelWatchFired && t.cancelWatchNodeId == nodeId && t.cancelWatchBranchId == branchId {
				t.cancelWatchFired = true
				return
			}

			if t.isWaiting && !t.isWaitSatisfied && t.waitNodeId == nodeId && t.waitBranchId == branchId {
				t.isWaitSatisfied = true
				return
			}

			if t.nodeId != nodeId || t.branchId != branchId {
				continue
			}

			if err := d.applyCompletion(ctx, t, m.EntryCompleted.GetIsFailure(), m.EntryCompleted.GetErrorMessage(), m.EntryCompleted.GetPayload()); err != nil {
				d.err = err
			}

			return
		}
	}
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

func (d *dag) isDone() bool {
	if d.err != nil {
		return true
	}

	for _, t := range d.tasks {
		if !t.isCompleted {
			return false
		}
	}

	return true
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

func (d *dag) registerCondition(ctx context.Context, t *task, kind conditionKind, satisfiedGroups ...map[uuid.UUID]bool) {
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

	d.requestCh <- &v1contracts.DurableTaskRequest{
		Message: &v1contracts.DurableTaskRequest_WaitFor{
			WaitFor: &v1contracts.DurableTaskWaitForRequest{
				DurableTaskExternalId: d.externalId.String(),
				InvocationCount:       d.invocationCount,
				WaitForConditions:     conditions,
			},
		},
	}

	d.pendingWaitAcks = append(d.pendingWaitAcks, &pendingWaitAck{task: t, kind: kind})
}
