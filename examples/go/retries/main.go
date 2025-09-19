package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/hatchet-dev/hatchet/pkg/worker"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type RetriesInput struct{}
type RetriesResult struct{}

// Simple retries example that always fails
func Retries(client *hatchet.Client) *hatchet.StandaloneTask {
	// > Simple Step Retries
	retries := client.NewStandaloneTask("retries-task", func(ctx hatchet.Context, input RetriesInput) (*RetriesResult, error) {
		return nil, errors.New("intentional failure")
	}, hatchet.WithRetries(3))

	return retries
}

type RetriesWithCountInput struct{}
type RetriesWithCountResult struct {
	Message string `json:"message"`
}

// Retries example that succeeds after a certain number of retries
func RetriesWithCount(client *hatchet.Client) *hatchet.StandaloneTask {
	// > Retries with Count
	retriesWithCount := client.NewStandaloneTask("fail-twice-task", func(ctx hatchet.Context, input RetriesWithCountInput) (*RetriesWithCountResult, error) {
		// Get the current retry count
		retryCount := ctx.RetryCount()

		fmt.Printf("Retry count: %d\n", retryCount)

		if retryCount < 2 {
			return nil, errors.New("intentional failure")
		}

		return &RetriesWithCountResult{
			Message: "success",
		}, nil
	}, hatchet.WithRetries(3))

	return retriesWithCount
}

type BackoffInput struct{}
type BackoffResult struct{}

// Retries example with simple backoff (no configuration in this API version)
func WithBackoff(client *hatchet.Client) *hatchet.StandaloneTask {
	// > Retries with Backoff
	withBackoff := client.NewStandaloneTask("with-backoff-task", func(ctx hatchet.Context, input BackoffInput) (*BackoffResult, error) {
		return nil, errors.New("intentional failure")
	}, hatchet.WithRetries(3), hatchet.WithRetryBackoff(2, 10))

	return withBackoff
}

type NonRetryableInput struct{}
type NonRetryableResult struct{}

// NonRetryableError returns a workflow which throws a non-retryable error
func NonRetryableError(client *hatchet.Client) *hatchet.StandaloneTask {
	// > Non Retryable Error
	retries := client.NewStandaloneTask("non-retryable-task", func(ctx hatchet.Context, input NonRetryableInput) (*NonRetryableResult, error) {
		return nil, worker.NewNonRetryableError(errors.New("intentional failure"))
	}, hatchet.WithRetries(3))

	return retries
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	worker, err := client.NewWorker(
		"retries-worker",
		hatchet.WithWorkflows(Retries(client), RetriesWithCount(client), WithBackoff(client), NonRetryableError(client)),
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	if err := worker.StartBlocking(context.Background()); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}
