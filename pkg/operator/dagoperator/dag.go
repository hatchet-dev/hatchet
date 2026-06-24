package dagoperator

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/operator"
)

type dag struct {
	requestCh chan<- *v1contracts.DurableTaskRequest

	triggerStep func(ctx context.Context, actionId, workflowName string, childIndex int32, parentRunIds []string) (*operator.DAGStepTriggerResult, error)

	// important: task ordering must be the same between instances
	tasks           []*task
	externalId      string
	invocationCount int32
	input           string
	err             error // first child failure, if any
}

type task struct {
	conditions   []*condition
	id           uuid.UUID
	actionId     string
	workflowName string
	index        int32 // stable position; used as ChildIndex for deduplication
	parents      []*task
	isCompleted  bool
	isFailed     bool
	isTriggered  bool
	errorMessage string

	nodeId                int64
	branchId              int64
	workflowRunExternalId string
}

type condition struct {
	*v1contracts.TaskConditions
	isSatisfied bool
	isTriggered bool // nolint:unused
}

type failurePayload struct {
	IsFailure    bool   `json:"is_failure"`
	ErrorMessage string `json:"error_message"`
}

func dagDurableTask(
	ctx context.Context,
	tasks []*task,
	externalId string,
	invocationCount int32,
	input string,
	requestCh chan<- *v1contracts.DurableTaskRequest,
	responseCh <-chan *v1contracts.DurableTaskResponse,
	triggerStep func(ctx context.Context, actionId, workflowName string, childIndex int32, parentRunIds []string) (*operator.DAGStepTriggerResult, error),
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

	return d.err
}

func (d *dag) taskEmitter(ctx context.Context) error {
	if d.err != nil {
		return nil
	}

	for _, t := range d.tasks {
		if t.isTriggered {
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

		var parentRunIds []string
		for _, p := range d.tasks {
			if p.isCompleted && !p.isFailed && p.workflowRunExternalId != "" {
				parentRunIds = append(parentRunIds, p.workflowRunExternalId)
			}
		}

		result, err := d.triggerStep(ctx, t.actionId, t.workflowName, t.index, parentRunIds)
		if err != nil {
			d.err = fmt.Errorf("failed to trigger step %q: %w", t.actionId, err)
			return d.err
		}

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
	case *v1contracts.DurableTaskResponse_EntryCompleted:
		ref := m.EntryCompleted.GetRef()
		if ref == nil {
			return
		}

		for _, t := range d.tasks {
			if t.nodeId != ref.GetNodeId() || t.branchId != ref.GetBranchId() {
				continue
			}

			t.isCompleted = true

			if payload := m.EntryCompleted.GetPayload(); len(payload) > 0 {
				var fp failurePayload
				if err := json.Unmarshal(payload, &fp); err == nil && fp.IsFailure {
					t.isFailed = true
					t.errorMessage = fp.ErrorMessage
					if d.err == nil {
						d.err = fmt.Errorf("child task %q failed: %s", t.actionId, fp.ErrorMessage)
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
