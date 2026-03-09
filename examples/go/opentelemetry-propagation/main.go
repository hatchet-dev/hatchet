package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"go.opentelemetry.io/otel"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	hatchetotel "github.com/hatchet-dev/hatchet/sdks/go/opentelemetry"
)

// This example demonstrates cross-workflow trace propagation.
// A parent task spawns a child task via .Run(), and both tasks' spans
// appear under the same trace in the UI — even if they run on different workers.

type ParentInput struct {
	Name string `json:"name"`
}

type ParentOutput struct {
	ChildResult string `json:"child_result"`
}

type ChildInput struct {
	Greeting string `json:"greeting"`
}

type ChildOutput struct {
	Message string `json:"message"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	instrumentor, err := hatchetotel.NewInstrumentor(
		hatchetotel.EnableHatchetCollector(),
	)
	if err != nil {
		log.Fatalf("failed to create instrumentor: %v", err)
	}

	tracer := otel.Tracer("otel-propagation-example")

	// generateSpanTree creates a nested span subtree, returning how many spans were created.
	var generateSpanTree func(ctx context.Context, count *int, limit int, depth int, prefix string)
	generateSpanTree = func(ctx context.Context, count *int, limit int, depth int, prefix string) {
		numChildren := 1 + rand.Intn(5) // 1-5 children at this level
		for i := range numChildren {
			if *count >= limit {
				return
			}
			name := fmt.Sprintf("%s.%d", prefix, i)
			childCtx, span := tracer.Start(ctx, name)
			time.Sleep(time.Duration(1+rand.Intn(3)) * time.Millisecond)
			*count++

			// Recurse deeper with probability that decreases with depth
			if *count < limit && depth < 8 && rand.Float64() > float64(depth)*0.12 {
				generateSpanTree(childCtx, count, limit, depth+1, name)
			}

			span.End()
		}
	}

	// Child task — a standalone task that will be spawned by the parent.
	childTask := client.NewStandaloneTask(
		"otel-child-task",
		func(ctx hatchet.Context, input ChildInput) (ChildOutput, error) {
			target := 200 + rand.Intn(101) // 200-300 spans
			count := 0
			// Keep spawning top-level subtrees until we hit the target.
			round := 0
			for count < target {
				generateSpanTree(ctx.GetContext(), &count, target, 0, fmt.Sprintf("child.r%d", round))
				round++
			}

			return ChildOutput{
				Message: fmt.Sprintf("Hello from child: %s (generated %d spans)", input.Greeting, count),
			}, nil
		},
	)

	// Parent task — spawns the child task via .Run(), which propagates the traceparent.
	parentTask := client.NewStandaloneTask(
		"otel-parent-task",
		func(ctx hatchet.Context, input ParentInput) (ParentOutput, error) {
			_, span := tracer.Start(ctx.GetContext(), "parent.prepare")
			time.Sleep(30 * time.Millisecond)
			span.End()

			// This .Run() call automatically injects traceparent into AdditionalMetadata,
			// so the child task's spans will appear under the same trace.
			result, err := childTask.Run(ctx, ChildInput{
				Greeting: fmt.Sprintf("greetings from %s", input.Name),
			})
			if err != nil {
				return ParentOutput{}, fmt.Errorf("child task failed: %w", err)
			}

			var childOutput ChildOutput
			if err := result.Into(&childOutput); err != nil {
				return ParentOutput{}, fmt.Errorf("failed to parse child output: %w", err)
			}

			return ParentOutput{
				ChildResult: childOutput.Message,
			}, nil
		},
	)

	worker, err := client.NewWorker(
		"otel-propagation-worker",
		hatchet.WithWorkflows(parentTask, childTask),
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	worker.Use(instrumentor.Middleware())

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	fmt.Println("Starting worker with OTel trace propagation...")
	fmt.Println("Trigger the parent task to see linked parent → child traces in the UI.")

	go func() {
		<-interruptCtx.Done()
		if shutdownErr := instrumentor.Shutdown(context.Background()); shutdownErr != nil {
			log.Printf("failed to shutdown instrumentor: %v", shutdownErr)
		}
	}()

	if startErr := worker.StartBlocking(interruptCtx); startErr != nil {
		log.Printf("worker error: %v", startErr)
	}
}
