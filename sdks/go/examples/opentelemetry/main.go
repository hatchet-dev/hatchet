package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2" //nolint:gosec // G404: example code, not security-sensitive
	"time"

	"go.opentelemetry.io/otel"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	hatchetotel "github.com/hatchet-dev/hatchet/sdks/go/opentelemetry"
)

type PipelineInput struct {
	URL string `json:"url"`
}

type FetchOutput struct {
	Data string `json:"data"`
}

type ValidateOutput struct {
	Valid      bool `json:"valid"`
	FieldCount int  `json:"field_count"`
}

type ProcessOutput struct {
	ProcessedData string `json:"processed_data"`
	RecordCount   int    `json:"record_count"`
}

type SaveOutput struct {
	Location     string `json:"location"`
	RecordsSaved int    `json:"records_saved"`
}

func randMillis(base, jitter int) time.Duration {
	return time.Duration(base+rand.IntN(jitter)) * time.Millisecond //nolint:gosec // G404
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	// Set up OpenTelemetry instrumentation.
	// EnableHatchetCollector() auto-configures from the same env vars as the client
	// (HATCHET_CLIENT_HOST_PORT, HATCHET_CLIENT_TOKEN, HATCHET_CLIENT_TLS_STRATEGY).
	instrumentor, err := hatchetotel.NewInstrumentor(
		hatchetotel.EnableHatchetCollector(),
	)
	if err != nil {
		log.Fatalf("failed to create instrumentor: %v", err)
	}

	tracer := otel.Tracer("otel-example")

	// Create a multi-task workflow
	workflow := client.NewWorkflow("otel-data-pipeline")

	fetchData := workflow.NewTask("fetch-data", func(ctx hatchet.Context, input PipelineInput) (*FetchOutput, error) {
		_, span := tracer.Start(ctx.GetContext(), fmt.Sprintf("GET %s", input.URL))
		time.Sleep(randMillis(10, 20))
		span.End()

		_, parseSpan := tracer.Start(ctx.GetContext(), "json.parse")
		time.Sleep(randMillis(5, 10))
		parseSpan.End()

		return &FetchOutput{
			Data: `{"users": [{"name": "Alice"}, {"name": "Bob"}]}`,
		}, nil
	})

	validateData := workflow.NewTask("validate-data", func(ctx hatchet.Context, input PipelineInput) (*ValidateOutput, error) {
		var parentOutput FetchOutput
		if parentErr := ctx.ParentOutput(fetchData, &parentOutput); parentErr != nil {
			return nil, parentErr
		}

		_, span := tracer.Start(ctx.GetContext(), "schema.validate")
		time.Sleep(randMillis(5, 10))

		var parsed map[string]any
		if unmarshalErr := json.Unmarshal([]byte(parentOutput.Data), &parsed); unmarshalErr != nil {
			span.End()
			return nil, fmt.Errorf("invalid JSON: %w", unmarshalErr)
		}
		span.End()

		return &ValidateOutput{
			Valid:      true,
			FieldCount: len(parsed),
		}, nil
	}, hatchet.WithParents(fetchData))

	processData := workflow.NewTask("process-data", func(ctx hatchet.Context, input PipelineInput) (*ProcessOutput, error) {
		var validateOutput ValidateOutput
		if parentErr := ctx.ParentOutput(validateData, &validateOutput); parentErr != nil {
			return nil, parentErr
		}

		_, span := tracer.Start(ctx.GetContext(), "data.transform")
		time.Sleep(randMillis(10, 15))
		span.End()

		_, enrichSpan := tracer.Start(ctx.GetContext(), "data.enrich")
		time.Sleep(randMillis(5, 10))
		enrichSpan.End()

		return &ProcessOutput{
			ProcessedData: "transformed_and_enriched",
			RecordCount:   validateOutput.FieldCount,
		}, nil
	}, hatchet.WithParents(validateData))

	workflow.NewTask("save-results", func(ctx hatchet.Context, input PipelineInput) (*SaveOutput, error) {
		var processOutput ProcessOutput
		if parentErr := ctx.ParentOutput(processData, &processOutput); parentErr != nil {
			return nil, parentErr
		}

		_, span := tracer.Start(ctx.GetContext(), "db.insert")
		time.Sleep(randMillis(10, 20))
		span.End()

		return &SaveOutput{
			RecordsSaved: processOutput.RecordCount,
			Location:     "postgresql://localhost/pipeline_results",
		}, nil
	}, hatchet.WithParents(processData))

	// Create worker and register the OTel middleware
	worker, err := client.NewWorker("otel-worker", hatchet.WithWorkflows(workflow))
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	worker.Use(instrumentor.Middleware())

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	fmt.Println("Starting worker with OpenTelemetry instrumentation...")

	go func() {
		<-interruptCtx.Done()
		// Flush remaining spans before exit
		if shutdownErr := instrumentor.Shutdown(context.Background()); shutdownErr != nil {
			log.Printf("failed to shutdown instrumentor: %v", shutdownErr)
		}
	}()

	if startErr := worker.StartBlocking(interruptCtx); startErr != nil {
		log.Printf("worker error: %v", startErr)
	}
}
