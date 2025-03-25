package v1_workflows

import (
	"math/rand"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client/types"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/task"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type ConcurrencyInput struct {
	Message  string
	GroupKey string
}

type TransformedOutput struct {
	TransformedMessage string
}

type ConcurrencyResult struct {
	ToLower TransformedOutput
}

func ConcurrencyRoundRobin(hatchet *v1.HatchetClient) workflow.WorkflowDeclaration[ConcurrencyInput, ConcurrencyResult] {
	strategy := types.GroupRoundRobin
	concurrency := v1.WorkflowFactory[ConcurrencyInput, ConcurrencyResult](
		workflow.CreateOpts[ConcurrencyInput]{
			Name: "simple-concurrency",
			Concurrency: &types.Concurrency{
				Expression:    "input.GroupKey",
				MaxRuns:       &[]int32{100}[0],
				LimitStrategy: &strategy,
			},
		},
		hatchet,
	)

	concurrency.Task(
		task.CreateOpts[ConcurrencyInput]{
			Name: "to-lower",
			Fn: func(input ConcurrencyInput, ctx worker.HatchetContext) (*TransformedOutput, error) {
				// Random sleep between 200ms and 1000ms
				time.Sleep(time.Duration(200+rand.Intn(800)) * time.Millisecond)

				return &TransformedOutput{
					TransformedMessage: input.Message,
				}, nil
			},
		},
	)

	return concurrency
}
