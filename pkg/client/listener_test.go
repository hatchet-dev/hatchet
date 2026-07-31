// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
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

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/client/retry"
)

func TestWorkflowRunsListenerAddWorkflowRunSendsOnceWhenStarting(t *testing.T) {
	logger := zerolog.Nop()
	recvChan := make(chan *dispatchercontracts.WorkflowRunEvent)

	client := &mockSubscribeClient{
		recvChan: recvChan,
	}

	listener := newTestWorkflowRunsListener(t, &logger, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		return client, nil
	}, client)

	require.NoError(t, listener.AddWorkflowRun("run-1", "session-1", func(event WorkflowRunEvent) error {
		return nil
	}))
	assert.Equal(t, int32(1), client.sendCount.Load())

	require.NoError(t, listener.Close())
	close(recvChan)
}

func TestWorkflowRunsListenerAddWorkflowRunRollsBackHandlerWhenSendFails(t *testing.T) {
	logger := zerolog.Nop()
	client := &mockSubscribeClient{
		sendErr: status.Error(codes.Unavailable, "send failed"),
		recvErr: io.EOF,
	}

	listener := newTestWorkflowRunsListener(t, &logger, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		return client, nil
	}, client)

	err := listener.AddWorkflowRun("run-1", "session-1", func(event WorkflowRunEvent) error {
		return nil
	})
	require.Error(t, err)
	assert.False(t, listener.reg.hasAny())

	require.NoError(t, listener.Close())
	require.Eventually(t, func() bool {
		return !listener.isListening()
	}, time.Second, 10*time.Millisecond)
}

func TestWorkflowRunsListenerAddWorkflowRunKeepsHandlerDuringRecovery(t *testing.T) {
	logger := zerolog.Nop()
	oldClient := &mockSubscribeClient{}
	recoveredClient := &mockSubscribeClient{
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}

	var listener *WorkflowRunsListener
	reconnectFailures := atomic.Int32{}
	recoveredSends := atomic.Int32{}
	missingHandlerDuringRecovery := atomic.Bool{}

	oldClient.sendFn = func(req *dispatchercontracts.SubscribeToWorkflowRunsRequest) error {
		listener.gate.stop()
		return status.Error(codes.Unavailable, "stream broken")
	}
	recoveredClient.sendFn = func(req *dispatchercontracts.SubscribeToWorkflowRunsRequest) error {
		recoveredSends.Add(1)
		if !listener.reg.hasAny() {
			missingHandlerDuringRecovery.Store(true)
		}
		return nil
	}

	listener = newTestWorkflowRunsListener(t, &logger, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		if reconnectFailures.Add(1) <= int32(2*retry.StreamSyncMaxAttempts-1) {
			return nil, status.Error(codes.Unavailable, "engine down")
		}

		return recoveredClient, nil
	}, oldClient)
	require.True(t, listener.gate.tryStart(false))

	err := listener.AddWorkflowRun("run-1", "session-1", func(event WorkflowRunEvent) error {
		return nil
	})
	require.NoError(t, err)
	assert.False(t, missingHandlerDuringRecovery.Load())
	assert.True(t, listener.reg.hasAny())
	assert.GreaterOrEqual(t, recoveredSends.Load(), int32(1))

	require.NoError(t, listener.Close())
	close(recoveredClient.recvChan)
	require.Eventually(t, func() bool {
		return !listener.isListening()
	}, time.Second, 10*time.Millisecond)
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

func TestWorkflowRunsListenerHandlerErrorDoesNotFailUnrelatedWaiters(t *testing.T) {
	logger := zerolog.Nop()
	recvChan := make(chan *dispatchercontracts.WorkflowRunEvent, 2)

	client := &mockSubscribeClient{
		recvChan: recvChan,
	}

	listener := newTestWorkflowRunsListener(t, &logger, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		return client, nil
	}, client)

	onErrorCalled := atomic.Bool{}
	listener.reg.store("run-1", "session-1", func(event WorkflowRunEvent) error {
		return errors.New("handler failed")
	}, func(err error) {
		onErrorCalled.Store(true)
	})

	receivedRun2 := make(chan WorkflowRunEvent, 1)
	listener.reg.store("run-2", "session-2", func(event WorkflowRunEvent) error {
		receivedRun2 <- event
		return nil
	}, func(err error) {
		t.Errorf("unrelated onError should not be invoked: %v", err)
	})

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- listener.Listen(context.Background())
	}()

	recvChan <- &dispatchercontracts.WorkflowRunEvent{
		WorkflowRunId: "run-1",
	}

	require.Eventually(t, func() bool {
		return listener.reg.hasAny() && !onErrorCalled.Load()
	}, time.Second, 10*time.Millisecond, "handler error should not fail unrelated waiters")

	recvChan <- &dispatchercontracts.WorkflowRunEvent{
		WorkflowRunId: "run-2",
	}

	select {
	case event := <-receivedRun2:
		assert.Equal(t, "run-2", event.WorkflowRunId)
	case <-time.After(time.Second):
		t.Fatal("expected later event to reach unaffected waiter")
	}

	listener.reg.removeSession("run-1", "session-1")
	listener.reg.removeSession("run-2", "session-2")
	close(recvChan)

	err := <-listenErr
	assert.NoError(t, err)
}

