package main

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"

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
	wf := &worker.WorkflowJob{
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
	}

	handler := w.WebhookHttpHandler(worker.WebhookHandlerOptions{
		Secret: "secret",
	}, wf)
	port := "8741"
	err = run("webhook-demo", w, port, handler, c, workflow, event)
	if err != nil {
		panic(err)
	}
}
