package client

import (
	"context"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
)

func TestWorkerActionsSurvivesMoreThanFiveTransientFailures(t *testing.T) {
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

	listener := newTestActionListener(t, streamClient)
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

func TestWorkerActionsNoProgressConnectFailuresSurfaceOnErrCh(t *testing.T) {
	constructorCalls := atomic.Int32{}

	streamClient := &mockListenV2Client{
		recvFn: func() (*dispatchercontracts.AssignedAction, error) {
			return nil, fmt.Errorf("plain recv error")
		},
	}

	listener := newTestActionListener(t, streamClient)
	listener.client = &mockWorkerDispatcherClient{
		listenV2Fn: func(ctx context.Context, in *dispatchercontracts.WorkerListenRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_ListenV2Client, error) {
			constructorCalls.Add(1)
			return nil, fmt.Errorf("plain connect error")
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, errCh, err := listener.Actions(ctx)
	require.NoError(t, err)

	select {
	case err := <-errCh:
		require.Error(t, err)
		assert.Contains(t, err.Error(), "made no progress")
	case <-time.After(10 * time.Second):
		t.Fatal("expected terminal error on errCh after no-progress cap")
	}

	assert.GreaterOrEqual(t, constructorCalls.Load(), int32(maxConsecutiveStreamNoProgress-2))
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

	listener := newTestActionListener(t, &mockListenV2Client{
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
	listener := newTestActionListener(t, &mockListenV2Client{
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

	listener := newTestActionListener(t, &mockListenV2Client{
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
	subscribeCalls := atomic.Int32{}
	streamClient := &mockListenV2Client{
		recvFn: func() (*dispatchercontracts.AssignedAction, error) {
			return nil, io.EOF
		},
	}

	listener := newTestActionListener(t, streamClient)
	listener.client = &mockWorkerDispatcherClient{
		listenV2Fn: func(ctx context.Context, in *dispatchercontracts.WorkerListenRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_ListenV2Client, error) {
			require.NotEmpty(t, opts)
			subscribeCalls.Add(1)
			return streamClient, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, errCh, err := listener.Actions(ctx)
	require.NoError(t, err)

	select {
	case err, ok := <-errCh:
		if ok && err != nil {
			t.Fatalf("unexpected terminal error: %v", err)
		}
	case <-time.After(2 * time.Second):
	}

	assert.GreaterOrEqual(t, subscribeCalls.Load(), int32(1))
}

func TestWorkerActionsReconnectsOnEOFWhileContextActive(t *testing.T) {
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

	listener := newTestActionListener(t, streamClient)
	listener.client = &mockWorkerDispatcherClient{
		listenV2Fn: func(ctx context.Context, in *dispatchercontracts.WorkerListenRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_ListenV2Client, error) {
			require.NotEmpty(t, opts)
			subscribeCalls.Add(1)
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

	listener := newTestActionListener(t, streamClient)
	listener.client = &mockWorkerDispatcherClient{
		listenV2Fn: func(ctx context.Context, in *dispatchercontracts.WorkerListenRequest, opts ...grpc.CallOption) (dispatchercontracts.Dispatcher_ListenV2Client, error) {
			require.NotEmpty(t, opts)
			subscribeCalls.Add(1)
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
