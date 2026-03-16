package main

import (
	"context"
	"fmt"
	"log"
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

	tracer := otel.Tracer("otel-dag-example")

	workflow := client.NewWorkflow("otel-order-processing")

	validateOrder := workflow.NewTask(
		"validate-order",
		func(ctx hatchet.Context, input OrderInput) (ValidateOrderOutput, error) {
			_, span := tracer.Start(ctx.GetContext(), "order.validate.schema")
			time.Sleep(10 * time.Millisecond)
			span.End()

			_, span = tracer.Start(ctx.GetContext(), "order.validate.fraud-check")
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

			payCtx, paySpan := tracer.Start(ctx.GetContext(), "payment.process")

			_, tokenSpan := tracer.Start(payCtx, "payment.tokenize-card")
			time.Sleep(15 * time.Millisecond)
			tokenSpan.End()

			_, chargeSpan := tracer.Start(payCtx, "payment.charge")
			time.Sleep(30 * time.Millisecond)
			chargeSpan.End()

			paySpan.End()

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
			_, span := tracer.Start(ctx.GetContext(), "inventory.check-availability")
			time.Sleep(10 * time.Millisecond)
			span.End()

			_, span = tracer.Start(ctx.GetContext(), "inventory.reserve")
			time.Sleep(15 * time.Millisecond)
			span.End()

			return ReserveInventoryOutput{
				ReservationID: fmt.Sprintf("res-%s", input.OrderID),
				ItemsReserved: 3,
			}, nil
		},
		hatchet.WithParents(validateOrder),
	)

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

			_, span := tracer.Start(ctx.GetContext(), "notification.render-template")
			time.Sleep(5 * time.Millisecond)
			span.End()

			_, span = tracer.Start(ctx.GetContext(), "notification.send-email")
			time.Sleep(20 * time.Millisecond)
			span.End()

			return SendConfirmationOutput{Sent: true}, nil
		},
		hatchet.WithParents(chargePayment, reserveInventory),
	)

	worker, err := client.NewWorker(
		"otel-dag-worker",
		hatchet.WithWorkflows(workflow),
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
