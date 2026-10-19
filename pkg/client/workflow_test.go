// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/client/retry"
	"github.com/hatchet-dev/hatchet/pkg/client/streaming"
)

func TestWorkflowResultSingleAddWorkflowRunAttempt(t *testing.T) {
	logger := zerolog.Nop()
	constructorCalls := atomic.Int32{}

	listener := newTestWorkflowRunsListener(t, &logger, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		constructorCalls.Add(1)
		return nil, status.Error(codes.Unavailable, "engine down")
	}, nil)

	workflow := NewWorkflow("run-1", listener)
	_, err := workflow.Result()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to listen for workflow events")
	assert.LessOrEqual(t, constructorCalls.Load(), int32(retry.StreamSyncMaxAttempts))
	assert.Greater(t, constructorCalls.Load(), int32(0))

	assert.False(t, listener.reg.hasAny())
}

func TestWorkflowResultFailsWhenListenerDiesPermanently(t *testing.T) {
	logger := zerolog.Nop()
	runID := "run-permanent-fail"

	client := &mockSubscribeClient{
		recvFn: func() (*dispatchercontracts.WorkflowRunEvent, error) {
			return nil, status.Error(codes.Unauthenticated, "auth failed")
		},
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}

	listener := newTestWorkflowRunsListener(t, &logger, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		return client, nil
	}, client)

	workflow := NewWorkflow(runID, listener)

	resultDone := make(chan struct{})
	var resultErr error
	go func() {
		_, resultErr = workflow.Result()
		close(resultDone)
	}()

	require.Eventually(t, func() bool {
		select {
		case <-resultDone:
			return true
		default:
			return false
		}
	}, 5*time.Second, 10*time.Millisecond)

	require.Error(t, resultErr)
	assert.Contains(t, resultErr.Error(), runID)
	assert.ErrorIs(t, resultErr, status.Error(codes.Unauthenticated, "auth failed"))

	require.NoError(t, listener.Close())
}

// newSubscribedWorkflow returns a Workflow whose listener accepts
// subscriptions and delivers events pushed onto the returned client's
// recvChan.
func newSubscribedWorkflow(t *testing.T, runID string) (*Workflow, *WorkflowRunsListener, *mockSubscribeClient) {
	t.Helper()

	logger := zerolog.Nop()
	client := &mockSubscribeClient{
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent, 1),
	}
	listener := newTestWorkflowRunsListener(t, &logger, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		return client, nil
	}, client)
	t.Cleanup(func() {
		_ = listener.Close()
	})

	return NewWorkflow(runID, listener), listener, client
}

func TestWorkflowResultWithContextReturnsWhenContextAlreadyCancelled(t *testing.T) {
	workflow, listener, client := newSubscribedWorkflow(t, "run-ctx-cancelled")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	res, err := workflow.ResultWithContext(ctx)
	require.Nil(t, res)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Contains(t, err.Error(), "run-ctx-cancelled")
	assert.Less(t, time.Since(start), time.Second)

	// No subscription is sent for a caller whose context already ended, and
	// nothing is left registered.
	assert.Equal(t, int32(0), client.sendCount.Load())
	assert.False(t, listener.reg.hasAny())
}

func TestWorkflowResultWithContextReturnsOnDeadline(t *testing.T) {
	workflow, listener, _ := newSubscribedWorkflow(t, "run-ctx-deadline")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	res, err := workflow.ResultWithContext(ctx)
	require.Nil(t, res)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.False(t, listener.reg.hasAny())
}

func TestWorkflowResultWithContextDeliversResultWhileContextLive(t *testing.T) {
	workflow, listener, client := newSubscribedWorkflow(t, "run-ctx-live")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output := `{"ok":true}`
	go func() {
		// Wait until the subscription is registered before delivering.
		require.Eventually(t, listener.reg.hasAny, time.Second, 5*time.Millisecond)
		client.recvChan <- &dispatchercontracts.WorkflowRunEvent{
			WorkflowRunId: "run-ctx-live",
			Results: []*dispatchercontracts.StepRunResult{
				{TaskName: "step1", Output: &output},
			},
		}
	}()

	res, err := workflow.ResultWithContext(ctx)
	require.NoError(t, err)
	require.NotNil(t, res)

	var got map[string]bool
	require.NoError(t, res.StepOutput("step1", &got))
	assert.True(t, got["ok"])
	assert.False(t, listener.reg.hasAny())
}

