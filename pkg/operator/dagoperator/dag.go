package dagoperator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
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

type dag struct {
	requestCh   chan<- *v1contracts.DurableTaskRequest
	matchRepo   repository.MatchRepository
	triggerStep func(ctx context.Context, actionId, workflowName string, childIndex int32, parentTaskRunIds []uuid.UUID, isSkipped, isCancelled bool) (*operator.DAGStepTriggerResult, error)

	tasks           []*task
	externalId      uuid.UUID
	invocationCount int32
	input           string
	err             error

	pendingWaitAcks []*pendingWaitAck
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
	matchRepo repository.MatchRepository,
	triggerStep func(ctx context.Context, actionId, workflowName string, childIndex int32, parentTaskRunIds []uuid.UUID, isSkipped, isCancelled bool) (*operator.DAGStepTriggerResult, error),
) error {
	d := &dag{
		tasks:           tasks,
		requestCh:       requestCh,
		matchRepo:       matchRepo,
		externalId:      externalId,
		invocationCount: invocationCount,
		input:           input,
		triggerStep:     triggerStep,
	}

	for !d.isDone() {
		if err := d.taskEmitter(ctx); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case resp, ok := <-responseCh:
			if !ok {
				return fmt.Errorf("durable task session closed")
			}
			d.taskConsumer(resp)
		}
	}

	if d.err != nil {
		return d.err
	}

	for _, t := range d.tasks {
		if t.isFailed {
			return fmt.Errorf("child task %q failed: %s", t.actionId, t.errorMessage)
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
	if d.err != nil {
		return nil
	}

	for _, t := range d.tasks {
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
			var err error
			skip, cancelled, err = d.evaluateParentConditions(ctx, t)
			if err != nil {
				d.err = fmt.Errorf("failed to evaluate conditions for task %q: %w", t.actionId, err)
				return d.err
			}

			if !cancelled {
				if d.hasEventOrSleepConditions(t, conditionKindSkip) && !t.skipWatchRegistered {
					d.registerCondition(t, conditionKindSkip)
					t.skipWatchRegistered = true
				}

				if d.hasEventOrSleepConditions(t, conditionKindCancel) && !t.cancelWatchRegistered {
					d.registerCondition(t, conditionKindCancel)
					t.cancelWatchRegistered = true
				}

				if t.cancelWatchFired {
					cancelled = true
				} else if t.skipWatchFired {
					skip = true
				}

				if !skip && !cancelled && d.hasEventOrSleepConditions(t, conditionKindWait) && !t.isWaitSatisfied {
					if !t.isWaiting {
						d.registerCondition(t, conditionKindWait)
						t.isWaiting = true
					}
					continue
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
			return d.err
		}

		if cancelled {
			t.isCancelled = true
		} else if skip {
			t.isSkipped = true
			t.output = map[string]interface{}{"skipped": true}
		}

		t.nodeId = result.NodeId
		t.branchId = result.BranchId
		t.workflowRunExternalId = &result.WorkflowRunExternalId
		t.isTriggered = true
	}

	return nil
}

func (d *dag) taskConsumer(resp *v1contracts.DurableTaskResponse) {
	if resp == nil || resp.Message == nil {
		return
	}

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

			t.isCompleted = true

			if m.EntryCompleted.GetIsFailure() && !t.isCancelled {
				t.isFailed = true
				t.errorMessage = m.EntryCompleted.GetErrorMessage()
			} else if payload := m.EntryCompleted.GetPayload(); len(payload) > 0 {
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

			return
		}
	}
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

func (d *dag) evaluateParentConditions(ctx context.Context, t *task) (skip bool, cancel bool, err error) {
	parentByReadableId := make(map[string]*task, len(d.tasks))
	for _, p := range d.tasks {
		parentByReadableId[p.readableId] = p
	}

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

		parentReadableId := cond.ParentReadableID.String
		parent, ok := parentByReadableId[parentReadableId]
		if !ok || parent.output == nil {
			continue
		}

		expr := cond.Expression.String
		if expr == "" {
			expr = "true"
		}

		matched, evalErr := d.matchRepo.EvalBoolExpr(ctx, expr, map[string]interface{}{"output": parent.output})
		if evalErr != nil {
			return false, false, fmt.Errorf("CEL eval error for task %q condition %q: %w", t.actionId, expr, evalErr)
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
		return false, true, nil
	}
	if skipTotal > 0 && len(skipGroups) == skipTotal {
		return true, false, nil
	}

	return false, false, nil
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

func (d *dag) registerCondition(t *task, kind conditionKind) {
	action := getMatchConditionActionForWatchKind(kind)
	conditions := &v1contracts.DurableEventListenerConditions{}

	for _, c := range t.stepConditions {
		if c.Action != action {
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
