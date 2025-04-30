package v1_workflows

import (
	"math/rand"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client/create"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/hatchet-dev/hatchet/pkg/worker/condition"
)

// StepOutput represents the output of most tasks in this workflow
type StepOutput struct {
	RandomNumber int `json:"randomNumber"`
}

// RandomSum represents the output of the sum task
type RandomSum struct {
	Sum int `json:"sum"`
}

// TaskConditionWorkflowResult represents the aggregate output of all tasks
type TaskConditionWorkflowResult struct {
	Start        StepOutput `json:"start"`
	WaitForSleep StepOutput `json:"waitForSleep"`
	WaitForEvent StepOutput `json:"waitForEvent"`
	SkipOnEvent  StepOutput `json:"skipOnEvent"`
	LeftBranch   StepOutput `json:"leftBranch"`
	RightBranch  StepOutput `json:"rightBranch"`
	Sum          RandomSum  `json:"sum"`
}

// taskOpts is a type alias for workflow task options
type taskOpts = create.WorkflowTask[struct{}, TaskConditionWorkflowResult]

func TaskConditionWorkflow(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[struct{}, TaskConditionWorkflowResult] {
	// > Create a workflow
	wf := factory.NewWorkflow[struct{}, TaskConditionWorkflowResult](
		create.WorkflowCreateOpts[struct{}]{
			Name: "TaskConditionWorkflow",
		},
		hatchet,
	)
	// !!

	// > Add base task
	start := wf.Task(
		taskOpts{
			Name: "start",
		},
		func(ctx worker.HatchetContext, _ struct{}) (interface{}, error) {
			return &StepOutput{
				RandomNumber: rand.Intn(100) + 1,
			}, nil
		},
	)
	// !!

	// > Add wait for sleep
	waitForSleep := wf.Task(
		taskOpts{
			Name:    "waitForSleep",
			Parents: []create.NamedTask{start},
			WaitFor: condition.SleepCondition(time.Second * 10),
		},
		func(ctx worker.HatchetContext, _ struct{}) (interface{}, error) {
			return &StepOutput{
				RandomNumber: rand.Intn(100) + 1,
			}, nil
		},
	)
	// !!

	// > Add skip on event
	skipOnEvent := wf.Task(
		taskOpts{
			Name:    "skipOnEvent",
			Parents: []create.NamedTask{start},
			WaitFor: condition.SleepCondition(time.Second * 30),
			SkipIf:  condition.UserEventCondition("skip_on_event:skip", "true"),
		},
		func(ctx worker.HatchetContext, _ struct{}) (interface{}, error) {
			return &StepOutput{
				RandomNumber: rand.Intn(100) + 1,
			}, nil
		},
	)
	// !!

	// > Add branching
	leftBranch := wf.Task(
		taskOpts{
			Name:    "leftBranch",
			Parents: []create.NamedTask{waitForSleep},
			SkipIf:  condition.ParentCondition(waitForSleep, "output.randomNumber > 50"),
		},
		func(ctx worker.HatchetContext, _ struct{}) (interface{}, error) {
			return &StepOutput{
				RandomNumber: rand.Intn(100) + 1,
			}, nil
		},
	)

	rightBranch := wf.Task(
		taskOpts{
			Name:    "rightBranch",
			Parents: []create.NamedTask{waitForSleep},
			SkipIf:  condition.ParentCondition(waitForSleep, "output.randomNumber <= 50"),
		},
		func(ctx worker.HatchetContext, _ struct{}) (interface{}, error) {
			return &StepOutput{
				RandomNumber: rand.Intn(100) + 1,
			}, nil
		},
	)
	// !!

	// > Add wait for event
	waitForEvent := wf.Task(
		taskOpts{
			Name:    "waitForEvent",
			Parents: []create.NamedTask{start},
			WaitFor: condition.Or(
				condition.SleepCondition(time.Minute),
				condition.UserEventCondition("wait_for_event:start", "true"),
			),
		},
		func(ctx worker.HatchetContext, _ struct{}) (interface{}, error) {
			return &StepOutput{
				RandomNumber: rand.Intn(100) + 1,
			}, nil
		},
	)
	// !!

	// > Add sum
	wf.Task(
		taskOpts{
			Name: "sum",
			Parents: []create.NamedTask{
				start,
				waitForSleep,
				waitForEvent,
				skipOnEvent,
				leftBranch,
				rightBranch,
			},
		},
		func(ctx worker.HatchetContext, _ struct{}) (interface{}, error) {
			var startOutput StepOutput
			if err := ctx.ParentOutput(start, &startOutput); err != nil {
				return nil, err
			}

			var waitForSleepOutput StepOutput
			if err := ctx.ParentOutput(waitForSleep, &waitForSleepOutput); err != nil {
				return nil, err
			}

			var waitForEventOutput StepOutput
			ctx.ParentOutput(waitForEvent, &waitForEventOutput)

			// Handle potentially skipped tasks
			var skipOnEventOutput StepOutput
			var four int

			err := ctx.ParentOutput(skipOnEvent, &skipOnEventOutput)

			if err != nil {
				four = 0
			} else {
				four = skipOnEventOutput.RandomNumber
			}

			var leftBranchOutput StepOutput
			var five int

			err = ctx.ParentOutput(leftBranch, leftBranchOutput)
			if err != nil {
				five = 0
			} else {
				five = leftBranchOutput.RandomNumber
			}

			var rightBranchOutput StepOutput
			var six int

			err = ctx.ParentOutput(rightBranch, rightBranchOutput)
			if err != nil {
				six = 0
			} else {
				six = rightBranchOutput.RandomNumber
			}

			return &RandomSum{
				Sum: startOutput.RandomNumber + waitForEventOutput.RandomNumber +
					waitForSleepOutput.RandomNumber + four + five + six,
			}, nil
		},
	)
	// !!

	return wf
}
