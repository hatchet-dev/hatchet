package client

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
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
			return nil, status.Error(codes.Unavailable, "stream broken")
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

	addErr := make(chan error, 1)
	go func() {
		addErr <- listener.AddSignal("task-1", "signal-1", func(e DurableEvent) error {
			return nil
		})
	}()

	// Wait until AddSignal has definitely attempted to use the broken stream
	// before forcing Listen down the reconnect path.
	require.Eventually(t, func() bool {
		select {
		case <-sendAttempted:
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- listener.Listen(ctx)
	}()

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

// TestDurableEventsListenerDeliversEventAfterReconnectDuringRetryBackoff
// verifies the user-visible symptom: after a disconnect during AddSignal's
// retry window, the listener should reconnect and deliver the durable event
// from the new stream without waiting for retrySend to exhaust all attempts.
func TestDurableEventsListenerDeliversEventAfterReconnectDuringRetryBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := zerolog.Nop()
	sendAttempted := make(chan struct{})
	var closeSendAttempted sync.Once

	// The first stream simulates a dead gRPC stream: sends fail, and the receive
	// loop observes the disconnect that should trigger reconnection.
	oldClient := &mockDurableEventClient{
		sendFn: func(req *contracts.ListenForDurableEventRequest) error {
			closeSendAttempted.Do(func() {
				close(sendAttempted)
			})
			return status.Error(codes.Unavailable, "stream broken")
		},
		recvFn: func() (*contracts.DurableEvent, error) {
			return nil, status.Error(codes.Unavailable, "stream broken")
		},
	}

	newClient := &mockDurableEventClient{
		recvCh: make(chan *contracts.DurableEvent, 1),
	}

	// Reconnection immediately makes the awaited event available on the new stream.
	// If reconnect is blocked by retrySend, this event will not reach the handler.
	listener := &DurableEventsListener{
		constructor: func(ctx context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
			newClient.recvCh <- &contracts.DurableEvent{
				TaskId:    "task-1",
				SignalKey: "signal-1",
				Data:      []byte(`{"ok":true}`),
			}
			return newClient, nil
		},
		client: oldClient,
		l:      &logger,
	}

	received := make(chan DurableEvent, 1)
	addErr := make(chan error, 1)
	go func() {
		addErr <- listener.AddSignal("task-1", "signal-1", func(e DurableEvent) error {
			received <- e
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

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- listener.Listen(ctx)
	}()

	select {
	case event := <-received:
		require.Equal(t, "task-1", event.TaskId)
		require.Equal(t, "signal-1", event.SignalKey)
		require.JSONEq(t, `{"ok":true}`, string(event.Data))
	case err := <-addErr:
		require.NoError(t, err)
	case err := <-listenErr:
		require.NoError(t, err)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected durable event to be delivered after reconnect during retrySend backoff")
	}
}
