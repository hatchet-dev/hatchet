package main

import (
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/task"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	"github.com/joho/godotenv"
)

type userCreateEvent struct {
	Username string            `json:"username"`
	UserID   string            `json:"user_id"`
	Data     map[string]string `json:"data"`
}

type stepOutput struct {
	Message string `json:"message"`
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

	simple := hatchet.Workflow(workflow.CreateOpts{
		Name: "simple",
	})

	toLower := simple.Task(task.CreateOpts{
		Name: "to_lower",
		Fn: func(ctx worker.HatchetContext) error {
			events <- "to_lower"
			return nil
		},
	})

	println(toLower.Name)

	return nil
}
