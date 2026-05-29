package client

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

type mockDurableEventClient struct {
	sendFn      func(req *contracts.ListenForDurableEventRequest) error
	recvFn      func() (*contracts.DurableEvent, error)
	closeSendFn func() error
	recvCh      chan *contracts.DurableEvent
	sendCount   atomic.Int32
}

func (m *mockDurableEventClient) Send(req *contracts.ListenForDurableEventRequest) error {
	m.sendCount.Add(1)
	if m.sendFn != nil {
		return m.sendFn(req)
	}

	return nil
}

func (m *mockDurableEventClient) Recv() (*contracts.DurableEvent, error) {
	if m.recvFn != nil {
		return m.recvFn()
	}

	if m.recvCh == nil {
		return nil, io.EOF
	}

	event, ok := <-m.recvCh
	if !ok {
		return nil, io.EOF
	}

	return event, nil
}

func (m *mockDurableEventClient) CloseSend() error {
	if m.closeSendFn != nil {
		return m.closeSendFn()
	}

	return nil
}

func (m *mockDurableEventClient) Header() (metadata.MD, error) {
	return nil, nil
}

func (m *mockDurableEventClient) Trailer() metadata.MD {
	return nil
}

func (m *mockDurableEventClient) Context() context.Context {
	return context.Background()
}

func (m *mockDurableEventClient) SendMsg(msg interface{}) error {
	return nil
}

func (m *mockDurableEventClient) RecvMsg(msg interface{}) error {
	return nil
}

func TestDurableEventsListenerDeliversEventAfterSendTriggeredReconnectWhileOldRecvBlocks(t *testing.T) {
	logger := zerolog.Nop()

	oldRecvStarted := make(chan struct{})
	oldClosed := make(chan struct{})
	sendAttempted := make(chan struct{})
	var closeOldRecvStarted sync.Once
	var closeOldClosed sync.Once
	var closeSendAttempted sync.Once

	oldClient := &mockDurableEventClient{
		sendFn: func(req *contracts.ListenForDurableEventRequest) error {
			closeSendAttempted.Do(func() {
				close(sendAttempted)
			})
			return status.Error(codes.Unavailable, "stream send failed")
		},
		recvFn: func() (*contracts.DurableEvent, error) {
			closeOldRecvStarted.Do(func() {
				close(oldRecvStarted)
			})
			<-oldClosed
			return nil, status.Error(codes.Canceled, "old stream canceled")
		},
		closeSendFn: func() error {
			closeOldClosed.Do(func() {
				close(oldClosed)
			})
			return nil
		},
	}

	newRequests := make(chan *contracts.ListenForDurableEventRequest, 2)
	newClient := &mockDurableEventClient{
		recvCh: make(chan *contracts.DurableEvent, 1),
		sendFn: func(req *contracts.ListenForDurableEventRequest) error {
			newRequests <- req
			return nil
		},
	}

	constructorCalls := atomic.Int32{}
	listener := newDurableEventsListener(func(ctx context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
		constructorCalls.Add(1)
		return newClient, nil
	}, &logger)
	listener.reconnectingListener.client = oldClient

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- listener.Listen(context.Background())
	}()
	waitForTestValue(t, oldRecvStarted)

	received := make(chan DurableEvent, 1)
	addErr := make(chan error, 1)
	go func() {
		addErr <- listener.AddSignal("task-1", "signal-1", func(event DurableEvent) error {
			received <- event
			return nil
		})
	}()

	waitForTestValue(t, sendAttempted)

	req := waitForTestValue(t, newRequests)
	require.Equal(t, "task-1", req.TaskId)
	require.Equal(t, "signal-1", req.SignalKey)
	require.Equal(t, int32(1), constructorCalls.Load())

	newClient.recvCh <- &contracts.DurableEvent{
		TaskId:    "task-1",
		SignalKey: "signal-1",
		Data:      []byte(`{"ok":true}`),
	}
	close(newClient.recvCh)

	event := waitForTestValue(t, received)
	require.Equal(t, "task-1", event.TaskId)
	require.Equal(t, "signal-1", event.SignalKey)
	require.JSONEq(t, `{"ok":true}`, string(event.Data))

	require.NoError(t, waitForTestValue(t, addErr))
	require.NoError(t, waitForTestValue(t, listenErr))
}

func TestDurableEventsListenerListenReturnsAfterSingleFailedReconnectCycle(t *testing.T) {
	logger := zerolog.Nop()
	constructorCalls := atomic.Int32{}

	listener := newDurableEventsListener(func(ctx context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
		constructorCalls.Add(1)
		return nil, status.Error(codes.Unavailable, "dispatcher unavailable")
	}, &logger)
	listener.reconnectingListener.client = &mockDurableEventClient{
		recvFn: func() (*contracts.DurableEvent, error) {
			return nil, status.Error(codes.Internal, "stream dropped")
		},
	}
	listener.reconnectingListener.retryPolicy.subscribeRetryCount = 1

	err := listener.Listen(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resubscribe after 1 consecutive errors")
	assert.Equal(t, int32(1), constructorCalls.Load())
}
