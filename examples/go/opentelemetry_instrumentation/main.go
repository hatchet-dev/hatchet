package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go.opentelemetry.io/otel"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	hatchetotel "github.com/hatchet-dev/hatchet/sdks/go/opentelemetry"
)

type OrderInput struct {
	OrderID    string `json:"orderId"`
	CustomerID string `json:"customerId"`
	Amount     int    `json:"amount"`
}

type ValidateOrderOutput struct {
	OrderID string `json:"orderId"`
	Valid   bool   `json:"valid"`
}

type ChargePaymentOutput struct {
	TransactionID string `json:"transactionId"`
	Charged       int    `json:"charged"`
}

type ReserveInventoryOutput struct {
	ReservationID string `json:"reservationId"`
	ItemsReserved int    `json:"itemsReserved"`
}

type SendConfirmationOutput struct {
	Sent bool `json:"sent"`
}

type NotifyInput struct {
	OrderID       string `json:"orderId"`
	TransactionID string `json:"transactionId"`
	Channel       string `json:"channel"`
}

type NotifyOutput struct {
	Delivered bool   `json:"delivered"`
	Channel   string `json:"channel"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	// > Setup
	// Set up the OTel instrumentor — by default it creates a TracerProvider that
	// sends spans to the Hatchet engine's OTLP collector. The instrumentor also
	// provides middleware that creates a root span per task run and propagates
	// hatchet.* attributes to all child spans.
	instrumentor, err := hatchetotel.NewInstrumentor()
	if err != nil {
		log.Fatalf("failed to create instrumentor: %v", err)
	}

	// Use the global tracer for creating custom child spans inside tasks.
	// These will inherit hatchet.* attributes from the parent task run span.
	tracer := otel.Tracer("otel-instrumentation-example")

	// Standalone task for sending notifications — spawned as a child workflow
	// from send-confirmation below. Each notification gets its own trace subtree.
	notifyTask := client.NewStandaloneTask(
		"otel-send-notification",
		func(ctx hatchet.Context, input NotifyInput) (NotifyOutput, error) {
			// Render the notification template
			_, span := tracer.Start(ctx, "notification.render-template")
			time.Sleep(2 * time.Second)
			span.End()

			// Deliver via the requested channel (email, sms, etc.)
			_, span = tracer.Start(ctx, fmt.Sprintf("notification.deliver.%s", input.Channel))
			time.Sleep(3 * time.Second)
			span.End()

			return NotifyOutput{
				Delivered: true,
				Channel:   input.Channel,
			}, nil
		},
	)

	workflow := client.NewWorkflow("otel-order-processing")

	// > Custom Spans
	// Step 1: Validate the incoming order (schema + fraud check).
	validateOrder := workflow.NewTask(
		"validate-order",
		func(ctx hatchet.Context, input OrderInput) (ValidateOrderOutput, error) {
			// Validate the order schema
			_, span := tracer.Start(ctx, "order.validate.schema")
			time.Sleep(2 * time.Second)
			span.End()

			// Run a fraud check against an external service
			_, span = tracer.Start(ctx, "order.validate.fraud-check")
			time.Sleep(3 * time.Second)
			span.End()

			return ValidateOrderOutput{
				Valid:   true,
				OrderID: input.OrderID,
			}, nil
		},
	)

	// Step 2a: Charge the customer's payment method (runs after validate-order).
	chargePayment := workflow.NewTask(
		"charge-payment",
		func(ctx hatchet.Context, input OrderInput) (ChargePaymentOutput, error) {
			var validated ValidateOrderOutput
			if err := ctx.ParentOutput(validateOrder, &validated); err != nil {
				return ChargePaymentOutput{}, err
			}

			// Parent span wrapping the full payment flow
			payCtx, paySpan := tracer.Start(ctx, "payment.process")
			defer paySpan.End()

			// Tokenize the card before charging
			_, tokenSpan := tracer.Start(payCtx, "payment.tokenize-card")
			time.Sleep(2 * time.Second)
			tokenSpan.End()

			// Charge the tokenized card
			_, chargeSpan := tracer.Start(payCtx, "payment.charge")
			time.Sleep(4 * time.Second)
			chargeSpan.End()

			return ChargePaymentOutput{
				TransactionID: fmt.Sprintf("txn-%s", validated.OrderID),
				Charged:       input.Amount,
			}, nil
		},
		hatchet.WithParents(validateOrder),
	)

	// Step 2b: Reserve inventory (runs in parallel with charge-payment after validate-order).
	reserveInventory := workflow.NewTask(
		"reserve-inventory",
		func(ctx hatchet.Context, input OrderInput) (ReserveInventoryOutput, error) {
			// Check if items are available
			_, span := tracer.Start(ctx, "inventory.check-availability")
			time.Sleep(2 * time.Second)
			span.End()

			// Lock the inventory for this order
			_, span = tracer.Start(ctx, "inventory.reserve")
			time.Sleep(3 * time.Second)
			span.End()

			return ReserveInventoryOutput{
				ReservationID: fmt.Sprintf("res-%s", input.OrderID),
				ItemsReserved: 3,
			}, nil
		},
		hatchet.WithParents(validateOrder),
	)

	// Step 3: Send order confirmation (runs after both payment and inventory are done).
	// Spawns multiple child workflows concurrently to test parallel Run() spans.
	_ = workflow.NewTask(
		"send-confirmation",
		func(ctx hatchet.Context, input OrderInput) (SendConfirmationOutput, error) {
			var payment ChargePaymentOutput
			if err := ctx.ParentOutput(chargePayment, &payment); err != nil {
				return SendConfirmationOutput{}, err
			}

			var inventory ReserveInventoryOutput
			if err := ctx.ParentOutput(reserveInventory, &inventory); err != nil {
				return SendConfirmationOutput{}, err
			}

			channels := []string{"email", "sms", "push", "slack", "webhook"}
			numNotifications := 10

			var wg sync.WaitGroup
			var mu sync.Mutex
			var firstErr error

			for i := 0; i < numNotifications; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()

					channel := channels[idx%len(channels)]
					_, runErr := notifyTask.Run(ctx, NotifyInput{
						OrderID:       input.OrderID,
						TransactionID: payment.TransactionID,
						Channel:       channel,
					})
					if runErr != nil {
						mu.Lock()
						if firstErr == nil {
							firstErr = runErr
						}
						mu.Unlock()
					}
				}(i)
			}

			wg.Wait()

			if firstErr != nil {
				return SendConfirmationOutput{}, fmt.Errorf("notification failed: %w", firstErr)
			}

			return SendConfirmationOutput{Sent: true}, nil
		},
		hatchet.WithParents(chargePayment, reserveInventory),
	)

	// > Middleware
	worker, err := client.NewWorker(
		"otel-instrumentation-worker",
		hatchet.WithWorkflows(workflow, notifyTask),
	)
	if err != nil {
		log.Fatalf("failed to create worker: %v", err)
	}

	// Register the OTel middleware so every task run gets a root span
	worker.Use(instrumentor.Middleware())

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	// > Shutdown
	// Flush remaining spans on shutdown
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
