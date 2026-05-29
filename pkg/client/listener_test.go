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
	"google.golang.org/protobuf/types/known/timestamppb"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
)

type mockCoreStream struct {
	sendErr     error
	recvFn      func() (*dispatchercontracts.WorkflowRunEvent, error)
	closeSendFn func() error
	sendCount   atomic.Int32
}

func (m *mockCoreStream) Send(req *dispatchercontracts.SubscribeToWorkflowRunsRequest) error {
	m.sendCount.Add(1)
	return m.sendErr
}

func (m *mockCoreStream) Recv() (*dispatchercontracts.WorkflowRunEvent, error) {
	if m.recvFn != nil {
		return m.recvFn()
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

	_, gen0 := listener.getClientSnapshot()
	assert.Equal(t, uint64(0), gen0)

	require.NoError(t, listener.retrySubscribe(context.Background()))
	_, gen1 := listener.getClientSnapshot()
	assert.Equal(t, uint64(1), gen1)

	require.NoError(t, listener.retrySubscribe(context.Background()))
	_, gen2 := listener.getClientSnapshot()
	assert.Equal(t, uint64(2), gen2)
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
