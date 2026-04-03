package olap

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

func Test_buildWorkflowRunRootSpan_creates_when_no_sdk_traceparent(t *testing.T) {
	tenantID := uuid.New()
	wfRunID := uuid.New()
	wfID := uuid.New()
	now := time.Now()

	span := buildWorkflowRunRootSpan(tenantID, wfRunID, wfID, "my-workflow", now, nil)

	expectedTraceID := v1.DeriveWorkflowRunTraceID(wfRunID)
	expectedSpanID := v1.DeriveWorkflowRunSpanID(wfRunID)

	assert.Equal(t, expectedTraceID, span.TraceID)
	assert.Equal(t, expectedSpanID, span.SpanID)
	assert.Nil(t, span.ParentSpanID, "no parent when SDK didn't inject traceparent")
	assert.Equal(t, "hatchet.engine.workflow_run", span.Name)
}

func Test_buildWorkflowRunRootSpan_inherits_sdk_traceparent(t *testing.T) {
	tenantID := uuid.New()
	wfRunID := uuid.New()
	wfID := uuid.New()
	now := time.Now()

	sdkTraceID := "4bf92f3577b34da6a3ce929d0e0e4736"
	sdkSpanID := "00f067aa0ba902b7"

	meta, _ := json.Marshal(map[string]string{
		"traceparent": fmt.Sprintf("00-%s-%s-01", sdkTraceID, sdkSpanID),
	})

	span := buildWorkflowRunRootSpan(tenantID, wfRunID, wfID, "my-workflow", now, meta)

	assert.Equal(t, hexMustDecode(t, sdkTraceID), span.TraceID,
		"engine span inherits SDK trace_id")
	assert.Equal(t, v1.DeriveWorkflowRunSpanID(wfRunID), span.SpanID,
		"engine span uses its own derived span_id")
	assert.Equal(t, hexMustDecode(t, sdkSpanID), span.ParentSpanID,
		"engine span's parent is the SDK producer span")
}

func Test_buildWorkflowRunRootSpan_ignores_synthetic_self_referencing_traceparent(t *testing.T) {
	tenantID := uuid.New()
	wfRunID := uuid.New()
	wfID := uuid.New()
	now := time.Now()

	traceID := hex.EncodeToString(v1.DeriveWorkflowRunTraceID(wfRunID))
	spanID := hex.EncodeToString(v1.DeriveWorkflowRunSpanID(wfRunID))
	meta, _ := json.Marshal(map[string]string{
		"traceparent": fmt.Sprintf("00-%s-%s-01", traceID, spanID),
	})

	span := buildWorkflowRunRootSpan(tenantID, wfRunID, wfID, "my-workflow", now, meta)

	assert.Equal(t, v1.DeriveWorkflowRunSpanID(wfRunID), span.SpanID)
	assert.Nil(t, span.ParentSpanID,
		"must not self-reference when synthetic traceparent reuses the span's own ID")
}

func Test_buildWorkflowRunRootSpan_child_with_synthetic_traceparent_nests_under_step_run(t *testing.T) {
	tenantID := uuid.New()
	parentWfRunID := uuid.New()
	parentStepRunID := uuid.New()
	childWfRunID := uuid.New()
	wfID := uuid.New()
	now := time.Now()

	parentTraceID := hex.EncodeToString(v1.DeriveWorkflowRunTraceID(parentWfRunID))
	childSpanID := hex.EncodeToString(v1.DeriveWorkflowRunSpanID(childWfRunID))

	meta, _ := json.Marshal(map[string]string{
		"traceparent":                     fmt.Sprintf("00-%s-%s-01", parentTraceID, childSpanID),
		"hatchet__parent_workflow_run_id": parentWfRunID.String(),
		"hatchet__parent_step_run_id":     parentStepRunID.String(),
	})

	span := buildWorkflowRunRootSpan(tenantID, childWfRunID, wfID, "child-wf", now, meta)

	assert.Equal(t, v1.DeriveWorkflowRunSpanID(childWfRunID), span.SpanID)
	assert.Equal(t, deriveStepRunSpanID(parentStepRunID, 0, "step_run"), span.ParentSpanID,
		"child must nest under parent's step_run span, not self-reference via synthetic traceparent")
}

