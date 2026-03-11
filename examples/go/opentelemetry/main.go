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

	tracer := otel.Tracer("otel-example")

	var generateSpanTree func(ctx context.Context, count *int, limit int, depth int, prefix string)
	generateSpanTree = func(ctx context.Context, count *int, limit int, depth int, prefix string) {
		numChildren := 1 + rand.Intn(5)
		for i := range numChildren {
			if *count >= limit {
				return
			}
			name := fmt.Sprintf("%s.%d", prefix, i)
			childCtx, span := tracer.Start(ctx, name)
			time.Sleep(time.Duration(1+rand.Intn(3)) * time.Millisecond)
			*count++

			if *count < limit && depth < 8 && rand.Float64() > float64(depth)*0.12 {
				generateSpanTree(childCtx, count, limit, depth+1, name)
			}

			span.End()
		}
	}

	childTask := client.NewStandaloneTask(
		"otel-child-task",
		func(ctx hatchet.Context, input ChildInput) (ChildOutput, error) {
			target := 200 + rand.Intn(101)
			count := 0
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

	parentTask := client.NewStandaloneTask(
		"otel-parent-task",
		func(ctx hatchet.Context, input ParentInput) (ParentOutput, error) {
			_, span := tracer.Start(ctx.GetContext(), "parent.prepare")
			time.Sleep(30 * time.Millisecond)
			span.End()

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
		"otel-worker",
		hatchet.WithWorkflows(parentTask, childTask),
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	worker.Use(instrumentor.Middleware())

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

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
