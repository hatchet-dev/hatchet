package v1_workflows

import (
	"errors"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client/create"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
	"github.com/hatchet-dev/hatchet/pkg/v1/factory"
	"github.com/hatchet-dev/hatchet/pkg/v1/workflow"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type RetriesInput struct{}
type RetriesResult struct{}

// Simple retries example that always fails
func Retries(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[RetriesInput, RetriesResult] {
	retries := factory.NewTask(
		create.StandaloneTask{
			Name:    "retries-task",
			Retries: 3,
		}, func(ctx worker.HatchetContext, input RetriesInput) (*RetriesResult, error) {
			return nil, errors.New("intentional failure")
		},
		hatchet,
	)

	return retries
}

type RetriesWithCountInput struct{}
type RetriesWithCountResult struct {
	Message string `json:"message"`
}

// Retries example that succeeds after a certain number of retries
func RetriesWithCount(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[RetriesWithCountInput, RetriesWithCountResult] {
	retriesWithCount := factory.NewTask(
		create.StandaloneTask{
			Name:    "retries-with-count-task",
			Retries: 3,
		}, func(ctx worker.HatchetContext, input RetriesWithCountInput) (*RetriesWithCountResult, error) {
			// Get the current retry count
			retryCount := ctx.RetryCount()

			fmt.Printf("Retry count: %d\n", retryCount)

			if retryCount < 2 {
				return nil, errors.New("intentional failure")
			}

			return &RetriesWithCountResult{
				Message: "success",
			}, nil
		},
		hatchet,
	)

	return retriesWithCount
}

type BackoffInput struct{}
type BackoffResult struct{}

// Retries example with simple backoff (no configuration in this API version)
func WithBackoff(hatchet v1.HatchetClient) workflow.WorkflowDeclaration[BackoffInput, BackoffResult] {
	withBackoff := factory.NewTask(
		create.StandaloneTask{
			Name:                   "with-backoff-task",
			Retries:                3,
			RetryBackoffFactor:     2,
			RetryMaxBackoffSeconds: 10,
		}, func(ctx worker.HatchetContext, input BackoffInput) (*BackoffResult, error) {
			// Sleep for a short time to simulate backing off
			time.Sleep(100 * time.Millisecond)
			return nil, errors.New("intentional failure")
		},
		hatchet,
	)

	return withBackoff
}
