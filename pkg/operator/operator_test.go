//go:build !e2e && !load && !rampup && !integration

package operator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// captureSession is the part of Session the shared operator drives: it records every step
// action event and every action delta.
type captureSession struct {
	Session

	workerId uuid.UUID
	events   []*contracts.StepActionEvent
	added    [][]string
	removed  [][]string
	flushes  int
	addErr   error
}

func (c *captureSession) Registration() Registration {
	return Registration{WorkerId: c.workerId}
}

func (c *captureSession) SendStepActionEvent(_ context.Context, ev *contracts.StepActionEvent) error {
	c.events = append(c.events, ev)
	return nil
}

func (c *captureSession) AddActions(_ context.Context, ids []string) error {
	if c.addErr != nil {
		return c.addErr
	}

	c.added = append(c.added, append([]string(nil), ids...))

	return nil
}

func (c *captureSession) RemoveActions(_ context.Context, ids []string) error {
	c.removed = append(c.removed, append([]string(nil), ids...))
	return nil
}

func (c *captureSession) Flush(context.Context) error {
	c.flushes++
	return nil
}

// captureWriter is the engine-internal writer, recording the cancel-with-message events.
type captureWriter struct {
	events []*contracts.StepActionEvent
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

func startedOperator(t *testing.T) (*SharedOperator[struct{}], *captureSession) {
	t.Helper()

	session := &captureSession{workerId: uuid.New()}
	s := &SharedOperator[struct{}]{lastActions: map[string]struct{}{}}

	if err := s.Start(context.Background(), session); err != nil {
		t.Fatalf("Start: %v", err)
	}

	return s, session
}

// TestSendStartedAt_UsesProvidedTimestamp verifies the STARTED event carries the caller's
// timestamp verbatim rather than the moment the (possibly delayed) report is sent, and the
// session's worker id.
func TestSendStartedAt_UsesProvidedTimestamp(t *testing.T) {
	s, session := startedOperator(t)

	at := time.Now().Add(-5 * time.Second).UTC()

	if err := s.SendStartedAt(testAssignedAction(), at); err != nil {
		t.Fatalf("SendStartedAt: %v", err)
	}

	if len(session.events) != 1 {
		t.Fatalf("want 1 event, got %d", len(session.events))
	}
	got := session.events[0].EventTimestamp.AsTime()
	if !got.Equal(at) {
		t.Fatalf("STARTED timestamp = %s, want %s", got, at)
	}
	if session.events[0].WorkerId != session.workerId.String() {
		t.Fatalf("worker id = %s, want the session's %s", session.events[0].WorkerId, session.workerId)
	}
}

// TestSendStartedAt_OrdersBeforeLaterCompleted is the regression: a STARTED timestamp captured
// synchronously before the task body must precede a COMPLETED reported after it, even though the
// STARTED report is delivered later.
func TestSendStartedAt_OrdersBeforeLaterCompleted(t *testing.T) {
	s, session := startedOperator(t)
	s.inFlight = map[string]context.CancelFunc{}

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
	for _, e := range session.events {
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

// An operator that has not been started has no session to report through; the error says so
// rather than panicking on the delivery goroutine.
func TestSendBeforeStart(t *testing.T) {
	s := &SharedOperator[struct{}]{}

	if err := s.SendStarted(testAssignedAction()); err == nil {
		t.Fatal("expected an error before Start")
	}

	if s.WorkerId() != uuid.Nil {
		t.Fatal("expected no worker before Start")
	}
}

// SendCancelledWithMessage is the one report that stays engine-internal.
func TestSendCancelledWithMessage(t *testing.T) {
	s, session := startedOperator(t)
	w := &captureWriter{}
	s.taskEventWriter = w

	if err := s.SendCancelledWithMessage(testAssignedAction(), "because"); err != nil {
		t.Fatalf("SendCancelledWithMessage: %v", err)
	}

	if len(w.events) != 1 || w.events[0].EventPayload != "because" {
		t.Fatalf("want the cancel on the writer, got %v", w.events)
	}
	if len(session.events) != 0 {
		t.Fatal("the cancel must not go through the session")
	}
}

// UpdateWorkerActions sends the difference from the last advertised set and flushes; an
// unchanged set sends nothing; a failed delta leaves the advertised set as it was.
func TestUpdateWorkerActionsDiffs(t *testing.T) {
	s, session := startedOperator(t)

	changed, err := s.UpdateWorkerActions(context.Background(), []string{"a", "b", "b"})
	if err != nil || !changed {
		t.Fatalf("first update: changed=%v err=%v", changed, err)
	}
	if len(session.added) != 1 || len(session.added[0]) != 2 || len(session.removed) != 0 || session.flushes != 1 {
		t.Fatalf("first update: added=%v removed=%v flushes=%d", session.added, session.removed, session.flushes)
	}

	changed, err = s.UpdateWorkerActions(context.Background(), []string{"b", "a"})
	if err != nil || changed {
		t.Fatalf("unchanged set: changed=%v err=%v", changed, err)
	}
	if session.flushes != 1 {
		t.Fatal("an unchanged set must not flush")
	}

	changed, err = s.UpdateWorkerActions(context.Background(), []string{"b", "c"})
	if err != nil || !changed {
		t.Fatalf("diff update: changed=%v err=%v", changed, err)
	}
	if len(session.added) != 2 || session.added[1][0] != "c" || len(session.removed) != 1 || session.removed[0][0] != "a" {
		t.Fatalf("diff update: added=%v removed=%v", session.added, session.removed)
	}

	session.addErr = errors.New("budget")

	if _, err := s.UpdateWorkerActions(context.Background(), []string{"b", "c", "d"}); err == nil {
		t.Fatal("expected the delta error")
	}

	session.addErr = nil

	changed, err = s.UpdateWorkerActions(context.Background(), []string{"b", "c", "d"})
	if err != nil || !changed || session.added[len(session.added)-1][0] != "d" {
		t.Fatalf("the failed delta is repeated: changed=%v err=%v added=%v", changed, err, session.added)
	}
}

// TestRecordTaskDrains verifies Drain blocks until every recorded task has been released, and
// that RecordTask is a no-op once the drain has begun.
func TestRecordTaskDrains(t *testing.T) {
	s := &SharedOperator[struct{}]{}

	release := s.RecordTask()

	drained := make(chan struct{})

	go func() {
		s.Drain(context.Background())
		close(drained)
	}()

	// Drain must not return while the task is still in flight.
	select {
	case <-drained:
		t.Fatal("Drain returned before the in-flight task was released")
	case <-time.After(50 * time.Millisecond):
	}

	release()

	select {
	case <-drained:
	case <-time.After(time.Second):
		t.Fatal("Drain did not return after the task was released")
	}

	// After the drain began, RecordTask is a no-op and its release is safe to call.
	s.RecordTask()()
}

// Drain gives up when its context ends, so a task that never finishes cannot hold a shutdown.
func TestDrainHonoursContext(t *testing.T) {
	s := &SharedOperator[struct{}]{}

	release := s.RecordTask()
	defer release()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	done := make(chan struct{})

	go func() {
		s.Drain(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Drain did not return when its context ended")
	}
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
		s.Drain(context.Background())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Drain blocked despite all tasks being released")
	}
}
