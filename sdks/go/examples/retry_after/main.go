package main

import (
	"context"
	"errors"
	"log"
	"math/rand/v2"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type UpstreamInput struct {
	FailingAttempts int `json:"failingAttempts"`
}

type UpstreamResult struct {
	Attempt int `json:"attempt"`
}

func UpstreamDelay(client *hatchet.Client) *hatchet.StandaloneTask {
	// > Retry after an upstream-provided delay
	task := client.NewStandaloneTask("retry-after-upstream-delay", func(ctx hatchet.Context, input UpstreamInput) (*UpstreamResult, error) {
		if ctx.RetryCount() < input.FailingAttempts {
			return nil, hatchet.NewRetryAfterError(2*time.Second, errors.New("upstream rate limited"))
		}

		return &UpstreamResult{Attempt: ctx.RetryCount()}, nil
	}, hatchet.WithRetries(5))
	// !!

	return task
}

func ExponentialBackoff(client *hatchet.Client) *hatchet.StandaloneTask {
	// > Exponential backoff with full jitter
	task := client.NewStandaloneTask("retry-after-exponential-backoff", func(ctx hatchet.Context, input UpstreamInput) (*UpstreamResult, error) {
		if ctx.RetryCount() < input.FailingAttempts {
			ceiling := min(time.Duration(1<<ctx.RetryCount())*time.Second, time.Minute)

			return nil, hatchet.NewRetryAfterError(rand.N(ceiling), errors.New("upstream unavailable"))
		}

		return &UpstreamResult{Attempt: ctx.RetryCount()}, nil
	}, hatchet.WithRetries(3))
	// !!

	return task
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	worker, err := client.NewWorker(
		"retry-after-worker",
		hatchet.WithWorkflows(UpstreamDelay(client), ExponentialBackoff(client)),
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	if err := worker.StartBlocking(context.Background()); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}
