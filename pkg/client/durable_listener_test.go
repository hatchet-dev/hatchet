package client

import (
	"context"
	"errors"
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
	"google.golang.org/grpc/status"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/client/retry"
)

func TestDurableEventsListenerAddSignalSendsOnceWhenStarting(t *testing.T) {
	logger := zerolog.Nop()
	recvCh := make(chan *contracts.DurableEvent)

	client := &mockDurableEventClient{
		recvCh: recvCh,
	}

	listener := newTestDurableEventsListener(t, &logger, func(ctx context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
		return client, nil
	}, client)

	require.NoError(t, listener.AddSignal("task-1", "signal-1", func(e DurableEvent) error {
		return nil
	}))
	require.Equal(t, int32(1), client.sendCount.Load())

	require.NoError(t, listener.Close())
	close(recvCh)
}

func TestDurableEventsListenerAddSignalRollsBackHandlerWhenSendFails(t *testing.T) {
	logger := zerolog.Nop()
	client := &mockDurableEventClient{
		sendFn: func(req *contracts.ListenForDurableEventRequest) error {
			return status.Error(codes.Unavailable, "send failed")
		},
		recvFn: func() (*contracts.DurableEvent, error) {
			return nil, io.EOF
		},
	}

	listener := newTestDurableEventsListener(t, &logger, func(ctx context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
		return client, nil
	}, client)

	err := listener.AddSignal("task-1", "signal-1", func(e DurableEvent) error {
		return nil
	})
	require.Error(t, err)
	assert.False(t, listener.reg.hasAny())

	require.NoError(t, listener.Close())
	require.Eventually(t, func() bool {
		return !listener.isListening()
	}, time.Second, 10*time.Millisecond)
}

func TestDurableEventsListenerAddSignalKeepsHandlerDuringRecovery(t *testing.T) {
	logger := zerolog.Nop()
	oldClient := &mockDurableEventClient{}
	recoveredClient := &mockDurableEventClient{
		recvCh: make(chan *contracts.DurableEvent),
	}

	var listener *DurableEventsListener
	reconnectFailures := atomic.Int32{}
	recoveredSends := atomic.Int32{}
	missingHandlerDuringRecovery := atomic.Bool{}

	oldClient.sendFn = func(req *contracts.ListenForDurableEventRequest) error {
		listener.gate.stop()
		return status.Error(codes.Unavailable, "stream broken")
	}
	recoveredClient.sendFn = func(req *contracts.ListenForDurableEventRequest) error {
		recoveredSends.Add(1)
		if !listener.reg.hasAny() {
			missingHandlerDuringRecovery.Store(true)
		}
		return nil
	}

	listener = newTestDurableEventsListener(t, &logger, func(ctx context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
		if reconnectFailures.Add(1) <= int32(2*retry.StreamSyncMaxAttempts-1) {
			return nil, status.Error(codes.Unavailable, "engine down")
		}

		return recoveredClient, nil
	}, oldClient)
	require.True(t, listener.gate.tryStart(false))

	err := listener.AddSignal("task-1", "signal-1", func(e DurableEvent) error {
		return nil
	})
	require.NoError(t, err)
	assert.False(t, missingHandlerDuringRecovery.Load())
	assert.True(t, listener.reg.hasAny())
	assert.GreaterOrEqual(t, recoveredSends.Load(), int32(1))

	require.NoError(t, listener.Close())
	close(recoveredClient.recvCh)
	require.Eventually(t, func() bool {
		return !listener.isListening()
	}, time.Second, 10*time.Millisecond)
}

func TestDurableEventsListenerHandlerErrorDoesNotStopLaterEvents(t *testing.T) {
	logger := zerolog.Nop()
	recvCh := make(chan *contracts.DurableEvent, 2)

	client := &mockDurableEventClient{
		recvCh: recvCh,
	}

	listener := newTestDurableEventsListener(t, &logger, func(ctx context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
		return client, nil
	}, client)

	failTuple := listenTuple{taskId: "task-1", signalKey: "signal-fail"}
	okTuple := listenTuple{taskId: "task-1", signalKey: "signal-ok"}

	listener.reg.store(failTuple, "fail", func(e DurableEvent) error {
		return errors.New("handler failed")
	}, nil)

	received := make(chan DurableEvent, 1)
	listener.reg.store(okTuple, "ok", func(e DurableEvent) error {
		received <- e
		return nil
	}, nil)

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- listener.Listen(context.Background())
	}()

	recvCh <- &contracts.DurableEvent{
		TaskId:    failTuple.taskId,
		SignalKey: failTuple.signalKey,
	}

	require.Eventually(t, listener.reg.hasAny, time.Second, 10*time.Millisecond,
		"handler error should not tear down the listener")

	recvCh <- &contracts.DurableEvent{
		TaskId:    okTuple.taskId,
		SignalKey: okTuple.signalKey,
		Data:      []byte(`{"ok":true}`),
	}

	select {
	case event := <-received:
		assert.Equal(t, "signal-ok", event.SignalKey)
	case <-time.After(time.Second):
		t.Fatal("expected later durable event after handler error")
	}

	listener.reg.removeSession(failTuple, "fail")
	close(recvCh)
	assert.NoError(t, <-listenErr)
}

