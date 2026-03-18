package main

import (
	"context"
	"fmt"
	"log"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	hatchetotel "github.com/hatchet-dev/hatchet/sdks/go/opentelemetry"
)

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	// > Trigger
	instrumentor, err := hatchetotel.NewInstrumentor()
	if err != nil {
		log.Fatalf("failed to create instrumentor: %v", err)
	}
	defer func() {
		if shutdownErr := instrumentor.Shutdown(context.Background()); shutdownErr != nil {
			log.Printf("failed to shutdown instrumentor: %v", shutdownErr)
		}
	}()

	// Trigger the workflow — this creates a hatchet.run_workflow span on the
	// client side with traceparent injected into additional_metadata. The worker
	// side picks up that traceparent and creates child spans under the same trace.
	ref, err := client.Run(context.Background(), "otel-order-processing", map[string]any{
		"orderId":    "order-123",
		"customerId": "cust-456",
		"amount":     4200,
	})
	if err != nil {
		log.Fatalf("failed to trigger workflow: %v", err)
	}

	fmt.Printf("triggered workflow run: %s\n", ref.RunId)

	// Wait a bit for spans to flush
	time.Sleep(2 * time.Second)
	fmt.Println("done — check the trace view for both trigger and worker spans")
}
