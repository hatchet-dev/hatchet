package v1_workflows

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/pkg/client/create"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type StickyInput struct{}

type StickyResult struct {
	Result string `json:"result"`
}

type StickyDagResult struct {
	StickyTask1 StickyResult `json:"sticky-task-1"`
	StickyTask2 StickyResult `json:"sticky-task-2"`
}

func StickyDag(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[StickyInput, StickyDagResult] {
	stickyDag := factory.NewWorkflow[StickyInput, StickyDagResult](
		create.WorkflowCreateOpts[StickyInput]{
			Name: "sticky-dag",
		},
		hatchet,
	)

	stickyDag.Task(
		create.WorkflowTask[StickyInput, StickyDagResult]{
			Name: "sticky-task",
		},
		func(ctx worker.HatchetContext, input StickyInput) (interface{}, error) {
			workerId := ctx.Worker().ID()

			return &StickyResult{
				Result: workerId,
			}, nil
		},
	)

	stickyDag.Task(
		create.WorkflowTask[StickyInput, StickyDagResult]{
			Name: "sticky-task-2",
		},
		func(ctx worker.HatchetContext, input StickyInput) (interface{}, error) {
			workerId := ctx.Worker().ID()

			return &StickyResult{
				Result: workerId,
			}, nil
		},
	)

	return stickyDag
}

func Sticky(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[StickyInput, StickyResult] {
	sticky := factory.NewTask(
		create.StandaloneTask{
			Name:    "sticky-task",
			Retries: 3,
		}, func(ctx worker.HatchetContext, input StickyInput) (*StickyResult, error) {
			// Run a child workflow on the same worker
			childWorkflow := Child(hatchet)
			childResult, err := workflow.RunChildWorkflow(ctx, childWorkflow, ChildInput{N: 1}) // 	 workflow.RunAsChildOpts{
			// 	Sticky: &[]bool{true}[0],
			// }

			if err != nil {
				return nil, err
			}

			return &StickyResult{
				Result: fmt.Sprintf("child-result-%d", childResult.Value),
			}, nil
		},
		hatchet,
	)

	return sticky
}
