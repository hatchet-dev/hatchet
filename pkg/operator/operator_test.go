//go:build !e2e && !load && !rampup && !integration

package operator

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// captureWriter records every step action event reported through it.
type captureWriter struct {
	events []*contracts.StepActionEvent
}

func (c *captureWriter) SendStepActionEvent(_ context.Context, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	c.events = append(c.events, request)
	return &contracts.ActionEventResponse{}, nil
}

func (c *captureWriter) CancelTaskEvent(_ context.Context, request *contracts.StepActionEvent) (*contracts.ActionEventResponse, error) {
	c.events = append(c.events, request)
	return &contracts.ActionEventResponse{}, nil
}

func (c *captureWriter) RegisterDurableTask(_ context.Context, _ uuid.UUID) (chan<- *v1contracts.DurableTaskRequest, <-chan *v1contracts.DurableTaskResponse, error) {
	return nil, nil, nil
}

func (c *captureWriter) TriggerDAGStep(_ context.Context, _ uuid.UUID, _ *DAGStepTriggerRequest) (*DAGStepTriggerResult, error) {
	return nil, nil
}

func (c *captureWriter) CancelDAGChildren(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
	return nil
}

func testAssignedAction() *contracts.AssignedAction {
	return &contracts.AssignedAction{
		TaskId:            "task-1",
		TaskRunExternalId: "run-1",
		ActionId:          "action-1",
		RetryCount:        0,
	}
}

// TestSendStartedAt_UsesProvidedTimestamp verifies the STARTED event carries the caller's
// timestamp verbatim rather than the moment the (possibly delayed) report is sent.
func TestSendStartedAt_UsesProvidedTimestamp(t *testing.T) {
	w := &captureWriter{}
	s := &SharedOperator[struct{}]{taskEventWriter: w, workerId: uuid.New()}

	at := time.Now().Add(-5 * time.Second).UTC()

	if err := s.SendStartedAt(testAssignedAction(), at); err != nil {
		t.Fatalf("SendStartedAt: %v", err)
	}

	if len(w.events) != 1 {
		t.Fatalf("want 1 event, got %d", len(w.events))
	}
	got := w.events[0].EventTimestamp.AsTime()
	if !got.Equal(at) {
		t.Fatalf("STARTED timestamp = %s, want %s", got, at)
	}
}

// TestSendStartedAt_OrdersBeforeLaterCompleted is the regression: a STARTED timestamp captured
// synchronously before the task body must precede a COMPLETED reported after it, even though the
// STARTED report is delivered later.
func TestSendStartedAt_OrdersBeforeLaterCompleted(t *testing.T) {
	w := &captureWriter{}
	s := &SharedOperator[struct{}]{taskEventWriter: w, workerId: uuid.New(), inFlight: map[string]context.CancelFunc{}}

	action := testAssignedAction()

	startedAt := time.Now() // captured before the "work"
	time.Sleep(2 * time.Millisecond)

	if err := s.SendCompleted(action, []byte(`{}`)); err != nil {
		t.Fatalf("SendCompleted: %v", err)
	}
	// STARTED is reported only now, after the work finished
	if err := s.SendStartedAt(action, startedAt); err != nil {
		t.Fatalf("SendStartedAt: %v", err)
	}

	var started, completed time.Time
	for _, e := range w.events {
		switch e.EventType {
		case contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED:
			started = e.EventTimestamp.AsTime()
		case contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED:
			completed = e.EventTimestamp.AsTime()
		}
	}

	if started.IsZero() || completed.IsZero() {
		t.Fatalf("missing events: started=%v completed=%v", started, completed)
	}
	if !started.Before(completed) {
		t.Fatalf("STARTED %s not before COMPLETED %s", started, completed)
	}
}

// TestRecordTaskDrainsOnCleanup verifies Cleanup blocks until every recorded task has been
// released, and that RecordTask is a no-op once shutdown has begun.
func TestRecordTaskDrainsOnCleanup(t *testing.T) {
	s := &SharedOperator[struct{}]{}

	release := s.RecordTask()

	cleanupDone := make(chan struct{})

	go func() {
		s.Cleanup()
		close(cleanupDone)
	}()

	// Cleanup must not return while the task is still in flight.
	select {
	case <-cleanupDone:
		t.Fatal("Cleanup returned before the in-flight task was released")
	case <-time.After(50 * time.Millisecond):
	}

	release()

	select {
	case <-cleanupDone:
	case <-time.After(time.Second):
		t.Fatal("Cleanup did not return after the task was released")
	}

	// After shutdown, RecordTask is a no-op and its release is safe to call.
	s.RecordTask()()
}

// TestReleaseIsIdempotent ensures calling release more than once does not over-decrement the
// task counter (which would panic the WaitGroup).
func TestReleaseIsIdempotent(t *testing.T) {
	s := &SharedOperator[struct{}]{}

	release := s.RecordTask()
	release()
	release()

	done := make(chan struct{})

	go func() {
		s.Cleanup()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Cleanup blocked despite all tasks being released")
	}
}
