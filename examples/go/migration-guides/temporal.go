// Package migrationguides holds the Go code rendered into the migration
// guides. Each snippet here is the source of truth for a code block in the
// docs; edit the code, not the .mdx.
package migrationguides

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/hatchet-dev/hatchet/sdks/go/features"
)

// ValidateOutput is returned by the validate-order task.
type ValidateOutput struct {
	Valid bool `json:"valid"`
}

// FulfillOutput is returned by the fulfill-order task and by the ProcessOrder flow.
type FulfillOutput struct {
	Fulfilled   bool   `json:"fulfilled"`
	TrackingID  string `json:"tracking_id"`
	AmountCents int64  `json:"amount_cents"`
}

// OnboardingInput is the input to the onboarding durable task.
type OnboardingInput struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}

// OnboardingOutput reports how many emails the onboarding flow sent.
type OnboardingOutput struct {
	EmailsSent int `json:"emails_sent"`
}

// ApprovalEvent is the payload carried by an approval:granted event.
type ApprovalEvent struct {
	CorrelationID string `json:"correlation_id"`
	ApprovedBy    string `json:"approved_by"`
}

// ApprovalOutput is returned by the durable task that waits for an approval.
type ApprovalOutput struct {
	Approved   bool   `json:"approved"`
	ApprovedBy string `json:"approved_by"`
}

