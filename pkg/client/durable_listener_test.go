package client

import (
	"context"
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

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/client/retry"
)

type mockDurableEventClient struct {
	sendFn      func(req *contracts.ListenForDurableEventRequest) error
	recvFn      func() (*contracts.DurableEvent, error)
	closeSendFn func() error
	recvCh      chan *contracts.DurableEvent
	sendCount   atomic.Int32
}

type mockV1DispatcherClient struct {
	listenForDurableEventFn func(ctx context.Context, opts ...grpc.CallOption) (contracts.V1Dispatcher_ListenForDurableEventClient, error)
}

func (m *mockV1DispatcherClient) DurableTask(ctx context.Context, opts ...grpc.CallOption) (contracts.V1Dispatcher_DurableTaskClient, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockV1DispatcherClient) RegisterDurableEvent(ctx context.Context, in *contracts.RegisterDurableEventRequest, opts ...grpc.CallOption) (*contracts.RegisterDurableEventResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (m *mockV1DispatcherClient) ListenForDurableEvent(ctx context.Context, opts ...grpc.CallOption) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
	if m.listenForDurableEventFn != nil {
		return m.listenForDurableEventFn(ctx, opts...)
	}

	return nil, status.Error(codes.Unimplemented, "not implemented")
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

func TestDurableEventsListenerAddSignalSendsOnceWhenStarting(t *testing.T) {
	logger := zerolog.Nop()
	recvCh := make(chan *contracts.DurableEvent)

	client := &mockDurableEventClient{
		recvCh: recvCh,
	}

	listener := &DurableEventsListener{
		constructor: func(ctx context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
			return client, nil
		},
		l: &logger,
	}

	require.NoError(t, listener.AddSignal("task-1", "signal-1", func(e DurableEvent) error {
		return nil
	}))
	require.Equal(t, int32(1), client.sendCount.Load())

	require.NoError(t, listener.Close())
	close(recvCh)
}

func TestGetDurableEventsListenerImmediateAddDoesNotOpenSecondStream(t *testing.T) {
	logger := zerolog.Nop()
	closeCh := make(chan struct{})
	var closeOnce sync.Once

	client := &mockDurableEventClient{
		recvFn: func() (*contracts.DurableEvent, error) {
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
		clientv1: &mockV1DispatcherClient{
			listenForDurableEventFn: func(ctx context.Context, opts ...grpc.CallOption) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
				constructorCalls.Add(1)
				return client, nil
			},
		},
		l:   &logger,
		ctx: newContextLoader("", nil),
	}

	listener, err := subscriber.getDurableEventsListener(context.Background())
	require.NoError(t, err)
	require.True(t, listener.isListening())

	require.NoError(t, listener.AddSignal("task-1", "signal-1", func(e DurableEvent) error {
		return nil
	}))

	require.Equal(t, int32(1), constructorCalls.Load())
	require.Equal(t, int32(1), client.sendCount.Load())

	require.NoError(t, listener.Close())
	require.Eventually(t, func() bool {
		return !listener.isListening()
	}, time.Second, 10*time.Millisecond)
}

func TestGetDurableEventsListenerInternalCleanupDoesNotTerminallyCloseListener(t *testing.T) {
	logger := zerolog.Nop()

	initialClient := &mockDurableEventClient{
		recvFn: func() (*contracts.DurableEvent, error) {
			return nil, io.EOF
		},
	}
	replacementClient := &mockDurableEventClient{
		recvCh: make(chan *contracts.DurableEvent, 1),
	}
	constructorCalls := atomic.Int32{}

	subscriber := &subscribeClientImpl{
		clientv1: &mockV1DispatcherClient{
			listenForDurableEventFn: func(ctx context.Context, opts ...grpc.CallOption) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
				if constructorCalls.Add(1) == 1 {
					return initialClient, nil
				}

				return replacementClient, nil
			},
		},
		l:   &logger,
		ctx: newContextLoader("", nil),
	}

	listener, err := subscriber.getDurableEventsListener(context.Background())
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return !listener.isListening()
	}, time.Second, 10*time.Millisecond)

	received := make(chan DurableEvent, 1)
	require.NoError(t, listener.AddSignal("task-1", "signal-1", func(e DurableEvent) error {
		received <- e
		return nil
	}))

	require.Equal(t, int32(2), constructorCalls.Load())
	require.Equal(t, int32(1), replacementClient.sendCount.Load())

	replacementClient.recvCh <- &contracts.DurableEvent{
		TaskId:    "task-1",
		SignalKey: "signal-1",
		Data:      []byte(`{"ok":true}`),
	}

	select {
	case event := <-received:
		require.Equal(t, "task-1", event.TaskId)
		require.Equal(t, "signal-1", event.SignalKey)
		require.JSONEq(t, `{"ok":true}`, string(event.Data))
	case <-time.After(time.Second):
		t.Fatal("expected durable event after internal cleanup")
	}

	close(replacementClient.recvCh)
	require.Eventually(t, func() bool {
		return !listener.isListening()
	}, time.Second, 10*time.Millisecond)
}

