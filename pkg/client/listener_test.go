// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/client/retry"
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

type mockDispatcherClient struct {
	subscribeToWorkflowRunsFn   func(ctx context.Context, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error)
	subscribeToWorkflowEventsFn func(ctx context.Context, in *dispatchercontracts.SubscribeToWorkflowEventsRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_SubscribeToWorkflowEventsClient, error)
}

func (m *mockDispatcherClient) Register(ctx context.Context, in *dispatchercontracts.WorkerRegisterRequest, opts ...grpc.CallOption) (*dispatchercontracts.WorkerRegisterResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) Listen(ctx context.Context, in *dispatchercontracts.WorkerListenRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_ListenClient, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) ListenV2(ctx context.Context, in *dispatchercontracts.WorkerListenRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_ListenV2Client, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) Heartbeat(ctx context.Context, in *dispatchercontracts.HeartbeatRequest, opts ...grpc.CallOption) (*dispatchercontracts.HeartbeatResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) SubscribeToWorkflowEvents(ctx context.Context, in *dispatchercontracts.SubscribeToWorkflowEventsRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_SubscribeToWorkflowEventsClient, error) {
	if m.subscribeToWorkflowEventsFn != nil {
		return m.subscribeToWorkflowEventsFn(ctx, in, opts...)
	}

	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) SubscribeToWorkflowRuns(ctx context.Context, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
	if m.subscribeToWorkflowRunsFn != nil {
		return m.subscribeToWorkflowRunsFn(ctx, opts...)
	}

	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) SendStepActionEvent(ctx context.Context, in *dispatchercontracts.StepActionEvent, opts ...grpc.CallOption) (*dispatchercontracts.ActionEventResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) SendGroupKeyActionEvent(ctx context.Context, in *dispatchercontracts.GroupKeyActionEvent, opts ...grpc.CallOption) (*dispatchercontracts.ActionEventResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) PutOverridesData(ctx context.Context, in *dispatchercontracts.OverridesData, opts ...grpc.CallOption) (*dispatchercontracts.OverridesDataResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) Unsubscribe(ctx context.Context, in *dispatchercontracts.WorkerUnsubscribeRequest, opts ...grpc.CallOption) (*dispatchercontracts.WorkerUnsubscribeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) RefreshTimeout(ctx context.Context, in *dispatchercontracts.RefreshTimeoutRequest, opts ...grpc.CallOption) (*dispatchercontracts.RefreshTimeoutResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) ReleaseSlot(ctx context.Context, in *dispatchercontracts.ReleaseSlotRequest, opts ...grpc.CallOption) (*dispatchercontracts.ReleaseSlotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) RestoreEvictedTask(ctx context.Context, in *dispatchercontracts.RestoreEvictedTaskRequest, opts ...grpc.CallOption) (*dispatchercontracts.RestoreEvictedTaskResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) UpsertWorkerLabels(ctx context.Context, in *dispatchercontracts.UpsertWorkerLabelsRequest, opts ...grpc.CallOption) (*dispatchercontracts.UpsertWorkerLabelsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockDispatcherClient) GetVersion(ctx context.Context, in *dispatchercontracts.GetVersionRequest, opts ...grpc.CallOption) (*dispatchercontracts.GetVersionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
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

func TestWorkflowRunsListenerAddWorkflowRunSendsOnceWhenStarting(t *testing.T) {
	logger := zerolog.Nop()
	recvChan := make(chan *dispatchercontracts.WorkflowRunEvent)

	client := &mockSubscribeClient{
		recvChan: recvChan,
	}

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			return client, nil
		},
		l: &logger,
	}

	require.NoError(t, listener.AddWorkflowRun("run-1", "session-1", func(event WorkflowRunEvent) error {
		return nil
	}))
	assert.Equal(t, int32(1), client.sendCount.Load())

	require.NoError(t, listener.Close())
	close(recvChan)
}

