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

	if err := setup(c); err != nil {
		panic(fmt.Errorf("error setting up webhook: %w", err))
	}

	err = run(c, worker.WorkflowJob{
		Name:        "webhook",
		Description: "webhook",
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
	})
	if err != nil {
		panic(err)
	}
}