func Test_buildWorkflowRunRootSpan_child_with_inherited_parent_traceparent_nests_under_step_run(t *testing.T) {
	tenantID := uuid.New()
	parentWfRunID := uuid.New()
	parentStepRunID := uuid.New()
	childWfRunID := uuid.New()
	wfID := uuid.New()
	now := time.Now()

	parentTraceID := hex.EncodeToString(v1.DeriveWorkflowRunTraceID(parentWfRunID))
	parentSpanID := hex.EncodeToString(v1.DeriveWorkflowRunSpanID(parentWfRunID))

	meta, _ := json.Marshal(map[string]string{
		"traceparent":                     fmt.Sprintf("00-%s-%s-01", parentTraceID, parentSpanID),
		"hatchet__parent_workflow_run_id": parentWfRunID.String(),
		"hatchet__parent_step_run_id":     parentStepRunID.String(),
	})

	span := buildWorkflowRunRootSpan(tenantID, childWfRunID, wfID, "child-wf", now, meta)

	assert.Equal(t, v1.DeriveWorkflowRunSpanID(childWfRunID), span.SpanID)
	assert.Equal(t, deriveStepRunSpanID(parentStepRunID, 0, "step_run"), span.ParentSpanID,
		"child must nest under parent's step_run span, not the parent's workflow_run root via inherited traceparent")
}

func Test_buildWorkflowRunRootSpan_child_with_grandparent_traceparent_nests_under_step_run(t *testing.T) {
	tenantID := uuid.New()
	grandparentWfRunID := uuid.New()
	parentWfRunID := uuid.New()
	parentStepRunID := uuid.New()
	childWfRunID := uuid.New()
	wfID := uuid.New()
	now := time.Now()

	gpTraceID := hex.EncodeToString(v1.DeriveWorkflowRunTraceID(grandparentWfRunID))
	gpSpanID := hex.EncodeToString(v1.DeriveWorkflowRunSpanID(grandparentWfRunID))

	meta, _ := json.Marshal(map[string]string{
		"traceparent":                     fmt.Sprintf("00-%s-%s-01", gpTraceID, gpSpanID),
		"hatchet__parent_workflow_run_id": parentWfRunID.String(),
		"hatchet__parent_step_run_id":     parentStepRunID.String(),
	})

	span := buildWorkflowRunRootSpan(tenantID, childWfRunID, wfID, "child-wf", now, meta)

	assert.Equal(t, deriveStepRunSpanID(parentStepRunID, 0, "step_run"), span.ParentSpanID,
		"child must nest under parent's step_run span even when traceparent is from a grandparent workflow")
}

func Test_buildEventSpan_creates_when_no_sdk_traceparent(t *testing.T) {
	tenantID := uuid.New()
	eventID := uuid.New()
	wfRunID := uuid.New()
	now := time.Now()

	span := buildEventSpan(tenantID, eventID, "user.created", now, wfRunID, nil)

	expectedTraceID := v1.DeriveWorkflowRunTraceID(wfRunID)

	assert.Equal(t, expectedTraceID, span.TraceID)
	assert.Nil(t, span.ParentSpanID, "no parent when SDK didn't inject traceparent")
	assert.Equal(t, "hatchet.engine.event", span.Name)
}

func Test_buildEventSpan_inherits_sdk_traceparent(t *testing.T) {
	tenantID := uuid.New()
	eventID := uuid.New()
	wfRunID := uuid.New()
	now := time.Now()

	sdkTraceID := "4bf92f3577b34da6a3ce929d0e0e4736"
	sdkSpanID := "00f067aa0ba902b7"

	meta, _ := json.Marshal(map[string]string{
		"traceparent": fmt.Sprintf("00-%s-%s-01", sdkTraceID, sdkSpanID),
	})

	span := buildEventSpan(tenantID, eventID, "user.created", now, wfRunID, meta)

	assert.Equal(t, hexMustDecode(t, sdkTraceID), span.TraceID,
		"event span inherits SDK trace_id")
	assert.Equal(t, hexMustDecode(t, sdkSpanID), span.ParentSpanID,
		"event span's parent is the SDK producer span")
}

func hexMustDecode(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	require.NoError(t, err)
	return b
}