func TestGetWorkflowRunsListenerImmediateAddDoesNotOpenSecondStream(t *testing.T) {
	logger := zerolog.Nop()
	closeCh := make(chan struct{})
	var closeOnce sync.Once

	client := &mockSubscribeClient{
		recvFn: func() (*dispatchercontracts.WorkflowRunEvent, error) {
			<-closeCh
			return nil, io.EOF
		},
		closeSendFn: func() error {
			closeOnce.Do(func() {
				close(closeCh)
			})
			return nil
		},
	}

	constructorCalls := atomic.Int32{}

	subscriber := &subscribeClientImpl{
		client: &mockDispatcherClient{
			subscribeToWorkflowRunsFn: func(ctx context.Context, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
				constructorCalls.Add(1)
				return client, nil
			},
		},
		l:   &logger,
		ctx: newContextLoader("", nil),
	}

	listener, err := subscriber.getWorkflowRunsListener(context.Background())
	require.NoError(t, err)
	require.True(t, listener.isListening())

	require.NoError(t, listener.AddWorkflowRun("run-1", "session-1", func(event WorkflowRunEvent) error {
		return nil
	}))

	require.Equal(t, int32(1), constructorCalls.Load())
	require.Equal(t, int32(1), client.sendCount.Load())

	require.NoError(t, listener.Close())
	require.Eventually(t, func() bool {
		return !listener.isListening()
	}, time.Second, 10*time.Millisecond)
}

func TestGetWorkflowRunsListenerInternalCleanupDoesNotTerminallyCloseListener(t *testing.T) {
	logger := zerolog.Nop()

	initialClient := &mockSubscribeClient{
		recvErr: io.EOF,
	}
	replacementClient := &mockSubscribeClient{
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent, 1),
	}
	constructorCalls := atomic.Int32{}

	subscriber := &subscribeClientImpl{
		client: &mockDispatcherClient{
			subscribeToWorkflowRunsFn: func(ctx context.Context, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
				if constructorCalls.Add(1) == 1 {
					return initialClient, nil
				}

				return replacementClient, nil
			},
		},
		l:   &logger,
		ctx: newContextLoader("", nil),
	}

	listener, err := subscriber.getWorkflowRunsListener(context.Background())
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return !listener.isListening()
	}, time.Second, 10*time.Millisecond)

	received := make(chan WorkflowRunEvent, 1)
	require.NoError(t, listener.AddWorkflowRun("run-1", "session-1", func(event WorkflowRunEvent) error {
		received <- event
		return nil
	}))

	require.Equal(t, int32(2), constructorCalls.Load())
	require.Equal(t, int32(1), replacementClient.sendCount.Load())

	replacementClient.recvChan <- &dispatchercontracts.WorkflowRunEvent{
		WorkflowRunId: "run-1",
	}

	select {
	case event := <-received:
		assert.Equal(t, "run-1", event.WorkflowRunId)
	case <-time.After(time.Second):
		t.Fatal("expected workflow run event after internal cleanup")
	}

	listener.RemoveWorkflowRun("run-1", "session-1")
	close(replacementClient.recvChan)
	require.Eventually(t, func() bool {
		return !listener.isListening()
	}, time.Second, 10*time.Millisecond)
}

func TestRetrySend_ResubscribesOnSendFailure(t *testing.T) {
	logger := zerolog.Nop()

	constructorCalls := atomic.Int32{}

	failingClient := &mockSubscribeClient{
		sendErr:  fmt.Errorf("stream broken"),
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}

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

	err := listener.retrySend("test-workflow-run-id")

	require.NoError(t, err, "retrySend should succeed after resubscribing")
	assert.GreaterOrEqual(t, constructorCalls.Load(), int32(1), "constructor should be called for resubscribe")
	assert.GreaterOrEqual(t, failingClient.sendCount.Load(), int32(1), "failing client should have received send attempts")
	assert.Equal(t, int32(1), workingClient.sendCount.Load(), "working client should have received exactly one send")
}

func TestWorkflowRunsListenerReconnectDoesNotBlockClientSnapshot(t *testing.T) {
	logger := zerolog.Nop()
	constructorEntered := make(chan struct{})
	releaseConstructor := make(chan struct{})
	var closeConstructorEntered sync.Once

	oldClient := &mockSubscribeClient{
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}
	newClient := &mockSubscribeClient{
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			closeConstructorEntered.Do(func() {
				close(constructorEntered)
			})
			<-releaseConstructor
			return newClient, nil
		},
		client: oldClient,
		l:      &logger,
	}

	retryErr := make(chan error, 1)
	go func() {
		retryErr <- listener.retrySubscribeBackground(context.Background())
	}()

	select {
	case <-constructorEntered:
	case <-time.After(time.Second):
		t.Fatal("reconnect did not reach constructor")
	}

	snapshotRead := make(chan struct {
		client     dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient
		generation uint64
	}, 1)
	go func() {
		client, generation := listener.getClientSnapshot()
		snapshotRead <- struct {
			client     dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient
			generation uint64
		}{client: client, generation: generation}
	}()

	select {
	case snapshot := <-snapshotRead:
		require.Same(t, oldClient, snapshot.client)
		require.Equal(t, uint64(0), snapshot.generation)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("getClientSnapshot blocked behind reconnect")
	}

	close(releaseConstructor)

	require.NoError(t, <-retryErr)
	require.NoError(t, listener.Close())
}

