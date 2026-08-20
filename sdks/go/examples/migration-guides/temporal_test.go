//go:build !e2e && !load && !rampup && !integration

package migrationguides

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/client/create"
	"github.com/hatchet-dev/hatchet/pkg/worker"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// offlineToken builds an unsigned JWT carrying the claims the client config loader
// reads. The claims are never verified client-side, so this is enough to construct
// a client without a tenant or a running engine.
func offlineToken(t *testing.T) string {
	t.Helper()

	claims, err := json.Marshal(map[string]any{
		"server_url":             "http://localhost:8080",
		"grpc_broadcast_address": "localhost:7070",
		"sub":                    "00000000-0000-0000-0000-000000000000",
		"exp":                    time.Now().Add(time.Hour).Unix(),
	})
	require.NoError(t, err)

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))

	return header + "." + base64.RawURLEncoding.EncodeToString(claims) + ".not-a-signature"
}

// newOfflineClient returns a client suitable for declaring tasks in a test. The
// gRPC connection is lazy, so nothing here dials the engine.
func newOfflineClient(t *testing.T) *hatchet.Client {
	t.Helper()

	t.Setenv("HATCHET_CLIENT_TOKEN", offlineToken(t))
	t.Setenv("HATCHET_CLIENT_TLS_STRATEGY", "none")
	t.Setenv("HATCHET_CLIENT_LOG_LEVEL", "error")

	client, err := hatchet.NewClient()
	require.NoError(t, err)

	return client
}

// remarshal moves a value through JSON, the way Hatchet moves task inputs and
// outputs between runs.
func remarshal(from any, to any) error {
	encoded, err := json.Marshal(from)
	if err != nil {
		return err
	}

	return json.Unmarshal(encoded, to)
}

// fakeContext supplies the handful of context methods a task body actually uses.
// The embedded interface is nil, so any other method panics rather than lying.
type fakeContext struct {
	worker.HatchetContext

	input   any
	parents map[string]any
	logs    []string
}

func newFakeContext(input any) *fakeContext {
	return &fakeContext{input: input, parents: map[string]any{}}
}

func (f *fakeContext) WorkflowInput(target interface{}) error {
	return remarshal(f.input, target)
}

func (f *fakeContext) ParentOutput(parent create.NamedTask, output interface{}) error {
	value, ok := f.parents[parent.GetName()]
	if !ok {
		return fmt.Errorf("parent %s not found", parent.GetName())
	}

	return remarshal(value, output)
}

func (f *fakeContext) Log(message string) {
	f.logs = append(f.logs, message)
}

// runStandaloneTask executes a standalone task's body against ctx and decodes its
// output into out.
func runStandaloneTask(t *testing.T, task *hatchet.StandaloneTask, ctx *fakeContext, out any) error {
	t.Helper()

	_, fns, _, _ := task.Dump()
	require.Len(t, fns, 1, "expected exactly one registered function")

	result, err := fns[0].Fn(ctx)
	if err != nil {
		return err
	}

	return remarshal(result, out)
}

// runWorkflowTask executes one named task of a DAG workflow against ctx.
func runWorkflowTask(t *testing.T, workflow *hatchet.Workflow, name string, ctx *fakeContext, out any) error {
	t.Helper()

	req, fns, _, _ := workflow.Dump()

	for i, task := range req.Tasks {
		if task.ReadableId != name {
			continue
		}

		result, err := fns[i].Fn(ctx)
		if err != nil {
			return err
		}

		return remarshal(result, out)
	}

	t.Fatalf("task %q not found in workflow %q", name, req.Name)

	return nil
}

// taskOpts returns the registration options for a single-task workflow.
func taskOpts(t *testing.T, req *v1.CreateWorkflowVersionRequest) *v1.CreateTaskOpts {
	t.Helper()

	require.Len(t, req.Tasks, 1)

	return req.Tasks[0]
}

func TestChargeOrderProducesTypedOutput(t *testing.T) {
	client := newOfflineClient(t)

	var output ChargeOutput
	err := runStandaloneTask(t, NewChargeOrder(client), newFakeContext(OrderInput{OrderID: "order-1"}), &output)

	require.NoError(t, err)
	assert.Equal(t, ChargeOutput{Charged: true, AmountCents: 4999}, output)
}

func TestChargeOrderWrapsFailures(t *testing.T) {
	client := newOfflineClient(t)

	var output ChargeOutput
	err := runStandaloneTask(t, NewChargeOrder(client), newFakeContext(OrderInput{}), &output)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "charging order")
}

