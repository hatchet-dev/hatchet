package client

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

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
	recvFn      func() (*dispatchercontracts.WorkflowRunEvent, error)
}

func (m *mockSubscribeClient) Send(req *dispatchercontracts.SubscribeToWorkflowRunsRequest) error {
	m.sendCount.Add(1)
	if m.sendFn != nil {
		return m.sendFn(req)
	}
	return m.sendErr
}

func (m *mockSubscribeClient) Recv() (*dispatchercontracts.WorkflowRunEvent, error) {
	if m.recvFn != nil {
		return m.recvFn()
	}
	if m.recvErr != nil {
		return nil, m.recvErr
	}
	event, ok := <-m.recvChan
	if !ok {
		return nil, io.EOF
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

func TestListen_DispatchesEventsToHandlers(t *testing.T) {
	// Verifies that Listen receives events and dispatches them to registered handlers.

	logger := zerolog.Nop()
	recvChan := make(chan *dispatchercontracts.WorkflowRunEvent, 1)

	client := &mockSubscribeClient{
		recvChan: recvChan,
	}

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			return client, nil
		},
		client: client,
		l:      &logger,
	}

	var receivedEvent atomic.Value
	listener.AddWorkflowRun("run-1", "session-1", func(event WorkflowRunEvent) error {
		receivedEvent.Store(event)
		return nil
	})

	// Start Listen in a goroutine
	listenErr := make(chan error, 1)
	go func() {
		listenErr <- listener.Listen(context.Background())
	}()

	// Send an event
	recvChan <- &dispatchercontracts.WorkflowRunEvent{
		WorkflowRunId: "run-1",
	}

	// Close the channel to trigger EOF and cleanly end Listen
	close(recvChan)

	err := <-listenErr
	assert.NoError(t, err, "Listen should exit cleanly on EOF")

	// Verify handler was called
	ev := receivedEvent.Load()
	require.NotNil(t, ev, "handler should have been called")
	assert.Equal(t, "run-1", ev.(WorkflowRunEvent).WorkflowRunId)
}

func TestListen_ExitsOnEOF(t *testing.T) {
	// Verifies that Listen returns nil (clean exit) on io.EOF.

	logger := zerolog.Nop()

	client := &mockSubscribeClient{
		recvErr: io.EOF,
	}

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			return client, nil
		},
		client: client,
		l:      &logger,
	}

	err := listener.Listen(context.Background())
	assert.NoError(t, err, "Listen should return nil on EOF")
}

func TestListen_ExitsOnCanceled(t *testing.T) {
	// Verifies that Listen returns nil on gRPC Canceled status.

	logger := zerolog.Nop()

	client := &mockSubscribeClient{
		recvErr: status.Error(codes.Canceled, "context canceled"),
	}

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			return client, nil
		},
		client: client,
		l:      &logger,
	}

	err := listener.Listen(context.Background())
	assert.NoError(t, err, "Listen should return nil on Canceled")
}

func TestListen_ReconnectsAndUsesNewClient(t *testing.T) {
	// Verifies that after a Recv error, Listen reconnects and reads from the new client.

	logger := zerolog.Nop()

	newRecvChan := make(chan *dispatchercontracts.WorkflowRunEvent, 1)

	// First client errors immediately on Recv
	brokenClient := &mockSubscribeClient{
		recvErr: fmt.Errorf("stream broken"),
	}

	// Second client delivers an event then closes cleanly (EOF)
	workingClient := &mockSubscribeClient{
		recvChan: newRecvChan,
	}

	constructorCalls := atomic.Int32{}

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			constructorCalls.Add(1)
			return workingClient, nil
		},
		client: brokenClient,
		l:      &logger,
	}

	var receivedEvent atomic.Value
	listener.AddWorkflowRun("run-1", "session-1", func(event WorkflowRunEvent) error {
		receivedEvent.Store(event)
		return nil
	})

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- listener.Listen(context.Background())
	}()

	// Send an event on the new client, then close to trigger EOF (clean exit)
	newRecvChan <- &dispatchercontracts.WorkflowRunEvent{
		WorkflowRunId: "run-1",
	}
	close(newRecvChan)

	err := <-listenErr
	assert.NoError(t, err, "Listen should exit cleanly on EOF after reconnection")

	// Verify the constructor was called for reconnection
	assert.GreaterOrEqual(t, constructorCalls.Load(), int32(1), "constructor should have been called to reconnect")

	// Verify handler received the event from the new client
	ev := receivedEvent.Load()
	require.NotNil(t, ev, "handler should have received event from new client")
	assert.Equal(t, "run-1", ev.(WorkflowRunEvent).WorkflowRunId)
}