func TestRetrySend_FailsAfterMaxRetries(t *testing.T) {
	logger := zerolog.Nop()

	failingClient := &mockSubscribeClient{
		sendErr:  fmt.Errorf("stream broken"),
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			return failingClient, nil
		},
		client: failingClient,
		l:      &logger,
	}

	err := listener.retrySend("test-workflow-run-id")

	require.Error(t, err, "retrySend should fail after exhausting retries")
	assert.Contains(t, err.Error(), "could not send to the worker after", "error should indicate retry exhaustion")
	assert.Equal(t, int32(retry.StreamSyncMaxAttempts), failingClient.sendCount.Load(),
		"should have attempted send StreamSyncMaxAttempts times")
}

func TestRetrySend_SucceedsOnFirstAttempt(t *testing.T) {
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

	assert.Equal(t, int32(numGoroutines), workingClient.sendCount.Load(), "all concurrent sends should complete")
}

func TestListen_DispatchesEventsToHandlers(t *testing.T) {
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
	listener.handlers.Store("run-1", &threadSafeHandlers{
		handlers: map[string]WorkflowRunEventHandler{
			"session-1": func(event WorkflowRunEvent) error {
				receivedEvent.Store(event)
				return nil
			},
		},
	})

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- listener.Listen(context.Background())
	}()

	recvChan <- &dispatchercontracts.WorkflowRunEvent{
		WorkflowRunId: "run-1",
	}

	require.Eventually(t, func() bool {
		return receivedEvent.Load() != nil
	}, time.Second, 10*time.Millisecond)
	listener.handlers.Delete("run-1")

	close(recvChan)

	err := <-listenErr
	assert.NoError(t, err, "Listen should exit cleanly on EOF")

	ev := receivedEvent.Load()
	require.NotNil(t, ev, "handler should have been called")
	assert.Equal(t, "run-1", ev.(WorkflowRunEvent).WorkflowRunId)
}

func TestListen_ExitsOnEOF(t *testing.T) {
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

func TestWorkflowRunsListenerRestartsAfterListenExits(t *testing.T) {
	logger := zerolog.Nop()

	initialClient := &mockSubscribeClient{
		recvErr: io.EOF,
	}
	replacementClient := &mockSubscribeClient{
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent, 1),
	}

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			return replacementClient, nil
		},
		client: initialClient,
		l:      &logger,
	}

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- listener.Listen(context.Background())
	}()

	require.NoError(t, <-listenErr)
	require.False(t, listener.isListening())

	received := make(chan WorkflowRunEvent, 1)
	require.NoError(t, listener.AddWorkflowRun("run-1", "session-1", func(event WorkflowRunEvent) error {
		received <- event
		return nil
	}))

	replacementClient.recvChan <- &dispatchercontracts.WorkflowRunEvent{
		WorkflowRunId: "run-1",
	}

	select {
	case event := <-received:
		assert.Equal(t, "run-1", event.WorkflowRunId)
	case <-time.After(time.Second):
		t.Fatal("expected workflow run event after listener restarted")
	}

	listener.RemoveWorkflowRun("run-1", "session-1")
	close(replacementClient.recvChan)
}

