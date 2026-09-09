package client

import (
	"context"
	"errors"
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