// Item is a single line item on an order.
type Item struct {
	SKU      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

// ItemInput is the input to the per-item child task.
type ItemInput struct {
	Item Item `json:"item"`
}

// ItemOutput is returned by the per-item child task.
type ItemOutput struct {
	SKU     string `json:"sku"`
	Shipped bool   `json:"shipped"`
}

// ShipmentInput is the input to the task that fans out over line items.
type ShipmentInput struct {
	OrderID string `json:"order_id"`
	Items   []Item `json:"items"`
}

// ShipmentOutput reports how many line items shipped.
type ShipmentOutput struct {
	Shipped int `json:"shipped"`
}

// ReportInput selects which report to generate.
type ReportInput struct {
	Kind string `json:"kind"`
}

// ReportOutput reports how many rows the report contained.
type ReportOutput struct {
	Rows int `json:"rows"`
}

// SyncInput identifies the customer whose records are being synced.
type SyncInput struct {
	CustomerID string `json:"customer_id"`
}

// SyncOutput reports how many records were synced.
type SyncOutput struct {
	Records int `json:"records"`
}

// PromptInput is the input to the rate-limited model call.
type PromptInput struct {
	Prompt string `json:"prompt"`
}

// ModelOutput is the completion returned by the model call.
type ModelOutput struct {
	Completion string `json:"completion"`
}

// validateOrderRecord stands in for whatever a Temporal validate_order activity did.
func validateOrderRecord(orderID string) (bool, error) {
	if orderID == "" {
		return false, errors.New("order id is required")
	}

	return true, nil
}

// capturePayment stands in for the payment provider call and returns the amount
// captured, in cents.
func capturePayment(orderID string) (int64, error) {
	if orderID == "" {
		return 0, errors.New("order id is required")
	}

	return 4999, nil
}

// shipOrder stands in for the fulfilment provider call and returns a tracking id.
func shipOrder(orderID string) (string, error) {
	if orderID == "" {
		return "", errors.New("order id is required")
	}

	return "trk_" + orderID, nil
}

// shipItem stands in for shipping a single line item.
func shipItem(item Item) (bool, error) {
	if item.Quantity <= 0 {
		return false, fmt.Errorf("item %q has no quantity", item.SKU)
	}

	return true, nil
}

// sendEmail stands in for the transactional email provider.
func sendEmail(address, template string) error {
	if address == "" {
		return errors.New("email address is required")
	}

	if template == "" {
		return errors.New("email template is required")
	}

	return nil
}

// generateReport stands in for the reporting query and returns a row count.
func generateReport(kind string) (int, error) {
	if kind == "" {
		return 0, errors.New("report kind is required")
	}

	return 128, nil
}

// syncCustomerRecords stands in for the per-customer sync and returns a record count.
func syncCustomerRecords(customerID string) (int, error) {
	if customerID == "" {
		return 0, errors.New("customer id is required")
	}

	return 42, nil
}

// completePrompt stands in for the model provider call.
func completePrompt(prompt string) (string, error) {
	if prompt == "" {
		return "", errors.New("prompt is required")
	}

	return "completion for: " + prompt, nil
}

// RunOrderWorker starts a worker serving the order tasks and the durable task
// that orchestrates them. This is the Hatchet replacement for a Temporal client
// plus a Worker bound to a task queue.
func RunOrderWorker() error {
	// > Hatchet worker
	client, err := hatchet.NewClient()
	if err != nil {
		return fmt.Errorf("creating hatchet client: %w", err)
	}

	validateOrder := NewValidateOrder(client)
	chargeOrder := NewChargeOrder(client)
	fulfillOrder := NewFulfillOrder(client)
	processOrder := NewProcessOrder(client, validateOrder, chargeOrder, fulfillOrder)

	worker, err := client.NewWorker("order-worker",
		hatchet.WithWorkflows(validateOrder, chargeOrder, fulfillOrder, processOrder),
		hatchet.WithSlots(10),
	)
	if err != nil {
		return fmt.Errorf("creating worker: %w", err)
	}

	interruptCtx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	return worker.StartBlocking(interruptCtx)
}

// > Hatchet task definition

// OrderInput is the structured input every task in the order flow receives.
// Temporal activities take positional arguments; Hatchet tasks take one value.
type OrderInput struct {
	OrderID       string `json:"order_id"`
	CorrelationID string `json:"correlation_id"`
}

// ChargeOutput is returned by the charge-order task.
type ChargeOutput struct {
	Charged     bool  `json:"charged"`
	AmountCents int64 `json:"amount_cents"`
}

// NewChargeOrder declares the charge-order task: ordinary code, retried by the
// engine, and runnable on its own without a workflow to orchestrate it.
func NewChargeOrder(client *hatchet.Client) *hatchet.StandaloneTask {
	chargeOrder := client.NewStandaloneTask("charge-order",
		func(ctx hatchet.Context, input OrderInput) (ChargeOutput, error) {
			amount, err := capturePayment(input.OrderID)
			if err != nil {
				return ChargeOutput{}, fmt.Errorf("charging order %q: %w", input.OrderID, err)
			}

			return ChargeOutput{Charged: true, AmountCents: amount}, nil
		},
	)

	return chargeOrder
}

// NewValidateOrder declares the validate-order task.
func NewValidateOrder(client *hatchet.Client) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("validate-order",
		func(ctx hatchet.Context, input OrderInput) (ValidateOutput, error) {
			valid, err := validateOrderRecord(input.OrderID)
			if err != nil {
				return ValidateOutput{}, fmt.Errorf("validating order %q: %w", input.OrderID, err)
			}

			return ValidateOutput{Valid: valid}, nil
		},
	)
}

// NewFulfillOrder declares the fulfill-order task.
func NewFulfillOrder(client *hatchet.Client) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("fulfill-order",
		func(ctx hatchet.Context, input OrderInput) (FulfillOutput, error) {
			trackingID, err := shipOrder(input.OrderID)
			if err != nil {
				return FulfillOutput{}, fmt.Errorf("fulfilling order %q: %w", input.OrderID, err)
			}

			return FulfillOutput{Fulfilled: true, TrackingID: trackingID}, nil
		},
	)
}

// NewProcessOrder declares the 1:1 translation of a Temporal workflow: a durable
// task whose body calls the three order tasks in the order the workflow called
// its activities.
func NewProcessOrder(client *hatchet.Client, validateOrder, chargeOrder, fulfillOrder *hatchet.StandaloneTask) *hatchet.StandaloneTask {
	// > Hatchet workflow as durable task
	processOrder := client.NewStandaloneDurableTask("ProcessOrder",
		func(ctx hatchet.DurableContext, input OrderInput) (FulfillOutput, error) {
			if _, err := validateOrder.Run(ctx, input); err != nil {
				return FulfillOutput{}, fmt.Errorf("validating order: %w", err)
			}

			if _, err := chargeOrder.Run(ctx, input); err != nil {
				return FulfillOutput{}, fmt.Errorf("charging order: %w", err)
			}

			result, err := fulfillOrder.Run(ctx, input)
			if err != nil {
				return FulfillOutput{}, fmt.Errorf("fulfilling order: %w", err)
			}

			var fulfilled FulfillOutput
			if err := result.Into(&fulfilled); err != nil {
				return FulfillOutput{}, fmt.Errorf("decoding fulfillment: %w", err)
			}

			return fulfilled, nil
		},
	)

	return processOrder
}

