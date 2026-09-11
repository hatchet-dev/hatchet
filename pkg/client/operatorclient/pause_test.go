package operatorclient

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

func startStepRun(taskRunExternalId string) *dispatchercontracts.AssignedAction {
	return &dispatchercontracts.AssignedAction{
		ActionId:          "svc:a",
		ActionType:        dispatchercontracts.ActionType_START_STEP_RUN,
		TaskRunExternalId: taskRunExternalId,
	}
}

func completed(taskRunExternalId string) *dispatchercontracts.StepActionEvent {
	return &dispatchercontracts.StepActionEvent{
		TaskRunExternalId: taskRunExternalId,
		EventType:         dispatchercontracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED,
	}
}

// Pause and Resume name the session's current worker and carry the operator id, so they keep
// working across a reconnect that gave the session a new worker.
func TestOperatorSessionPauseAndResume(t *testing.T) {
	client := &fakeOperatorServiceClient{registrations: []*v1.OperatorRegisterResponse{registeredAs("worker-1", false)}}
	s, _ := connectTestOperatorSession(t, client, true)

	require.NoError(t, s.Pause(context.Background()))
	require.NoError(t, s.Resume(context.Background()))

	pauses := client.pauseRequests()
	require.Len(t, pauses, 2)
	assert.Equal(t, "worker-1", pauses[0].WorkerId)
	assert.True(t, pauses[0].Paused)
	assert.False(t, pauses[1].Paused)
}

// Close is pause then drain: the pause goes out first, and the hang-up waits for the actions
// the consumer already holds to be reported.
func TestOperatorSessionCloseDrainsInFlightActions(t *testing.T) {
	client := &fakeOperatorServiceClient{registrations: []*v1.OperatorRegisterResponse{registeredAs("worker-1", false)}}
	s, _ := newTestOperatorSession(t, client, true)
	require.NoError(t, s.connect(context.Background()))

	actions, _, err := s.Actions(context.Background())
	require.NoError(t, err)

	client.stream(0).deliver(startStepRun("run-1"))

	action := <-actions
	require.Equal(t, "run-1", action.TaskRunExternalId)

	closed := make(chan error, 1)
	go func() { closed <- s.Close() }()

	waitFor(t, func() bool { return len(client.pauseRequests()) == 1 }, "the worker was not paused")
	assert.True(t, client.pauseRequests()[0].Paused)

	select {
	case <-closed:
		t.Fatal("Close returned while an action was still in flight")
	case <-time.After(100 * time.Millisecond):
	}

	_, err = s.SendStepActionEvent(context.Background(), completed("run-1"))
	require.NoError(t, err)

	select {
	case err := <-closed:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not return after the action was reported")
	}
}

// A drain that runs out of time hangs up anyway: the engine retries the task once it times out.
func TestOperatorSessionCloseDrainTimeout(t *testing.T) {
	client := &fakeOperatorServiceClient{registrations: []*v1.OperatorRegisterResponse{registeredAs("worker-1", false)}}
	s, _ := newTestOperatorSession(t, client, true)
	require.NoError(t, s.connect(context.Background()))

	actions, _, err := s.Actions(context.Background())
	require.NoError(t, err)

	client.stream(0).deliver(startStepRun("run-1"))
	<-actions

	require.NoError(t, s.Close(WithDrainTimeout(50*time.Millisecond)))
	assert.Len(t, client.pauseRequests(), 1)
}

// WithoutDrain is the immediate hang-up: nothing is paused and nothing is waited for.
func TestOperatorSessionCloseWithoutDrain(t *testing.T) {
	client := &fakeOperatorServiceClient{registrations: []*v1.OperatorRegisterResponse{registeredAs("worker-1", false)}}
	s, _ := newTestOperatorSession(t, client, true)
	require.NoError(t, s.connect(context.Background()))

	actions, _, err := s.Actions(context.Background())
	require.NoError(t, err)

	client.stream(0).deliver(startStepRun("run-1"))
	<-actions

	require.NoError(t, s.Close(WithoutDrain()))
	assert.Empty(t, client.pauseRequests())
}

