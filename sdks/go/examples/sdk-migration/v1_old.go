package main

import (
	"log"
	"strings"

	"github.com/hatchet-dev/hatchet/pkg/client/create"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
	"github.com/hatchet-dev/hatchet/pkg/v1/worker"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	v0Worker "github.com/hatchet-dev/hatchet/pkg/worker"
)

func V1Old() {
	hatchet, err := v1.NewHatchetClient()
	if err != nil {
		log.Fatal(err)
	}

	simple := factory.NewTask(
		create.StandaloneTask{Name: "simple-task", OnEvents: []string{"user:create"}},
		func(ctx v0Worker.HatchetContext, input SimpleInput) (*SimpleResult, error) {
			return &SimpleResult{TransformedMessage: strings.ToLower(input.Message)}, nil
		},
		hatchet,
	)

	_, err = hatchet.Worker(worker.WorkerOpts{
		Name:      "worker",
		Workflows: []workflow.WorkflowBase{simple},
	})
	if err != nil {
		log.Fatal(err)
	}
}
