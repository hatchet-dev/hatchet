//go:build !e2e && !load && !rampup && !integration

package dispatcher

import (
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func genEvent(payload string, hangup bool, eventIndex int64) *contracts.WorkflowEvent {
	return &contracts.WorkflowEvent{
		WorkflowRunId:  "test-run-id",
		ResourceId:     "test-step-run-id",
		ResourceType:   contracts.ResourceType_RESOURCE_TYPE_STEP_RUN,
		EventType:      contracts.ResourceEventType_RESOURCE_EVENT_TYPE_STREAM,
		EventTimestamp: timestamppb.Now(),
		EventPayload:   payload,
		Hangup:         hangup,
		EventIndex:     &eventIndex,
	}
}

func TestStreamBuffer_BasicEventRelease(t *testing.T) {
	buffer := NewStreamEventBuffer(5 * time.Second)

	event := genEvent("test_payload", false, 0) // Events are zero-indexed

	releasedEvents := buffer.AddEvent(event)

	assert.Equal(t, 1, len(releasedEvents))
	assert.Equal(t, event, releasedEvents[0])
}

func TestStreamBuffer_OutOfOrderRelease(t *testing.T) {
	buffer := NewStreamEventBuffer(5 * time.Second)

	event2 := genEvent("test_payload", false, 1)

	releasedEvents := buffer.AddEvent(event2)

	assert.Equal(t, 0, len(releasedEvents))

	event3 := genEvent("test_payload", false, 2)
	releasedEvents2 := buffer.AddEvent(event3)

	assert.Equal(t, 0, len(releasedEvents2))

	event1 := genEvent("test_payload", false, 0)
	releasedEvents3 := buffer.AddEvent(event1)

	assert.Equal(t, 3, len(releasedEvents3))

	assert.Equal(t, event1, releasedEvents3[0])
	assert.Equal(t, event2, releasedEvents3[1])
	assert.Equal(t, event3, releasedEvents3[2])
}

func TestStreamBuffer_Timeout(t *testing.T) {
	buffer := NewStreamEventBuffer(1 * time.Second)

	event2 := genEvent("test_payload", false, 1)
	releasedEvents := buffer.AddEvent(event2)
	assert.Equal(t, 0, len(releasedEvents))

	event3 := genEvent("test_payload", false, 2)
	releasedEvents2 := buffer.AddEvent(event3)
	assert.Equal(t, 0, len(releasedEvents2))

	time.Sleep(2 * time.Second)

	timedOutEvents := buffer.GetTimedOutEvents()

	assert.Equal(t, 2, len(timedOutEvents))
	assert.Equal(t, event2, timedOutEvents[0])
	assert.Equal(t, event3, timedOutEvents[1])

	event1 := genEvent("test_payload", false, 0)
	releasedEvents3 := buffer.AddEvent(event1)

	// This should be released immediately
	assert.Equal(t, 1, len(releasedEvents3))
	assert.Equal(t, event1, releasedEvents3[0])
}

func TestStreamBuffer_TimeoutWithSubsequentOrdering(t *testing.T) {
	buffer := NewStreamEventBuffer(500 * time.Millisecond)

	event1 := genEvent("payload1", false, 1)
	releasedEvents := buffer.AddEvent(event1)
	assert.Equal(t, 0, len(releasedEvents))

	event2 := genEvent("payload2", false, 2)
	releasedEvents2 := buffer.AddEvent(event2)
	assert.Equal(t, 0, len(releasedEvents2))

	time.Sleep(1 * time.Second)

	timedOutEvents := buffer.GetTimedOutEvents()
	assert.Equal(t, 2, len(timedOutEvents))
	assert.Equal(t, event1, timedOutEvents[0])
	assert.Equal(t, event2, timedOutEvents[1])

	// Now start a new sequence - event 5 should start a fresh sequence
	event5 := genEvent("payload5", false, 5)
	releasedEvents3 := buffer.AddEvent(event5)
	assert.Equal(t, 1, len(releasedEvents3))
	assert.Equal(t, event5, releasedEvents3[0])

	// Event 6 should be released immediately as it's the next in sequence
	event6 := genEvent("payload6", false, 6)
	releasedEvents4 := buffer.AddEvent(event6)
	assert.Equal(t, 1, len(releasedEvents4))
	assert.Equal(t, event6, releasedEvents4[0])
}

func TestStreamBuffer_HangupHandling(t *testing.T) {
	buffer := NewStreamEventBuffer(500 * time.Millisecond)

	event2 := genEvent("first-event", false, 1)
	event3 := genEvent("second-event", false, 2)

	releasedEvents := buffer.AddEvent(event2)
	assert.Equal(t, 0, len(releasedEvents))

	releasedEvents2 := buffer.AddEvent(event3)
	assert.Equal(t, 0, len(releasedEvents2))

	eventHangup := genEvent("hangup-event", true, 3)
	releasedEvents3 := buffer.AddEvent(eventHangup)
	assert.Equal(t, 0, len(releasedEvents3))

	event0 := genEvent("first-event", false, 0)
	releasedEvents4 := buffer.AddEvent(event0)
	assert.Equal(t, 4, len(releasedEvents4))

	assert.Equal(t, event0, releasedEvents4[0])
	assert.Equal(t, event2, releasedEvents4[1])
	assert.Equal(t, event3, releasedEvents4[2])
	assert.Equal(t, eventHangup, releasedEvents4[3])
}
