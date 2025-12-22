package client

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
)

// mockSubscribeClient implements dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient
type mockSubscribeClient struct {
	sendErr     error
	sendCount   atomic.Int32
	recvErr     error
	recvChan    chan *dispatchercontracts.WorkflowRunEvent
	closeSendFn func() error
	sendFn      func(req *dispatchercontracts.SubscribeToWorkflowRunsRequest) error
}

func (m *mockSubscribeClient) Send(req *dispatchercontracts.SubscribeToWorkflowRunsRequest) error {
	m.sendCount.Add(1)
	if m.sendFn != nil {
		return m.sendFn(req)
	}
	return m.sendErr
}

func (m *mockSubscribeClient) Recv() (*dispatchercontracts.WorkflowRunEvent, error) {
	if m.recvErr != nil {
		return nil, m.recvErr
	}
	event, ok := <-m.recvChan
	if !ok {
		return nil, fmt.Errorf("channel closed")
	}
	return event, nil
}

func (m *mockSubscribeClient) CloseSend() error {
	if m.closeSendFn != nil {
		return m.closeSendFn()
	}
	return nil
}

func (m *mockSubscribeClient) Header() (metadata.MD, error) {
	return nil, nil
}

func (m *mockSubscribeClient) Trailer() metadata.MD {
	return nil
}

func (m *mockSubscribeClient) Context() context.Context {
	return context.Background()
}

func (m *mockSubscribeClient) SendMsg(msg interface{}) error {
	return nil
}

func (m *mockSubscribeClient) RecvMsg(msg interface{}) error {
	return nil
}

func TestRetrySend_ResubscribesOnSendFailure(t *testing.T) {
	// This test verifies that when Send() fails on a broken stream,
	// retrySend will call retrySubscribe to establish a new stream
	// and then successfully send on the new stream.

	logger := zerolog.Nop()

	// Track how many times the constructor is called
	constructorCalls := atomic.Int32{}

	// First client will fail on Send
	failingClient := &mockSubscribeClient{
		sendErr:  fmt.Errorf("stream broken"),
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}

	// Second client will succeed
	workingClient := &mockSubscribeClient{
		sendErr:  nil,
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			calls := constructorCalls.Add(1)
			if calls == 1 {
				// First call during initial setup
				return failingClient, nil
			}
			// Subsequent calls after resubscribe
			return workingClient, nil
		},
		client: failingClient,
		l:      &logger,
	}

	// Call retrySend - it should fail on first attempt, resubscribe, then succeed
	err := listener.retrySend("test-workflow-run-id")

	require.NoError(t, err, "retrySend should succeed after resubscribing")

	// Verify constructor was called (for resubscribe)
	assert.GreaterOrEqual(t, constructorCalls.Load(), int32(1), "constructor should be called for resubscribe")

	// Verify the failing client received at least one send attempt
	assert.GreaterOrEqual(t, failingClient.sendCount.Load(), int32(1), "failing client should have received send attempts")

	// Verify the working client received a send
	assert.Equal(t, int32(1), workingClient.sendCount.Load(), "working client should have received exactly one send")
}

func TestRetrySend_FailsAfterMaxRetries(t *testing.T) {
	// This test verifies that retrySend returns an error after
	// exhausting all retry attempts when resubscribe also fails.

	logger := zerolog.Nop()

	// Client that always fails
	failingClient := &mockSubscribeClient{
		sendErr:  fmt.Errorf("stream broken"),
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			// Constructor also returns a failing client
			return failingClient, nil
		},
		client: failingClient,
		l:      &logger,
	}

	// Call retrySend - it should fail after all retries
	err := listener.retrySend("test-workflow-run-id")

	require.Error(t, err, "retrySend should fail after exhausting retries")
	assert.Contains(t, err.Error(), "could not send to the worker after", "error should indicate retry exhaustion")

	// Verify multiple send attempts were made
	assert.Equal(t, int32(DefaultActionListenerRetryCount), failingClient.sendCount.Load(),
		"should have attempted send DefaultActionListenerRetryCount times")
}

func TestRetrySend_SucceedsOnFirstAttempt(t *testing.T) {
	// This test verifies that retrySend returns immediately
	// when Send() succeeds on the first attempt.

	logger := zerolog.Nop()

	workingClient := &mockSubscribeClient{
		sendErr:  nil,
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}

	constructorCalls := atomic.Int32{}

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			constructorCalls.Add(1)
			return workingClient, nil
		},
		client: workingClient,
		l:      &logger,
	}

	err := listener.retrySend("test-workflow-run-id")

	require.NoError(t, err, "retrySend should succeed on first attempt")
	assert.Equal(t, int32(1), workingClient.sendCount.Load(), "should have sent exactly once")
	assert.Equal(t, int32(0), constructorCalls.Load(), "constructor should not be called when first send succeeds")
}

func TestRetrySend_HandlesNilClient(t *testing.T) {
	// This test verifies that retrySend returns an error when client is nil

	logger := zerolog.Nop()

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			return nil, fmt.Errorf("connection failed")
		},
		client: nil,
		l:      &logger,
	}

	err := listener.retrySend("test-workflow-run-id")

	require.Error(t, err, "retrySend should fail when client is nil")
	assert.Contains(t, err.Error(), "client is not connected", "error should indicate client not connected")
}

func TestRetrySend_ConcurrentSafety(t *testing.T) {
	// This test verifies that retrySend is safe to call concurrently

	logger := zerolog.Nop()

	workingClient := &mockSubscribeClient{
		sendErr:  nil,
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
		sendFn: func(req *dispatchercontracts.SubscribeToWorkflowRunsRequest) error {
			time.Sleep(10 * time.Millisecond) // Simulate some work
			return nil
		},
	}

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			return workingClient, nil
		},
		client: workingClient,
		l:      &logger,
	}

	// Launch multiple concurrent retrySend calls
	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			err := listener.retrySend(fmt.Sprintf("workflow-run-%d", id))
			assert.NoError(t, err, "concurrent retrySend should succeed")
		}(i)
	}

	wg.Wait()

	// All sends should have completed
	assert.Equal(t, int32(numGoroutines), workingClient.sendCount.Load(), "all concurrent sends should complete")
}