// NewSendWelcomeEmail declares the task that sends the welcome email.
func NewSendWelcomeEmail(client *hatchet.Client) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("send-welcome-email",
		func(ctx hatchet.Context, input OnboardingInput) (OnboardingOutput, error) {
			if err := sendEmail(input.Email, "welcome"); err != nil {
				return OnboardingOutput{}, fmt.Errorf("sending welcome email: %w", err)
			}

			return OnboardingOutput{EmailsSent: 1}, nil
		},
	)
}

// NewSendFollowupEmail declares the task that sends the follow-up email.
func NewSendFollowupEmail(client *hatchet.Client) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("send-followup-email",
		func(ctx hatchet.Context, input OnboardingInput) (OnboardingOutput, error) {
			if err := sendEmail(input.Email, "followup"); err != nil {
				return OnboardingOutput{}, fmt.Errorf("sending follow-up email: %w", err)
			}

			return OnboardingOutput{EmailsSent: 1}, nil
		},
	)
}

// NewOnboardingFlow declares a durable task that waits between two emails. A run
// is evicted while it sleeps, so its worker slot is released for the three days.
func NewOnboardingFlow(client *hatchet.Client, sendWelcomeEmail, sendFollowupEmail *hatchet.StandaloneTask) *hatchet.StandaloneTask {
	// > Hatchet durable task with sleep
	onboardingFlow := client.NewStandaloneDurableTask("OnboardingFlow",
		func(ctx hatchet.DurableContext, input OnboardingInput) (OnboardingOutput, error) {
			if _, err := sendWelcomeEmail.Run(ctx, input); err != nil {
				return OnboardingOutput{}, fmt.Errorf("sending welcome email: %w", err)
			}

			if _, err := ctx.SleepFor(72 * time.Hour); err != nil {
				return OnboardingOutput{}, fmt.Errorf("sleeping between emails: %w", err)
			}

			if _, err := sendFollowupEmail.Run(ctx, input); err != nil {
				return OnboardingOutput{}, fmt.Errorf("sending follow-up email: %w", err)
			}

			return OnboardingOutput{EmailsSent: 2}, nil
		},
		// The execution timeout has to cover the sleep, not just the work.
		hatchet.WithExecutionTimeout(168*time.Hour),
	)

	return onboardingFlow
}

// TriggerProcessOrder enqueues an order without waiting for it, then collects the
// result. This is the Hatchet replacement for start_workflow plus handle.result().
func TriggerProcessOrder(processOrder *hatchet.StandaloneTask) (FulfillOutput, error) {
	// > Hatchet task invocation
	runRef, err := processOrder.RunNoWait(context.Background(), OrderInput{OrderID: "123"})
	if err != nil {
		return FulfillOutput{}, fmt.Errorf("triggering process-order: %w", err)
	}

	// Store this somewhere durable if you need to reattach to the run later.
	runID := runRef.RunId

	result, err := runRef.Result()
	if err != nil {
		return FulfillOutput{}, fmt.Errorf("waiting for run %s: %w", runID, err)
	}

	var output FulfillOutput
	if err := result.TaskOutput("ProcessOrder").Into(&output); err != nil {
		return FulfillOutput{}, fmt.Errorf("decoding run %s output: %w", runID, err)
	}

	return output, nil
}

