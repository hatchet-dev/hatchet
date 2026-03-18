package hatchetotel

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/attribute"

	"github.com/hatchet-dev/hatchet/pkg/worker"
)

// OTelAttribute represents a hatchet span attribute that can be included or excluded.
type OTelAttribute string

const (
	AttrActionName          OTelAttribute = "action_name"
	AttrStepName            OTelAttribute = "step_name"
	AttrChildWorkflowIndex  OTelAttribute = "child_workflow_index"
	AttrChildWorkflowKey    OTelAttribute = "child_workflow_key"
	AttrParentWorkflowRunID OTelAttribute = "parent_workflow_run_id"
	AttrRetryCount          OTelAttribute = "retry_count"
	AttrStepID              OTelAttribute = "step_id"
	AttrStepRunID           OTelAttribute = "step_run_id"
	AttrTenantID            OTelAttribute = "tenant_id"
	AttrWorkerID            OTelAttribute = "worker_id"
	AttrWorkflowID          OTelAttribute = "workflow_id"
	AttrWorkflowRunID       OTelAttribute = "workflow_run_id"
	AttrWorkflowVersionID   OTelAttribute = "workflow_version_id"
	AttrActionID            OTelAttribute = "action_id"
	AttrPayload             OTelAttribute = "payload"
)

type hatchetAttrsKeyType struct{}

var hatchetAttrsKey = hatchetAttrsKeyType{}

// hatchetAttributes builds the set of hatchet.* span attributes from a HatchetContext.
// Attributes whose name (without the "hatchet." prefix) appears in excluded are omitted.
func hatchetAttributes(ctx worker.HatchetContext, excluded map[OTelAttribute]bool) []attribute.KeyValue {
	all := []attribute.KeyValue{
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
		all = append(all, attribute.String("hatchet.workflow_id", *wfID))
	}

	if wfVersionID := ctx.WorkflowVersionId(); wfVersionID != nil {
		all = append(all, attribute.String("hatchet.workflow_version_id", *wfVersionID))
	}

	if parentID := ctx.ParentWorkflowRunId(); parentID != nil {
		all = append(all, attribute.String("hatchet.parent_workflow_run_id", *parentID))
	}

	if childIdx := ctx.ChildIndex(); childIdx != nil {
		all = append(all, attribute.Int("hatchet.child_workflow_index", int(*childIdx)))
	}

	if childKey := ctx.ChildKey(); childKey != nil {
		all = append(all, attribute.String("hatchet.child_workflow_key", *childKey))
	}

	if len(excluded) == 0 {
		return all
	}

	filtered := make([]attribute.KeyValue, 0, len(all))
	for _, attr := range all {
		key := string(attr.Key)
		// Only filter hatchet.* attributes; "instrumentor" is never excluded.
		if strings.HasPrefix(key, "hatchet.") {
			attrName := OTelAttribute(strings.TrimPrefix(key, "hatchet."))
			if excluded[attrName] {
				continue
			}
		}
		filtered = append(filtered, attr)
	}
	return filtered
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
