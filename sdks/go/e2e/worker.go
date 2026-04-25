//go:build e2e

package e2e

import (
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/worker"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

const (
	sleepTime            = 5
	replayResetSleepTime = 3
	eventKey             = "durable-example:event"
	evictionTTLSeconds   = 5
	longSleepSeconds     = 15
	evictionEventKey     = "durable-eviction:event"
)

// --- Durable test workflows ---

type AwaitedEvent struct {
	ID string `json:"id"`
}

type DurableBulkSpawnInput struct {
	N int `json:"n"`
}

type NonDeterminismOutput struct {
	AttemptNumber          int  `json:"attempt_number"`
	SleepTime              int  `json:"sleep_time"`
	NonDeterminismDetected bool `json:"non_determinism_detected"`
	NodeID                 *int `json:"node_id,omitempty"`
}

type ReplayResetResponse struct {
	Sleep1Duration float64 `json:"sleep_1_duration"`
	Sleep2Duration float64 `json:"sleep_2_duration"`
	Sleep3Duration float64 `json:"sleep_3_duration"`
}

type MemoInput struct {
	Message string `json:"message"`
}

type SleepResult struct {
	Message  string  `json:"message"`
	Duration float64 `json:"duration"`
}

type EmptyInput struct{}

// Durable test workflow definitions and worker tasks
var (
	testDurableWorkflow         *hatchet.Workflow
	testDurableTask             *hatchet.Task
	testWaitForOrGroup1         *hatchet.Task
	testWaitForOrGroup2         *hatchet.Task
	testWaitForSleepTwice       *hatchet.StandaloneTask
	testSpawnChildTask          *hatchet.StandaloneTask
	testDurableWithSpawn        *hatchet.StandaloneTask
	testDurableWithBulkSpawn    *hatchet.StandaloneTask
	testDurableSleepEventSpawn  *hatchet.StandaloneTask
	testDurableNonDeterminism   *hatchet.StandaloneTask
	testDurableReplayReset      *hatchet.StandaloneTask
	testMemoTask                *hatchet.StandaloneTask
	testMemoNowCaching          *hatchet.StandaloneTask
	testDurableSpawnDAG         *hatchet.StandaloneTask
	testDagChildWorkflow        *hatchet.Workflow
	testEvictableSleep          *hatchet.StandaloneTask
	testEvictableWaitForEvent   *hatchet.StandaloneTask
	testEvictableChildSpawn     *hatchet.StandaloneTask
	testEvictableChildBulkSpawn *hatchet.StandaloneTask
	testMultipleEviction        *hatchet.StandaloneTask
	testCapacityEvictableSleep  *hatchet.StandaloneTask
	testNonEvictableSleep       *hatchet.StandaloneTask
	testEvictionChildTask       *hatchet.StandaloneTask
	testEvictionBulkChildTask   *hatchet.StandaloneTask
)

func registerAllWorkflows(client *hatchet.Client) {
	evictionPolicy := &hatchet.EvictionPolicy{
		TTL:                   5 * time.Second,
		AllowCapacityEviction: true,
		Priority:              0,
	}

	capacityEvictionPolicy := &hatchet.EvictionPolicy{
		AllowCapacityEviction: true,
		Priority:              0,
	}

	nonEvictablePolicy := &hatchet.EvictionPolicy{
		AllowCapacityEviction: false,
		Priority:              0,
	}

	// --- DAG child workflow for spawn DAG test ---
	testDagChildWorkflow = client.NewWorkflow("dag-child-workflow")

	dagChild1 := testDagChildWorkflow.NewTask("dag-child-1", func(ctx hatchet.Context, input EmptyInput) (map[string]string, error) {
		time.Sleep(1 * time.Second)
		return map[string]string{"result": "child1"}, nil
	})

	testDagChildWorkflow.NewTask("dag-child-2", func(ctx hatchet.Context, input EmptyInput) (map[string]string, error) {
		time.Sleep(5 * time.Second)
		return map[string]string{"result": "child2"}, nil
	}, hatchet.WithParents(dagChild1))

	// --- Durable workflow with mixed tasks ---
	testDurableWorkflow = client.NewWorkflow("DurableWorkflow")

	testDurableWorkflow.NewTask("ephemeral_task", func(ctx hatchet.Context, input EmptyInput) (any, error) {
		return nil, nil
	})

	testDurableTask = testDurableWorkflow.NewDurableTask("durable_task", func(ctx hatchet.DurableContext, input EmptyInput) (map[string]any, error) {
		_, err := ctx.SleepFor(time.Duration(sleepTime) * time.Second)
		if err != nil {
			return nil, err
		}

		event, err := ctx.WaitForEvent(eventKey, "true")
		if err != nil {
			return nil, err
		}

		var evtData AwaitedEvent
		if err := event.Unmarshal(&evtData); err != nil {
			evtData = AwaitedEvent{ID: ""}
		}

		return map[string]any{
			"status":                 "success",
			"event_id":               evtData.ID,
			"sleep_duration_seconds": sleepTime,
		}, nil
	})

	testWaitForOrGroup1 = testDurableWorkflow.NewDurableTask("wait_for_or_group_1", func(ctx hatchet.DurableContext, input EmptyInput) (map[string]any, error) {
		start := time.Now()
		waitResult, err := ctx.WaitFor(
			hatchet.OrCondition(
				hatchet.SleepCondition(time.Duration(sleepTime)*time.Second),
				hatchet.UserEventCondition(eventKey, ""),
			),
		)
		if err != nil {
			return nil, err
		}

		key, eventID := extractWaitResult(waitResult)
		return map[string]any{
			"runtime":  time.Since(start).Seconds(),
			"key":      key,
			"event_id": eventID,
		}, nil
	})

	testWaitForOrGroup2 = testDurableWorkflow.NewDurableTask("wait_for_or_group_2", func(ctx hatchet.DurableContext, input EmptyInput) (map[string]any, error) {
		start := time.Now()
		waitResult, err := ctx.WaitFor(
			hatchet.OrCondition(
				hatchet.SleepCondition(time.Duration(6*sleepTime)*time.Second),
				hatchet.UserEventCondition(eventKey, ""),
			),
		)
		if err != nil {
			return nil, err
		}

		key, eventID := extractWaitResult(waitResult)
		return map[string]any{
			"runtime":  time.Since(start).Seconds(),
			"key":      key,
			"event_id": eventID,
		}, nil
	})

	// --- Standalone durable tasks ---

	testWaitForSleepTwice = client.NewStandaloneDurableTask("wait-for-sleep-twice", func(ctx hatchet.DurableContext, input EmptyInput) (map[string]float64, error) {
		start := time.Now()
		if _, err := ctx.SleepFor(time.Duration(sleepTime) * time.Second); err != nil {
			return map[string]float64{"runtime": -1.0}, nil
		}
		return map[string]float64{"runtime": time.Since(start).Seconds()}, nil
	})

	testSpawnChildTask = client.NewStandaloneTask("spawn-child-task", func(ctx hatchet.Context, input DurableBulkSpawnInput) (map[string]string, error) {
		return map[string]string{"message": fmt.Sprintf("hello from child %d", input.N)}, nil
	})

	testDurableWithSpawn = client.NewStandaloneDurableTask("durable-with-spawn",
		func(ctx hatchet.DurableContext, input EmptyInput) (map[string]any, error) {
			childResult, err := testSpawnChildTask.Run(ctx, DurableBulkSpawnInput{N: 1})
			if err != nil {
				return nil, err
			}
			var childMap map[string]any
			if err := childResult.Into(&childMap); err != nil {
				return nil, err
			}
			return map[string]any{"child_output": childMap}, nil
		},
		hatchet.WithExecutionTimeout(10*time.Second),
	)

	testDurableWithBulkSpawn = client.NewStandaloneDurableTask("durable-with-bulk-spawn",
		func(ctx hatchet.DurableContext, input DurableBulkSpawnInput) (map[string]any, error) {
			inputs := make([]hatchet.RunManyOpt, input.N)
			for i := 0; i < input.N; i++ {
				inputs[i] = hatchet.RunManyOpt{Input: DurableBulkSpawnInput{N: i}}
			}
			refs, err := testSpawnChildTask.RunMany(ctx, inputs)
			if err != nil {
				return nil, err
			}
			outputs := make([]map[string]string, len(refs))
			for i, ref := range refs {
				result, err := ref.Result()
				if err != nil {
					return nil, err
				}
				var m map[string]string
				if err := result.TaskOutput("spawn-child-task").Into(&m); err != nil {
					return nil, err
				}
				outputs[i] = m
			}
			return map[string]any{"child_outputs": outputs}, nil
		},
	)

	testDurableSleepEventSpawn = client.NewStandaloneDurableTask("durable-sleep-event-spawn",
		func(ctx hatchet.DurableContext, input EmptyInput) (map[string]any, error) {
			start := time.Now()
			if _, err := ctx.SleepFor(time.Duration(sleepTime) * time.Second); err != nil {
				return nil, err
			}
			if _, err := ctx.WaitForEvent(eventKey, "true"); err != nil {
				return nil, err
			}
			childResult, err := testSpawnChildTask.Run(ctx, DurableBulkSpawnInput{N: 1})
			if err != nil {
				return nil, err
			}
			var childMap map[string]any
			if err := childResult.Into(&childMap); err != nil {
				return nil, err
			}
			return map[string]any{
				"runtime":      time.Since(start).Seconds(),
				"child_output": childMap,
			}, nil
		},
	)

	testDurableNonDeterminism = client.NewStandaloneDurableTask("durable-non-determinism",
		func(ctx hatchet.DurableContext, input EmptyInput) (NonDeterminismOutput, error) {
			sleepTimeSec := int(ctx.InvocationCount()) * 2

			err := func() error {
				_, err := ctx.SleepFor(time.Duration(sleepTimeSec) * time.Second)
				return err
			}()

			if err != nil {
				if nde, ok := hatchet.IsNonDeterminismError(err); ok {
					return NonDeterminismOutput{
						AttemptNumber:          int(ctx.InvocationCount()),
						SleepTime:              sleepTimeSec,
						NonDeterminismDetected: true,
						NodeID:                 intPtr(int(nde.NodeID)),
					}, nil
				}
				return NonDeterminismOutput{}, err
			}

			return NonDeterminismOutput{
				AttemptNumber: int(ctx.InvocationCount()),
				SleepTime:     sleepTimeSec,
			}, nil
		},
		hatchet.WithExecutionTimeout(10*time.Second),
	)

	testDurableReplayReset = client.NewStandaloneDurableTask("durable-replay-reset",
		func(ctx hatchet.DurableContext, input EmptyInput) (ReplayResetResponse, error) {
			start := time.Now()
			if _, err := ctx.SleepFor(time.Duration(replayResetSleepTime) * time.Second); err != nil {
				return ReplayResetResponse{}, err
			}
			sleep1 := time.Since(start).Seconds()

			start = time.Now()
			if _, err := ctx.SleepFor(time.Duration(replayResetSleepTime) * time.Second); err != nil {
				return ReplayResetResponse{}, err
			}
			sleep2 := time.Since(start).Seconds()

			start = time.Now()
			if _, err := ctx.SleepFor(time.Duration(replayResetSleepTime) * time.Second); err != nil {
				return ReplayResetResponse{}, err
			}
			sleep3 := time.Since(start).Seconds()

			return ReplayResetResponse{
				Sleep1Duration: sleep1,
				Sleep2Duration: sleep2,
				Sleep3Duration: sleep3,
			}, nil
		},
		hatchet.WithExecutionTimeout(20*time.Second),
	)

	testMemoTask = client.NewStandaloneDurableTask("memo-task",
		func(ctx hatchet.DurableContext, input MemoInput) (SleepResult, error) {
			start := time.Now()

			result, err := ctx.Memo("expensive-computation", func() (any, error) {
				time.Sleep(time.Duration(sleepTime) * time.Second)
				return SleepResult{Message: input.Message, Duration: float64(sleepTime)}, nil
			})
			if err != nil {
				return SleepResult{}, err
			}

			var sr SleepResult
			if m, ok := result.(SleepResult); ok {
				sr = m
			}

			return SleepResult{Message: sr.Message, Duration: time.Since(start).Seconds()}, nil
		},
	)

	testMemoNowCaching = client.NewStandaloneDurableTask("memo-now-caching",
		func(ctx hatchet.DurableContext, input EmptyInput) (map[string]string, error) {
			now, err := ctx.Now()
			if err != nil {
				return nil, err
			}
			return map[string]string{"start_time": now.Format(time.RFC3339Nano)}, nil
		},
	)

	testDurableSpawnDAG = client.NewStandaloneDurableTask("durable-spawn-dag",
		func(ctx hatchet.DurableContext, input EmptyInput) (map[string]any, error) {
			sleepStart := time.Now()
			sleepResult, err := ctx.SleepFor(1 * time.Second)
			if err != nil {
				return nil, err
			}
			sleepDuration := time.Since(sleepStart).Seconds()

			spawnStart := time.Now()
			spawnResult, err := testDagChildWorkflow.Run(ctx, EmptyInput{})
			if err != nil {
				return nil, err
			}
			spawnDuration := time.Since(spawnStart).Seconds()

			return map[string]any{
				"sleep_duration": sleepDuration,
				"sleep_result":   sleepResult,
				"spawn_duration": spawnDuration,
				"spawn_result":   spawnResult.Raw(),
			}, nil
		},
		hatchet.WithExecutionTimeout(10*time.Second),
	)

	// --- Eviction test workflows ---

	testEvictionChildTask = client.NewStandaloneTask("eviction-child-task",
		func(ctx hatchet.Context, input EmptyInput) (map[string]any, error) {
			time.Sleep(time.Duration(longSleepSeconds) * time.Second)
			return map[string]any{"child_status": "completed"}, nil
		},
	)

	type BulkChildInput struct {
		SleepFor int `json:"sleep_for"`
	}

	testEvictionBulkChildTask = client.NewStandaloneTask("eviction-bulk-child-task",
		func(ctx hatchet.Context, input BulkChildInput) (map[string]any, error) {
			time.Sleep(time.Duration(input.SleepFor) * time.Second)
			return map[string]any{"sleep_for": input.SleepFor, "status": "completed"}, nil
		},
	)

	testEvictableSleep = client.NewStandaloneDurableTask("evictable-sleep",
		func(ctx hatchet.DurableContext, input EmptyInput) (map[string]any, error) {
			if _, err := ctx.SleepFor(time.Duration(longSleepSeconds) * time.Second); err != nil {
				return nil, err
			}
			return map[string]any{"status": "completed"}, nil
		},
		hatchet.WithExecutionTimeout(5*time.Minute),
		hatchet.WithEvictionPolicy(evictionPolicy),
	)

	testEvictableWaitForEvent = client.NewStandaloneDurableTask("evictable-wait-for-event",
		func(ctx hatchet.DurableContext, input EmptyInput) (map[string]any, error) {
			if _, err := ctx.WaitForEvent(evictionEventKey, "true"); err != nil {
				return nil, err
			}
			return map[string]any{"status": "completed"}, nil
		},
		hatchet.WithExecutionTimeout(5*time.Minute),
		hatchet.WithEvictionPolicy(evictionPolicy),
	)

	testEvictableChildSpawn = client.NewStandaloneDurableTask("evictable-child-spawn",
		func(ctx hatchet.DurableContext, input EmptyInput) (map[string]any, error) {
			childResult, err := testEvictionChildTask.Run(ctx, EmptyInput{})
			if err != nil {
				return nil, err
			}
			var childMap map[string]any
			if err := childResult.Into(&childMap); err != nil {
				return nil, err
			}
			return map[string]any{"child": childMap, "status": "completed"}, nil
		},
		hatchet.WithExecutionTimeout(5*time.Minute),
		hatchet.WithEvictionPolicy(evictionPolicy),
	)

	testEvictableChildBulkSpawn = client.NewStandaloneDurableTask("evictable-child-bulk-spawn",
		func(ctx hatchet.DurableContext, input EmptyInput) (map[string]any, error) {
			inputs := make([]hatchet.RunManyOpt, 3)
			for i := 0; i < 3; i++ {
				inputs[i] = hatchet.RunManyOpt{
					Input: BulkChildInput{SleepFor: (evictionTTLSeconds + 5) * (i + 1)},
				}
			}
			refs, err := testEvictionBulkChildTask.RunMany(ctx, inputs)
			if err != nil {
				return nil, err
			}
			results := make([]map[string]any, len(refs))
			for i, ref := range refs {
				result, err := ref.Result()
				if err != nil {
					return nil, err
				}
				var m map[string]any
				if err := result.TaskOutput("eviction-bulk-child-task").Into(&m); err != nil {
					return nil, err
				}
				results[i] = m
			}
			return map[string]any{"child_results": results}, nil
		},
		hatchet.WithExecutionTimeout(5*time.Minute),
		hatchet.WithEvictionPolicy(evictionPolicy),
	)

	testMultipleEviction = client.NewStandaloneDurableTask("multiple-eviction",
		func(ctx hatchet.DurableContext, input EmptyInput) (map[string]any, error) {
			if _, err := ctx.SleepFor(time.Duration(longSleepSeconds) * time.Second); err != nil {
				return nil, err
			}
			if _, err := ctx.SleepFor(time.Duration(longSleepSeconds) * time.Second); err != nil {
				return nil, err
			}
			return map[string]any{"status": "completed"}, nil
		},
		hatchet.WithExecutionTimeout(5*time.Minute),
		hatchet.WithEvictionPolicy(evictionPolicy),
	)

	testCapacityEvictableSleep = client.NewStandaloneDurableTask("capacity-evictable-sleep",
		func(ctx hatchet.DurableContext, input EmptyInput) (map[string]any, error) {
			if _, err := ctx.SleepFor(20 * time.Second); err != nil {
				return nil, err
			}
			return map[string]any{"status": "completed"}, nil
		},
		hatchet.WithExecutionTimeout(5*time.Minute),
		hatchet.WithEvictionPolicy(capacityEvictionPolicy),
	)

	testNonEvictableSleep = client.NewStandaloneDurableTask("non-evictable-sleep",
		func(ctx hatchet.DurableContext, input EmptyInput) (map[string]any, error) {
			if _, err := ctx.SleepFor(10 * time.Second); err != nil {
				return nil, err
			}
			return map[string]any{"status": "completed"}, nil
		},
		hatchet.WithExecutionTimeout(5*time.Minute),
		hatchet.WithEvictionPolicy(nonEvictablePolicy),
	)
}

func startTestWorker(client *hatchet.Client) (*hatchet.Worker, func() error, error) {
	registerAllWorkflows(client)

	worker, err := client.NewWorker("e2e-durable-worker",
		hatchet.WithWorkflows(
			testDurableWorkflow,
			testDagChildWorkflow,
			testWaitForSleepTwice,
			testSpawnChildTask,
			testDurableWithSpawn,
			testDurableWithBulkSpawn,
			testDurableSleepEventSpawn,
			testDurableNonDeterminism,
			testDurableReplayReset,
			testMemoTask,
			testMemoNowCaching,
			testDurableSpawnDAG,
			testEvictableSleep,
			testEvictableWaitForEvent,
			testEvictableChildSpawn,
			testEvictableChildBulkSpawn,
			testMultipleEviction,
			testCapacityEvictableSleep,
			testNonEvictableSleep,
			testEvictionChildTask,
			testEvictionBulkChildTask,
		),
		hatchet.WithDurableSlots(10),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create worker: %w", err)
	}

	cleanup, err := worker.Start()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start worker: %w", err)
	}

	// Give the worker a moment to register
	time.Sleep(2 * time.Second)

	return worker, cleanup, nil
}

// extractWaitResult extracts the first key and first sub-key from a WaitResult.
func extractWaitResult(result *worker.WaitResult) (string, string) {
	if result == nil {
		return "", ""
	}
	keys := result.Keys()
	if len(keys) == 0 {
		return "", ""
	}
	key := keys[0]
	return key, key
}

func intPtr(v int) *int {
	return &v
}
