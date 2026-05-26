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
	I             int    `json:"i"`
}

type NotifyOutput struct {
	Delivered bool   `json:"delivered"`
	Channel   string `json:"channel"`
}

type AuditEvent struct {
	OrderID string `json:"orderId"`
	Action  string `json:"action"`
}

type AuditOutput struct {
	Logged bool `json:"logged"`
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	instrumentor, err := hatchetotel.NewInstrumentor()
	if err != nil {
		log.Fatalf("failed to create instrumentor: %v", err)
	}

	tracer := otel.Tracer("otel-instrumentation-example")

	// Child workflow for sending notifications via a specific channel
	notifyTask := client.NewStandaloneTask(
		"otel-send-notification",
		func(ctx hatchet.Context, input NotifyInput) (NotifyOutput, error) {
			_, span := tracer.Start(ctx, "notification.render-template")
			time.Sleep(2 * time.Second)
			span.End()

			_, span = tracer.Start(ctx, fmt.Sprintf("notification.deliver.%s", input.Channel))
			time.Sleep(3 * time.Second)
			span.End()

			if input.I == 10 {
				return NotifyOutput{}, fmt.Errorf("test error")
			}

			return NotifyOutput{
				Delivered: true,
				Channel:   input.Channel,
			}, nil
		},
	)

	// Child workflow for sending notifications via a specific channel
	otherTask := client.NewStandaloneTask(
		"otel-other-task",
		func(ctx hatchet.Context, input NotifyInput) (NotifyOutput, error) {
			_, span := tracer.Start(ctx, "notification.render-template")
			time.Sleep(50 * time.Millisecond)
			span.End()

			_, span = tracer.Start(ctx, fmt.Sprintf("notification.deliver.%s", input.Channel))
			time.Sleep(200 * time.Millisecond)
			span.End()

			return NotifyOutput{
				Delivered: true,
				Channel:   input.Channel,
			}, nil
		},
	)

	auditWorkflow := client.NewWorkflow(
		"otel-audit-log",
		hatchet.WithWorkflowEvents("order:payment_charged"),
	)

	auditWorkflow.NewTask(
		"write-audit-entry",
		func(ctx hatchet.Context, input AuditEvent) (AuditOutput, error) {
			_, span := tracer.Start(ctx, "audit.persist")
			time.Sleep(100 * time.Millisecond)
			span.End()

			return AuditOutput{Logged: true}, nil
		},
	)

	workflow := client.NewWorkflow("otel-order-processing")

	validateOrder := workflow.NewTask(
		"validate-order",
		func(ctx hatchet.Context, input OrderInput) (ValidateOrderOutput, error) {
			_, span := tracer.Start(ctx, "order.validate.schema")
			time.Sleep(10 * time.Millisecond)
			span.End()

			_, span = tracer.Start(ctx, "order.validate.fraud-check")
			time.Sleep(20 * time.Millisecond)
			span.End()

			return ValidateOrderOutput{
				Valid:   true,
				OrderID: input.OrderID,
			}, nil
		},
	)

	chargePayment := workflow.NewTask(
		"charge-payment",
		func(ctx hatchet.Context, input OrderInput) (ChargePaymentOutput, error) {
			var validated ValidateOrderOutput
			if err := ctx.ParentOutput(validateOrder, &validated); err != nil {
				return ChargePaymentOutput{}, err
			}

			payCtx, paySpan := tracer.Start(ctx, "payment.process")

			_, tokenSpan := tracer.Start(payCtx, "payment.tokenize-card")
			time.Sleep(200 * time.Millisecond)
			tokenSpan.End()

			_, chargeSpan := tracer.Start(payCtx, "payment.charge")
			time.Sleep(400 * time.Millisecond)
			chargeSpan.End()

			paySpan.End()

			if err := client.Events().Push(ctx, "order:payment_charged", AuditEvent{
				OrderID: validated.OrderID,
				Action:  "payment_charged",
			}); err != nil {
				log.Printf("failed to push audit event: %v", err)
			}

			return ChargePaymentOutput{
				TransactionID: fmt.Sprintf("txn-%s", validated.OrderID),
				Charged:       input.Amount,
			}, nil
		},
		hatchet.WithParents(validateOrder),
	)

	reserveInventory := workflow.NewTask(
		"reserve-inventory",
		func(ctx hatchet.Context, input OrderInput) (ReserveInventoryOutput, error) {
			_, span := tracer.Start(ctx, "inventory.check-availability")
			time.Sleep(2 * time.Second)
			span.End()

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
		"dag-confirmation",
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
						I:             idx,
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

			_, runErr := otherTask.Run(ctx, NotifyInput{
				OrderID:       input.OrderID,
				TransactionID: payment.TransactionID,
				Channel:       "email",
				I:             1,
			})
			if runErr != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = runErr
				}
				mu.Unlock()
			}

			wg.Wait()

			if firstErr != nil {
				return SendConfirmationOutput{}, fmt.Errorf("notification failed: %w", firstErr)
			}

			return SendConfirmationOutput{Sent: true}, nil
		},
		hatchet.WithParents(chargePayment, reserveInventory),
	)

	worker, err := client.NewWorker(
		"otel-instrumentation-worker",
		hatchet.WithWorkflows(workflow, notifyTask, otherTask, auditWorkflow),
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
