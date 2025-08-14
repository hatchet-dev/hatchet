package main

import (
	"context"
	"fmt"
	"log"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
)

type APIRequest struct {
	UserID    string `json:"user_id"`
	Action    string `json:"action"`
	Timestamp string `json:"timestamp"`
}

type APIResponse struct {
	UserID      string    `json:"user_id"`
	Action      string    `json:"action"`
	ProcessedAt time.Time `json:"processed_at"`
	Success     bool      `json:"success"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create hatchet client: %v", err)
	}

	// Create workflow with static rate limiting (global limit)
	staticRateLimitWorkflow := client.NewWorkflow("static-rate-limit-demo",
		hatchet.WithWorkflowDescription("Demonstrates static rate limiting"),
		hatchet.WithWorkflowVersion("1.0.0"),
	)

	// Task with static rate limit - 5 requests per second globally
	units := 1
	staticRateLimitWorkflow.NewTask("api-call",
		func(ctx hatchet.Context, input APIRequest) (APIResponse, error) {
			log.Printf("Processing API call for user %s, action: %s", input.UserID, input.Action)
			
			// Simulate API processing time
			time.Sleep(100 * time.Millisecond)
			
			return APIResponse{
				UserID:      input.UserID,
				Action:      input.Action,
				ProcessedAt: time.Now(),
				Success:     true,
			}, nil
		},
		hatchet.WithRateLimits(&types.RateLimit{
			Key:   "global-api-limit",
			Units: &units,
		}),
	)

	// Create workflow with dynamic rate limiting (per-user limit)
	dynamicRateLimitWorkflow := client.NewWorkflow("dynamic-rate-limit-demo",
		hatchet.WithWorkflowDescription("Demonstrates dynamic rate limiting per user"),
		hatchet.WithWorkflowVersion("1.0.0"),
	)

	// Task with dynamic rate limit - 3 requests per second per user
	userUnits := 1
	perSecond := types.Second
	keyExpression := "input.user_id"
	dynamicRateLimitWorkflow.NewTask("user-api-call",
		func(ctx hatchet.Context, input APIRequest) (APIResponse, error) {
			log.Printf("Processing user-specific API call for user %s, action: %s", input.UserID, input.Action)
			
			// Simulate API processing time
			time.Sleep(200 * time.Millisecond)
			
			return APIResponse{
				UserID:      input.UserID,
				Action:      input.Action,
				ProcessedAt: time.Now(),
				Success:     true,
			}, nil
		},
		hatchet.WithRateLimits(&types.RateLimit{
			KeyExpr:  &keyExpression,
			Units:    &userUnits,
			Duration: &perSecond,
		}),
	)

	// Create workflow with multiple rate limits (both global and per-user)
	multiRateLimitWorkflow := client.NewWorkflow("multi-rate-limit-demo",
		hatchet.WithWorkflowDescription("Demonstrates multiple rate limits on a single task"),
		hatchet.WithWorkflowVersion("1.0.0"),
	)

	// Task with both global and per-user rate limits
	globalUnits := 2
	userSpecificUnits := 1
	multiRateLimitWorkflow.NewTask("premium-api-call",
		func(ctx hatchet.Context, input APIRequest) (APIResponse, error) {
			log.Printf("Processing premium API call for user %s, action: %s", input.UserID, input.Action)
			
			// Simulate more complex processing for premium API
			time.Sleep(300 * time.Millisecond)
			
			return APIResponse{
				UserID:      input.UserID,
				Action:      "PREMIUM_" + input.Action,
				ProcessedAt: time.Now(),
				Success:     true,
			}, nil
		},
		hatchet.WithRateLimits(
			// Global rate limit - 10 requests per second across all users
			&types.RateLimit{
				Key:   "premium-global-limit",
				Units: &globalUnits,
			},
			// Per-user rate limit - 2 requests per second per user
			&types.RateLimit{
				KeyExpr:  &keyExpression,
				Units:    &userSpecificUnits,
				Duration: &perSecond,
			},
		),
	)

	// Create a worker with all rate-limited workflows
	worker, err := client.NewWorker("rate-limit-worker",
		hatchet.WithWorkflows(staticRateLimitWorkflow, dynamicRateLimitWorkflow, multiRateLimitWorkflow),
		hatchet.WithSlots(10), // Allow more slots to see rate limiting in action
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	// Function to submit multiple requests rapidly to test rate limiting
	submitRequests := func(workflowName string, userPrefix string, count int, delay time.Duration) {
		for i := 0; i < count; i++ {
			go func(index int) {
				time.Sleep(time.Duration(index) * delay)
				
				userID := fmt.Sprintf("%s-user-%d", userPrefix, (index%3)+1) // Cycle through 3 users
				
				_, err := client.Run(context.Background(), workflowName, APIRequest{
					UserID:    userID,
					Action:    fmt.Sprintf("action-%d", index+1),
					Timestamp: time.Now().Format(time.RFC3339),
				})
				if err != nil {
					log.Printf("Failed to submit request %d: %v", index+1, err)
				}
			}(i)
		}
	}

	// Demonstrate rate limiting behavior
	go func() {
		time.Sleep(3 * time.Second)

		log.Println("\n=== Static Rate Limiting Demo ===")
		log.Println("Submitting 10 requests rapidly to global rate limit workflow")
		log.Println("Watch how they get processed at the rate limit speed...")
		submitRequests("static-rate-limit-demo", "static", 10, 50*time.Millisecond)

		time.Sleep(8 * time.Second)

		log.Println("\n=== Dynamic Rate Limiting Demo ===")
		log.Println("Submitting 15 requests from 3 different users")
		log.Println("Each user has their own rate limit bucket...")
		submitRequests("dynamic-rate-limit-demo", "dynamic", 15, 30*time.Millisecond)

		time.Sleep(10 * time.Second)

		log.Println("\n=== Multi Rate Limiting Demo ===")
		log.Println("Submitting requests with both global and per-user limits")
		log.Println("Requests will be throttled by both limits...")
		submitRequests("multi-rate-limit-demo", "multi", 12, 25*time.Millisecond)
	}()

	log.Println("Starting worker for rate limiting demos...")
	log.Println("Features demonstrated:")
	log.Println("  - Static/global rate limiting")
	log.Println("  - Dynamic rate limiting with key expressions")
	log.Println("  - Per-user rate limiting")
	log.Println("  - Multiple rate limits on a single task")
	log.Println("  - Rate limit units and duration configuration")

	if err := worker.StartBlocking(); err != nil {
		log.Fatalf("failed to start worker: %v", err)
	}
}