func TestDurableEventsListenerRollbackHandlersByStableID(t *testing.T) {
	tuple := listenTuple{taskId: "task-1", signalKey: "signal-1"}

	tests := []struct {
		name           string
		rollbackFirst  bool
		expectedFirst  int32
		expectedSecond int32
	}{
		{
			name:           "first handler rolls back before second",
			rollbackFirst:  true,
			expectedSecond: 1,
		},
		{
			name:          "second handler rolls back before first",
			expectedFirst: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zerolog.Nop()
			listener := newDurableEventsListener(&logger, func(ctx context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
				return &mockDurableEventClient{}, nil
			})
			firstCalls := atomic.Int32{}
			secondCalls := atomic.Int32{}

			rollbackFirst := listener.reg.store(tuple, "", func(e DurableEvent) error {
				firstCalls.Add(1)
				return nil
			}, nil)
			rollbackSecond := listener.reg.store(tuple, "", func(e DurableEvent) error {
				secondCalls.Add(1)
				return nil
			}, nil)

			if tt.rollbackFirst {
				rollbackFirst()
			} else {
				rollbackSecond()
			}

			require.NoError(t, listener.dispatch(&contracts.DurableEvent{
				TaskId:    tuple.taskId,
				SignalKey: tuple.signalKey,
			}))

			assert.Equal(t, tt.expectedFirst, firstCalls.Load())
			assert.Equal(t, tt.expectedSecond, secondCalls.Load())
			assert.False(t, listener.reg.hasAny())
		})
	}
}

func TestDurableEventsListenerConcurrentStoreAndDispatchDoesNotOrphanHandlers(t *testing.T) {
	logger := zerolog.Nop()
	listener := newDurableEventsListener(&logger, func(ctx context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
		return &mockDurableEventClient{}, nil
	})
	tuple := listenTuple{taskId: "task-1", signalKey: "signal-1"}
	event := &contracts.DurableEvent{
		TaskId:    tuple.taskId,
		SignalKey: tuple.signalKey,
	}

	for i := 0; i < 500; i++ {
		fired := atomic.Bool{}

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			assert.NoError(t, listener.dispatch(event))
		}()
		go func() {
			defer wg.Done()
			listener.reg.store(tuple, "", func(e DurableEvent) error {
				fired.Store(true)
				return nil
			}, nil)
		}()

		wg.Wait()

		if !fired.Load() {
			require.NoError(t, listener.dispatch(event))
			require.True(t, fired.Load(), "handler stored concurrently with dispatch was orphaned")
		}
	}

	assert.False(t, listener.reg.hasAny())
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

	listener := newTestDurableEventsListener(t, &logger, func(ctx context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
		closeConstructorCalled.Do(func() {
			close(constructorCalled)
		})
		return newClient, nil
	}, oldClient)

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

	listener := newTestDurableEventsListener(t, &logger, func(ctx context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
		return replacementClient, nil
	}, initialClient)

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

	listener := newTestDurableEventsListener(t, &logger, func(ctx context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
		constructorCalls.Add(1)
		return replacementClient, nil
	}, initialClient)

	received := make(chan DurableEvent, 1)
	listener.reg.store(listenTuple{taskId: "task-1", signalKey: "signal-1"}, "", func(e DurableEvent) error {
		received <- e
		return nil
	}, nil)

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

	listener := newTestDurableEventsListener(t, &logger, func(ctx context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
		constructorCalls.Add(1)
		return nil, status.Error(codes.Unavailable, "engine down")
	}, nil)

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

	listener := newTestDurableEventsListener(t, &logger, func(ctx context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
		constructorCalls.Add(1)
		return &mockDurableEventClient{recvCh: make(chan *contracts.DurableEvent)}, nil
	}, &mockDurableEventClient{
		recvFn: func() (*contracts.DurableEvent, error) {
			return nil, io.EOF
		},
	})

	require.NoError(t, listener.Listen(context.Background()))
	assert.Equal(t, int32(0), constructorCalls.Load())
}