func TestAwaitWorkflowResultNilContextDoneKeepsWaiting(t *testing.T) {
	resChan := make(chan *WorkflowResult, 1)
	failChan := make(chan error, 1)

	done := make(chan struct{})
	go func() {
		defer close(done)
		res, err := awaitWorkflowResult("run-nil-ctx", resChan, failChan, nil, nil)
		assert.NoError(t, err)
		assert.NotNil(t, res)
	}()

	select {
	case <-done:
		t.Fatal("wait ended without a result or failure")
	case <-time.After(50 * time.Millisecond):
	}

	resChan <- &WorkflowResult{workflowRun: &workflowRunEvent{WorkflowRunId: "run-nil-ctx"}}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("wait did not end after result delivery")
	}
}

// blockBackoffUntilCancelled makes the stream's backoff sleep wait for its
// context instead of a timer, so a test can prove that the caller's context is
// what ends a retry loop. The fallback keeps a regression from hanging the
// suite.
func blockBackoffUntilCancelled[C any](t *testing.T, s *streaming.ReconnectingStream[C]) {
	t.Helper()
	s.SetSleep(func(ctx context.Context, _ int) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			return errors.New("test backoff sleep was not cancelled")
		}
	})
}

func TestWorkflowResultWithContextEndsConnectRetriesOnDeadline(t *testing.T) {
	logger := zerolog.Nop()
	constructorCalls := atomic.Int32{}

	listener := newTestWorkflowRunsListener(t, &logger, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		constructorCalls.Add(1)
		return nil, status.Error(codes.Unavailable, "engine down")
	}, nil)
	blockBackoffUntilCancelled(t, listener.stream)
	t.Cleanup(func() { _ = listener.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	start := time.Now()
	res, err := NewWorkflow("run-connect-retry", listener).ResultWithContext(ctx)
	require.Nil(t, res)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Contains(t, err.Error(), "run-connect-retry")
	assert.Less(t, time.Since(start), time.Second)

	// One attempt, then the backoff sleep was cut short by the caller.
	assert.Equal(t, int32(1), constructorCalls.Load())
	assert.False(t, listener.reg.hasAny())
	// The caller's deadline must not end the shared listener.
	assert.False(t, listener.stream.IsClosed())
	assert.NoError(t, listener.stream.LifecycleContext().Err())
}

func TestWorkflowResultWithContextEndsSendRetriesOnCancelAndKeepsListenerUsable(t *testing.T) {
	logger := zerolog.Nop()
	sendFails := atomic.Bool{}
	sendFails.Store(true)

	client := &mockSubscribeClient{
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent, 1),
	}
	client.sendFn = func(req *dispatchercontracts.SubscribeToWorkflowRunsRequest) error {
		if sendFails.Load() {
			return status.Error(codes.Unavailable, "stream broken")
		}
		return nil
	}
	listener := newTestWorkflowRunsListener(t, &logger, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		return client, nil
	}, client)
	blockBackoffUntilCancelled(t, listener.stream)
	t.Cleanup(func() { _ = listener.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		// Cancel once the first send has failed and the loop is in backoff.
		require.Eventually(t, func() bool { return client.sendCount.Load() >= 1 }, time.Second, time.Millisecond)
		cancel()
	}()

	res, err := NewWorkflow("run-send-retry", listener).ResultWithContext(ctx)
	require.Nil(t, res)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.False(t, listener.reg.hasAny())
	assert.False(t, listener.stream.IsClosed())

	// A later waiter on the same listener still gets its result.
	sendFails.Store(false)
	assertListenerStillServes(t, listener, client, "run-after-cancel")
}

// hasRegistration reports whether any handler is registered for runID.
func hasRegistration(listener *WorkflowRunsListener, runID string) bool {
	return len(listener.reg.snapshot(runID)) > 0
}

// subscribeAfterRelease returns a constructor that blocks until release is
// closed, then hands out client. It models a dial that hangs past the
// caller's deadline.
func subscribeAfterRelease(release <-chan struct{}, client *mockSubscribeClient, calls *atomic.Int32) func(context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
	return func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		calls.Add(1)
		<-release
		return client, nil
	}
}

