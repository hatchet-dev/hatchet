package dagoperator

import (
	"context"

	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

type dag struct {
	requestCh chan<- *v1contracts.DurableTaskRequest

	// important: task ordering must be the same between instances
	tasks []*task
}

type task struct {
	conditions  []*condition
	parents     []*task
	isCompleted bool
	isTriggered bool // nolint:unused
}

type condition struct {
	*v1contracts.TaskConditions
	isSatisfied bool
	isTriggered bool // nolint:unused
}

func dagDurableTask(ctx context.Context, tasks []*task, requestCh chan<- *v1contracts.DurableTaskRequest, responseCh <-chan *v1contracts.DurableTaskResponse) {
	dag := &dag{
		tasks:     tasks,
		requestCh: requestCh,
	}

	for !dag.isCompleted() {

		dag.taskEmitter()

		select {
		case <-ctx.Done():
			return
		case resp := <-responseCh:
			dag.taskConsumer(resp)
		}
	}
}

// taskConsumer updates the state of the DAG based on a received task request.
func (d *dag) taskConsumer(resp *v1contracts.DurableTaskResponse) {
	panic("not implemented yet")
}

func (d *dag) taskEmitter() {
	// iterate through tasks and figure out which have parent conditions satisfied; those that do, emit a request
	for _, task := range d.tasks {
		// if the task has all parents satisfied, emit requests for its conditions
		// if no conditions, trigger the task
		areParentsSatisfied := true

		for _, parent := range task.parents {
			if !parent.isCompleted {
				areParentsSatisfied = false
				break
			}
		}

		if areParentsSatisfied {
			areConditionsSatisfied := true

			for _, condition := range task.conditions {
				areConditionsSatisfied = areConditionsSatisfied && condition.isSatisfied

				// emit a request for the condition if it's not triggered
				// TODO: emit a request for the condition
				// if !condition.isTriggered {

				// }
			}

			if areConditionsSatisfied {
				// all conditions are satisfied, so we can trigger the task
				d.requestCh <- &v1contracts.DurableTaskRequest{
					Message: &v1contracts.DurableTaskRequest_TriggerRuns{
						// TODO: trigger the task
						TriggerRuns: &v1contracts.DurableTaskTriggerRunsRequest{},
					},
				}
			}
		}
	}
}

func (d *dag) isCompleted() bool {
	// if every task is completed, then the DAG is completed
	for _, task := range d.tasks {
		if !task.isCompleted {
			return false
		}
	}
	return true
}
