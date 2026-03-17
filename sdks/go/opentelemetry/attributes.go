package opentelemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/hatchet-dev/hatchet/pkg/worker"
)

type hatchetAttrsKeyType struct{}

var hatchetAttrsKey = hatchetAttrsKeyType{}

// hatchetAttributes builds the set of hatchet.* span attributes from a HatchetContext.
func hatchetAttributes(ctx worker.HatchetContext) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		attribute.String("instrumentor", "hatchet"),
		attribute.String("hatchet.tenant_id", ctx.TenantId()),
		attribute.String("hatchet.worker_id", ctx.WorkerId()),
		attribute.String("hatchet.workflow_run_id", ctx.WorkflowRunId()),
		attribute.String("hatchet.step_run_id", ctx.StepRunId()),
		attribute.String("hatchet.step_id", ctx.StepId()),
		attribute.String("hatchet.action_id", ctx.ActionId()),
		attribute.String("hatchet.action_name", ctx.ActionId()),
		attribute.String("hatchet.step_name", ctx.StepName()),
		attribute.Int("hatchet.retry_count", ctx.RetryCount()),
	}

	if wfID := ctx.WorkflowId(); wfID != nil {
		attrs = append(attrs, attribute.String("hatchet.workflow_id", *wfID))
	}

	if wfVersionID := ctx.WorkflowVersionId(); wfVersionID != nil {
		attrs = append(attrs, attribute.String("hatchet.workflow_version_id", *wfVersionID))
	}

	if parentID := ctx.ParentWorkflowRunId(); parentID != nil {
		attrs = append(attrs, attribute.String("hatchet.parent_workflow_run_id", *parentID))
	}

	if childIdx := ctx.ChildIndex(); childIdx != nil {
		attrs = append(attrs, attribute.Int("hatchet.child_workflow_index", int(*childIdx)))
	}

	if childKey := ctx.ChildKey(); childKey != nil {
		attrs = append(attrs, attribute.String("hatchet.child_workflow_key", *childKey))
	}

	return attrs
}

// withHatchetAttributes stores hatchet attributes in the context so the
// SpanProcessor can inject them into child spans.
func withHatchetAttributes(ctx context.Context, attrs []attribute.KeyValue) context.Context {
	return context.WithValue(ctx, hatchetAttrsKey, attrs)
}

// getHatchetAttributes retrieves hatchet attributes from the context.
func getHatchetAttributes(ctx context.Context) []attribute.KeyValue {
	attrs, _ := ctx.Value(hatchetAttrsKey).([]attribute.KeyValue)
	return attrs
}
