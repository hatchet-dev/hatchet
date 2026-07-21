// Command hatchet-loadtest-go-worker is a basic, standalone Go SDK worker
// you can run alongside `cmd/hatchet-loadtest --externalWorker`.
package main

import (
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"strconv"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// LoadTestInput mirrors cmd/hatchet-loadtest's `Event` struct (emit.go)
type LoadTestInput struct {
	CreatedAt time.Time `json:"created_at"`
	Payload   string    `json:"payload"`
	ID        int64     `json:"id"`
}

type LoadTestOutput struct {
	Message string `json:"message"`
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func envFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

func run() error {
	client, err := hatchet.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create hatchet client: %w", err)
	}

	taskName := envOr("HATCHET_LOADTEST_WORKFLOW_NAME", "load-test-0")
	eventKey := envOr("HATCHET_LOADTEST_EVENT_KEY", "load-test:event")
	delayMs := envInt("HATCHET_LOADTEST_DELAY_MS", 0)
	failureRate := envFloat("HATCHET_LOADTEST_FAILURE_RATE", 0)
	workerName := envOr("HATCHET_LOADTEST_WORKER_NAME", "load-test-worker")

	task := client.NewStandaloneTask(taskName, func(ctx hatchet.Context, input LoadTestInput) (LoadTestOutput, error) {
		took := time.Since(input.CreatedAt)
		log.Printf("executing %d took %s", input.ID, took)

		if delayMs > 0 {
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
		}

		if failureRate > 0 && rand.Float64() < failureRate { //nolint:gosec // simulated failure rate, not security-sensitive
			return LoadTestOutput{}, fmt.Errorf("random failure")
		}

		return LoadTestOutput{
			Message: "This ran at: " + time.Now().Format(time.RFC3339Nano),
		}, nil
	},
		hatchet.WithWorkflowEvents(eventKey),
	)

	worker, err := client.NewWorker(
		workerName,
		hatchet.WithWorkflows(task),
	)
	if err != nil {
		return fmt.Errorf("failed to create worker: %w", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	if err := worker.StartBlocking(interruptCtx); err != nil {
		return fmt.Errorf("failed to start worker: %w", err)
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
