package main

import (
	"time"

	"github.com/joho/godotenv"

	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type userCreateEvent struct {
	Username string            `json:"username"`
	UserID   string            `json:"user_id"`
	Data     map[string]string `json:"data"`
}

type stepOneOutput struct {
	Message string `json:"message"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	err = run(worker.WorkflowJob{
		Name:        "webhook",
		Description: "webhook",
		Steps: []*worker.WorkflowStep{
			worker.Fn(func(ctx worker.HatchetContext) (result *stepOneOutput, err error) {
				time.Sleep(time.Second * 60)
				return nil, nil
			}).SetName("step-one").SetTimeout("10s"),
		},
	})
	if err != nil {
		panic(err)
	}
}