func TestOrderTaskNames(t *testing.T) {
	client := newOfflineClient(t)

	assert.Equal(t, "validate-order", NewValidateOrder(client).GetName())
	assert.Equal(t, "charge-order", NewChargeOrder(client).GetName())
	assert.Equal(t, "fulfill-order", NewFulfillOrder(client).GetName())
	assert.Equal(t, "process-item", NewProcessItem(client).GetName())
	assert.Equal(t, "weekly-report", NewWeeklyReport(client).GetName())
}

// The DAG in step 13 has to describe the same order flow as the durable task in
// step 4, so the two are checked against the same task names.
func TestDAGWorkflowMatchesDurableTask(t *testing.T) {
	client := newOfflineClient(t)

	durable := NewProcessOrder(client,
		NewValidateOrder(client),
		NewChargeOrder(client),
		NewFulfillOrder(client),
	)

	workflow := NewOrderWorkflow(client)
	req, _, _, _ := workflow.Dump()

	assert.Equal(t, "ProcessOrder", durable.GetName())
	assert.Equal(t, "ProcessOrderDag", workflow.GetName())

	names := make([]string, 0, len(req.Tasks))
	parents := map[string][]string{}

	for _, task := range req.Tasks {
		names = append(names, task.ReadableId)
		parents[task.ReadableId] = task.Parents
	}

	assert.Equal(t, []string{"validate", "charge", "fulfill"}, names)
	assert.Empty(t, parents["validate"])
	assert.Equal(t, []string{"validate"}, parents["charge"])
	assert.Equal(t, []string{"charge"}, parents["fulfill"])

	for _, task := range req.Tasks {
		assert.Equal(t, "30s", task.Timeout, "task %s", task.ReadableId)
	}
}

func TestDAGWorkflowDataFlow(t *testing.T) {
	client := newOfflineClient(t)
	workflow := NewOrderWorkflow(client)

	input := OrderInput{OrderID: "order-1"}
	ctx := newFakeContext(input)

	var validated ValidateOutput
	require.NoError(t, runWorkflowTask(t, workflow, "validate", ctx, &validated))
	assert.True(t, validated.Valid)

	ctx.parents["validate"] = validated

	var charged ChargeOutput
	require.NoError(t, runWorkflowTask(t, workflow, "charge", ctx, &charged))
	assert.Equal(t, ChargeOutput{Charged: true, AmountCents: 4999}, charged)

	ctx.parents["charge"] = charged

	var fulfilled FulfillOutput
	require.NoError(t, runWorkflowTask(t, workflow, "fulfill", ctx, &fulfilled))

	// The amount only reaches fulfill through ctx.ParentOutput, which is the whole
	// point of the DAG rewrite.
	assert.Equal(t, FulfillOutput{
		Fulfilled:   true,
		TrackingID:  "trk_order-1",
		AmountCents: 4999,
	}, fulfilled)
}

func TestDAGChargeRejectsInvalidOrder(t *testing.T) {
	client := newOfflineClient(t)
	workflow := NewOrderWorkflow(client)

	ctx := newFakeContext(OrderInput{OrderID: "order-1"})
	ctx.parents["validate"] = ValidateOutput{Valid: false}

	var charged ChargeOutput
	err := runWorkflowTask(t, workflow, "charge", ctx, &charged)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed validation")
}

func TestDAGChargeFailsWithoutParentOutput(t *testing.T) {
	client := newOfflineClient(t)
	workflow := NewOrderWorkflow(client)

	var charged ChargeOutput
	err := runWorkflowTask(t, workflow, "charge", newFakeContext(OrderInput{OrderID: "order-1"}), &charged)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading validate output")
}

func TestRetriesAndTimeouts(t *testing.T) {
	client := newOfflineClient(t)

	retryingTask := NewChargeOrderWithRetries(client)
	req, _, _, _ := retryingTask.Dump()
	opts := taskOpts(t, req)

	// The retrying variant needs its own name: "charge-order" is already taken by
	// the plain task, and two workflows cannot share a name on one worker.
	assert.Equal(t, "charge-order-with-retries", retryingTask.GetName())
	assert.Equal(t, int32(10), opts.Retries)
	assert.Equal(t, float32(2), opts.GetBackoffFactor())
	assert.Equal(t, int32(10), opts.GetBackoffMaxSeconds())
	assert.Equal(t, "30s", opts.Timeout)
	assert.Equal(t, "600s", opts.GetScheduleTimeout())
}

