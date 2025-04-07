package v1_workflows

import (
	"github.com/hatchet-dev/hatchet/pkg/client/create"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type DagInput struct {
	Message string
	UserID  string `json:"user_id"`
}

type SimpleOutput struct {
	Step int
}

type DagResult struct {
	Step1 SimpleOutput
	Step2 SimpleOutput
}

func DagWorkflow(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[DagInput, DagResult] {
	// ❓ Declaring a Workflow
	grr := types.GroupRoundRobin
	var maxRuns int32 = 50

	simple := factory.NewWorkflow[DagInput, DagResult](
		create.WorkflowCreateOpts[DagInput]{
			Name: "simple-dag",
			Concurrency: &types.Concurrency{
				Expression:    "input.user_id",
				LimitStrategy: &grr,
				MaxRuns:       &maxRuns,
			},
		},
		hatchet,
	)
	// ‼️

	simple.OnFailure(
		create.WorkflowOnFailureTask[DagInput, DagResult]{},
		func(ctx worker.HatchetContext, input DagInput) (interface{}, error) {
			// Handle failure
			return nil, nil
		},
	)

	// ❓ Defining a Task
	step1 := simple.Task(
		create.WorkflowTask[DagInput, DagResult]{
			Name:    "step1",
			Retries: 3,
			// RetryBackoffFactor:     2.0,
			// RetryMaxBackoffSeconds: 60,
		}, func(ctx worker.HatchetContext, input DagInput) (interface{}, error) {
			// return nil, errors.New("intentional failure")

			// // 50% chance of failure
			// if rand.Intn(2) == 0 {
			// 	return nil, errors.New("intentional failure")
			// }

			return &SimpleOutput{
				Step: 1,
			}, nil
		},
	)
	// ‼️

	// ❓ Adding a Task with a parent
	step2 := simple.Task(
		create.WorkflowTask[DagInput, DagResult]{
			Name:    "step2",
			Retries: 3,
			// RetryBackoffFactor:     2.0,
			// RetryMaxBackoffSeconds: 60,
		}, func(ctx worker.HatchetContext, input DagInput) (interface{}, error) {
			// return nil, errors.New("intentional failure")
			// if rand.Intn(2) == 0 {
			// 	return nil, errors.New("intentional failure")
			// }

			return &SimpleOutput{
				Step: 1,
			}, nil
		},
	)

	simple.Task(
		create.WorkflowTask[DagInput, DagResult]{
			Name:    "step3",
			Retries: 4,
			// RetryBackoffFactor:     2.0,
			// RetryMaxBackoffSeconds: 60,
			Parents: []create.NamedTask{step1, step2},
		}, func(ctx worker.HatchetContext, input DagInput) (interface{}, error) {
			// if rand.Intn(2) == 0 {
			// 	return nil, errors.New("intentional failure")
			// }

			return &SimpleOutput{
				Step: 1,
			}, nil
		},
	)
	// ‼️

	return simple
}