// NewChargeOrderWithRetries declares a charge-order variant carrying the settings
// that a Temporal RetryPolicy and activity options used to carry at the call site.
func NewChargeOrderWithRetries(client *hatchet.Client) *hatchet.StandaloneTask {
	// > Hatchet retries and timeouts
	chargeOrderWithRetries := client.NewStandaloneTask("charge-order-with-retries",
		func(ctx hatchet.Context, input OrderInput) (ChargeOutput, error) {
			amount, err := capturePayment(input.OrderID)
			if err != nil {
				return ChargeOutput{}, fmt.Errorf("charging order %q: %w", input.OrderID, err)
			}

			return ChargeOutput{Charged: true, AmountCents: amount}, nil
		},
		// retries counts retries, not attempts: maximum_attempts=11 becomes 10.
		hatchet.WithRetries(10),
		// factor, maxBackoffSeconds
		hatchet.WithRetryBackoff(2, 10),
		hatchet.WithExecutionTimeout(30*time.Second),
		hatchet.WithScheduleTimeout(10*time.Minute),
	)

	return chargeOrderWithRetries
}

// dailyDigest is the body of a durable task that waits a day before doing its work.
func dailyDigest(ctx hatchet.DurableContext, input OnboardingInput) (OnboardingOutput, error) {
	// > Hatchet durable sleep
	if _, err := ctx.SleepFor(24 * time.Hour); err != nil {
		return OnboardingOutput{}, fmt.Errorf("sleeping for a day: %w", err)
	}

	if err := sendEmail(input.Email, "daily-digest"); err != nil {
		return OnboardingOutput{}, fmt.Errorf("sending daily digest: %w", err)
	}

	return OnboardingOutput{EmailsSent: 1}, nil
}

// NewDailyDigest declares the durable task whose body is dailyDigest.
func NewDailyDigest(client *hatchet.Client) *hatchet.StandaloneTask {
	return client.NewStandaloneDurableTask("DailyDigest", dailyDigest,
		hatchet.WithExecutionTimeout(48*time.Hour),
	)
}

// PushApprovalGranted pushes the event that releases a waiting ApprovalFlow run.
// Unlike a Temporal signal this is not addressed to a run, so the payload carries
// the correlation id that the waiting run filters on.
func PushApprovalGranted(client *hatchet.Client, correlationID string) error {
	// > Hatchet event push
	err := client.Events().Push(context.Background(), "approval:granted", map[string]any{
		"correlation_id": correlationID,
		"approved_by":    "finance@example.com",
	})
	if err != nil {
		return fmt.Errorf("pushing approval:granted: %w", err)
	}

	return nil
}

// NewApprovalFlow declares the waiting side of a Temporal signal: a durable task
// that blocks on an event matching this run's correlation id.
func NewApprovalFlow(client *hatchet.Client) *hatchet.StandaloneTask {
	// > Hatchet durable event wait
	approvalFlow := client.NewStandaloneDurableTask("ApprovalFlow",
		func(ctx hatchet.DurableContext, input OrderInput) (ApprovalOutput, error) {
			// The expression is compiled as CEL and passed through unescaped, so
			// correlate on an opaque id you generate rather than on user input.
			expr := fmt.Sprintf("input.correlation_id == '%s'", input.CorrelationID)

			event, err := ctx.WaitForEvent("approval:granted", expr)
			if err != nil {
				return ApprovalOutput{}, fmt.Errorf("waiting for approval: %w", err)
			}

			var payload ApprovalEvent
			if err := hatchet.EventInto(event, &payload); err != nil {
				return ApprovalOutput{}, fmt.Errorf("decoding approval event: %w", err)
			}

			return ApprovalOutput{Approved: true, ApprovedBy: payload.ApprovedBy}, nil
		},
		hatchet.WithExecutionTimeout(24*time.Hour),
	)

	return approvalFlow
}

// NewProcessItem declares the child task spawned once per line item.
func NewProcessItem(client *hatchet.Client) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("process-item",
		func(ctx hatchet.Context, input ItemInput) (ItemOutput, error) {
			shipped, err := shipItem(input.Item)
			if err != nil {
				return ItemOutput{}, fmt.Errorf("shipping item %q: %w", input.Item.SKU, err)
			}

			return ItemOutput{SKU: input.Item.SKU, Shipped: shipped}, nil
		},
	)
}