func TestClose_NilClient(t *testing.T) {
	// Verifies that Close returns nil when client is nil.

	logger := zerolog.Nop()

	listener := &WorkflowRunsListener{
		client: nil,
		l:      &logger,
	}

	err := listener.Close()
	assert.NoError(t, err, "Close should return nil for nil client")
}

func TestClose_CallsCloseSend(t *testing.T) {
	// Verifies that Close calls CloseSend on the client.

	logger := zerolog.Nop()
	closeCalled := atomic.Bool{}

	client := &mockSubscribeClient{
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
		closeSendFn: func() error {
			closeCalled.Store(true)
			return nil
		},
	}

	listener := &WorkflowRunsListener{
		client: client,
		l:      &logger,
	}

	err := listener.Close()
	assert.NoError(t, err)
	assert.True(t, closeCalled.Load(), "CloseSend should have been called")
}

func TestRetrySubscribe_SingleflightCoalescesConcurrentCalls(t *testing.T) {
	// Verifies that concurrent retrySubscribe calls are coalesced into one
	// actual reconnection via singleflight.

	logger := zerolog.Nop()
	constructorCalls := atomic.Int32{}

	client := &mockSubscribeClient{
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			constructorCalls.Add(1)
			// Simulate some latency so concurrent calls overlap
			time.Sleep(50 * time.Millisecond)
			return client, nil
		},
		client: client,
		l:      &logger,
	}

	numGoroutines := 10
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := listener.retrySubscribe(context.Background())
			assert.NoError(t, err)
		}()
	}

	wg.Wait()

	// singleflight should coalesce all concurrent calls into 1 constructor invocation
	assert.Equal(t, int32(1), constructorCalls.Load(),
		"concurrent retrySubscribe calls should be coalesced into a single reconnection")
}

func TestRetrySubscribe_GenerationIncrements(t *testing.T) {
	// Verifies that the generation counter increments on each successful reconnection.

	logger := zerolog.Nop()

	client := &mockSubscribeClient{
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			return client, nil
		},
		client: client,
		l:      &logger,
	}

	_, gen0 := listener.getClientSnapshot()
	assert.Equal(t, uint64(0), gen0, "initial generation should be 0")

	err := listener.retrySubscribe(context.Background())
	require.NoError(t, err)

	_, gen1 := listener.getClientSnapshot()
	assert.Equal(t, uint64(1), gen1, "generation should be 1 after first reconnect")

	err = listener.retrySubscribe(context.Background())
	require.NoError(t, err)

	_, gen2 := listener.getClientSnapshot()
	assert.Equal(t, uint64(2), gen2, "generation should be 2 after second reconnect")
}

func TestGetClientSnapshot_ReturnsCurrentClient(t *testing.T) {
	// Verifies that getClientSnapshot returns the current client and generation.

	logger := zerolog.Nop()

	client1 := &mockSubscribeClient{recvChan: make(chan *dispatchercontracts.WorkflowRunEvent)}

	listener := &WorkflowRunsListener{
		client: client1,
		l:      &logger,
	}

	got, gen := listener.getClientSnapshot()
	assert.Equal(t, client1, got)
	assert.Equal(t, uint64(0), gen)
}

func TestWorkflowEventToDeprecatedWorkflowEvent_Success(t *testing.T) {
	// Verifies successful conversion of WorkflowEvent to deprecated WorkflowEvent

	workflowRunId := "test-workflow-run-123"
	stepId := "test-step-456"
	eventPayload := "test-payload"

	event := &dispatchercontracts.WorkflowEvent{
		WorkflowRunId:  workflowRunId,
		ResourceType:   dispatchercontracts.ResourceType_RESOURCE_TYPE_STEP_RUN,
		ResourceId:     stepId,
		EventType:      dispatchercontracts.ResourceEventType_RESOURCE_EVENT_TYPE_STARTED,
		EventPayload:   eventPayload,
		EventTimestamp: timestamppb.New(time.Unix(1234567890, 123456789)),
	}

	deprecated, err := workflowEventToDeprecatedWorkflowEvent(event)

	require.NoError(t, err, "conversion should succeed")
	require.NotNil(t, deprecated, "deprecated event should not be nil")

	// Verify key fields are preserved
	assert.Equal(t, workflowRunId, deprecated.WorkflowRunId)
	assert.Equal(t, stepId, deprecated.ResourceId)
	assert.Equal(t, eventPayload, deprecated.EventPayload)
}

