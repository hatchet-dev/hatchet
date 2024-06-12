package main

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type userCreateEvent struct {
	Username string            `json:"username"`
	UserID   string            `json:"user_id"`
	Data     map[string]string `json:"data"`
}

type output struct {
	Message string `json:"message"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	prisma := db.NewClient()
	if err := prisma.Connect(); err != nil {
		panic(fmt.Errorf("error connecting to database: %w", err))
	}
	defer func() {
		if err := prisma.Disconnect(); err != nil {
			panic(fmt.Errorf("error disconnecting from database: %w", err))
		}
	}()

	c, err := client.New()
	if err != nil {
		panic(fmt.Errorf("error creating client: %w", err))
	}

	w, err := worker.NewWorker(
		worker.WithClient(
			c,
		),
	)
	if err != nil {
		panic(fmt.Errorf("error creating worker: %w", err))
	}

	workflow := "webhook"
	event := "user:create:webhook"
	err = initialize(w, worker.WorkflowJob{
		Name:        workflow,
		Description: workflow,
		Steps: []*worker.WorkflowStep{
			worker.Fn(func(ctx worker.HatchetContext) (result *output, err error) {
				log.Printf("step name: %s", ctx.StepName())
				return &output{
					Message: "hi from " + ctx.StepName(),
				}, nil
			}).SetName("webhook-step-one").SetTimeout("10s"),
			worker.Fn(func(ctx worker.HatchetContext) (result *output, err error) {
				log.Printf("step name: %s", ctx.StepName())
				return &output{
					Message: "hi from " + ctx.StepName(),
				}, nil
			}).SetName("webhook-step-one").SetTimeout("10s"),
		},
	}, event)
	if err != nil {
		panic(err)
	}

	handler := w.WebhookHttpHandler(worker.WebhookHandlerOptions{
		Secret: "secret",
	})
	err = run(prisma, handler, c, workflow, event)
	if err != nil {
		panic(err)
	}
}
