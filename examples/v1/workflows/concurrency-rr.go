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
	Message  string
	GroupKey string
}

type TransformedOutput struct {
	TransformedMessage string
}

func ConcurrencyRoundRobin(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[ConcurrencyInput, TransformedOutput] {
	strategy := types.GroupRoundRobin

	// ❓ Concurrency Strategy With Key
	concurrency := factory.NewTask(
		create.StandaloneTask{
			Name: "simple-concurrency",
			Concurrency: []*types.Concurrency{
				{
					Expression:    "input.GroupKey",
					MaxRuns:       &[]int32{1}[0],
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
	// !!

	return concurrency
}
