package dagoperator

import (
	"context"
	"encoding/json"
	"fmt"

	cel "github.com/google/cel-go/cel"
	"github.com/google/uuid"

	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type dag struct {
	requestCh chan<- *v1contracts.DurableTaskRequest

	triggerStep func(ctx context.Context, actionId, workflowName string, childIndex int32, parentRunIds []string, isSkipped, isCancelled bool) (*operator.DAGStepTriggerResult, error)

	// important: task ordering must be the same between instances
	tasks           []*task
	externalId      string
	invocationCount int32
	input           string
	err             error // first child failure, if any

	pendingWaitAcks []*pendingWaitAck
}

type pendingWaitAck struct {
	task   *task
	isSkip bool
}

type task struct {
	conditions   []*condition
	id           uuid.UUID
	actionId     string
	workflowName string
	readableId   string
	index        int32 // stable position; used as ChildIndex for deduplication
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

	isSkipWaiting       bool
	isSkipWaitSatisfied bool
	skipWaitNodeId      int64
	skipWaitBranchId    int64

	stepConditions []*sqlcv1.V1StepMatchCondition

	nodeId                int64
	branchId              int64
	workflowRunExternalId string
}

type condition struct {
	*v1contracts.TaskConditions
	isSatisfied bool
	isTriggered bool // nolint:unused
}

func dagDurableTask(
	ctx context.Context,
	tasks []*task,
	externalId string,
	invocationCount int32,
	input string,
	requestCh chan<- *v1contracts.DurableTaskRequest,
	responseCh <-chan *v1contracts.DurableTaskResponse,
	triggerStep func(ctx context.Context, actionId, workflowName string, childIndex int32, parentRunIds []string, isSkipped, isCancelled bool) (*operator.DAGStepTriggerResult, error),
) error {
	d := &dag{
		tasks:           tasks,
		requestCh:       requestCh,
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
		case resp := <-responseCh:
			d.taskConsumer(resp)
		}
	}

	if d.err == nil {
		for _, t := range d.tasks {
			if t.isCancelled {
				d.err = fmt.Errorf("dag cancelled: task %q was cancelled", t.actionId)
				break
			}
		}
	}

	return d.err
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
			if p.isCancelled {
				cancelled = true
				break
			}
		}

		if !cancelled {
			var skip bool
			var err error
			skip, cancelled, err = d.evaluateParentConditions(ctx, t)
			if err != nil {
				d.err = fmt.Errorf("failed to evaluate conditions for task %q: %w", t.actionId, err)
				return d.err
			}

			if !cancelled {
				if d.hasSkipWaitConditions(t) && !t.isSkipWaiting {
					if err := d.registerSkipWaitFor(t); err != nil {
						d.err = fmt.Errorf("failed to register skip_if wait for task %q: %w", t.actionId, err)
						return d.err
					}
					t.isSkipWaiting = true
				}

				if t.isSkipWaitSatisfied {
					skip = true
				}

				if !skip && d.hasWaitConditions(t) && !t.isWaitSatisfied {
					if !t.isWaiting {
						if err := d.registerWaitFor(t); err != nil {
							d.err = fmt.Errorf("failed to register wait_for for task %q: %w", t.actionId, err)
							return d.err
						}
						t.isWaiting = true
					}
					continue
				}

				var parentRunIds []string
				for _, p := range d.tasks {
					if p.isCompleted && !p.isFailed && p.workflowRunExternalId != "" {
						parentRunIds = append(parentRunIds, p.workflowRunExternalId)
					}
				}

				result, err := d.triggerStep(ctx, t.actionId, t.workflowName, t.index, parentRunIds, skip, false)
				if err != nil {
					d.err = fmt.Errorf("failed to trigger step %q: %w", t.actionId, err)
					return d.err
				}

				if skip {
					t.isSkipped = true
				}

				t.nodeId = result.NodeId
				t.branchId = result.BranchId
				t.workflowRunExternalId = result.WorkflowRunExternalId
				t.isTriggered = true
				continue
			}
		}

		var parentRunIds []string
		for _, p := range d.tasks {
			if p.isCompleted && !p.isFailed && p.workflowRunExternalId != "" {
				parentRunIds = append(parentRunIds, p.workflowRunExternalId)
			}
		}

		result, err := d.triggerStep(ctx, t.actionId, t.workflowName, t.index, parentRunIds, false, true)
		if err != nil {
			d.err = fmt.Errorf("failed to trigger step %q: %w", t.actionId, err)
			return d.err
		}

		t.isCancelled = true

		t.nodeId = result.NodeId
		t.branchId = result.BranchId
		t.workflowRunExternalId = result.WorkflowRunExternalId
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
		if ack.isSkip {
			ack.task.skipWaitNodeId = ref.GetNodeId()
			ack.task.skipWaitBranchId = ref.GetBranchId()
		} else {
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
			if t.isSkipWaiting && !t.isSkipWaitSatisfied && t.skipWaitNodeId == nodeId && t.skipWaitBranchId == branchId {
				t.isSkipWaitSatisfied = true
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
				if d.err == nil {
					d.err = fmt.Errorf("child task %q failed: %s", t.actionId, t.errorMessage)
				}
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

		matched, evalErr := evalCELExpr(ctx, expr, parent.output)
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

func (d *dag) hasWaitConditions(t *task) bool {
	for _, c := range t.stepConditions {
		if c.Action == sqlcv1.V1MatchConditionActionSKIP {
			continue
		}
		if c.Kind == sqlcv1.V1StepMatchConditionKindSLEEP || c.Kind == sqlcv1.V1StepMatchConditionKindUSEREVENT {
			return true
		}
	}
	return false
}

func (d *dag) registerWaitFor(t *task) error {
	conditions := &v1contracts.DurableEventListenerConditions{}

	for _, c := range t.stepConditions {
		if c.Action == sqlcv1.V1MatchConditionActionSKIP {
			continue
		}
		switch c.Kind {
		case sqlcv1.V1StepMatchConditionKindSLEEP:
			sleepFor := c.SleepDuration.String
			conditions.SleepConditions = append(conditions.SleepConditions, &v1contracts.SleepMatchCondition{
				Base: &v1contracts.BaseMatchCondition{
					ReadableDataKey: c.ReadableDataKey,
					OrGroupId:       c.OrGroupID.String(),
				},
				SleepFor: sleepFor,
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
				DurableTaskExternalId: d.externalId,
				InvocationCount:       d.invocationCount,
				WaitForConditions:     conditions,
			},
		},
	}

	d.pendingWaitAcks = append(d.pendingWaitAcks, &pendingWaitAck{task: t, isSkip: false})
	return nil
}

func (d *dag) hasSkipWaitConditions(t *task) bool {
	for _, c := range t.stepConditions {
		if c.Kind == sqlcv1.V1StepMatchConditionKindUSEREVENT && c.Action == sqlcv1.V1MatchConditionActionSKIP {
			return true
		}
	}
	return false
}

func (d *dag) registerSkipWaitFor(t *task) error {
	conditions := &v1contracts.DurableEventListenerConditions{}

	for _, c := range t.stepConditions {
		if c.Kind != sqlcv1.V1StepMatchConditionKindUSEREVENT || c.Action != sqlcv1.V1MatchConditionActionSKIP {
			continue
		}
		conditions.UserEventConditions = append(conditions.UserEventConditions, &v1contracts.UserEventMatchCondition{
			Base: &v1contracts.BaseMatchCondition{
				ReadableDataKey: c.ReadableDataKey,
				OrGroupId:       c.OrGroupID.String(),
				Expression:      c.Expression.String,
			},
			UserEventKey: c.EventKey.String,
		})
	}

	d.requestCh <- &v1contracts.DurableTaskRequest{
		Message: &v1contracts.DurableTaskRequest_WaitFor{
			WaitFor: &v1contracts.DurableTaskWaitForRequest{
				DurableTaskExternalId: d.externalId,
				InvocationCount:       d.invocationCount,
				WaitForConditions:     conditions,
			},
		},
	}

	d.pendingWaitAcks = append(d.pendingWaitAcks, &pendingWaitAck{task: t, isSkip: true})
	return nil
}

func evalCELExpr(ctx context.Context, expr string, output map[string]interface{}) (bool, error) {
	env, err := cel.NewEnv(
		cel.Variable("output", cel.MapType(cel.StringType, cel.DynType)),
	)
	if err != nil {
		return false, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	ast, issues := env.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return false, fmt.Errorf("failed to compile CEL expression %q: %w", expr, issues.Err())
	}

	prg, err := env.Program(ast)
	if err != nil {
		return false, fmt.Errorf("failed to create CEL program for %q: %w", expr, err)
	}

	out, _, err := prg.ContextEval(ctx, map[string]interface{}{"output": output})
	if err != nil {
		return false, fmt.Errorf("failed to evaluate CEL expression %q: %w", expr, err)
	}

	b, ok := out.Value().(bool)
	return ok && b, nil
}
