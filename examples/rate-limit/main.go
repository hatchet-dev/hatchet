package main

import (
	"fmt"
	"time"

	"github.com/joho/godotenv"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type rateLimitInput struct {
	Index int `json:"index"`
}

type stepOneOutput struct {
	Message string `json:"message"`
}

func StepOne(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
	input := &rateLimitInput{}

	err = ctx.WorkflowInput(input)

	if err != nil {
		return nil, err
	}

	ctx.StreamEvent([]byte(fmt.Sprintf("This is a stream event %d", input.Index)))

	return &stepOneOutput{
		Message: fmt.Sprintf("This ran at %s", time.Now().String()),
	}, nil
}

func main() {
	err := godotenv.Load()

	if err != nil {
		panic(err)
	}

	c, err := client.New()

	if err != nil {
		panic(err)
	}

	err = c.Admin().PutRateLimit("api1", &types.RateLimitOpts{
		Max:      3,
		Duration: "second",
	})

	if err != nil {
		panic(err)
	}

	w, err := worker.NewWorker(
		worker.WithClient(
			c,
		),
	)

	if err != nil {
		panic(err)
	}

	err = w.On(
		worker.NoTrigger(),
		&worker.WorkflowJob{
			Name:        "rate-limit-workflow",
			Description: "This illustrates rate limiting.",
			Steps: []*worker.WorkflowStep{
				worker.Fn(StepOne).SetName("step-one").SetRateLimit(
					worker.RateLimit{
						Units: 1,
						Key:   "api1",
					},
				),
			},
		},
	)

	if err != nil {
		panic(err)
	}

	for i := 0; i < 12; i++ {
		_, err = c.Admin().RunWorkflow("rate-limit-workflow", &rateLimitInput{
			Index: i,
		})

		if err != nil {
			panic(err)
		}
	}

	interrupt := cmdutils.InterruptChan()

	cleanup, err := w.Start()
	if err != nil {
		panic(err)
	}

	<-interrupt

	if err := cleanup(); err != nil {
		panic(fmt.Errorf("error cleaning up: %w", err))
	}
}
