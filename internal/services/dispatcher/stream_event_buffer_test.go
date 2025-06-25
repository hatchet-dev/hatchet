//go:build !e2e && !load && !rampup && !integration

package dispatcher

import (
	"testing"
	"time"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func genEvent(payload string, hangup bool, eventIndex *int64) *contracts.WorkflowEvent {
	return &contracts.WorkflowEvent{
		WorkflowRunId:  "test-run-id",
		ResourceId:     "test-step-run-id",
		ResourceType:   contracts.ResourceType_RESOURCE_TYPE_STEP_RUN,
		EventType:      contracts.ResourceEventType_RESOURCE_EVENT_TYPE_STREAM,
		EventTimestamp: timestamppb.Now(),
		EventPayload:   payload,
		Hangup:         hangup,
		EventIndex:     eventIndex,
	}
}

func TestStreamBuffer_BasicEventRelease(t *testing.T) {
	buffer := NewStreamEventBuffer(5 * time.Second)
	defer buffer.Close()

	ix := int64(0)

	event := genEvent("test_payload", false, &ix)

	buffer.AddEvent(event)

	select {
	case receivedEvent := <-buffer.Events():
		assert.Equal(t, event, receivedEvent)
	case <-time.After(1 * time.Second):
		t.Fatal("Expected event was not received")
	}
}

func TestStreamBuffer_OutOfOrderRelease(t *testing.T) {
	buffer := NewStreamEventBuffer(5 * time.Second)
	defer buffer.Close()

	ix0 := int64(0)
	ix1 := int64(1)
	ix2 := int64(2)

	event2 := genEvent("test_payload", false, &ix1)

	buffer.AddEvent(event2)

	select {
	case <-buffer.Events():
		t.Fatal("Should not receive out-of-order event")
	case <-time.After(100 * time.Millisecond):
		// Expected - no event should be received
	}

	event3 := genEvent("test_payload", false, &ix2)
	buffer.AddEvent(event3)

	select {
	case <-buffer.Events():
		t.Fatal("Should not receive out-of-order event")
	case <-time.After(100 * time.Millisecond):
		// Expected - no event should be received
	}

	event1 := genEvent("test_payload", false, &ix0)
	buffer.AddEvent(event1)

	receivedEvents := make([]*contracts.WorkflowEvent, 0, 3)
	for i := 0; i < 3; i++ {
		select {
		case event := <-buffer.Events():
			receivedEvents = append(receivedEvents, event)
		case <-time.After(1 * time.Second):
			t.Fatalf("Expected to receive event %d", i)
		}
	}

	assert.Equal(t, 3, len(receivedEvents))
	assert.Equal(t, event1, receivedEvents[0])
	assert.Equal(t, event2, receivedEvents[1])
	assert.Equal(t, event3, receivedEvents[2])
}

func TestStreamBuffer_Timeout(t *testing.T) {
	buffer := NewStreamEventBuffer(1 * time.Second)
	defer buffer.Close()

	ix1 := int64(1)
	ix2 := int64(2)
	ix0 := int64(0)

	event2 := genEvent("test_payload", false, &ix1)
	buffer.AddEvent(event2)

	select {
	case <-buffer.Events():
		t.Fatal("Should not receive out-of-order event")
	case <-time.After(100 * time.Millisecond):
		// Expected - no event should be received
	}

	event3 := genEvent("test_payload", false, &ix2)
	buffer.AddEvent(event3)

	select {
	case <-buffer.Events():
		t.Fatal("Should not receive out-of-order event")
	case <-time.After(100 * time.Millisecond):
		// Expected - no event should be received
	}

	time.Sleep(2 * time.Second)

	receivedEvents := make([]*contracts.WorkflowEvent, 0, 2)
	for i := 0; i < 2; i++ {
		select {
		case event := <-buffer.Events():
			receivedEvents = append(receivedEvents, event)
		case <-time.After(1 * time.Second):
			t.Fatalf("Expected to receive timed out event %d", i)
		}
	}

	assert.Equal(t, 2, len(receivedEvents))
	assert.Equal(t, event2, receivedEvents[0])
	assert.Equal(t, event3, receivedEvents[1])

	event1 := genEvent("test_payload", false, &ix0)
	buffer.AddEvent(event1)

	// This should be released immediately (fresh sequence after timeout)
	select {
	case receivedEvent := <-buffer.Events():
		assert.Equal(t, event1, receivedEvent)
	case <-time.After(1 * time.Second):
		t.Fatal("Expected event was not received")
	}
}

func TestStreamBuffer_TimeoutWithSubsequentOrdering(t *testing.T) {
	buffer := NewStreamEventBuffer(500 * time.Millisecond)
	defer buffer.Close()

	ix1 := int64(1)
	ix2 := int64(2)
	ix5 := int64(5)
	ix6 := int64(6)

	event1 := genEvent("payload1", false, &ix1)
	buffer.AddEvent(event1)

	select {
	case <-buffer.Events():
		t.Fatal("Should not receive out-of-order event")
	case <-time.After(100 * time.Millisecond):
		// Expected - no event should be received
	}

	event2 := genEvent("payload2", false, &ix2)
	buffer.AddEvent(event2)

	select {
	case <-buffer.Events():
		t.Fatal("Should not receive out-of-order event")
	case <-time.After(100 * time.Millisecond):
		// Expected - no event should be received
	}

	time.Sleep(1 * time.Second)

	receivedEvents := make([]*contracts.WorkflowEvent, 0, 2)
	for i := 0; i < 2; i++ {
		select {
		case event := <-buffer.Events():
			receivedEvents = append(receivedEvents, event)
		case <-time.After(1 * time.Second):
			t.Fatalf("Expected to receive timed out event %d", i)
		}
	}

	assert.Equal(t, 2, len(receivedEvents))
	assert.Equal(t, event1, receivedEvents[0])
	assert.Equal(t, event2, receivedEvents[1])

	// Now start a new sequence - event 5 should start a fresh sequence
	event5 := genEvent("payload5", false, &ix5)
	buffer.AddEvent(event5)

	select {
	case receivedEvent := <-buffer.Events():
		assert.Equal(t, event5, receivedEvent)
	case <-time.After(1 * time.Second):
		t.Fatal("Expected event was not received")
	}

	// Event 6 should be released immediately as it's the next in sequence
	event6 := genEvent("payload6", false, &ix6)
	buffer.AddEvent(event6)

	select {
	case receivedEvent := <-buffer.Events():
		assert.Equal(t, event6, receivedEvent)
	case <-time.After(1 * time.Second):
		t.Fatal("Expected event was not received")
	}
}

func TestStreamBuffer_HangupHandling(t *testing.T) {
	buffer := NewStreamEventBuffer(500 * time.Millisecond)
	defer buffer.Close()

	ix0 := int64(0)
	ix1 := int64(1)
	ix2 := int64(2)
	ix3 := int64(3)

	event2 := genEvent("first-event", false, &ix1)
	event3 := genEvent("second-event", false, &ix2)

	buffer.AddEvent(event2)

	select {
	case <-buffer.Events():
		t.Fatal("Should not receive out-of-order event")
	case <-time.After(100 * time.Millisecond):
		// Expected - no event should be received
	}

	buffer.AddEvent(event3)

	select {
	case <-buffer.Events():
		t.Fatal("Should not receive out-of-order event")
	case <-time.After(100 * time.Millisecond):
		// Expected - no event should be received
	}

	eventHangup := genEvent("hangup-event", true, &ix3)
	buffer.AddEvent(eventHangup)

	select {
	case <-buffer.Events():
		t.Fatal("Should not receive out-of-order event")
	case <-time.After(100 * time.Millisecond):
		// Expected - no event should be received
	}

	event0 := genEvent("first-event", false, &ix0)
	buffer.AddEvent(event0)

	receivedEvents := make([]*contracts.WorkflowEvent, 0, 4)
	for i := 0; i < 4; i++ {
		select {
		case event := <-buffer.Events():
			receivedEvents = append(receivedEvents, event)
		case <-time.After(1 * time.Second):
			t.Fatalf("Expected to receive event %d", i)
		}
	}

	assert.Equal(t, 4, len(receivedEvents))
	assert.Equal(t, event0, receivedEvents[0])
	assert.Equal(t, event2, receivedEvents[1])
	assert.Equal(t, event3, receivedEvents[2])
	assert.Equal(t, eventHangup, receivedEvents[3])
}

func TestStreamBuffer_NoIndexSent(t *testing.T) {
	buffer := NewStreamEventBuffer(500 * time.Millisecond)
	defer buffer.Close()

	event1 := genEvent("first-event", false, nil)
	event2 := genEvent("second-event", false, nil)

	buffer.AddEvent(event2)

	select {
	case receivedEvent := <-buffer.Events():
		assert.Equal(t, event2, receivedEvent)
	case <-time.After(1 * time.Second):
		t.Fatal("Expected event was not received")
	}

	buffer.AddEvent(event1)

	select {
	case receivedEvent := <-buffer.Events():
		assert.Equal(t, event1, receivedEvent)
	case <-time.After(1 * time.Second):
		t.Fatal("Expected event was not received")
	}
}
