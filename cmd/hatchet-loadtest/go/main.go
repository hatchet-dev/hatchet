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

	// multistep dags to mirror some production workloads we have seen
	dagShapeEventKey := envOr("HATCHET_LOADTEST_DAG_SHAPES_EVENT_KEY", eventkeys.EventKeyDagShapes.String())
	dagShapeFailureRate := envFloat("HATCHET_LOADTEST_DAG_SHAPE_FAILURE_RATE", 0)
	dagShapeWorkflows := buildDagShapeWorkflows(client, dagShapeEventKey, dagShapeFailureRate)

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

	workflows := []hatchet.WorkflowBase{task, batchTask, durableTask, durableChildTask, dagWorkflow}
	for _, w := range dagShapeWorkflows {
		workflows = append(workflows, w)
	}

	worker, err := client.NewWorker(
		workerName,
		hatchet.WithWorkflows(workflows...),
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

type dagShapeOutput struct {
	Message          string `json:"message"`
	SkipNotify       bool   `json:"skipNotify"`
	SkipResolve      bool   `json:"skipResolve"`
	SkipUpdateStatus bool   `json:"skipUpdateStatus"`
	SkipWithdraw     bool   `json:"skipWithdraw"`
}

type dagShapeNode struct {
	name           string
	parents        []string
	retries        int
	backoffFactor  float32
	maxBackoffSecs int
	skipParent     string
	skipExpr       string
	failable       bool
}

func buildDagShapeWorkflow(client *hatchet.Client, wfName, eventKey string, onFailure bool, failureRate float64, nodes []dagShapeNode) *hatchet.Workflow {
	wf := client.NewWorkflow(wfName, hatchet.WithWorkflowEvents(eventKey))

	tasks := make(map[string]*hatchet.Task, len(nodes))

	for _, n := range nodes {
		opts := make([]hatchet.TaskOption, 0, 4)
		if len(n.parents) > 0 {
			parents := make([]*hatchet.Task, 0, len(n.parents))
			for _, p := range n.parents {
				parents = append(parents, tasks[p])
			}
			opts = append(opts, hatchet.WithParents(parents...))
		}
		if n.retries > 0 {
			opts = append(opts, hatchet.WithRetries(n.retries))
		}
		if n.backoffFactor > 0 {
			opts = append(opts, hatchet.WithRetryBackoff(n.backoffFactor, n.maxBackoffSecs))
		}
		if n.skipExpr != "" {
			opts = append(opts, hatchet.WithSkipIf(hatchet.ParentCondition(tasks[n.skipParent], n.skipExpr)))
		}

		failable := n.failable
		nodeName := n.name

		tasks[n.name] = wf.NewTask(wfName+"-"+n.name, func(ctx hatchet.Context, input LoadTestInput) (dagShapeOutput, error) {
			if failable && failureRate > 0 && rand.Float64() < failureRate { //nolint:gosec // load shaping, not security
				return dagShapeOutput{}, fmt.Errorf("injected failure at %s", nodeName)
			}
			// Vary which conditional branches skip per run so the SKIP path is
			// exercised without ever skipping all of them at once.
			return dagShapeOutput{
				Message:          nodeName + " ran at: " + time.Now().Format(time.RFC3339Nano),
				SkipNotify:       input.ID%2 == 0,
				SkipResolve:      input.ID%3 == 0,
				SkipUpdateStatus: input.ID%4 == 0,
				SkipWithdraw:     input.ID%5 == 0,
			}, nil
		}, opts...)
	}

	if onFailure {
		wf.OnFailure(func(ctx hatchet.Context, input LoadTestInput) (dagShapeOutput, error) {
			log.Printf("%s on-failure handler ran for input %d", wfName, input.ID)
			return dagShapeOutput{Message: "on-failure ran at: " + time.Now().Format(time.RFC3339Nano)}, nil
		})
	}

	return wf
}

// buildDagShapeWorkflows returns the four dag-shape workflows. Topologies (node
// count, exact parent edges, retry config, skip conditions, on-failure handler)
// are modelled 1:1 on real production DAGs; node names are structural.
func buildDagShapeWorkflows(client *hatchet.Client, eventKey string, failureRate float64) []*hatchet.Workflow {
	// Shape 1: dense fan-in - 10 nodes, 25 edges. Mostly-linear pipeline where
	// each node re-declares its transitive dependencies as direct parents, so
	// the runnable frontier stays 1-2 wide despite the edge count.
	denseFanin := buildDagShapeWorkflow(client, eventkeys.WorkflowDagShapeDenseFaninName, eventKey, false, failureRate, []dagShapeNode{
		{name: "n0"},
		{name: "n1", parents: []string{"n0"}},
		{name: "n2", parents: []string{"n0", "n1"}},
		{name: "n3", parents: []string{"n0", "n1", "n2"}},
		{name: "n4", parents: []string{"n0", "n1", "n3"}},
		{name: "n5", parents: []string{"n0", "n1", "n3", "n4"}},
		{name: "n6", parents: []string{"n0", "n4", "n5"}},
		{name: "n7", parents: []string{"n0", "n1", "n6"}},
		{name: "n8", parents: []string{"n0", "n2", "n6", "n7"}},
		{name: "n9", parents: []string{"n6", "n8"}},
	})

	// Shape 2: conditional fan-out - 1 root, 4 branches each gated by a
	// PARENT_OVERRIDE/SKIP condition on the root's output, retries 2-3/node.
	conditional := buildDagShapeWorkflow(client, eventkeys.WorkflowDagShapeConditionalName, eventKey, false, failureRate, []dagShapeNode{
		{name: "root", retries: 2},
		{name: "branch-a", parents: []string{"root"}, retries: 3, skipParent: "root", skipExpr: "output.skipNotify == true"},
		{name: "branch-b", parents: []string{"root"}, retries: 2, skipParent: "root", skipExpr: "output.skipResolve == true"},
		{name: "branch-c", parents: []string{"root"}, retries: 2, skipParent: "root", skipExpr: "output.skipUpdateStatus == true"},
		{name: "branch-d", parents: []string{"root"}, retries: 2, skipParent: "root", skipExpr: "output.skipWithdraw == true"},
	})

	// Shape 3: diamond topology, 7 nodes + an ON_FAILURE handler.
	onFailure := buildDagShapeWorkflow(client, eventkeys.WorkflowDagShapeOnFailureName, eventKey, true, failureRate, []dagShapeNode{
		{name: "f0"},
		{name: "f1", parents: []string{"f0"}},
		{name: "f2", parents: []string{"f1"}},
		{name: "f3", parents: []string{"f1", "f2"}},
		{name: "f4", parents: []string{"f2", "f3"}, failable: true},
		{name: "f5", parents: []string{"f4"}},
		{name: "f6", parents: []string{"f1", "f5"}},
	})

	// Shape 4: deep retry chain - 7 nodes, 3 mid-chain nodes with 10 retries
	// and exponential backoff (factor 1.5, max 10s).
	deepRetry := buildDagShapeWorkflow(client, eventkeys.WorkflowDagShapeDeepRetryName, eventKey, false, failureRate, []dagShapeNode{
		{name: "r0"},
		{name: "r1", parents: []string{"r0"}},
		{name: "r2", parents: []string{"r0", "r1"}},
		{name: "r3", parents: []string{"r0", "r2"}, retries: 10, backoffFactor: 1.5, maxBackoffSecs: 10, failable: true},
		{name: "r4", parents: []string{"r0", "r3"}, retries: 10, backoffFactor: 1.5, maxBackoffSecs: 10, failable: true},
		{name: "r5", parents: []string{"r0", "r3", "r4"}, retries: 10, backoffFactor: 1.5, maxBackoffSecs: 10, failable: true},
		{name: "r6", parents: []string{"r0", "r3", "r4", "r5"}},
	})

	return []*hatchet.Workflow{denseFanin, conditional, onFailure, deepRetry}
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