func TestWorkflowRunsListenerReconnectsOnEOFWithRegisteredHandlers(t *testing.T) {
	logger := zerolog.Nop()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	initialClient := &mockSubscribeClient{
		recvErr: io.EOF,
	}
	replacementClient := &mockSubscribeClient{
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent, 1),
	}
	constructorCalls := atomic.Int32{}

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			constructorCalls.Add(1)
			return replacementClient, nil
		},
		client: initialClient,
		l:      &logger,
	}

	received := make(chan WorkflowRunEvent, 1)
	listener.handlers.Store("run-1", &threadSafeHandlers{
		handlers: map[string]WorkflowRunEventHandler{
			"session-1": func(event WorkflowRunEvent) error {
				received <- event
				return nil
			},
		},
	})

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- listener.Listen(ctx)
	}()

	require.Eventually(t, func() bool {
		return constructorCalls.Load() == 1 && replacementClient.sendCount.Load() == 1
	}, time.Second, 10*time.Millisecond)

	replacementClient.recvChan <- &dispatchercontracts.WorkflowRunEvent{
		WorkflowRunId: "run-1",
	}

	select {
	case event := <-received:
		assert.Equal(t, "run-1", event.WorkflowRunId)
	case <-time.After(time.Second):
		t.Fatal("expected workflow run event after EOF reconnect")
	}

	listener.RemoveWorkflowRun("run-1", "session-1")
	close(replacementClient.recvChan)
	require.NoError(t, <-listenErr)
}

func TestWorkflowRunsListenerClosePreventsEOFReconnectAndAdd(t *testing.T) {
	logger := zerolog.Nop()
	constructorCalls := atomic.Int32{}

	initialClient := &mockSubscribeClient{
		recvErr: io.EOF,
	}

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			constructorCalls.Add(1)
			return &mockSubscribeClient{
				recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
			}, nil
		},
		client: initialClient,
		l:      &logger,
	}

	listener.handlers.Store("run-1", &threadSafeHandlers{
		handlers: map[string]WorkflowRunEventHandler{
			"session-1": func(event WorkflowRunEvent) error {
				return nil
			},
		},
	})

	require.NoError(t, listener.Close())
	require.NoError(t, listener.listen(context.Background()))
	assert.Equal(t, int32(0), constructorCalls.Load())

	err := listener.AddWorkflowRun("run-2", "session-2", func(event WorkflowRunEvent) error {
		return nil
	})
	require.ErrorIs(t, err, errListenerClosed)
}

func TestListen_ReconnectsAndUsesNewClient(t *testing.T) {
	logger := zerolog.Nop()

	newRecvChan := make(chan *dispatchercontracts.WorkflowRunEvent, 1)

	brokenClient := &mockSubscribeClient{
		recvErr: status.Error(codes.Unavailable, "stream broken"),
	}

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
	listener.handlers.Store("run-1", &threadSafeHandlers{
		handlers: map[string]WorkflowRunEventHandler{
			"session-1": func(event WorkflowRunEvent) error {
				receivedEvent.Store(event)
				return nil
			},
		},
	})

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- listener.Listen(context.Background())
	}()

	newRecvChan <- &dispatchercontracts.WorkflowRunEvent{
		WorkflowRunId: "run-1",
	}

	require.Eventually(t, func() bool {
		return receivedEvent.Load() != nil
	}, time.Second, 10*time.Millisecond)
	listener.handlers.Delete("run-1")
	close(newRecvChan)

	err := <-listenErr
	assert.NoError(t, err, "Listen should exit cleanly on EOF after reconnection")

	assert.GreaterOrEqual(t, constructorCalls.Load(), int32(1), "constructor should have been called to reconnect")

	ev := receivedEvent.Load()
	require.NotNil(t, ev, "handler should have received event from new client")
	assert.Equal(t, "run-1", ev.(WorkflowRunEvent).WorkflowRunId)
}

func TestClose_NilClient(t *testing.T) {
	logger := zerolog.Nop()

	listener := &WorkflowRunsListener{
		client: nil,
		l:      &logger,
	}

	err := listener.Close()
	assert.NoError(t, err, "Close should return nil for nil client")
}

func TestClose_CallsCloseSend(t *testing.T) {
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
	logger := zerolog.Nop()
	constructorCalls := atomic.Int32{}

	client := &mockSubscribeClient{
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			constructorCalls.Add(1)
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
			err := listener.retrySubscribeSync(context.Background())
			assert.NoError(t, err)
		}()
	}

	wg.Wait()

	assert.Equal(t, int32(1), constructorCalls.Load(),
		"concurrent retrySubscribe calls should be coalesced into a single reconnection")
}

func TestRetrySubscribe_GenerationIncrements(t *testing.T) {
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

	err := listener.retrySubscribeSync(context.Background())
	require.NoError(t, err)

	_, gen1 := listener.getClientSnapshot()
	assert.Equal(t, uint64(1), gen1, "generation should be 1 after first reconnect")

	err = listener.retrySubscribeSync(context.Background())
	require.NoError(t, err)

	_, gen2 := listener.getClientSnapshot()
	assert.Equal(t, uint64(2), gen2, "generation should be 2 after second reconnect")
}

