//go:build !e2e && !load && !rampup && !integration

package dispatcher

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
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

// TestWorkflowRunFinishedConverterDropsNonTerminalStatuses reproduces the
// premature-hangup bug for DAG runs: the OLAP controller's DAG status updater
// used to publish QUEUED->RUNNING transitions as workflow-run-finished, and
// the converter coerced the unknown RUNNING status into COMPLETED, hanging up
// stream subscribers the moment the run started. Non-terminal statuses must
// produce no event at all.
func TestWorkflowRunFinishedConverterDropsNonTerminalStatuses(t *testing.T) {
	t.Parallel()

	convert := workflowEventConverters[msgqueue.MsgIDWorkflowRunFinished]
	require.NotNil(t, convert)

	runId := uuid.New()

	payload := func(status sqlcv1.V1ReadableStatusOlap) []byte {
		body, err := json.Marshal(tasktypes.NotifyFinalizedPayload{
			ExternalId: runId,
			Status:     status,
		})
		require.NoError(t, err)
		return body
	}

	events := convert([][]byte{
		payload(sqlcv1.V1ReadableStatusOlapQUEUED),
		payload(sqlcv1.V1ReadableStatusOlapRUNNING),
		payload(sqlcv1.V1ReadableStatusOlapCOMPLETED),
	})

	require.Len(t, events, 1)
	assert.Equal(t, contracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED, events[0].EventType)
	assert.Equal(t, runId.String(), events[0].WorkflowRunId)
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
	case <-buffer.Events():
		t.Fatal("workflow hangup must wait for the quiet period")
	case <-time.After(100 * time.Millisecond):
	}

	select {
	case got := <-buffer.Events():
		require.True(t, got.Hangup)
		assert.Equal(t, contracts.ResourceType_RESOURCE_TYPE_WORKFLOW_RUN, got.ResourceType)
		assert.Equal(t, contracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED, got.EventType)
		assert.Equal(t, WORKFLOW_RUN_ID, got.WorkflowRunId)
	case <-time.After(2 * time.Second):
		t.Fatal("expected hangup event after quiet period")
	}
}

func TestStreamBuffer_WorkflowHangupWaitsForMissingIndex(t *testing.T) {
	t.Parallel()

	buffer := newStreamEventBufferForTest(5*time.Second, 50*time.Millisecond, 5*time.Second)
	defer buffer.Close()

	ix0, ix1, ix2 := int64(0), int64(1), int64(2)
	event1 := genEvent("c01", false, &ix1)
	event2 := genEvent("c02", false, &ix2)
	event0 := genEvent("c00", false, &ix0)
	hangup := genWorkflowHangup()

	buffer.AddEvent(event1)
	buffer.AddEvent(event2)
	buffer.AddEvent(hangup)

	select {
	case <-buffer.Events():
		t.Fatal("should not emit hangup or out-of-order chunks before index 0")
	case <-time.After(80 * time.Millisecond):
	}

	buffer.AddEvent(event0)

	got := make([]*contracts.WorkflowEvent, 0, 4)
	for i := 0; i < 4; i++ {
		select {
		case e := <-buffer.Events():
			got = append(got, e)
		case <-time.After(2 * time.Second):
			t.Fatalf("expected event %d, got %d", i, len(got))
		}
	}

	require.Len(t, got, 4)
	assert.Equal(t, event0, got[0])
	assert.Equal(t, event1, got[1])
	assert.Equal(t, event2, got[2])
	require.True(t, got[3].Hangup)
}

// TestStreamBuffer_ChunksAfterTerminalEventAreDelivered reproduces the
// production loss pattern: the step-run terminal event leapfrogs late stream
// chunks (separate per-msgId flush buffers), and the stragglers arriving after
// it must still be delivered — in order, since the buffer holds chunks past
// the hole until the straggler fills it rather than flushing on the terminal
// event.
func TestStreamBuffer_ChunksAfterTerminalEventAreDelivered(t *testing.T) {
	t.Parallel()

	buffer := newStreamEventBufferForTest(5*time.Second, 100*time.Millisecond, 5*time.Second)
	defer buffer.Close()

	ix0, ix1, ix2, ix3 := int64(0), int64(1), int64(2), int64(3)

	terminal := &contracts.WorkflowEvent{
		WorkflowRunId:  WORKFLOW_RUN_ID,
		ResourceId:     RESOURCE,
		ResourceType:   contracts.ResourceType_RESOURCE_TYPE_STEP_RUN,
		EventType:      contracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED,
		EventTimestamp: timestamppb.Now(),
	}

	// c00 flows through; c02/c03 buffer waiting on the missing c01; the
	// terminal event then flushes the buffer while c01 is still in flight
	buffer.AddEvent(genEvent("c00", false, &ix0))
	buffer.AddEvent(genEvent("c02", false, &ix2))
	buffer.AddEvent(genEvent("c03", false, &ix3))
	buffer.AddEvent(terminal)

	// the straggler arrives after the terminal event
	buffer.AddEvent(genEvent("c01", false, &ix1))
	buffer.AddEvent(genWorkflowHangup())

	payloads := make([]string, 0, 4)
	sawHangup := false

	for i := 0; i < 6; i++ {
		select {
		case e := <-buffer.Events():
			if e.Hangup {
				sawHangup = true
				continue
			}
			if e.EventType == contracts.ResourceEventType_RESOURCE_EVENT_TYPE_STREAM {
				payloads = append(payloads, e.EventPayload)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out after %d events: %v", len(payloads), payloads)
		}
	}

	// the straggler fills the hole, so everything drains in order
	assert.Equal(t, []string{"c00", "c01", "c02", "c03"}, payloads)
	require.True(t, sawHangup)
}

func TestStreamBuffer_WorkflowHangupAcceptsLateStreamChunks(t *testing.T) {
	t.Parallel()

	buffer := newStreamEventBufferForTest(5*time.Second, 150*time.Millisecond, 5*time.Second)
	defer buffer.Close()

	ix0, ix1 := int64(0), int64(1)
	event0 := genEvent("c00", false, &ix0)
	event1 := genEvent("c01", false, &ix1)
	hangup := genWorkflowHangup()

	buffer.AddEvent(event0)
	buffer.AddEvent(hangup)
	time.Sleep(40 * time.Millisecond)
	buffer.AddEvent(event1)

	got := make([]*contracts.WorkflowEvent, 0, 3)
	for i := 0; i < 3; i++ {
		select {
		case e := <-buffer.Events():
			got = append(got, e)
		case <-time.After(2 * time.Second):
			t.Fatalf("expected event %d, got %d", i, len(got))
		}
	}

	require.Len(t, got, 3)
	assert.Equal(t, event0, got[0])
	assert.Equal(t, event1, got[1])
	require.True(t, got[2].Hangup)
}
