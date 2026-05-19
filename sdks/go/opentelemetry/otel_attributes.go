package opentelemetry

// Span attribute key constants for Hatchet OpenTelemetry instrumentation.
// These mirror the OTelAttribute enum in the Python SDK and the OTelAttribute
// object in the TypeScript SDK.
const (
	// Instrumentor identifies the span source.
	AttrInstrumentor = "instrumentor"

	// AttrInstrumentorValue is the standard value for the instrumentor attribute.
	AttrInstrumentorValue = "hatchet"

	// Shared attributes
	AttrNamespace          = "hatchet.namespace"
	AttrAdditionalMetadata = "hatchet.additional_metadata"
	AttrWorkflowName       = "hatchet.workflow_name"
	AttrPriority           = "hatchet.priority"
	AttrActionPayload      = "hatchet.payload"

	// Action / consumer span attributes
	AttrActionName          = "hatchet.action_name"
	AttrStepName            = "hatchet.step_name"
	AttrChildWorkflowIndex  = "hatchet.child_workflow_index"
	AttrChildWorkflowKey    = "hatchet.child_workflow_key"
	AttrParentWorkflowRunID = "hatchet.parent_workflow_run_id"
	AttrRetryCount          = "hatchet.retry_count"
	AttrStepID              = "hatchet.step_id"
	AttrStepRunID           = "hatchet.step_run_id"
	AttrTenantID            = "hatchet.tenant_id"
	AttrWorkerID            = "hatchet.worker_id"
	AttrWorkflowID          = "hatchet.workflow_id"
	AttrWorkflowRunID       = "hatchet.workflow_run_id"
	AttrWorkflowVersionID   = "hatchet.workflow_version_id"

	// Trigger / producer span attributes
	AttrChildWorkflowRunID = "hatchet.child_workflow_run_id"
	AttrNumWorkflows       = "hatchet.num_workflows"
	AttrTriggerAt          = "hatchet.trigger_at"

	// Span names
	SpanStartStepRun     = "hatchet.start_step_run"
	SpanCancelStepRun    = "hatchet.cancel_step_run"
	SpanRunWorkflow      = "hatchet.run_workflow"
	SpanRunWorkflows     = "hatchet.run_workflows"
	SpanScheduleWorkflow = "hatchet.schedule_workflow"
)