// A cancel carries no work of its own and is never reported, so it must not hold the drain up.
func TestOperatorSessionCloseIgnoresCancelActions(t *testing.T) {
	client := &fakeOperatorServiceClient{registrations: []*v1.OperatorRegisterResponse{registeredAs("worker-1", false)}}
	s, _ := newTestOperatorSession(t, client, true)
	require.NoError(t, s.connect(context.Background()))

	actions, _, err := s.Actions(context.Background())
	require.NoError(t, err)

	client.stream(0).deliver(&dispatchercontracts.AssignedAction{
		ActionId:          "svc:a",
		ActionType:        dispatchercontracts.ActionType_CANCEL_STEP_RUN,
		TaskRunExternalId: "run-1",
	})
	<-actions

	done := make(chan error, 1)
	go func() { done <- s.Close(WithDrainTimeout(5 * time.Second)) }()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("a cancel action held the drain up")
	}
}

// A pause the engine refuses is logged, not fatal: the session still drains what it holds and
// closes.
func TestOperatorSessionClosesWhenPauseFails(t *testing.T) {
	client := &fakeOperatorServiceClient{
		registrations: []*v1.OperatorRegisterResponse{registeredAs("worker-1", false)},
		pauseErr:      errors.New("engine is gone"),
	}
	s, _ := newTestOperatorSession(t, client, true)
	require.NoError(t, s.connect(context.Background()))

	require.NoError(t, s.Close(WithDrainTimeout(time.Second)))
}

// A consumer that reports an action the instant it takes it must still leave the session idle:
// the action is recorded as in flight before it is handed over, so the report cannot clear an
// entry that has not been added yet.
func TestOperatorSessionCloseDrainsImmediateReports(t *testing.T) {
	client := &fakeOperatorServiceClient{registrations: []*v1.OperatorRegisterResponse{registeredAs("worker-1", false)}}
	s, _ := newTestOperatorSession(t, client, true)
	require.NoError(t, s.connect(context.Background()))

	actions, _, err := s.Actions(context.Background())
	require.NoError(t, err)

	const runs = 50

	reported := make(chan struct{})

	go func() {
		defer close(reported)

		for action := range actions {
			_, _ = s.SendStepActionEvent(context.Background(), completed(action.TaskRunExternalId))
		}
	}()

	for i := 0; i < runs; i++ {
		client.stream(0).deliver(startStepRun(fmt.Sprintf("run-%d", i)))
	}

	waitFor(t, func() bool { return len(client.stepEventsSent()) == runs }, "the consumer did not report every action")

	started := time.Now()
	require.NoError(t, s.Close(WithDrainTimeout(10*time.Second)))
	assert.Less(t, time.Since(started), 5*time.Second, "the drain waited for an action that was already reported")

	<-reported
}

// A consumer that cancels its Actions context will never report the runs it holds, so Close
// stops waiting for them instead of draining until the timeout.
func TestOperatorSessionCloseDoesNotDrainAnEndedConsumer(t *testing.T) {
	client := &fakeOperatorServiceClient{registrations: []*v1.OperatorRegisterResponse{registeredAs("worker-1", false)}}
	s, _ := newTestOperatorSession(t, client, true)
	require.NoError(t, s.connect(context.Background()))

	consumerCtx, cancelConsumer := context.WithCancel(context.Background())

	actions, _, err := s.Actions(consumerCtx)
	require.NoError(t, err)

	client.stream(0).deliver(startStepRun("run-1"))
	<-actions

	cancelConsumer()

	// the delivery loop closes the channel on its way out, which is when the runs it handed
	// over stop counting
	for range actions {
	}

	started := time.Now()
	require.NoError(t, s.Close(WithDrainTimeout(10*time.Second)))
	assert.Less(t, time.Since(started), 5*time.Second, "Close waited for a consumer that had gone away")
}