// TestDurableEventsListenerReconnectsWhileRetrySendBacksOff verifies that a
// stream disconnect can reconnect while AddSignal is retrying a send on the
// broken stream. This is the lock-contention repro: retrySend must not hold a
// reader lock for the full retry/backoff window.
func TestDurableEventsListenerReconnectsWhileRetrySendBacksOff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := zerolog.Nop()
	sendAttempted := make(chan struct{})
	var closeSendAttempted sync.Once

	// The old stream fails both Send and Recv so AddSignal enters retrySend while
	// Listen tries to recover the stream through retryListen.
	oldClient := &mockDurableEventClient{
		sendFn: func(req *contracts.ListenForDurableEventRequest) error {
			closeSendAttempted.Do(func() {
				close(sendAttempted)
			})
			return status.Error(codes.Unavailable, "stream broken")
		},
		recvFn: func() (*contracts.DurableEvent, error) {
			<-sendAttempted
			return nil, status.Error(codes.Internal, "stream broken")
		},
	}

	newClient := &mockDurableEventClient{
		recvFn: func() (*contracts.DurableEvent, error) {
			<-ctx.Done()
			return nil, status.Error(codes.Canceled, "context canceled")
		},
	}

	// The constructor should be reachable immediately after Recv reports the
	// disconnect, even while the sender is waiting to retry.
	constructorCalled := make(chan struct{})
	var closeConstructorCalled sync.Once

	listener := &DurableEventsListener{
		constructor: func(ctx context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
			closeConstructorCalled.Do(func() {
				close(constructorCalled)
			})
			return newClient, nil
		},
		client: oldClient,
		l:      &logger,
	}

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- listener.Listen(ctx)
	}()

	require.Eventually(t, listener.isListening, time.Second, 10*time.Millisecond)

	addErr := make(chan error, 1)
	go func() {
		addErr <- listener.AddSignal("task-1", "signal-1", func(e DurableEvent) error {
			return nil
		})
	}()

	require.Eventually(t, func() bool {
		select {
		case <-sendAttempted:
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)

	select {
	case <-constructorCalled:
	case err := <-addErr:
		require.NoError(t, err)
	case err := <-listenErr:
		require.NoError(t, err)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected durable event listener to reconnect while retrySend is backing off")
	}
}

func TestDurableEventsListenerReconnectDoesNotBlockClientSnapshot(t *testing.T) {
	logger := zerolog.Nop()
	constructorEntered := make(chan struct{})
	releaseConstructor := make(chan struct{})
	var closeConstructorEntered sync.Once

	oldClient := &mockDurableEventClient{
		recvCh: make(chan *contracts.DurableEvent),
	}
	newClient := &mockDurableEventClient{
		recvCh: make(chan *contracts.DurableEvent),
	}

	listener := &DurableEventsListener{
		constructor: func(ctx context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
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
		retryErr <- listener.retryListenBackground(context.Background())
	}()

	select {
	case <-constructorEntered:
	case <-time.After(time.Second):
		t.Fatal("reconnect did not reach constructor")
	}

	snapshotRead := make(chan struct {
		client     contracts.V1Dispatcher_ListenForDurableEventClient
		generation uint64
	}, 1)
	go func() {
		client, generation := listener.getClientSnapshot()
		snapshotRead <- struct {
			client     contracts.V1Dispatcher_ListenForDurableEventClient
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

func TestDurableEventsListenerRestartsAfterListenExits(t *testing.T) {
	logger := zerolog.Nop()

	initialClient := &mockDurableEventClient{
		recvFn: func() (*contracts.DurableEvent, error) {
			return nil, io.EOF
		},
	}
	replacementClient := &mockDurableEventClient{
		recvCh: make(chan *contracts.DurableEvent, 1),
	}

	listener := &DurableEventsListener{
		constructor: func(ctx context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
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

	received := make(chan DurableEvent, 1)
	require.NoError(t, listener.AddSignal("task-1", "signal-1", func(e DurableEvent) error {
		received <- e
		return nil
	}))

	replacementClient.recvCh <- &contracts.DurableEvent{
		TaskId:    "task-1",
		SignalKey: "signal-1",
		Data:      []byte(`{"ok":true}`),
	}
	close(replacementClient.recvCh)

	select {
	case event := <-received:
		require.Equal(t, "task-1", event.TaskId)
		require.Equal(t, "signal-1", event.SignalKey)
		require.JSONEq(t, `{"ok":true}`, string(event.Data))
	case <-time.After(time.Second):
		t.Fatal("expected durable event after listener restarted")
	}
}

func TestDurableEventsListenerReconnectsOnEOFWithRegisteredHandlers(t *testing.T) {
	logger := zerolog.Nop()

	initialClient := &mockDurableEventClient{
		recvFn: func() (*contracts.DurableEvent, error) {
			return nil, io.EOF
		},
	}
	replacementClient := &mockDurableEventClient{
		recvCh: make(chan *contracts.DurableEvent, 1),
	}
	constructorCalls := atomic.Int32{}

	listener := &DurableEventsListener{
		constructor: func(ctx context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
			constructorCalls.Add(1)
			return replacementClient, nil
		},
		client: initialClient,
		l:      &logger,
	}

	received := make(chan DurableEvent, 1)
	listener.handlers.Store(listenTuple{taskId: "task-1", signalKey: "signal-1"}, &threadSafeDurableEventHandlers{
		handlers: []DurableEventHandler{
			func(e DurableEvent) error {
				received <- e
				return nil
			},
		},
	})

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- listener.Listen(context.Background())
	}()

	require.Eventually(t, func() bool {
		return constructorCalls.Load() == 1 && replacementClient.sendCount.Load() == 1
	}, time.Second, 10*time.Millisecond)

	replacementClient.recvCh <- &contracts.DurableEvent{
		TaskId:    "task-1",
		SignalKey: "signal-1",
		Data:      []byte(`{"ok":true}`),
	}

	select {
	case event := <-received:
		require.Equal(t, "task-1", event.TaskId)
		require.Equal(t, "signal-1", event.SignalKey)
		require.JSONEq(t, `{"ok":true}`, string(event.Data))
	case <-time.After(time.Second):
		t.Fatal("expected durable event after EOF reconnect")
	}

	close(replacementClient.recvCh)
	require.NoError(t, <-listenErr)
}

func TestAddSignalBoundedWhenEngineDown(t *testing.T) {
	logger := zerolog.Nop()
	constructorCalls := atomic.Int32{}

	listener := &DurableEventsListener{
		constructor: func(ctx context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
			constructorCalls.Add(1)
			return nil, status.Error(codes.Unavailable, "engine down")
		},
		l: &logger,
	}

	err := listener.AddSignal("task-1", "signal-1", func(e DurableEvent) error {
		return nil
	})

	require.Error(t, err)
	assert.LessOrEqual(t, constructorCalls.Load(), int32(retry.StreamSyncMaxAttempts+1))
}

func TestListenForDurableEventsRespectsCancelledContext(t *testing.T) {
	logger := zerolog.Nop()

	subscriber := &subscribeClientImpl{
		clientv1: &mockV1DispatcherClient{
			listenForDurableEventFn: func(ctx context.Context, opts ...grpc.CallOption) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
				t.Fatal("constructor should not run when caller context is already cancelled")
				return nil, nil
			},
		},
		l:   &logger,
		ctx: newContextLoader("", nil),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := subscriber.ListenForDurableEvents(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestListenForDurableEventsPassesCallerContextToInitialListen(t *testing.T) {
	logger := zerolog.Nop()
	callerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subscriber := &subscribeClientImpl{
		clientv1: &mockV1DispatcherClient{
			listenForDurableEventFn: func(ctx context.Context, opts ...grpc.CallOption) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
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

	_, err := subscriber.ListenForDurableEvents(callerCtx)
	require.Error(t, err)
}

func TestListenForDurableEventsRespectsDeadlineContext(t *testing.T) {
	logger := zerolog.Nop()
	constructorCalls := atomic.Int32{}

	subscriber := &subscribeClientImpl{
		clientv1: &mockV1DispatcherClient{
			listenForDurableEventFn: func(ctx context.Context, opts ...grpc.CallOption) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
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

	_, err := subscriber.ListenForDurableEvents(ctx)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Equal(t, int32(1), constructorCalls.Load())
}

func TestDurableListenEOFWithoutHandlersDoesNotReconnect(t *testing.T) {
	logger := zerolog.Nop()
	constructorCalls := atomic.Int32{}

	listener := &DurableEventsListener{
		constructor: func(ctx context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
			constructorCalls.Add(1)
			return &mockDurableEventClient{recvCh: make(chan *contracts.DurableEvent)}, nil
		},
		client: &mockDurableEventClient{
			recvFn: func() (*contracts.DurableEvent, error) {
				return nil, io.EOF
			},
		},
		l: &logger,
	}

	require.NoError(t, listener.Listen(context.Background()))
	assert.Equal(t, int32(0), constructorCalls.Load())
}
