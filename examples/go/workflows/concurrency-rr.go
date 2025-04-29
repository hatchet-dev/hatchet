package v1_workflows

import (
	"math/rand"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client/create"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type ConcurrencyInput struct {
	Message string
	Tier    string
	Account string
}

type TransformedOutput struct {
	TransformedMessage string
}

func ConcurrencyRoundRobin(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[ConcurrencyInput, TransformedOutput] {
	// > Concurrency Strategy With Key
	var maxRuns int32 = 1
	strategy := types.GroupRoundRobin

	concurrency := factory.NewTask(
		create.StandaloneTask{
			Name: "simple-concurrency",
			Concurrency: []*types.Concurrency{
				{
					Expression:    "input.GroupKey",
					MaxRuns:       &maxRuns,
					LimitStrategy: &strategy,
				},
			},
		}, func(ctx worker.HatchetContext, input ConcurrencyInput) (*TransformedOutput, error) {
			// Random sleep between 200ms and 1000ms
			time.Sleep(time.Duration(200+rand.Intn(800)) * time.Millisecond)

			return &TransformedOutput{
				TransformedMessage: input.Message,
			}, nil
		},
		hatchet,
	)
	

	return concurrency
}

func MultipleConcurrencyKeys(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[ConcurrencyInput, TransformedOutput] {
	// > Multiple Concurrency Keys
	strategy := types.GroupRoundRobin
	var maxRuns int32 = 20

	concurrency := factory.NewTask(
		create.StandaloneTask{
			Name: "simple-concurrency",
			Concurrency: []*types.Concurrency{
				{
					Expression:    "input.Tier",
					MaxRuns:       &maxRuns,
					LimitStrategy: &strategy,
				},
				{
					Expression:    "input.Account",
					MaxRuns:       &maxRuns,
					LimitStrategy: &strategy,
				},
			},
		}, func(ctx worker.HatchetContext, input ConcurrencyInput) (*TransformedOutput, error) {
			// Random sleep between 200ms and 1000ms
			time.Sleep(time.Duration(200+rand.Intn(800)) * time.Millisecond)

			return &TransformedOutput{
				TransformedMessage: input.Message,
			}, nil
		},
		hatchet,
	)
	

	return concurrency
}
