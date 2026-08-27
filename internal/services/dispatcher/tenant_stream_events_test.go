//go:build !e2e && !load && !rampup && !integration

package dispatcher

import (
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestWorkflowRunEventTypeForOlapStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status       sqlcv1.V1ReadableStatusOlap
		wantType     contracts.ResourceEventType
		wantTerminal bool
	}{
		{sqlcv1.V1ReadableStatusOlapCOMPLETED, contracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED, true},
		{sqlcv1.V1ReadableStatusOlapFAILED, contracts.ResourceEventType_RESOURCE_EVENT_TYPE_FAILED, true},
		{sqlcv1.V1ReadableStatusOlapCANCELLED, contracts.ResourceEventType_RESOURCE_EVENT_TYPE_CANCELLED, true},
		{sqlcv1.V1ReadableStatusOlapRUNNING, 0, false},
		{sqlcv1.V1ReadableStatusOlapQUEUED, 0, false},
		{sqlcv1.V1ReadableStatusOlapEVICTED, 0, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			t.Parallel()

			gotType, gotTerminal := workflowRunEventTypeForOlapStatus(tt.status)
			assert.Equal(t, tt.wantTerminal, gotTerminal)
			assert.Equal(t, tt.wantType, gotType)
		})
	}
}

func TestWorkflowRunEventTypeFromOutputEvents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		events   []*v1.TaskOutputEvent
		wantType contracts.ResourceEventType
	}{
		{
			name:     "empty defaults to completed",
			events:   nil,
			wantType: contracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED,
		},
		{
			name: "all completed",
			events: []*v1.TaskOutputEvent{
				{EventType: sqlcv1.V1TaskEventTypeCOMPLETED},
			},
			wantType: contracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED,
		},
		{
			name: "failed wins over completed",
			events: []*v1.TaskOutputEvent{
				{EventType: sqlcv1.V1TaskEventTypeCOMPLETED},
				{EventType: sqlcv1.V1TaskEventTypeFAILED},
			},
			wantType: contracts.ResourceEventType_RESOURCE_EVENT_TYPE_FAILED,
		},
		{
			name: "cancelled",
			events: []*v1.TaskOutputEvent{
				{EventType: sqlcv1.V1TaskEventTypeCANCELLED},
			},
			wantType: contracts.ResourceEventType_RESOURCE_EVENT_TYPE_CANCELLED,
		},
		{
			name: "failed wins over cancelled",
			events: []*v1.TaskOutputEvent{
				{EventType: sqlcv1.V1TaskEventTypeCANCELLED},
				{EventType: sqlcv1.V1TaskEventTypeFAILED},
			},
			wantType: contracts.ResourceEventType_RESOURCE_EVENT_TYPE_FAILED,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantType, workflowRunEventTypeFromOutputEvents(tt.events))
		})
	}
}

func TestStreamBuffer_WorkflowRunHangupIsReleased(t *testing.T) {
	t.Parallel()

	buffer := NewStreamEventBuffer(5 * time.Second)
	defer buffer.Close()

	hangup := &contracts.WorkflowEvent{
		WorkflowRunId:  WORKFLOW_RUN_ID,
		ResourceType:   contracts.ResourceType_RESOURCE_TYPE_WORKFLOW_RUN,
		ResourceId:     WORKFLOW_RUN_ID,
		EventType:      contracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED,
		Hangup:         true,
		EventTimestamp: timestamppb.Now(),
	}

	buffer.AddEvent(hangup)

	select {
	case got := <-buffer.Events():
		require.True(t, got.Hangup)
		assert.Equal(t, contracts.ResourceType_RESOURCE_TYPE_WORKFLOW_RUN, got.ResourceType)
		assert.Equal(t, contracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED, got.EventType)
		assert.Equal(t, WORKFLOW_RUN_ID, got.WorkflowRunId)
	case <-time.After(time.Second):
		t.Fatal("expected hangup event to be released immediately")
	}
}
