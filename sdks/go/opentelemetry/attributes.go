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
		attribute.String(AttrInstrumentor, AttrInstrumentorValue),
		attribute.String(AttrTenantID, ctx.TenantId()),
		attribute.String(AttrWorkerID, ctx.WorkerId()),
		attribute.String(AttrWorkflowRunID, ctx.WorkflowRunId()),
		attribute.String(AttrStepRunID, ctx.StepRunId()),
		attribute.String(AttrStepID, ctx.StepId()),
		attribute.String(AttrActionName, ctx.ActionId()),
		attribute.String(AttrStepName, ctx.StepName()),
		attribute.Int(AttrRetryCount, ctx.RetryCount()),
	}

	if wfID := ctx.WorkflowId(); wfID != nil {
		attrs = append(attrs, attribute.String(AttrWorkflowID, *wfID))
	}

	if wfVersionID := ctx.WorkflowVersionId(); wfVersionID != nil {
		attrs = append(attrs, attribute.String(AttrWorkflowVersionID, *wfVersionID))
	}

	if parentID := ctx.ParentWorkflowRunId(); parentID != nil {
		attrs = append(attrs, attribute.String(AttrParentWorkflowRunID, *parentID))
	}

	if childIdx := ctx.ChildIndex(); childIdx != nil {
		attrs = append(attrs, attribute.Int(AttrChildWorkflowIndex, int(*childIdx)))
	}

	if childKey := ctx.ChildKey(); childKey != nil {
		attrs = append(attrs, attribute.String(AttrChildWorkflowKey, *childKey))
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
