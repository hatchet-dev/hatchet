package main

import (
	"fmt"
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

	events := make(chan string, 50)

	// > TimeoutStep
	cleanup, err := run(events, worker.WorkflowJob{
		Name:        "timeout",
		Description: "timeout",
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

	<-events

	if err := cleanup(); err != nil {
		panic(fmt.Errorf("cleanup() error = %v", err))
	}
}