// assertListenerStillServes proves a later waiter on listener receives its
// result through client. The event is pushed only once runID's own handler is
// registered, so a stale registration from an earlier waiter cannot satisfy
// the wait and let the event be dropped.
func assertListenerStillServes(t *testing.T, listener *WorkflowRunsListener, client *mockSubscribeClient, runID string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		require.Eventually(t, func() bool { return hasRegistration(listener, runID) }, time.Second, time.Millisecond)
		client.recvChan <- &dispatchercontracts.WorkflowRunEvent{WorkflowRunId: runID}
	}()

	res, err := NewWorkflow(runID, listener).ResultWithContext(ctx)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.False(t, hasRegistration(listener, runID))
	// Earlier waiters' detached cleanups may still be finishing.
	require.Eventually(t, func() bool { return !listener.reg.hasAny() }, time.Second, time.Millisecond)
}

func TestWorkflowResultWithContextReturnsWhileDialBlocked(t *testing.T) {
	logger := zerolog.Nop()
	release := make(chan struct{})
	constructorCalls := atomic.Int32{}
	client := &mockSubscribeClient{
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent, 1),
	}

	listener := newTestWorkflowRunsListener(t, &logger, subscribeAfterRelease(release, client, &constructorCalls), nil)
	// Release before Close: a failed assertion must not leave the dial
	// blocked while Close waits on it.
	releaseOnce := sync.OnceFunc(func() { close(release) })
	t.Cleanup(func() {
		releaseOnce()
		_ = listener.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	start := time.Now()
	res, err := NewWorkflow("run-dial-blocked", listener).ResultWithContext(ctx)
	require.Nil(t, res)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Contains(t, err.Error(), "run-dial-blocked")
	assert.Less(t, time.Since(start), time.Second)

	// The dial is still in flight: the caller did not wait for it.
	assert.Equal(t, int32(1), constructorCalls.Load())
	assert.False(t, listener.stream.IsClosed())

	// Let the dial finish. The detached subscribe connects the stream, then
	// stores its registration and removes it again when the send step sees
	// the cancelled wait context. The listener, now connected, still serves
	// a fresh waiter, and the helper's final check proves the stale
	// registration is gone.
	releaseOnce()
	require.Eventually(t, func() bool {
		_, _, connected := listener.stream.Snapshot()
		return connected && !hasRegistration(listener, "run-dial-blocked")
	}, time.Second, time.Millisecond)
	assertListenerStillServes(t, listener, client, "run-after-dial")
}

func TestWorkflowResultWithContextReturnsWhileSendBlocked(t *testing.T) {
	logger := zerolog.Nop()
	release := make(chan struct{})
	sendBlocks := atomic.Bool{}
	sendBlocks.Store(true)

	client := &mockSubscribeClient{
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent, 1),
	}
	client.sendFn = func(req *dispatchercontracts.SubscribeToWorkflowRunsRequest) error {
		if sendBlocks.Load() {
			<-release
		}
		return nil
	}
	listener := newTestWorkflowRunsListener(t, &logger, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		return client, nil
	}, client)
	// Release before Close: the blocked Send holds sendMu, which Close also
	// needs, so a failed assertion must not leave it blocked.
	releaseOnce := sync.OnceFunc(func() {
		sendBlocks.Store(false)
		close(release)
	})
	t.Cleanup(func() {
		releaseOnce()
		_ = listener.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	// Call the listener directly: Workflow.result also removes the session
	// itself on error, which would hide the late cleanup under test here.
	start := time.Now()
	err := listener.addWorkflowRunContext(ctx, "run-send-blocked", "session-1",
		func(WorkflowRunEvent) error { return nil }, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Contains(t, err.Error(), "run-send-blocked")
	assert.Less(t, time.Since(start), time.Second)

	// The Send is still in flight and its registration is still stored.
	assert.Equal(t, int32(1), client.sendCount.Load())
	assert.True(t, hasRegistration(listener, "run-send-blocked"))
	assert.False(t, listener.stream.IsClosed())

	// Let the Send finish: it succeeds late and the detached cleanup removes
	// the registration.
	releaseOnce()
	require.Eventually(t, func() bool { return !hasRegistration(listener, "run-send-blocked") }, time.Second, time.Millisecond)
	assertListenerStillServes(t, listener, client, "run-after-send")
}
