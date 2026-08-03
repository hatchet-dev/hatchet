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
	"github.com/hatchet-dev/hatchet/pkg/worker/condition"
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

// DurableChildInput/Output are exercised by the durable load-test task's child fan-out.
type DurableChildInput struct {
	Index int `json:"index"`
}

type DurableChildOutput struct {
	Index   int    `json:"index"`
	Message string `json:"message"`
}

// DurableLoadTestOutput summarizes a single durable task run.
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

func durationPtr(d time.Duration) *time.Duration {
	return &d
}

func run() error {
	client, err := hatchet.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create hatchet client: %w", err)
	}

	taskName := envOr("HATCHET_LOADTEST_WORKFLOW_NAME", "load-test-0")
	eventKey := envOr("HATCHET_LOADTEST_EVENT_KEY", "load-test:event")
	durableTaskEventKey := envOr("HATCHET_LOADTEST_DURABLE_EVENT_KEY", "load-test:durable-event")
	delayMs := envInt("HATCHET_LOADTEST_DELAY_MS", 0)
	failureRate := envFloat("HATCHET_LOADTEST_FAILURE_RATE", 0)
	workerName := envOr("HATCHET_LOADTEST_WORKER_NAME", "load-test-worker")
	batchTaskName := envOr("HATCHET_LOADTEST_BATCH_WORKFLOW_NAME", "load-test-batch")

	durableTaskName := envOr("HATCHET_LOADTEST_DURABLE_TASK_NAME", "load-test-durable")
	durableChildTaskName := envOr("HATCHET_LOADTEST_DURABLE_CHILD_TASK_NAME", "load-test-durable-child")
	durableChildren := envInt("HATCHET_LOADTEST_DURABLE_CHILDREN", 3)
	durableChildDurationMs := envInt("HATCHET_LOADTEST_DURABLE_CHILD_DURATION_MS", 1000)
	durableSleepMs := envInt("HATCHET_LOADTEST_DURABLE_SLEEP_MS", 1000)
	durableSlots := envInt("HATCHET_LOADTEST_DURABLE_SLOTS", 10)

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

	// Preview: batch tasks are in beta and may change in future releases.
	//
	// batchTask subscribes to the same event as the standalone task above, so every load
	// test run also exercises the batch scheduler side by side with normal task scheduling
	// - a canary for scheduling interference, not a benchmarked workflow. Its name
	// deliberately doesn't match the "load-test-%d" pattern that cmd/hatchet-loadtest's
	// expectedWorkflowNames() (do.go) resolves, so the benchmark's TimingCollector never
	// discovers or polls it and its timings never affect the pass/fail thresholds.
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
			MaxInterval: durationPtr(500 * time.Millisecond),
		},
		hatchet.WithWorkflowEvents(eventKey),
	)

	durableChildTask := client.NewStandaloneTask(durableChildTaskName, func(ctx hatchet.Context, input DurableChildInput) (DurableChildOutput, error) {
		time.Sleep(time.Duration(durableChildDurationMs) * time.Millisecond)

		return DurableChildOutput{
			Index:   input.Index,
			Message: "child ran at: " + time.Now().Format(time.RFC3339Nano),
		}, nil
	})

	// durableTask exercises durable execution under load: it durably sleeps, fans out to a
	// batch of child tasks (waiting on their results), durably sleeps again, then returns.
	// It deliberately does NOT match the "load-test-%d" pattern that cmd/hatchet-loadtest's
	// expectedWorkflowNames() resolves, so it never affects the benchmark's pass/fail
	// thresholds - it's a companion durable-execution stressor, not a benchmarked workflow.
	durableTask := client.NewStandaloneDurableTask(
		durableTaskName,
		func(ctx hatchet.DurableContext, input LoadTestInput) (DurableLoadTestOutput, error) {
			log.Printf("durable task %d starting", input.ID)

			if _, err := durableChildTask.Run(ctx, DurableChildInput{Index: 1}); err != nil {
				return DurableLoadTestOutput{}, fmt.Errorf("durable child %d failed: %w", 1, err)
			}

			if _, err = ctx.SleepFor(time.Duration(durableSleepMs) * time.Millisecond); err != nil {
				return DurableLoadTestOutput{}, fmt.Errorf("durable sleep before fan-out failed: %w", err)
			}

			if _, err = durableChildTask.Run(ctx, DurableChildInput{Index: 2}); err != nil {
				return DurableLoadTestOutput{}, fmt.Errorf("durable child %d failed: %w", 2, err)
			}

			if _, err = ctx.WaitFor(condition.Or(
				condition.SleepCondition(time.Duration(durableSleepMs)*time.Millisecond),
				condition.UserEventCondition("test-event-key", "'true'"),
			)); err != nil {
				return DurableLoadTestOutput{}, fmt.Errorf("durable sleep before fan-out failed: %w", err)
			}

			inputs := make([]hatchet.RunManyOpt, durableChildren)

			for i := range inputs {
				inputs[i] = hatchet.RunManyOpt{
					Input: DurableChildInput{Index: i + 2}, // offset by 2 because we already ran 2 children above
				}
			}

			if _, err = durableChildTask.RunMany(ctx, inputs); err != nil {
				return DurableLoadTestOutput{}, fmt.Errorf("durable child fan-out failed: %w", err)
			}

			if _, err = ctx.SleepFor(time.Duration(durableSleepMs) * time.Millisecond); err != nil {
				return DurableLoadTestOutput{}, fmt.Errorf("durable sleep after fan-out failed: %w", err)
			}

			return DurableLoadTestOutput{
				Children: durableChildren + 2, // +2 because we already ran 2 children manually at the start
				Message:  "durable task ran at: " + time.Now().Format(time.RFC3339Nano),
			}, nil
		},
		hatchet.WithWorkflowEvents(durableTaskEventKey),
	)

	worker, err := client.NewWorker(
		workerName,
		hatchet.WithWorkflows(task, batchTask, durableTask, durableChildTask),
		hatchet.WithDurableSlots(durableSlots),
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