func TestGetClientSnapshot_ReturnsCurrentClient(t *testing.T) {
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

	assert.Equal(t, workflowRunId, deprecated.WorkflowRunId)
	assert.Equal(t, stepId, deprecated.ResourceId)
	assert.Equal(t, eventPayload, deprecated.EventPayload)
}

func TestWorkflowEventToDeprecatedWorkflowEvent_WithNilTimestamp(t *testing.T) {
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
	event := &dispatchercontracts.WorkflowEvent{}

	deprecated, err := workflowEventToDeprecatedWorkflowEvent(event)

	require.NoError(t, err, "conversion should succeed with empty event")
	require.NotNil(t, deprecated, "deprecated event should not be nil")
}

func TestWorkflowRunEventToDeprecatedWorkflowRunEvent_Success(t *testing.T) {
	workflowRunId := "test-workflow-run-789"

	event := &dispatchercontracts.WorkflowRunEvent{
		WorkflowRunId:  workflowRunId,
		EventType:      dispatchercontracts.WorkflowRunEventType_WORKFLOW_RUN_EVENT_TYPE_FINISHED,
		EventTimestamp: timestamppb.New(time.Unix(9876543210, 987654321)),
	}

	deprecated, err := workflowRunEventToDeprecatedWorkflowRunEvent(event)

	require.NoError(t, err, "conversion should succeed")
	require.NotNil(t, deprecated, "deprecated event should not be nil")

	assert.Equal(t, workflowRunId, deprecated.WorkflowRunId)
	assert.NotNil(t, deprecated.EventTimestamp)
}

func TestWorkflowRunEventToDeprecatedWorkflowRunEvent_WithNilTimestamp(t *testing.T) {
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
	event := &dispatchercontracts.WorkflowRunEvent{}

	deprecated, err := workflowRunEventToDeprecatedWorkflowRunEvent(event)

	require.NoError(t, err, "conversion should succeed with empty event")
	require.NotNil(t, deprecated, "deprecated event should not be nil")
}

func TestWorkflowRunEventToDeprecatedWorkflowRunEvent_WithResults(t *testing.T) {
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

func TestAddWorkflowRunBoundedWhenEngineDown(t *testing.T) {
	logger := zerolog.Nop()
	constructorCalls := atomic.Int32{}

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			constructorCalls.Add(1)
			return nil, status.Error(codes.Unavailable, "engine down")
		},
		l: &logger,
	}

	err := listener.AddWorkflowRun("run-1", "session-1", func(event WorkflowRunEvent) error {
		return nil
	})

	require.Error(t, err)
	assert.LessOrEqual(t, constructorCalls.Load(), int32(retry.StreamSyncMaxAttempts+1))
}

func TestDoRetrySubscribeSyncStopsAtStreamSyncMaxAttempts(t *testing.T) {
	logger := zerolog.Nop()
	constructorCalls := atomic.Int32{}

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			constructorCalls.Add(1)
			return nil, status.Error(codes.Unavailable, "still down")
		},
		l: &logger,
	}

	err := listener.doRetrySubscribeSync(context.Background())
	require.Error(t, err)
	assert.Equal(t, int32(retry.StreamSyncMaxAttempts), constructorCalls.Load())
}

func TestDoRetrySubscribeBackgroundContinuesPastSyncCap(t *testing.T) {
	disableStreamBackoffForTest(t)

	logger := zerolog.Nop()
	constructorCalls := atomic.Int32{}
	ctx, cancel := context.WithCancel(context.Background())

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			if constructorCalls.Add(1) >= retry.StreamSyncMaxAttempts+1 {
				cancel()
			}
			return nil, status.Error(codes.Unavailable, "still down")
		},
		l: &logger,
	}

	done := make(chan struct{})
	go func() {
		_ = listener.doRetrySubscribeBackground(ctx)
		close(done)
	}()

	select {
	case <-ctx.Done():
	case <-time.After(30 * time.Second):
		t.Fatal("background reconnect did not continue past sync cap")
	}

	<-done
	assert.Greater(t, constructorCalls.Load(), int32(retry.StreamSyncMaxAttempts))
}

