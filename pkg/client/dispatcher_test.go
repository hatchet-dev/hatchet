package client

import (
	"context"
	"fmt"
	"io"
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

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/client/retry"
)

type mockListenV2Client struct {
	recvFn func() (*dispatchercontracts.AssignedAction, error)
}

func (m *mockListenV2Client) Recv() (*dispatchercontracts.AssignedAction, error) {
	if m.recvFn != nil {
		return m.recvFn()
	}
	return nil, io.EOF
}

func (m *mockListenV2Client) CloseSend() error              { return nil }
func (m *mockListenV2Client) Header() (metadata.MD, error)  { return nil, nil }
func (m *mockListenV2Client) Trailer() metadata.MD          { return nil }
func (m *mockListenV2Client) Context() context.Context      { return context.Background() }
func (m *mockListenV2Client) SendMsg(msg interface{}) error { return nil }
func (m *mockListenV2Client) RecvMsg(msg interface{}) error { return nil }

type mockWorkerDispatcherClient struct {
	listenV2Fn func(ctx context.Context, in *dispatchercontracts.WorkerListenRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_ListenV2Client, error)
	listenFn   func(ctx context.Context, in *dispatchercontracts.WorkerListenRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_ListenClient, error)
}

func (m *mockWorkerDispatcherClient) Register(ctx context.Context, in *dispatchercontracts.WorkerRegisterRequest, opts ...grpc.CallOption) (*dispatchercontracts.WorkerRegisterResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockWorkerDispatcherClient) Listen(ctx context.Context, in *dispatchercontracts.WorkerListenRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_ListenClient, error) {
	if m.listenFn != nil {
		return m.listenFn(ctx, in, opts...)
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockWorkerDispatcherClient) ListenV2(ctx context.Context, in *dispatchercontracts.WorkerListenRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_ListenV2Client, error) {
	if m.listenV2Fn != nil {
		return m.listenV2Fn(ctx, in, opts...)
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockWorkerDispatcherClient) Heartbeat(ctx context.Context, in *dispatchercontracts.HeartbeatRequest, opts ...grpc.CallOption) (*dispatchercontracts.HeartbeatResponse, error) {
	return &dispatchercontracts.HeartbeatResponse{}, nil
}

func (m *mockWorkerDispatcherClient) SubscribeToWorkflowEvents(ctx context.Context, in *dispatchercontracts.SubscribeToWorkflowEventsRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_SubscribeToWorkflowEventsClient, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockWorkerDispatcherClient) SubscribeToWorkflowRuns(ctx context.Context, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockWorkerDispatcherClient) SendStepActionEvent(ctx context.Context, in *dispatchercontracts.StepActionEvent, opts ...grpc.CallOption) (*dispatchercontracts.ActionEventResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockWorkerDispatcherClient) SendGroupKeyActionEvent(ctx context.Context, in *dispatchercontracts.GroupKeyActionEvent, opts ...grpc.CallOption) (*dispatchercontracts.ActionEventResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockWorkerDispatcherClient) PutOverridesData(ctx context.Context, in *dispatchercontracts.OverridesData, opts ...grpc.CallOption) (*dispatchercontracts.OverridesDataResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockWorkerDispatcherClient) Unsubscribe(ctx context.Context, in *dispatchercontracts.WorkerUnsubscribeRequest, opts ...grpc.CallOption) (*dispatchercontracts.WorkerUnsubscribeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockWorkerDispatcherClient) RefreshTimeout(ctx context.Context, in *dispatchercontracts.RefreshTimeoutRequest, opts ...grpc.CallOption) (*dispatchercontracts.RefreshTimeoutResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockWorkerDispatcherClient) ReleaseSlot(ctx context.Context, in *dispatchercontracts.ReleaseSlotRequest, opts ...grpc.CallOption) (*dispatchercontracts.ReleaseSlotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockWorkerDispatcherClient) RestoreEvictedTask(ctx context.Context, in *dispatchercontracts.RestoreEvictedTaskRequest, opts ...grpc.CallOption) (*dispatchercontracts.RestoreEvictedTaskResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockWorkerDispatcherClient) UpsertWorkerLabels(ctx context.Context, in *dispatchercontracts.UpsertWorkerLabelsRequest, opts ...grpc.CallOption) (*dispatchercontracts.UpsertWorkerLabelsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockWorkerDispatcherClient) GetVersion(ctx context.Context, in *dispatchercontracts.GetVersionRequest, opts ...grpc.CallOption) (*dispatchercontracts.GetVersionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func newTestActionListener(listenClient dispatchercontracts.Dispatcher_ListenV2Client) *actionListenerImpl {
	logger := zerolog.Nop()
	return &actionListenerImpl{
		client:           &mockWorkerDispatcherClient{},
		listenClient:     listenClient,
		workerId:         "worker-1",
		l:                &logger,
		listenerStrategy: ListenerStrategyV2,
		ctx:              newContextLoader("", nil),
	}
}

func TestWorkerActionsSurvivesMoreThanFiveTransientFailures(t *testing.T) {
	retry.SetStreamSleepHookForTesting(func(ctx context.Context, attempt int) error {
		return nil
	})
	t.Cleanup(retry.ResetStreamSleepHookForTesting)

	recvCalls := atomic.Int32{}
	subscribeCalls := atomic.Int32{}

	streamClient := &mockListenV2Client{
		recvFn: func() (*dispatchercontracts.AssignedAction, error) {
			if recvCalls.Add(1) <= DefaultActionListenerRetryCount+1 {
				return nil, status.Error(codes.Unavailable, "stream broken")
			}

			return &dispatchercontracts.AssignedAction{
				ActionType: dispatchercontracts.ActionType_START_STEP_RUN,
				ActionId:   "action-1",
			}, nil
		},
	}

	listener := newTestActionListener(streamClient)
	listener.client = &mockWorkerDispatcherClient{
		listenV2Fn: func(ctx context.Context, in *dispatchercontracts.WorkerListenRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_ListenV2Client, error) {
			require.NotEmpty(t, opts)
			subscribeCalls.Add(1)
			return streamClient, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	actions, errCh, err := listener.Actions(ctx)
	require.NoError(t, err)

	select {
	case action := <-actions:
		require.NotNil(t, action)
		assert.Equal(t, "action-1", action.ActionId)
	case err := <-errCh:
		t.Fatalf("unexpected terminal error: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for action after prolonged transient failures")
	}

	assert.Greater(t, recvCalls.Load(), int32(DefaultActionListenerRetryCount))
	assert.Greater(t, subscribeCalls.Load(), int32(0))
}

func TestWorkerRetrySubscribeStopsOnNoProgressError(t *testing.T) {
	retry.SetStreamSleepHookForTesting(func(ctx context.Context, attempt int) error {
		return nil
	})
	t.Cleanup(retry.ResetStreamSleepHookForTesting)

	constructorCalls := atomic.Int32{}

	listener := newTestActionListener(&mockListenV2Client{})
	listener.client = &mockWorkerDispatcherClient{
		listenV2Fn: func(ctx context.Context, in *dispatchercontracts.WorkerListenRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_ListenV2Client, error) {
			constructorCalls.Add(1)
			return nil, fmt.Errorf("plain subscribe error")
		},
	}

	err := listener.retrySubscribe(context.Background())
	require.Error(t, err)
	assert.Equal(t, int32(1), constructorCalls.Load())
}

func TestWorkerActionsFallsBackFromV2ToV1(t *testing.T) {
	listenCalls := atomic.Int32{}

	v1Client := &mockListenV2Client{
		recvFn: func() (*dispatchercontracts.AssignedAction, error) {
			return &dispatchercontracts.AssignedAction{
				ActionType: dispatchercontracts.ActionType_START_STEP_RUN,
				ActionId:   "action-v1",
			}, nil
		},
	}

	listener := newTestActionListener(&mockListenV2Client{
		recvFn: func() (*dispatchercontracts.AssignedAction, error) {
			return nil, status.Error(codes.Unimplemented, "v2 not supported")
		},
	})

	listener.client = &mockWorkerDispatcherClient{
		listenFn: func(ctx context.Context, in *dispatchercontracts.WorkerListenRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_ListenClient, error) {
			require.NotEmpty(t, opts)
			listenCalls.Add(1)
			return v1Client, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	actions, errCh, err := listener.Actions(ctx)
	require.NoError(t, err)

	select {
	case action := <-actions:
		require.Equal(t, "action-v1", action.ActionId)
		assert.Equal(t, ListenerStrategyV1, listener.listenerStrategy)
	case err := <-errCh:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for v1 fallback action")
	}

	assert.Equal(t, int32(1), listenCalls.Load())
}

func TestWorkerActionsV1UnimplementedIsTerminal(t *testing.T) {
	listener := newTestActionListener(&mockListenV2Client{
		recvFn: func() (*dispatchercontracts.AssignedAction, error) {
			return nil, status.Error(codes.Unimplemented, "v2 not supported")
		},
	})
	listener.listenerStrategy = ListenerStrategyV1

	listener.client = &mockWorkerDispatcherClient{
		listenFn: func(ctx context.Context, in *dispatchercontracts.WorkerListenRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_ListenClient, error) {
			return nil, status.Error(codes.Unimplemented, "v1 not supported")
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, errCh, err := listener.Actions(ctx)
	require.NoError(t, err)

	select {
	case err := <-errCh:
		require.Error(t, err)
		assert.Equal(t, codes.Unimplemented, status.Code(err))
	case <-time.After(5 * time.Second):
		t.Fatal("expected terminal error for v1 unimplemented")
	}
}

func TestWorkerActionsContextCancelStopsLoop(t *testing.T) {
	blockRecv := make(chan struct{})

	listener := newTestActionListener(&mockListenV2Client{
		recvFn: func() (*dispatchercontracts.AssignedAction, error) {
			<-blockRecv
			return nil, status.Error(codes.Unavailable, "stream broken")
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	_, errCh, err := listener.Actions(ctx)
	require.NoError(t, err)

	cancel()

	select {
	case err, ok := <-errCh:
		if ok {
			t.Fatalf("unexpected terminal error: %v", err)
		}
	case <-time.After(2 * time.Second):
	}
}

func TestWorkerListenDisablesGrpcRetry(t *testing.T) {
	listener := newTestActionListener(&mockListenV2Client{recvFn: func() (*dispatchercontracts.AssignedAction, error) {
		return nil, io.EOF
	}})

	listener.client = &mockWorkerDispatcherClient{
		listenV2Fn: func(ctx context.Context, in *dispatchercontracts.WorkerListenRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_ListenV2Client, error) {
			require.NotEmpty(t, opts)
			return &mockListenV2Client{recvFn: func() (*dispatchercontracts.AssignedAction, error) {
				return nil, io.EOF
			}}, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	require.NoError(t, listener.retrySubscribe(ctx))
}

func TestWorkerRetrySubscribeUsesStreamSyncMaxAttemptsConstant(t *testing.T) {
	require.Equal(t, 5, retry.StreamSyncMaxAttempts)
}

func TestWorkerActionsReconnectsOnEOFWhileContextActive(t *testing.T) {
	retry.SetStreamSleepHookForTesting(func(ctx context.Context, attempt int) error {
		return nil
	})
	t.Cleanup(retry.ResetStreamSleepHookForTesting)

	recvCalls := atomic.Int32{}
	subscribeCalls := atomic.Int32{}

	replacementClient := &mockListenV2Client{
		recvFn: func() (*dispatchercontracts.AssignedAction, error) {
			return &dispatchercontracts.AssignedAction{
				ActionType: dispatchercontracts.ActionType_START_STEP_RUN,
				ActionId:   "action-after-eof",
			}, nil
		},
	}

	streamClient := &mockListenV2Client{
		recvFn: func() (*dispatchercontracts.AssignedAction, error) {
			recvCalls.Add(1)
			return nil, io.EOF
		},
	}

	listener := newTestActionListener(streamClient)
	listener.client = &mockWorkerDispatcherClient{
		listenV2Fn: func(ctx context.Context, in *dispatchercontracts.WorkerListenRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_ListenV2Client, error) {
			require.NotEmpty(t, opts)
			subscribeCalls.Add(1)
			listener.listenClient = replacementClient
			return replacementClient, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	actions, errCh, err := listener.Actions(ctx)
	require.NoError(t, err)

	select {
	case action := <-actions:
		require.NotNil(t, action)
		assert.Equal(t, "action-after-eof", action.ActionId)
	case err := <-errCh:
		t.Fatalf("unexpected terminal error: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for action after EOF reconnect")
	}

	assert.Equal(t, int32(1), recvCalls.Load())
	assert.Equal(t, int32(1), subscribeCalls.Load())
}

func TestWorkerActionsRetriesNoProgressBeforeGivingUp(t *testing.T) {
	retry.SetStreamSleepHookForTesting(func(ctx context.Context, attempt int) error {
		return nil
	})
	t.Cleanup(retry.ResetStreamSleepHookForTesting)

	recvCalls := atomic.Int32{}
	subscribeCalls := atomic.Int32{}

	replacementClient := &mockListenV2Client{
		recvFn: func() (*dispatchercontracts.AssignedAction, error) {
			return &dispatchercontracts.AssignedAction{
				ActionType: dispatchercontracts.ActionType_START_STEP_RUN,
				ActionId:   "action-after-unknown",
			}, nil
		},
	}

	streamClient := &mockListenV2Client{
		recvFn: func() (*dispatchercontracts.AssignedAction, error) {
			recvCalls.Add(1)
			return nil, status.Error(codes.Unknown, "transient unknown")
		},
	}

	listener := newTestActionListener(streamClient)
	listener.client = &mockWorkerDispatcherClient{
		listenV2Fn: func(ctx context.Context, in *dispatchercontracts.WorkerListenRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_ListenV2Client, error) {
			require.NotEmpty(t, opts)
			subscribeCalls.Add(1)
			listener.listenClient = replacementClient
			return replacementClient, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	actions, errCh, err := listener.Actions(ctx)
	require.NoError(t, err)

	select {
	case action := <-actions:
		require.NotNil(t, action)
		assert.Equal(t, "action-after-unknown", action.ActionId)
	case err := <-errCh:
		t.Fatalf("unexpected terminal error: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for action after no-progress retry")
	}

	assert.Equal(t, int32(1), recvCalls.Load())
	assert.Equal(t, int32(1), subscribeCalls.Load())
}