func TestCronDeclaration(t *testing.T) {
	client := newOfflineClient(t)

	req, _, _, _ := NewWeeklyReport(client).Dump()

	assert.Equal(t, []string{"0 9 * * 1"}, req.CronTriggers)
	assert.JSONEq(t, `{"kind":"weekly"}`, req.GetCronInput())
}

func TestConcurrencyAndRateLimits(t *testing.T) {
	client := newOfflineClient(t)

	syncCustomer, callModel := NewFlowControlledTasks(client)

	syncReq, _, _, _ := syncCustomer.Dump()
	require.Len(t, syncReq.ConcurrencyArr, 1)
	assert.Equal(t, "input.customer_id", syncReq.ConcurrencyArr[0].Expression)
	assert.Equal(t, int32(1), syncReq.ConcurrencyArr[0].GetMaxRuns())
	assert.Equal(t, v1.ConcurrencyLimitStrategy_CANCEL_IN_PROGRESS, syncReq.ConcurrencyArr[0].GetLimitStrategy())

	modelReq, _, _, _ := callModel.Dump()
	modelOpts := taskOpts(t, modelReq)
	require.Len(t, modelOpts.RateLimits, 1)
	assert.Equal(t, "openai", modelOpts.RateLimits[0].Key)
	assert.Equal(t, int32(1), modelOpts.RateLimits[0].GetUnits())
}

// The durable tasks that sleep or wait cannot be executed without an engine, so
// this asserts their registration instead. Nothing here waits three days.
func TestDurableTaskRegistration(t *testing.T) {
	client := newOfflineClient(t)

	onboarding := NewOnboardingFlow(client, NewSendWelcomeEmail(client), NewSendFollowupEmail(client))

	for _, tc := range []struct {
		name    string
		task    *hatchet.StandaloneTask
		timeout string
	}{
		{name: "OnboardingFlow", task: onboarding, timeout: "604800s"},
		{name: "ApprovalFlow", task: NewApprovalFlow(client), timeout: "86400s"},
		{name: "DailyDigest", task: NewDailyDigest(client), timeout: "172800s"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, regular, durable, _ := tc.task.Dump()

			assert.Equal(t, tc.name, tc.task.GetName())
			assert.Empty(t, regular)
			assert.Len(t, durable, 1)
			assert.True(t, taskOpts(t, req).IsDurable)
			assert.Equal(t, tc.timeout, taskOpts(t, req).Timeout)
		})
	}
}

func TestContextLoggingWritesToLogSink(t *testing.T) {
	client := newOfflineClient(t)

	ctx := newFakeContext(OrderInput{OrderID: "order-1"})

	var output ChargeOutput
	require.NoError(t, runStandaloneTask(t, NewLoggedChargeOrder(client), ctx, &output))

	assert.Equal(t, []string{"charging order order-1"}, ctx.logs)
	assert.True(t, output.Charged)
}

func TestProcessItemShipsLineItem(t *testing.T) {
	client := newOfflineClient(t)

	var output ItemOutput
	err := runStandaloneTask(t, NewProcessItem(client),
		newFakeContext(ItemInput{Item: Item{SKU: "sku-1", Quantity: 2}}), &output)

	require.NoError(t, err)
	assert.Equal(t, ItemOutput{SKU: "sku-1", Shipped: true}, output)
}

func TestProcessItemRejectsEmptyLineItem(t *testing.T) {
	client := newOfflineClient(t)

	var output ItemOutput
	err := runStandaloneTask(t, NewProcessItem(client),
		newFakeContext(ItemInput{Item: Item{SKU: "sku-1"}}), &output)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "shipping item")
}

func TestWeeklyReportProducesRowCount(t *testing.T) {
	client := newOfflineClient(t)

	var output ReportOutput
	err := runStandaloneTask(t, NewWeeklyReport(client), newFakeContext(ReportInput{Kind: "weekly"}), &output)

	require.NoError(t, err)
	assert.Equal(t, ReportOutput{Rows: 128}, output)
}

func TestFlowControlledTaskBodies(t *testing.T) {
	client := newOfflineClient(t)

	syncCustomer, callModel := NewFlowControlledTasks(client)

	var synced SyncOutput
	require.NoError(t, runStandaloneTask(t, syncCustomer, newFakeContext(SyncInput{CustomerID: "acme"}), &synced))
	assert.Equal(t, SyncOutput{Records: 42}, synced)

	var completion ModelOutput
	require.NoError(t, runStandaloneTask(t, callModel, newFakeContext(PromptInput{Prompt: "hello"}), &completion))
	assert.Equal(t, ModelOutput{Completion: "completion for: hello"}, completion)
}
