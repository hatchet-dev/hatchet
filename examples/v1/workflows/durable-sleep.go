package v1_workflows

// import (
// 	"strings"
// 	"time"

// 	"github.com/hatchet-dev/hatchet/pkg/client/create"
// 	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
// 	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
// 	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
// 	"github.com/hatchet-dev/hatchet/pkg/worker"
// )

// type DurableSleepInput struct {
// 	Message string
// }

// type SleepOutput struct {
// 	TransformedMessage string
// }

// type DurableSleepOutput struct {
// 	Sleep SleepOutput
// }

// func DurableSleep(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[DurableSleepInput, DurableSleepOutput] {

// 	simple := factory.NewTask[DurableSleepInput, DurableSleepOutput](
// 		create.StandaloneTask[DurableSleepInput, DurableSleepOutput]{
// 			Name: "durable-sleep",
// 		},
// 		hatchet,
// 	)

// 	simple.DurableTask(
// 		create.StandaloneDurableTaskCreateOpts[DurableSleepInput, DurableSleepOutput]{
// 			Name: "Sleep",
// 			Fn: func(input DurableSleepInput, ctx worker.DurableHatchetContext) (*SleepOutput, error) {

// 				_, err := ctx.SleepFor(time.Minute)

// 				if err != nil {
// 					return nil, err
// 				}

// 				return &SleepOutput{
// 					TransformedMessage: strings.ToLower(input.Message),
// 				}, nil
// 			},
// 		},
// 	)

// 	return simple
// }
