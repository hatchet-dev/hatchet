package v1_workflows

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type StickyInput struct{}

type StickyResult struct {
	Result string `json:"result"`
}

type StickyDagResult struct {
	StickyTask1 StickyResult `json:"sticky-task-1"`
	StickyTask2 StickyResult `json:"sticky-task-2"`
}

// > Sticky DAG
func StickyDag(client *hatchet.Client) *hatchet.Workflow {
	stickyDag := client.NewWorkflow("sticky-dag",
		hatchet.WithWorkflowStickyStrategy(types.StickyStrategy_SOFT),
	)

	_ = stickyDag.NewTask("sticky-task",
		func(ctx worker.HatchetContext, input StickyInput) (interface{}, error) {
			workerId := ctx.Worker().ID()

			return &StickyResult{
				Result: workerId,
			}, nil
		},
	)

	_ = stickyDag.NewTask("sticky-task-2",
		func(ctx worker.HatchetContext, input StickyInput) (interface{}, error) {
			workerId := ctx.Worker().ID()

			return &StickyResult{
				Result: workerId,
			}, nil
		},
	)

	return stickyDag
}

type ChildInput struct {
	N int `json:"n"`
}

type ChildResult struct {
	Result string `json:"result"`
}

func Child(client *hatchet.Client) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("child-task",
		func(ctx hatchet.Context, input ChildInput) (*ChildResult, error) {
			return &ChildResult{Result: "child-result"}, nil
		},
	)
}

// > Sticky Child
func Sticky(client *hatchet.Client) *hatchet.StandaloneTask {
	sticky := client.NewStandaloneTask("sticky-task",
		func(ctx worker.HatchetContext, input StickyInput) (*StickyResult, error) {
			// Run a child workflow on the same worker
			childWorkflow := Child(client)
			childResult, err := childWorkflow.Run(ctx, ChildInput{N: 1}, hatchet.WithRunSticky(true))

			if err != nil {
				return nil, err
			}

			var childOutput ChildResult
			err = childResult.Into(&childOutput)
			if err != nil {
				return nil, err
			}

			return &StickyResult{
				Result: fmt.Sprintf("child-result-%s", childOutput.Result),
			}, nil
		},
	)

	return sticky
}
