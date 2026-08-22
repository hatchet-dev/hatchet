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
	"github.com/hatchet-dev/hatchet/pkg/loadtest/eventkeys"
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

type DurableChildInput struct {
	Index int `json:"index"`
}

type DurableChildOutput struct {
	Index   int    `json:"index"`
	Message string `json:"message"`
}

type DurableLoadTestOutput struct {
	Children int    `json:"children"`
	Message  string `json:"message"`
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
	taskName := envOr("HATCHET_LOADTEST_WORKFLOW_NAME", eventkeys.WorkflowStandardName(0))
	eventKey := envOr("HATCHET_LOADTEST_EVENT_KEY", eventkeys.EventKeyDefault.String())
	batchTaskEventKey := envOr("HATCHET_LOADTEST_BATCH_EVENT_KEY", eventkeys.EventKeyBatch.String())
	durableTaskEventKey := envOr("HATCHET_LOADTEST_DURABLE_EVENT_KEY", eventkeys.EventKeyDurable.String())
	delayMs := envInt("HATCHET_LOADTEST_DELAY_MS", 0)
	failureRate := envFloat("HATCHET_LOADTEST_FAILURE_RATE", 0)
	workerName := envOr("HATCHET_LOADTEST_WORKER_NAME", eventkeys.WorkerName)
	batchTaskName := envOr("HATCHET_LOADTEST_BATCH_WORKFLOW_NAME", eventkeys.WorkflowBatchName)

	durableTaskName := envOr("HATCHET_LOADTEST_DURABLE_TASK_NAME", eventkeys.WorkflowDurableName)
	durableChildTaskName := envOr("HATCHET_LOADTEST_DURABLE_CHILD_TASK_NAME", eventkeys.WorkflowDurableChildName)
	durableChildren := envInt("HATCHET_LOADTEST_DURABLE_CHILDREN", 3)
	durableChildDurationMs := envInt("HATCHET_LOADTEST_DURABLE_CHILD_DURATION_MS", 100)
	durableSlots := envInt("HATCHET_LOADTEST_DURABLE_SLOTS", 100)
	slots := envInt("HATCHET_LOADTEST_SLOTS", 100)
	dagTaskName := envOr("HATCHET_LOADTEST_DAG_WORKFLOW_NAME", eventkeys.WorkflowDagName)
	dagTaskEventKey := envOr("HATCHET_LOADTEST_DAG_EVENT_KEY", eventkeys.EventKeyDag.String())

	dagWorkflow := client.NewWorkflow(dagTaskName, hatchet.WithWorkflowEvents(dagTaskEventKey))
	step1 := dagWorkflow.NewTask(dagTaskName+"-step1", func(ctx hatchet.Context, input LoadTestInput) (LoadTestOutput, error) {
		return LoadTestOutput{
			Message: "This ran at: " + time.Now().Format(time.RFC3339Nano),
		}, nil
	})
	_ = dagWorkflow.NewTask(dagTaskName+"-step2", func(ctx hatchet.Context, input LoadTestInput) (LoadTestOutput, error) {
		return LoadTestOutput{
			Message: "This ran at: " + time.Now().Format(time.RFC3339Nano),
		}, nil
	}, hatchet.WithParents(step1))

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

	batchTask := client.NewStandaloneBatchTask(batchTaskName, func(ctx hatchet.Context, tasks map[string]LoadTestInput) (map[string]LoadTestOutput, error) {
		out := make(map[string]LoadTestOutput, len(tasks))
		for id := range tasks {
			out[id] = LoadTestOutput{
				Message: "This ran at: " + time.Now().Format(time.RFC3339Nano),
			}
		}
		return out, nil
	},
		hatchet.BatchConfig{
			MaxSize:     10,
			MaxInterval: new(500 * time.Millisecond),
		},
		hatchet.WithWorkflowEvents(batchTaskEventKey),
	)

	durableChildTask := client.NewStandaloneTask(durableChildTaskName, func(ctx hatchet.Context, input DurableChildInput) (DurableChildOutput, error) {
		time.Sleep(time.Duration(durableChildDurationMs) * time.Millisecond)

		return DurableChildOutput{
			Index:   input.Index,
			Message: "child ran at: " + time.Now().Format(time.RFC3339Nano),
		}, nil
	})

	durableTask := client.NewStandaloneDurableTask(
		durableTaskName,
		func(ctx hatchet.DurableContext, input LoadTestInput) (DurableLoadTestOutput, error) {
			log.Printf("durable task %d starting", input.ID)

			for i := range durableChildren {
				if _, err := durableChildTask.Run(ctx, DurableChildInput{Index: i}); err != nil {
					return DurableLoadTestOutput{}, fmt.Errorf("durable child fan-out failed: %w", err)
				}
			}

			return DurableLoadTestOutput{
				Children: durableChildren,
				Message:  "durable task ran at: " + time.Now().Format(time.RFC3339Nano),
			}, nil
		},
		hatchet.WithWorkflowEvents(durableTaskEventKey),
		hatchet.WithScheduleTimeout(60*time.Minute),
		hatchet.WithExecutionTimeout(5*time.Minute),
	)

	worker, err := client.NewWorker(
		workerName,
		hatchet.WithWorkflows(task, batchTask, durableTask, durableChildTask, dagWorkflow),
		hatchet.WithDurableSlots(durableSlots),
		hatchet.WithSlots(slots),
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
