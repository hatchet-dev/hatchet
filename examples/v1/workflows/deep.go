package v1_workflows

import (
	"time"

	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/task"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type DeepInput struct {
	Message string
	N       int
}

type TransformerOutput struct {
	Sum int
}

type DeepResult struct {
	Transformer TransformerOutput
}

func Child1(hatchet *v1.HatchetClient) workflow.WorkflowDeclaration[DeepInput, DeepResult] {
	child := v1.WorkflowFactory[DeepInput, DeepResult](
		workflow.CreateOpts[DeepInput]{
			Name: "child1",
		},
		hatchet,
	)

	child.Task(
		task.CreateOpts[DeepInput]{
			Name: "transformer",
			Fn: func(input DeepInput, ctx worker.HatchetContext) (*TransformerOutput, error) {
				time.Sleep(15 * time.Second)
				return &TransformerOutput{
					Sum: 1,
				}, nil
			},
		},
	)

	return child
}

func Child2(hatchet *v1.HatchetClient) workflow.WorkflowDeclaration[DeepInput, DeepResult] {
	child := v1.WorkflowFactory[DeepInput, DeepResult](
		workflow.CreateOpts[DeepInput]{
			Name: "child2",
		},
		hatchet,
	)

	child.Task(
		task.CreateOpts[DeepInput]{
			Name: "transformer",
			Fn: func(input DeepInput, ctx worker.HatchetContext) (*TransformerOutput, error) {
				sum := 0
				for i := 0; i < input.N; i++ {
					workflow, err := ctx.SpawnWorkflow("child1", input, nil)
					if err != nil {
						return nil, err
					}

					result, err := workflow.Result()
					if err != nil {
						return nil, err
					}

					var childResult DeepResult
					err = result.StepOutput("transformer", &childResult.Transformer)
					if err != nil {
						return nil, err
					}

					sum += childResult.Transformer.Sum
				}

				time.Sleep(15 * time.Second)
				return &TransformerOutput{
					Sum: sum,
				}, nil
			},
		},
	)

	return child
}

func Child3(hatchet *v1.HatchetClient) workflow.WorkflowDeclaration[DeepInput, DeepResult] {
	child := v1.WorkflowFactory[DeepInput, DeepResult](
		workflow.CreateOpts[DeepInput]{
			Name: "child3",
		},
		hatchet,
	)

	child.Task(
		task.CreateOpts[DeepInput]{
			Name: "transformer",
			Fn: func(input DeepInput, ctx worker.HatchetContext) (*TransformerOutput, error) {
				sum := 0
				for i := 0; i < input.N; i++ {
					workflow, err := ctx.SpawnWorkflow("child2", input, nil)
					if err != nil {
						return nil, err
					}

					result, err := workflow.Result()
					if err != nil {
						return nil, err
					}

					var childResult DeepResult
					err = result.StepOutput("transformer", &childResult.Transformer)
					if err != nil {
						return nil, err
					}

					sum += childResult.Transformer.Sum
				}

				return &TransformerOutput{
					Sum: sum,
				}, nil
			},
		},
	)

	return child
}

func Child4(hatchet *v1.HatchetClient) workflow.WorkflowDeclaration[DeepInput, DeepResult] {
	child := v1.WorkflowFactory[DeepInput, DeepResult](
		workflow.CreateOpts[DeepInput]{
			Name: "child4",
		},
		hatchet,
	)

	child.Task(
		task.CreateOpts[DeepInput]{
			Name: "transformer",
			Fn: func(input DeepInput, ctx worker.HatchetContext) (*TransformerOutput, error) {
				sum := 0
				for i := 0; i < input.N; i++ {
					workflow, err := ctx.SpawnWorkflow("child3", input, nil)
					if err != nil {
						return nil, err
					}

					result, err := workflow.Result()
					if err != nil {
						return nil, err
					}

					var childResult DeepResult
					err = result.StepOutput("transformer", &childResult.Transformer)
					if err != nil {
						return nil, err
					}

					sum += childResult.Transformer.Sum
				}

				return &TransformerOutput{
					Sum: sum,
				}, nil
			},
		},
	)

	return child
}

func Child5(hatchet *v1.HatchetClient) workflow.WorkflowDeclaration[DeepInput, DeepResult] {
	child := v1.WorkflowFactory[DeepInput, DeepResult](
		workflow.CreateOpts[DeepInput]{
			Name: "child5",
		},
		hatchet,
	)

	child.Task(
		task.CreateOpts[DeepInput]{
			Name: "transformer",
			Fn: func(input DeepInput, ctx worker.HatchetContext) (*TransformerOutput, error) {
				sum := 0
				for i := 0; i < input.N; i++ {
					workflow, err := ctx.SpawnWorkflow("child4", input, nil)
					if err != nil {
						return nil, err
					}

					result, err := workflow.Result()
					if err != nil {
						return nil, err
					}

					var childResult DeepResult
					err = result.StepOutput("transformer", &childResult.Transformer)
					if err != nil {
						return nil, err
					}

					sum += childResult.Transformer.Sum
				}

				return &TransformerOutput{
					Sum: sum,
				}, nil
			},
		},
	)

	return child
}

func DeepParent(hatchet *v1.HatchetClient) workflow.WorkflowDeclaration[DeepInput, DeepResult] {
	parent := v1.WorkflowFactory[DeepInput, DeepResult](
		workflow.CreateOpts[DeepInput]{
			Name: "parent",
		},
		hatchet,
	)

	parent.Task(
		task.CreateOpts[DeepInput]{
			Name: "parent",
			Fn: func(input DeepInput, ctx worker.HatchetContext) (*TransformerOutput, error) {
				sum := 0
				for i := 0; i < input.N; i++ {
					workflow, err := ctx.SpawnWorkflow("child5", input, nil)
					if err != nil {
						return nil, err
					}

					result, err := workflow.Result()
					if err != nil {
						return nil, err
					}

					var childResult DeepResult
					err = result.StepOutput("transformer", &childResult.Transformer)
					if err != nil {
						return nil, err
					}

					sum += childResult.Transformer.Sum
				}

				return &TransformerOutput{
					Sum: sum,
				}, nil
			},
		},
	)

	return parent
}