func TestListenEOFWithoutHandlersDoesNotReconnect(t *testing.T) {
	logger := zerolog.Nop()
	constructorCalls := atomic.Int32{}

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			constructorCalls.Add(1)
			return &mockSubscribeClient{recvChan: make(chan *dispatchercontracts.WorkflowRunEvent)}, nil
		},
		client: &mockSubscribeClient{recvErr: io.EOF},
		l:      &logger,
	}

	require.NoError(t, listener.Listen(context.Background()))
	assert.Equal(t, int32(0), constructorCalls.Load())
}

func TestListenPermanentErrorStops(t *testing.T) {
	logger := zerolog.Nop()

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			return nil, status.Error(codes.PermissionDenied, "denied")
		},
		client: &mockSubscribeClient{
			recvErr: status.Error(codes.Unavailable, "stream broken"),
		},
		l: &logger,
	}

	err := listener.Listen(context.Background())
	require.Error(t, err)
}

func TestRetrySubscribeSyncReplaysHandlers(t *testing.T) {
	logger := zerolog.Nop()
	sendCount := atomic.Int32{}

	client := &mockSubscribeClient{
		sendFn: func(req *dispatchercontracts.SubscribeToWorkflowRunsRequest) error {
			sendCount.Add(1)
			return nil
		},
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			return client, nil
		},
		l: &logger,
	}

	listener.handlers.Store("run-1", &threadSafeHandlers{
		handlers: map[string]WorkflowRunEventHandler{
			"session-1": func(event WorkflowRunEvent) error { return nil },
		},
	})

	require.NoError(t, listener.retrySubscribeSync(context.Background()))
	assert.Equal(t, int32(1), sendCount.Load())
}

func TestRetrySendStaleGenerationSkipsReconnect(t *testing.T) {
	logger := zerolog.Nop()
	constructorCalls := atomic.Int32{}

	workingClient := &mockSubscribeClient{recvChan: make(chan *dispatchercontracts.WorkflowRunEvent)}
	failingClient := &mockSubscribeClient{
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			constructorCalls.Add(1)
			return workingClient, nil
		},
		client: failingClient,
		l:      &logger,
	}

	failingClient.sendFn = func(req *dispatchercontracts.SubscribeToWorkflowRunsRequest) error {
		require.NoError(t, listener.streamCore().installClient(workingClient))
		return status.Error(codes.Unavailable, "send failed")
	}

	require.NoError(t, listener.retrySend("run-1"))
	assert.Equal(t, int32(0), constructorCalls.Load())
}

func TestSubscribeToWorkflowRunsDisablesGrpcRetry(t *testing.T) {
	logger := zerolog.Nop()

	subscriber := &subscribeClientImpl{
		client: &mockDispatcherClient{
			subscribeToWorkflowRunsFn: func(ctx context.Context, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
				require.NotEmpty(t, opts)
				return &mockSubscribeClient{recvChan: make(chan *dispatchercontracts.WorkflowRunEvent)}, nil
			},
		},
		l:   &logger,
		ctx: newContextLoader("", nil),
	}

	listener, err := subscriber.getWorkflowRunsListener(context.Background())
	require.NoError(t, err)
	require.NoError(t, listener.Close())
}

type mockWorkflowEventsClient struct {
	recvFn      func() (*dispatchercontracts.WorkflowEvent, error)
	recvCh      chan *dispatchercontracts.WorkflowEvent
	establishFn func(ctx context.Context, opts ...grpc.CallOption) error
}

func (m *mockWorkflowEventsClient) Recv() (*dispatchercontracts.WorkflowEvent, error) {
	if m.recvFn != nil {
		return m.recvFn()
	}
	event, ok := <-m.recvCh
	if !ok {
		return nil, io.EOF
	}
	return event, nil
}

func (m *mockWorkflowEventsClient) CloseSend() error              { return nil }
func (m *mockWorkflowEventsClient) Header() (metadata.MD, error)  { return nil, nil }
func (m *mockWorkflowEventsClient) Trailer() metadata.MD          { return nil }
func (m *mockWorkflowEventsClient) Context() context.Context      { return context.Background() }
func (m *mockWorkflowEventsClient) SendMsg(msg interface{}) error { return nil }
func (m *mockWorkflowEventsClient) RecvMsg(msg interface{}) error { return nil }