// NewShipOrderItems declares a durable task that spawns one child run per line
// item. Spawning from a durable task checkpoints each child, so a crash
// mid-fan-out resumes without re-running the children that already finished.
func NewShipOrderItems(client *hatchet.Client, processItem *hatchet.StandaloneTask) *hatchet.StandaloneTask {
	return client.NewStandaloneDurableTask("ShipOrderItems",
		func(ctx hatchet.DurableContext, input ShipmentInput) (ShipmentOutput, error) {
			items := input.Items

			// > Hatchet fan out children
			var wg sync.WaitGroup

			outputs := make([]ItemOutput, len(items))
			errs := make([]error, len(items))

			wg.Add(len(items))

			for i, item := range items {
				go func(i int, item Item) {
					defer wg.Done()

					result, err := processItem.Run(ctx, ItemInput{Item: item})
					if err != nil {
						errs[i] = fmt.Errorf("running item %d: %w", i, err)
						return
					}

					if err := result.Into(&outputs[i]); err != nil {
						errs[i] = fmt.Errorf("decoding result for item %d: %w", i, err)
					}
				}(i, item)
			}

			wg.Wait()

			if err := errors.Join(errs...); err != nil {
				return ShipmentOutput{}, err
			}

			return ShipmentOutput{Shipped: len(outputs)}, nil
		},
	)
}

// NewWeeklyReport declares a task Hatchet triggers on a cron schedule. Go and
// Python are the two SDKs that can attach a fixed input to a declared cron.
func NewWeeklyReport(client *hatchet.Client) *hatchet.StandaloneTask {
	// > Hatchet cron declaration
	weeklyReport := client.NewStandaloneTask("weekly-report",
		func(ctx hatchet.Context, input ReportInput) (ReportOutput, error) {
			rows, err := generateReport(input.Kind)
			if err != nil {
				return ReportOutput{}, fmt.Errorf("generating %q report: %w", input.Kind, err)
			}

			return ReportOutput{Rows: rows}, nil
		},
		hatchet.WithWorkflowCron("0 9 * * 1"),
		hatchet.WithWorkflowCronInput(ReportInput{Kind: "weekly"}),
	)

	return weeklyReport
}

// CreateReportSchedules creates the two runtime schedule kinds through the API,
// the replacement for Temporal's programmatic schedule client.
func CreateReportSchedules(client *hatchet.Client) error {
	ctx := context.Background()

	// > Hatchet runtime schedules
	// A recurring schedule, created at runtime.
	_, err := client.Crons().Create(ctx, "weekly-report", features.CreateCronTrigger{
		Name:       "weekly-report-acme",
		Expression: "0 9 * * 1",
		Input:      map[string]interface{}{"kind": "weekly"},
		AdditionalMetadata: map[string]interface{}{
			"customer_id": "acme",
		},
	})
	if err != nil {
		return fmt.Errorf("creating cron: %w", err)
	}

	// A one-shot future run.
	_, err = client.Schedules().Create(ctx, "weekly-report", features.CreateScheduledRunTrigger{
		TriggerAt: time.Now().Add(24 * time.Hour),
		Input:     map[string]interface{}{"kind": "weekly"},
	})
	if err != nil {
		return fmt.Errorf("creating scheduled run: %w", err)
	}

	return nil
}

// NewFlowControlledTasks declares the two engine-level flow controls that replace
// partitioned task queues and an external rate limiter in front of an activity.
// The "openai" key has to exist first: see client.RateLimits().Upsert.
func NewFlowControlledTasks(client *hatchet.Client) (syncCustomer, callModel *hatchet.StandaloneTask) {
	// > Hatchet concurrency and rate limits
	// One in-flight run per customer, newest cancels the oldest.
	var maxRuns int32 = 1
	strategy := types.CancelInProgress

	syncCustomer = client.NewStandaloneTask("SyncCustomer",
		func(ctx hatchet.Context, input SyncInput) (SyncOutput, error) {
			records, err := syncCustomerRecords(input.CustomerID)
			if err != nil {
				return SyncOutput{}, fmt.Errorf("syncing customer %q: %w", input.CustomerID, err)
			}

			return SyncOutput{Records: records}, nil
		},
		hatchet.WithWorkflowConcurrency(types.Concurrency{
			Expression:    "input.customer_id",
			MaxRuns:       &maxRuns,
			LimitStrategy: &strategy,
		}),
	)

	// A global budget shared by every worker, not per-process.
	units := 1

	callModel = client.NewStandaloneTask("call-model",
		func(ctx hatchet.Context, input PromptInput) (ModelOutput, error) {
			completion, err := completePrompt(input.Prompt)
			if err != nil {
				return ModelOutput{}, fmt.Errorf("calling model: %w", err)
			}

			return ModelOutput{Completion: completion}, nil
		},
		hatchet.WithRateLimits(&types.RateLimit{
			Key:   "openai",
			Units: &units,
		}),
	)

	return syncCustomer, callModel
}

