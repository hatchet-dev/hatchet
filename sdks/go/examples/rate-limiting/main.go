package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/hatchet-dev/hatchet/sdks/go/features"
)

type APIRequest struct {
	UserID string `json:"userId"`
	Action string `json:"action"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	err = client.RateLimits().Upsert(features.CreateRatelimitOpts{
		Key:      "api-service-rate-limit",
		Limit:    10,
		Duration: types.Second,
	})
	if err != nil {
		log.Fatalf("failed to create rate limit: %v", err)
	}

	const RATE_LIMIT_KEY = "api-service-rate-limit"

	units := 1
	staticTask := client.NewStandaloneTask("task1",
		func(ctx hatchet.Context, input APIRequest) (string, error) {
			log.Println("executed task1")

			return "completed", nil
		},
		hatchet.WithRateLimits(&types.RateLimit{
			Key:   RATE_LIMIT_KEY,
			Units: &units,
		}),
	)

	userUnits := 1
	userLimit := "10"
	duration := types.Minute
	dynamicTask := client.NewStandaloneTask("task2",
		func(ctx hatchet.Context, input APIRequest) (string, error) {
			log.Printf("executed task2 for user: %s", input.UserID)

			return "completed", nil
		},
		hatchet.WithRateLimits(&types.RateLimit{
			Key:            "input.userId",
			Units:          &userUnits,
			LimitValueExpr: &userLimit,
			Duration:       &duration,
		}),
	)

	worker, err := client.NewWorker("rate-limit-worker",
		hatchet.WithWorkflows(staticTask, dynamicTask),
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	go func() {
		time.Sleep(2 * time.Second)

		for i := 0; i < 5; i++ {
			_, err := client.RunNoWait(context.Background(), "task1", APIRequest{
				UserID: fmt.Sprintf("user-%d", i),
				Action: "test",
			})
			if err != nil {
				log.Printf("Failed to submit static request: %v", err)
			}
		}

		for i := 0; i < 5; i++ {
			_, err := client.RunNoWait(context.Background(), "task2", APIRequest{
				UserID: fmt.Sprintf("user-%d", i%2),
				Action: "test",
			})
			if err != nil {
				log.Printf("Failed to submit dynamic request: %v", err)
			}
		}
	}()

	log.Println("Starting worker for rate limiting demo...")

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	if err := worker.StartBlocking(interruptCtx); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}
