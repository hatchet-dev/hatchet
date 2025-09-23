package main

import (
	"log"

	"github.com/hatchet-dev/hatchet/pkg/client"
	v0Worker "github.com/hatchet-dev/hatchet/pkg/worker"
)

func V0() {
	c, err := client.New()
	if err != nil {
		log.Fatal(err)
	}

	worker, err := v0Worker.NewWorker(
		v0Worker.WithClient(c),
		v0Worker.WithName("worker"),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = worker.RegisterWorkflow(
		&v0Worker.WorkflowJob{
			On:   v0Worker.Event("user:create"),
			Name: "simple-workflow",
			Steps: []*v0Worker.WorkflowStep{
				{
					Name: "step1",
					Function: func(ctx v0Worker.HatchetContext) error {
						log.Println("executed step1")
						return nil
					},
				},
			},
		},
	)
	if err != nil {
		log.Fatal(err)
	}
}