// NewOrderWorkflow declares the same order flow as a DAG. It replaces the
// ProcessOrder durable task once the migration is green: the orchestration code
// is gone, and upstream results are read off the context rather than passed
// through local variables. It registers under its own name so that it and the
// durable task it replaces can both be served while the cutover is in progress.
func NewOrderWorkflow(client *hatchet.Client) *hatchet.Workflow {
	// > Hatchet DAG workflow
	workflow := client.NewWorkflow("ProcessOrderDag")

	validate := workflow.NewTask("validate",
		func(ctx hatchet.Context, input OrderInput) (ValidateOutput, error) {
			valid, err := validateOrderRecord(input.OrderID)
			if err != nil {
				return ValidateOutput{}, fmt.Errorf("validating order %q: %w", input.OrderID, err)
			}

			return ValidateOutput{Valid: valid}, nil
		},
		hatchet.WithExecutionTimeout(30*time.Second),
	)

	charge := workflow.NewTask("charge",
		func(ctx hatchet.Context, input OrderInput) (ChargeOutput, error) {
			var validated ValidateOutput
			if err := ctx.ParentOutput(validate, &validated); err != nil {
				return ChargeOutput{}, fmt.Errorf("reading validate output: %w", err)
			}

			if !validated.Valid {
				return ChargeOutput{}, fmt.Errorf("order %q failed validation", input.OrderID)
			}

			amount, err := capturePayment(input.OrderID)
			if err != nil {
				return ChargeOutput{}, fmt.Errorf("charging order %q: %w", input.OrderID, err)
			}

			return ChargeOutput{Charged: true, AmountCents: amount}, nil
		},
		hatchet.WithParents(validate),
		hatchet.WithExecutionTimeout(30*time.Second),
	)

	_ = workflow.NewTask("fulfill",
		func(ctx hatchet.Context, input OrderInput) (FulfillOutput, error) {
			var charged ChargeOutput
			if err := ctx.ParentOutput(charge, &charged); err != nil {
				return FulfillOutput{}, fmt.Errorf("reading charge output: %w", err)
			}

			trackingID, err := shipOrder(input.OrderID)
			if err != nil {
				return FulfillOutput{}, fmt.Errorf("fulfilling order %q: %w", input.OrderID, err)
			}

			return FulfillOutput{
				Fulfilled:   true,
				TrackingID:  trackingID,
				AmountCents: charged.AmountCents,
			}, nil
		},
		hatchet.WithParents(charge),
		hatchet.WithExecutionTimeout(30*time.Second),
	)

	return workflow
}

// NewLoggedChargeOrder declares a charge-order task that writes to the run's
// built-in log sink, which replaces the logging wiring around a Temporal activity.
func NewLoggedChargeOrder(client *hatchet.Client) *hatchet.StandaloneTask {
	return client.NewStandaloneTask("charge-order-logged",
		func(ctx hatchet.Context, input OrderInput) (ChargeOutput, error) {
			// > Hatchet context logging
			ctx.Log("charging order " + input.OrderID)

			amount, err := capturePayment(input.OrderID)
			if err != nil {
				return ChargeOutput{}, fmt.Errorf("charging order %q: %w", input.OrderID, err)
			}

			return ChargeOutput{Charged: true, AmountCents: amount}, nil
		},
	)
}
