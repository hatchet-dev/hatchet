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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
)

type mockCoreStream struct {
	sendErr     error
	sendFn      func(*dispatchercontracts.SubscribeToWorkflowRunsRequest) error
	recvFn      func() (*dispatchercontracts.WorkflowRunEvent, error)
	recvCh      chan *dispatchercontracts.WorkflowRunEvent
	closeSendFn func() error
	sendCount   atomic.Int32
}

func (m *mockCoreStream) Send(req *dispatchercontracts.SubscribeToWorkflowRunsRequest) error {
	m.sendCount.Add(1)
	if m.sendFn != nil {
		return m.sendFn(req)
	}

	return m.sendErr
}

func (m *mockCoreStream) Recv() (*dispatchercontracts.WorkflowRunEvent, error) {
	if m.recvFn != nil {
		return m.recvFn()
	}

	if m.recvCh != nil {
		event, ok := <-m.recvCh
		if !ok {
			return nil, io.EOF
		}

		return event, nil
	}

	return nil, io.EOF
}

func (m *mockCoreStream) CloseSend() error {
	if m.closeSendFn != nil {
		return m.closeSendFn()
	}

	return nil
}

func newTestReconnectingListener(
	client *mockCoreStream,
	constructor func(context.Context) (*mockCoreStream, error),
) *reconnectingListener[string, *dispatchercontracts.SubscribeToWorkflowRunsRequest, *dispatchercontracts.WorkflowRunEvent, *mockCoreStream] {
	logger := zerolog.Nop()

	return &reconnectingListener[string, *dispatchercontracts.SubscribeToWorkflowRunsRequest, *dispatchercontracts.WorkflowRunEvent, *mockCoreStream]{
		constructor: constructor,
		client:      client,
		l:           &logger,
		requestForKey: func(workflowRunId string) *dispatchercontracts.SubscribeToWorkflowRunsRequest {
			return &dispatchercontracts.SubscribeToWorkflowRunsRequest{
				WorkflowRunId: workflowRunId,
			}
		},
		keyForEvent: func(event *dispatchercontracts.WorkflowRunEvent) string {
			return event.WorkflowRunId
		},
	}
}

func TestReconnectingListener_SingleflightCoalescesConcurrentCalls(t *testing.T) {
	client := &mockCoreStream{}
	constructorCalls := atomic.Int32{}

	listener := newTestReconnectingListener(client, func(ctx context.Context) (*mockCoreStream, error) {
		constructorCalls.Add(1)
		time.Sleep(50 * time.Millisecond)
		return client, nil
	})

	numGoroutines := 10
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			assert.NoError(t, listener.retrySubscribe(context.Background()))
		}()
	}

	wg.Wait()
	assert.Equal(t, int32(1), constructorCalls.Load())
}

func TestReconnectingListener_GenerationIncrements(t *testing.T) {
	client := &mockCoreStream{}
	listener := newTestReconnectingListener(client, func(ctx context.Context) (*mockCoreStream, error) {
		return client, nil
	})

	_, gen0, _ := listener.getClientSnapshot()
	assert.Equal(t, uint64(0), gen0)

	require.NoError(t, listener.retrySubscribe(context.Background()))
	_, gen1, _ := listener.getClientSnapshot()
	assert.Equal(t, uint64(1), gen1)

	require.NoError(t, listener.retrySubscribe(context.Background()))
	_, gen2, _ := listener.getClientSnapshot()
	assert.Equal(t, uint64(2), gen2)
}

func TestReconnectingListener_RetrySubscribeClosesPreviousClient(t *testing.T) {
	previousClosed := make(chan struct{})
	var closePrevious sync.Once

	previous := &mockCoreStream{
		closeSendFn: func() error {
			closePrevious.Do(func() {
				close(previousClosed)
			})
			return nil
		},
	}
	next := &mockCoreStream{}
	listener := newTestReconnectingListener(previous, func(ctx context.Context) (*mockCoreStream, error) {
		return next, nil
	})

	require.NoError(t, listener.retrySubscribe(context.Background()))
	assert.Equal(t, next, listener.client)

	select {
	case <-previousClosed:
	case <-time.After(time.Second):
		t.Fatal("expected replaced stream to be closed")
	}
}

func TestReconnectingListener_RetrySubscribeClosesFailedReplayClient(t *testing.T) {
	failedReplayClosed := make(chan struct{})
	var closeFailedReplay sync.Once

	failedReplay := &mockCoreStream{
		sendErr: status.Error(codes.Unavailable, "replay failed"),
		closeSendFn: func() error {
			closeFailedReplay.Do(func() {
				close(failedReplayClosed)
			})
			return nil
		},
	}

	listener := newTestReconnectingListener(nil, func(ctx context.Context) (*mockCoreStream, error) {
		return failedReplay, nil
	})
	listener.retryPolicy.subscribeRetryCount = 1
	listener.handlers.Store("run-1", &handlerSet[*dispatchercontracts.WorkflowRunEvent]{
		handlers: map[string]func(*dispatchercontracts.WorkflowRunEvent) error{
			"session-1": func(event *dispatchercontracts.WorkflowRunEvent) error {
				return nil
			},
		},
	})

	require.Error(t, listener.retrySubscribe(context.Background()))

	select {
	case <-failedReplayClosed:
	case <-time.After(time.Second):
		t.Fatal("expected failed replay stream to be closed")
	}
}