func TestListen_DispatchesEventsToHandlers(t *testing.T) {
	logger := zerolog.Nop()
	recvChan := make(chan *dispatchercontracts.WorkflowRunEvent, 1)

	client := &mockSubscribeClient{
		recvChan: recvChan,
	}

	listener := newTestWorkflowRunsListener(t, &logger, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		return client, nil
	}, client)

	var receivedEvent atomic.Value
	listener.reg.store("run-1", "session-1", func(event WorkflowRunEvent) error {
		receivedEvent.Store(event)
		return nil
	}, nil)

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
	listener.reg.removeSession("run-1", "session-1")

	close(recvChan)

	err := <-listenErr
	assert.NoError(t, err, "Listen should exit cleanly on EOF")

	ev := receivedEvent.Load()
	require.NotNil(t, ev, "handler should have been called")
	assert.Equal(t, "run-1", ev.(WorkflowRunEvent).WorkflowRunId)
}

func TestWorkflowRunsListenerRestartsAfterListenExits(t *testing.T) {
	logger := zerolog.Nop()

	initialClient := &mockSubscribeClient{
		recvErr: io.EOF,
	}
	replacementClient := &mockSubscribeClient{
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent, 1),
	}

	listener := newTestWorkflowRunsListener(t, &logger, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		return replacementClient, nil
	}, initialClient)

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

	listener := newTestWorkflowRunsListener(t, &logger, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		constructorCalls.Add(1)
		return replacementClient, nil
	}, initialClient)

	received := make(chan WorkflowRunEvent, 1)
	listener.reg.store("run-1", "session-1", func(event WorkflowRunEvent) error {
		received <- event
		return nil
	}, nil)

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

	listener := newTestWorkflowRunsListener(t, &logger, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		constructorCalls.Add(1)
		return &mockSubscribeClient{
			recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
		}, nil
	}, initialClient)

	listener.reg.store("run-1", "session-1", func(event WorkflowRunEvent) error {
		return nil
	}, nil)

	require.NoError(t, listener.Close())
	require.NoError(t, listener.listen(context.Background()))
	assert.Equal(t, int32(0), constructorCalls.Load())

	err := listener.AddWorkflowRun("run-2", "session-2", func(event WorkflowRunEvent) error {
		return nil
	})
	require.ErrorIs(t, err, errListenerClosed)
}

func TestClose_NilClient(t *testing.T) {
	logger := zerolog.Nop()

	listener := newTestWorkflowRunsListener(t, &logger, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		return nil, nil
	}, nil)

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

	listener := newTestWorkflowRunsListener(t, &logger, nil, client)

	err := listener.Close()
	assert.NoError(t, err)
	assert.True(t, closeCalled.Load(), "CloseSend should have been called")
}

func TestAddWorkflowRunBoundedWhenEngineDown(t *testing.T) {
	logger := zerolog.Nop()
	constructorCalls := atomic.Int32{}

	listener := newTestWorkflowRunsListener(t, &logger, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		constructorCalls.Add(1)
		return nil, status.Error(codes.Unavailable, "engine down")
	}, nil)

	err := listener.AddWorkflowRun("run-1", "session-1", func(event WorkflowRunEvent) error {
		return nil
	})

	require.Error(t, err)
	assert.LessOrEqual(t, constructorCalls.Load(), int32(retry.StreamSyncMaxAttempts+1))
}

func TestListenEOFWithoutHandlersDoesNotReconnect(t *testing.T) {
	logger := zerolog.Nop()
	constructorCalls := atomic.Int32{}

	listener := newTestWorkflowRunsListener(t, &logger, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		constructorCalls.Add(1)
		return &mockSubscribeClient{recvChan: make(chan *dispatchercontracts.WorkflowRunEvent)}, nil
	}, &mockSubscribeClient{recvErr: io.EOF})

	require.NoError(t, listener.Listen(context.Background()))
	assert.Equal(t, int32(0), constructorCalls.Load())
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

	listener := newTestWorkflowRunsListener(t, &logger, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		return client, nil
	}, client)

	listener.reg.store("run-1", "session-1", func(event WorkflowRunEvent) error { return nil }, nil)

	require.NoError(t, listener.stream.connectSync(context.Background()))
	assert.Equal(t, int32(1), sendCount.Load())
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

func TestStreamByAdditionalMetadataReconnects(t *testing.T) {
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
		stream := subscriber.newMetadataStream(ctx, "k", "v")
		disableStreamBackoff(t, stream)
		defer func() { _ = stream.Close() }()
		done <- runMetadataStream(ctx, stream, func(event StreamEvent) error {
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
		stream := subscriber.newMetadataStream(ctx, "k", "v")
		disableStreamBackoff(t, stream)
		defer func() { _ = stream.Close() }()
		done <- runMetadataStream(ctx, stream, func(event StreamEvent) error {
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