func TestStreamByAdditionalMetadataReconnects(t *testing.T) {
	disableStreamBackoffForTest(t)

	logger := zerolog.Nop()
	establishCalls := atomic.Int32{}
	recvCh := make(chan *dispatchercontracts.WorkflowEvent, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subscriber := &subscribeClientImpl{
		client: &mockDispatcherClient{
			subscribeToWorkflowEventsFn: func(ctx context.Context, in *dispatchercontracts.SubscribeToWorkflowEventsRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_SubscribeToWorkflowEventsClient, error) {
				require.NotEmpty(t, opts)
				call := establishCalls.Add(1)
				if call == 1 {
					return &mockWorkflowEventsClient{
						recvFn: func() (*dispatchercontracts.WorkflowEvent, error) {
							return nil, status.Error(codes.Unavailable, "stream broken")
						},
					}, nil
				}

				return &mockWorkflowEventsClient{recvCh: recvCh}, nil
			},
		},
		l:   &logger,
		ctx: newContextLoader("", nil),
	}

	done := make(chan error, 1)
	go func() {
		done <- subscriber.StreamByAdditionalMetadata(ctx, "k", "v", func(event StreamEvent) error {
			cancel()
			return nil
		})
	}()

	recvCh <- &dispatchercontracts.WorkflowEvent{
		EventType:    dispatchercontracts.ResourceEventType_RESOURCE_EVENT_TYPE_STREAM,
		EventPayload: "hello",
	}

	err := <-done
	require.ErrorIs(t, err, context.Canceled)
	assert.GreaterOrEqual(t, establishCalls.Load(), int32(2))
}

func TestStreamByAdditionalMetadataReconnectsOnEOF(t *testing.T) {
	disableStreamBackoffForTest(t)

	logger := zerolog.Nop()
	establishCalls := atomic.Int32{}
	handlerCalls := atomic.Int32{}
	secondRecvCh := make(chan *dispatchercontracts.WorkflowEvent, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subscriber := &subscribeClientImpl{
		client: &mockDispatcherClient{
			subscribeToWorkflowEventsFn: func(ctx context.Context, in *dispatchercontracts.SubscribeToWorkflowEventsRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_SubscribeToWorkflowEventsClient, error) {
				require.NotEmpty(t, opts)
				call := establishCalls.Add(1)
				if call == 1 {
					return &mockWorkflowEventsClient{
						recvFn: func() (*dispatchercontracts.WorkflowEvent, error) {
							return nil, io.EOF
						},
					}, nil
				}

				return &mockWorkflowEventsClient{recvCh: secondRecvCh}, nil
			},
		},
		l:   &logger,
		ctx: newContextLoader("", nil),
	}

	done := make(chan error, 1)
	go func() {
		done <- subscriber.StreamByAdditionalMetadata(ctx, "k", "v", func(event StreamEvent) error {
			handlerCalls.Add(1)
			cancel()
			return nil
		})
	}()

	secondRecvCh <- &dispatchercontracts.WorkflowEvent{
		EventType:    dispatchercontracts.ResourceEventType_RESOURCE_EVENT_TYPE_STREAM,
		EventPayload: "hello",
	}

	err := <-done
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, int32(1), handlerCalls.Load())
	assert.GreaterOrEqual(t, establishCalls.Load(), int32(2))
}

func TestSubscribeToWorkflowRunEventsRespectsCancelledContext(t *testing.T) {
	logger := zerolog.Nop()

	subscriber := &subscribeClientImpl{
		client: &mockDispatcherClient{
			subscribeToWorkflowRunsFn: func(ctx context.Context, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
				t.Fatal("constructor should not run when caller context is already cancelled")
				return nil, nil
			},
		},
		l:   &logger,
		ctx: newContextLoader("", nil),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := subscriber.SubscribeToWorkflowRunEvents(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestSubscribeToWorkflowRunEventsPassesCallerContextToInitialSubscribe(t *testing.T) {
	logger := zerolog.Nop()
	callerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subscriber := &subscribeClientImpl{
		client: &mockDispatcherClient{
			subscribeToWorkflowRunsFn: func(ctx context.Context, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
				assert.NotEqual(t, context.Background(), ctx)
				cancel()
				select {
				case <-ctx.Done():
				case <-time.After(time.Second):
					t.Fatal("constructor context was not cancelled with caller context")
				}
				return nil, status.Error(codes.Unavailable, "engine down")
			},
		},
		l:   &logger,
		ctx: newContextLoader("", nil),
	}

	_, err := subscriber.SubscribeToWorkflowRunEvents(callerCtx)
	require.Error(t, err)
}

func TestSubscribeToWorkflowRunEventsRespectsDeadlineContext(t *testing.T) {
	logger := zerolog.Nop()
	constructorCalls := atomic.Int32{}

	subscriber := &subscribeClientImpl{
		client: &mockDispatcherClient{
			subscribeToWorkflowRunsFn: func(ctx context.Context, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
				constructorCalls.Add(1)
				<-ctx.Done()
				return nil, ctx.Err()
			},
		},
		l:   &logger,
		ctx: newContextLoader("", nil),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := subscriber.SubscribeToWorkflowRunEvents(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Equal(t, int32(1), constructorCalls.Load())
}

func TestDoRetrySubscribeBackgroundStopsOnNoProgressError(t *testing.T) {
	disableStreamBackoffForTest(t)

	logger := zerolog.Nop()
	constructorCalls := atomic.Int32{}

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			constructorCalls.Add(1)
			return nil, fmt.Errorf("plain subscribe error")
		},
		l: &logger,
	}

	err := listener.doRetrySubscribeBackground(context.Background())
	require.Error(t, err)
	assert.Equal(t, int32(1), constructorCalls.Load())
}

func TestReconnectSyncAndBackgroundSerializeConnect(t *testing.T) {
	logger := zerolog.Nop()
	releaseConstructor := make(chan struct{})
	var concurrentConnect atomic.Int32
	var maxConcurrent atomic.Int32

	client := &mockSubscribeClient{
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			current := concurrentConnect.Add(1)
			for {
				prev := maxConcurrent.Load()
				if current <= prev {
					break
				}
				if maxConcurrent.CompareAndSwap(prev, current) {
					break
				}
			}
			defer concurrentConnect.Add(-1)

			<-releaseConstructor
			return client, nil
		},
		client: client,
		l:      &logger,
	}

	syncDone := make(chan error, 1)
	backgroundDone := make(chan error, 1)

	go func() {
		syncDone <- listener.retrySubscribeSync(context.Background())
	}()
	go func() {
		backgroundDone <- listener.retrySubscribeBackground(context.Background())
	}()

	require.Eventually(t, func() bool {
		return concurrentConnect.Load() >= 1
	}, time.Second, 10*time.Millisecond)

	assert.Equal(t, int32(1), concurrentConnect.Load())
	assert.Equal(t, int32(1), maxConcurrent.Load())

	close(releaseConstructor)

	require.NoError(t, <-syncDone)
	require.NoError(t, <-backgroundDone)
}

func TestListenRetriesUnknownCodeBeforeGivingUp(t *testing.T) {
	disableStreamBackoffForTest(t)

	logger := zerolog.Nop()
	recvCalls := atomic.Int32{}
	constructorCalls := atomic.Int32{}

	replacementClient := &mockSubscribeClient{
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent, 1),
	}

	brokenClient := &mockSubscribeClient{
		recvFn: func() (*dispatchercontracts.WorkflowRunEvent, error) {
			recvCalls.Add(1)
			return nil, status.Error(codes.Unknown, "unknown stream error")
		},
	}

	listener := &WorkflowRunsListener{
		constructor: func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			constructorCalls.Add(1)
			return replacementClient, nil
		},
		client: brokenClient,
		l:      &logger,
	}

	listener.handlers.Store("run-1", &threadSafeHandlers{
		handlers: map[string]WorkflowRunEventHandler{
			"session-1": func(event WorkflowRunEvent) error { return nil },
		},
	})

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- listener.Listen(context.Background())
	}()

	require.Eventually(t, func() bool {
		return constructorCalls.Load() >= 1
	}, time.Second, 10*time.Millisecond)

	replacementClient.recvChan <- &dispatchercontracts.WorkflowRunEvent{
		WorkflowRunId: "run-1",
	}

	require.Eventually(t, func() bool {
		return len(replacementClient.recvChan) == 0
	}, time.Second, 10*time.Millisecond)

	listener.handlers.Delete("run-1")
	close(replacementClient.recvChan)

	err := <-listenErr
	assert.NoError(t, err)
	assert.Equal(t, int32(1), recvCalls.Load())
}
