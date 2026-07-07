// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
)

func TestWorkflowEventToDeprecatedWorkflowEvent_Success(t *testing.T) {
	workflowRunId := "test-workflow-run-123"
	stepId := "test-step-456"
	eventPayload := "test-payload"

	event := &dispatchercontracts.WorkflowEvent{
		WorkflowRunId:  workflowRunId,
		ResourceType:   dispatchercontracts.ResourceType_RESOURCE_TYPE_STEP_RUN,
		ResourceId:     stepId,
		EventType:      dispatchercontracts.ResourceEventType_RESOURCE_EVENT_TYPE_STARTED,
		EventPayload:   eventPayload,
		EventTimestamp: timestamppb.New(time.Unix(1234567890, 123456789)),
	}

	deprecated, err := workflowEventToDeprecatedWorkflowEvent(event)

	require.NoError(t, err, "conversion should succeed")
	require.NotNil(t, deprecated, "deprecated event should not be nil")

	assert.Equal(t, workflowRunId, deprecated.WorkflowRunId)
	assert.Equal(t, stepId, deprecated.ResourceId)
	assert.Equal(t, eventPayload, deprecated.EventPayload)
}

func TestWorkflowEventToDeprecatedWorkflowEvent_WithNilTimestamp(t *testing.T) {
	event := &dispatchercontracts.WorkflowEvent{
		WorkflowRunId:  "test-run",
		ResourceId:     "test-resource",
		EventType:      dispatchercontracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED,
		EventPayload:   "payload",
		EventTimestamp: nil,
	}

	deprecated, err := workflowEventToDeprecatedWorkflowEvent(event)

	require.NoError(t, err, "conversion should succeed even with nil timestamp")
	require.NotNil(t, deprecated, "deprecated event should not be nil")
	assert.Equal(t, "test-run", deprecated.WorkflowRunId)
}

func TestWorkflowEventToDeprecatedWorkflowEvent_EmptyEvent(t *testing.T) {
	event := &dispatchercontracts.WorkflowEvent{}

	deprecated, err := workflowEventToDeprecatedWorkflowEvent(event)

	require.NoError(t, err, "conversion should succeed with empty event")
	require.NotNil(t, deprecated, "deprecated event should not be nil")
}

func TestWorkflowRunEventToDeprecatedWorkflowRunEvent_Success(t *testing.T) {
	workflowRunId := "test-workflow-run-789"

	event := &dispatchercontracts.WorkflowRunEvent{
		WorkflowRunId:  workflowRunId,
		EventType:      dispatchercontracts.WorkflowRunEventType_WORKFLOW_RUN_EVENT_TYPE_FINISHED,
		EventTimestamp: timestamppb.New(time.Unix(9876543210, 987654321)),
	}

	deprecated, err := workflowRunEventToDeprecatedWorkflowRunEvent(event)

	require.NoError(t, err, "conversion should succeed")
	require.NotNil(t, deprecated, "deprecated event should not be nil")

	assert.Equal(t, workflowRunId, deprecated.WorkflowRunId)
	assert.NotNil(t, deprecated.EventTimestamp)
}

func TestWorkflowRunEventToDeprecatedWorkflowRunEvent_WithNilTimestamp(t *testing.T) {
	event := &dispatchercontracts.WorkflowRunEvent{
		WorkflowRunId:  "test-run-2",
		EventType:      dispatchercontracts.WorkflowRunEventType_WORKFLOW_RUN_EVENT_TYPE_FINISHED,
		EventTimestamp: nil,
	}

	deprecated, err := workflowRunEventToDeprecatedWorkflowRunEvent(event)

	require.NoError(t, err, "conversion should succeed even with nil timestamp")
	require.NotNil(t, deprecated, "deprecated event should not be nil")
	assert.Equal(t, "test-run-2", deprecated.WorkflowRunId)
}

func TestWorkflowRunEventToDeprecatedWorkflowRunEvent_EmptyEvent(t *testing.T) {
	event := &dispatchercontracts.WorkflowRunEvent{}

	deprecated, err := workflowRunEventToDeprecatedWorkflowRunEvent(event)

	require.NoError(t, err, "conversion should succeed with empty event")
	require.NotNil(t, deprecated, "deprecated event should not be nil")
}

func TestWorkflowRunEventToDeprecatedWorkflowRunEvent_WithResults(t *testing.T) {
	event := &dispatchercontracts.WorkflowRunEvent{
		WorkflowRunId:  "test-run-3",
		EventType:      dispatchercontracts.WorkflowRunEventType_WORKFLOW_RUN_EVENT_TYPE_FINISHED,
		EventTimestamp: timestamppb.New(time.Now()),
		Results: []*dispatchercontracts.StepRunResult{
			{
				TaskRunExternalId: "step-1",
			},
		},
	}

	deprecated, err := workflowRunEventToDeprecatedWorkflowRunEvent(event)

	require.NoError(t, err, "conversion should succeed with results")
	require.NotNil(t, deprecated, "deprecated event should not be nil")
	assert.Equal(t, "test-run-3", deprecated.WorkflowRunId)
	assert.NotNil(t, deprecated.Results)
}