func TestWorkflowEventToDeprecatedWorkflowEvent_WithNilTimestamp(t *testing.T) {
	// Verifies conversion works when timestamp is nil

	event := &dispatchercontracts.WorkflowEvent{
		WorkflowRunId:  "test-run",
		ResourceId:     "test-resource",
		EventType:      dispatchercontracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED,
		EventPayload:   "payload",
		EventTimestamp: nil,
	}

	deprecated, err := workflowEventToDeprecatedWorkflowEvent(event)

	require.NoError(t, err, "conversion should succeed even with nil timestamp")
	require.NotNil(t, deprecated, "deprecated event should not be nil")
	assert.Equal(t, "test-run", deprecated.WorkflowRunId)
}

func TestWorkflowEventToDeprecatedWorkflowEvent_EmptyEvent(t *testing.T) {
	// Verifies conversion works with an empty event

	event := &dispatchercontracts.WorkflowEvent{}

	deprecated, err := workflowEventToDeprecatedWorkflowEvent(event)

	require.NoError(t, err, "conversion should succeed with empty event")
	require.NotNil(t, deprecated, "deprecated event should not be nil")
}

func TestWorkflowRunEventToDeprecatedWorkflowRunEvent_Success(t *testing.T) {
	// Verifies successful conversion of WorkflowRunEvent to deprecated WorkflowRunEvent

	workflowRunId := "test-workflow-run-789"

	event := &dispatchercontracts.WorkflowRunEvent{
		WorkflowRunId:  workflowRunId,
		EventType:      dispatchercontracts.WorkflowRunEventType_WORKFLOW_RUN_EVENT_TYPE_FINISHED,
		EventTimestamp: timestamppb.New(time.Unix(9876543210, 987654321)),
	}

	deprecated, err := workflowRunEventToDeprecatedWorkflowRunEvent(event)

	require.NoError(t, err, "conversion should succeed")
	require.NotNil(t, deprecated, "deprecated event should not be nil")

	// Verify key fields are preserved
	assert.Equal(t, workflowRunId, deprecated.WorkflowRunId)
	assert.NotNil(t, deprecated.EventTimestamp)
}

func TestWorkflowRunEventToDeprecatedWorkflowRunEvent_WithNilTimestamp(t *testing.T) {
	// Verifies conversion works when timestamp is nil

	event := &dispatchercontracts.WorkflowRunEvent{
		WorkflowRunId:  "test-run-2",
		EventType:      dispatchercontracts.WorkflowRunEventType_WORKFLOW_RUN_EVENT_TYPE_FINISHED,
		EventTimestamp: nil,
	}

	deprecated, err := workflowRunEventToDeprecatedWorkflowRunEvent(event)

	require.NoError(t, err, "conversion should succeed even with nil timestamp")
	require.NotNil(t, deprecated, "deprecated event should not be nil")
	assert.Equal(t, "test-run-2", deprecated.WorkflowRunId)
}

func TestWorkflowRunEventToDeprecatedWorkflowRunEvent_EmptyEvent(t *testing.T) {
	// Verifies conversion works with an empty event

	event := &dispatchercontracts.WorkflowRunEvent{}

	deprecated, err := workflowRunEventToDeprecatedWorkflowRunEvent(event)

	require.NoError(t, err, "conversion should succeed with empty event")
	require.NotNil(t, deprecated, "deprecated event should not be nil")
}

func TestWorkflowRunEventToDeprecatedWorkflowRunEvent_WithResults(t *testing.T) {
	// Verifies conversion works with step run results

	event := &dispatchercontracts.WorkflowRunEvent{
		WorkflowRunId:  "test-run-3",
		EventType:      dispatchercontracts.WorkflowRunEventType_WORKFLOW_RUN_EVENT_TYPE_FINISHED,
		EventTimestamp: timestamppb.New(time.Now()),
		Results: []*dispatchercontracts.StepRunResult{
			{
				TaskRunExternalId: "step-1",
			},
		},
	}

	deprecated, err := workflowRunEventToDeprecatedWorkflowRunEvent(event)

	require.NoError(t, err, "conversion should succeed with results")
	require.NotNil(t, deprecated, "deprecated event should not be nil")
	assert.Equal(t, "test-run-3", deprecated.WorkflowRunId)
	assert.NotNil(t, deprecated.Results)
}
