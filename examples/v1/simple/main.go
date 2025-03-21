package main

import (
	"fmt"
	"strings"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/task"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/joho/godotenv"
)

type Input struct {
	Message string `json:"message"`
}

type LowerOutput struct {
	TransformedMessage string `json:"message"`
}

type ReverseOutput struct {
	TransformedMessage string `json:"message"`
}

type Result struct {
	ToLower LowerOutput   `json:"to_lower"` // to_lower is the task name
	Reverse ReverseOutput `json:"reverse"`  // reverse is the task name
}

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	events := make(chan string, 50)
	if err := run(cmdutils.InterruptChan(), events); err != nil {
		panic(err)
	}
}

func run(ch <-chan interface{}, events chan<- string) error {
	hatchet, err := v1.NewHatchetClient()

	if err != nil {
		return err
	}

	simple := v1.WorkflowFactory[Input, Result](
		workflow.CreateOpts{
			Name: "simple",
		},
		&hatchet,
	)

	lower := simple.Task(task.CreateOpts[Input, Result]{
		Name: "to_lower",
		Fn: func(input Input, ctx worker.HatchetContext) (*Result, error) {
			events <- "to_lower"

			// TODO: this is a hack to get the result out of the function
			result := &Result{
				ToLower: LowerOutput{
					TransformedMessage: strings.ToLower(input.Message),
				},
			}

			return result, nil
		},
	})

	simple.Task(task.CreateOpts[Input, Result]{
		Name:    "reverse",
		Parents: simple.WithParents(lower),
		Fn: func(input Input, ctx worker.HatchetContext) (*Result, error) {
			events <- "reverse"

			reversed := ""
			for _, char := range input.Message {
				reversed = string(char) + reversed
			}

			result := &Result{
				Reverse: ReverseOutput{
					TransformedMessage: reversed,
				},
			}

			return result, nil
		},
	})

	res, err := simple.Run(Input{
		Message: "Hello, World!",
	})

	if err != nil {
		return err
	}

	fmt.Println(res.Reverse.TransformedMessage)

	return nil
}