func TestReconnectingListener_ListenFollowsReplacedClient(t *testing.T) {
	oldClosed := make(chan struct{})
	var closeOld sync.Once

	oldClient := &mockCoreStream{
		recvFn: func() (*dispatchercontracts.WorkflowRunEvent, error) {
			<-oldClosed
			return nil, status.Error(codes.Canceled, "old stream canceled")
		},
		closeSendFn: func() error {
			closeOld.Do(func() {
				close(oldClosed)
			})
			return nil
		},
	}
	newEvents := make(chan *dispatchercontracts.WorkflowRunEvent, 1)
	newClient := &mockCoreStream{
		recvCh: newEvents,
	}

	listener := newTestReconnectingListener(oldClient, func(ctx context.Context) (*mockCoreStream, error) {
		return newClient, nil
	})

	received := make(chan *dispatchercontracts.WorkflowRunEvent, 1)
	listener.handlers.Store("run-1", &handlerSet[*dispatchercontracts.WorkflowRunEvent]{
		handlers: map[string]func(*dispatchercontracts.WorkflowRunEvent) error{
			"session-1": func(event *dispatchercontracts.WorkflowRunEvent) error {
				received <- event
				return nil
			},
		},
	})

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- listener.Listen(context.Background())
	}()

	require.NoError(t, listener.retrySubscribe(context.Background()))

	newEvents <- &dispatchercontracts.WorkflowRunEvent{
		WorkflowRunId: "run-1",
	}
	close(newEvents)

	require.Equal(t, "run-1", waitForTestValue(t, received).WorkflowRunId)
	require.NoError(t, waitForTestValue(t, listenErr))
}

func TestReconnectingListener_ClosePreventsListenReconnect(t *testing.T) {
	recvStarted := make(chan struct{})
	unblockRecv := make(chan struct{})
	var closeRecvStarted sync.Once

	client := &mockCoreStream{
		recvFn: func() (*dispatchercontracts.WorkflowRunEvent, error) {
			closeRecvStarted.Do(func() {
				close(recvStarted)
			})
			<-unblockRecv
			return nil, status.Error(codes.Internal, "stream closed")
		},
	}

	constructorCalls := atomic.Int32{}
	listener := newTestReconnectingListener(client, func(ctx context.Context) (*mockCoreStream, error) {
		constructorCalls.Add(1)
		return &mockCoreStream{}, nil
	})

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- listener.Listen(context.Background())
	}()

	waitForTestValue(t, recvStarted)
	require.NoError(t, listener.Close())
	close(unblockRecv)

	require.NoError(t, waitForTestValue(t, listenErr))
	assert.Equal(t, int32(0), constructorCalls.Load())
}

func TestReconnectingListener_ClosePreventsRetrySendReconnect(t *testing.T) {
	constructorCalls := atomic.Int32{}
	listener := newTestReconnectingListener(&mockCoreStream{}, func(ctx context.Context) (*mockCoreStream, error) {
		constructorCalls.Add(1)
		return &mockCoreStream{}, nil
	})

	require.NoError(t, listener.Close())

	err := listener.retrySend("run-1")
	require.ErrorIs(t, err, errListenerClosed)
	assert.Equal(t, int32(0), constructorCalls.Load())
}

func TestReconnectingListener_RetrySendHandlesNilClient(t *testing.T) {
	listener := newTestReconnectingListener(nil, func(ctx context.Context) (*mockCoreStream, error) {
		return nil, fmt.Errorf("connection failed")
	})

	err := listener.retrySend("run-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "client is not connected")
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

	require.NoError(t, err)
	require.NotNil(t, deprecated)
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

	require.NoError(t, err)
	require.NotNil(t, deprecated)
	assert.Equal(t, "test-run", deprecated.WorkflowRunId)
}

func TestWorkflowEventToDeprecatedWorkflowEvent_EmptyEvent(t *testing.T) {
	deprecated, err := workflowEventToDeprecatedWorkflowEvent(&dispatchercontracts.WorkflowEvent{})

	require.NoError(t, err)
	require.NotNil(t, deprecated)
}

func TestWorkflowRunEventToDeprecatedWorkflowRunEvent_Success(t *testing.T) {
	workflowRunId := "test-workflow-run-789"

	event := &dispatchercontracts.WorkflowRunEvent{
		WorkflowRunId:  workflowRunId,
		EventType:      dispatchercontracts.WorkflowRunEventType_WORKFLOW_RUN_EVENT_TYPE_FINISHED,
		EventTimestamp: timestamppb.New(time.Unix(9876543210, 987654321)),
	}

	deprecated, err := workflowRunEventToDeprecatedWorkflowRunEvent(event)

	require.NoError(t, err)
	require.NotNil(t, deprecated)
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

	require.NoError(t, err)
	require.NotNil(t, deprecated)
	assert.Equal(t, "test-run-2", deprecated.WorkflowRunId)
}

func TestWorkflowRunEventToDeprecatedWorkflowRunEvent_EmptyEvent(t *testing.T) {
	deprecated, err := workflowRunEventToDeprecatedWorkflowRunEvent(&dispatchercontracts.WorkflowRunEvent{})

	require.NoError(t, err)
	require.NotNil(t, deprecated)
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

	require.NoError(t, err)
	require.NotNil(t, deprecated)
	assert.Equal(t, "test-run-3", deprecated.WorkflowRunId)
	assert.NotNil(t, deprecated.Results)
}
