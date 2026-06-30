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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/pkg/client/retry"
)

type testListenEvent struct {
	value string
}

type testListenClient struct {
	recvFn      func() (testListenEvent, error)
	closeSendFn func() error
	closeCalled atomic.Bool
}

func (c *testListenClient) Recv() (testListenEvent, error) {
	if c.recvFn != nil {
		return c.recvFn()
	}
	return testListenEvent{}, io.EOF
}

func (c *testListenClient) CloseSend() error {
	c.closeCalled.Store(true)
	if c.closeSendFn != nil {
		return c.closeSendFn()
	}
	return nil
}

func newTestListenStream(
	t *testing.T,
	initial *testListenClient,
	constructor func(context.Context) (*testListenClient, error),
) *reconnectingStream[*testListenClient] {
	t.Helper()

	stream := newReconnectingStream(
		constructor,
		func(client *testListenClient) error {
			return client.CloseSend()
		},
		nil,
	)
	stream.setInitialClient(initial)
	return stream
}

func testListenConfig(
	stream *reconnectingStream[*testListenClient],
	reconnectCtx context.Context,
	shouldReconnectOnEOF func(context.Context) bool,
) streamListenConfig[*testListenClient, testListenEvent] {
	logger := zerolog.Nop()
	return streamListenConfig[*testListenClient, testListenEvent]{
		recv: func(client *testListenClient) (testListenEvent, error) {
			return client.Recv()
		},
		handle: func(event testListenEvent) error {
			return nil
		},
		shouldReconnectOnEOF: shouldReconnectOnEOF,
		reconnectContext:     reconnectCtx,
		labels: streamListenLabels{
			streamName:    "test listener",
			reconnectVerb: "resubscribe",
		},
		l: &logger,
	}
}

func TestListenReconnectingStreamHandlesEventsAndStopsOnEOF(t *testing.T) {
	recvChan := make(chan testListenEvent, 1)
	client := &testListenClient{
		recvFn: func() (testListenEvent, error) {
			select {
			case event, ok := <-recvChan:
				if !ok {
					return testListenEvent{}, io.EOF
				}
				return event, nil
			default:
				return testListenEvent{}, io.EOF
			}
		},
	}

	var handled atomic.Value
	stream := newTestListenStream(t, client, func(ctx context.Context) (*testListenClient, error) {
		return client, nil
	})

	cfg := testListenConfig(stream, context.Background(), func(context.Context) bool { return false })
	cfg.handle = func(event testListenEvent) error {
		handled.Store(event.value)
		return nil
	}

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- listenReconnectingStream(context.Background(), stream, cfg)
	}()

	recvChan <- testListenEvent{value: "event-1"}
	require.Eventually(t, func() bool {
		return handled.Load() == "event-1"
	}, time.Second, 10*time.Millisecond)

	close(recvChan)
	require.NoError(t, <-listenErr)
	assert.True(t, client.closeCalled.Load())
}

func TestListenReconnectingStreamStopDecisionReturnsNil(t *testing.T) {
	client := &testListenClient{
		recvFn: func() (testListenEvent, error) {
			return testListenEvent{}, status.Error(codes.Canceled, "canceled")
		},
	}

	stream := newTestListenStream(t, client, func(ctx context.Context) (*testListenClient, error) {
		return client, nil
	})

	err := listenReconnectingStream(context.Background(), stream, testListenConfig(stream, context.Background(), func(context.Context) bool {
		return true
	}))
	require.NoError(t, err)
}

func TestListenReconnectingStreamReconnectsOnEOFWhenPolicyAllows(t *testing.T) {
	disableStreamBackoffForTest(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	initialClient := &testListenClient{
		recvFn: func() (testListenEvent, error) {
			return testListenEvent{}, io.EOF
		},
	}
	replacementRecv := make(chan testListenEvent, 1)
	replacementClient := &testListenClient{
		recvFn: func() (testListenEvent, error) {
			event, ok := <-replacementRecv
			if !ok {
				return testListenEvent{}, io.EOF
			}
			return event, nil
		},
	}
	constructorCalls := atomic.Int32{}

	stream := newTestListenStream(t, initialClient, func(ctx context.Context) (*testListenClient, error) {
		constructorCalls.Add(1)
		return replacementClient, nil
	})

	var handled atomic.Value
	cfg := testListenConfig(stream, context.Background(), func(context.Context) bool { return ctx.Err() == nil })
	cfg.handle = func(event testListenEvent) error {
		handled.Store(event.value)
		cancel()
		return nil
	}

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- listenReconnectingStream(ctx, stream, cfg)
	}()

	require.Eventually(t, func() bool {
		return constructorCalls.Load() == 1
	}, time.Second, 10*time.Millisecond)

	replacementRecv <- testListenEvent{value: "after-reconnect"}
	require.Eventually(t, func() bool {
		return handled.Load() == "after-reconnect"
	}, time.Second, 10*time.Millisecond)

	close(replacementRecv)
	require.NoError(t, <-listenErr)
}

func TestListenReconnectingStreamChecksEOFPolicyEachRecvError(t *testing.T) {
	disableStreamBackoffForTest(t)

	policyCalls := atomic.Int32{}
	recvCalls := atomic.Int32{}
	client := &testListenClient{
		recvFn: func() (testListenEvent, error) {
			recvCalls.Add(1)
			return testListenEvent{}, io.EOF
		},
	}

	constructorCalls := atomic.Int32{}
	stream := newTestListenStream(t, client, func(ctx context.Context) (*testListenClient, error) {
		constructorCalls.Add(1)
		return client, nil
	})

	err := listenReconnectingStream(context.Background(), stream, testListenConfig(stream, context.Background(), func(context.Context) bool {
		return policyCalls.Add(1) == 1
	}))
	require.NoError(t, err)
	assert.Equal(t, int32(2), policyCalls.Load())
	assert.Equal(t, int32(2), recvCalls.Load())
	assert.Equal(t, int32(1), constructorCalls.Load())
}

func TestListenReconnectingStreamNoProgressReconnectsBeforeCap(t *testing.T) {
	disableStreamBackoffForTest(t)

	recvCalls := atomic.Int32{}
	constructorCalls := atomic.Int32{}

	client := &testListenClient{
		recvFn: func() (testListenEvent, error) {
			if recvCalls.Add(1) == 1 {
				return testListenEvent{}, fmt.Errorf("plain recv error")
			}
			return testListenEvent{}, io.EOF
		},
	}

	stream := newTestListenStream(t, client, func(ctx context.Context) (*testListenClient, error) {
		constructorCalls.Add(1)
		return client, nil
	})

	err := listenReconnectingStream(context.Background(), stream, testListenConfig(stream, context.Background(), func(context.Context) bool {
		return false
	}))
	require.NoError(t, err)
	assert.Equal(t, int32(1), constructorCalls.Load())
	assert.Equal(t, int32(2), recvCalls.Load())
}

func TestListenReconnectingStreamNoProgressStopsAtCap(t *testing.T) {
	disableStreamBackoffForTest(t)

	client := &testListenClient{
		recvFn: func() (testListenEvent, error) {
			return testListenEvent{}, fmt.Errorf("plain recv error")
		},
	}

	stream := newTestListenStream(t, client, func(ctx context.Context) (*testListenClient, error) {
		return nil, fmt.Errorf("plain connect error")
	})

	err := listenReconnectingStream(context.Background(), stream, testListenConfig(stream, context.Background(), func(context.Context) bool {
		return true
	}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resubscribe")
}

func TestListenReconnectingStreamUsesReconnectContext(t *testing.T) {
	disableStreamBackoffForTest(t)

	listenCtx := context.Background()
	reconnectCtx, cancelReconnect := context.WithCancel(context.Background())
	defer cancelReconnect()

	var constructorCtx atomic.Value
	recvCalls := atomic.Int32{}
	client := &testListenClient{
		recvFn: func() (testListenEvent, error) {
			if recvCalls.Add(1) == 1 {
				return testListenEvent{}, status.Error(codes.Unavailable, "broken")
			}
			return testListenEvent{}, io.EOF
		},
	}

	stream := newTestListenStream(t, client, func(ctx context.Context) (*testListenClient, error) {
		constructorCtx.Store(ctx)
		return client, nil
	})

	cfg := testListenConfig(stream, reconnectCtx, func(context.Context) bool {
		return recvCalls.Load() == 1
	})

	err := listenReconnectingStream(listenCtx, stream, cfg)
	require.NoError(t, err)

	storedCtx := constructorCtx.Load().(context.Context)
	assert.Equal(t, reconnectCtx, storedCtx)
}

func TestListenReconnectingStreamSleepCancellationReturnsNil(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	retry.SetStreamSleepHookForTesting(func(ctx context.Context, attempt int) error {
		return ctx.Err()
	})
	t.Cleanup(retry.ResetStreamSleepHookForTesting)

	reconnectAttempt := 1
	if reconnectAttempt > 0 {
		if sleepErr := retry.SleepStreamBackoff(ctx, reconnectAttempt-1); sleepErr != nil {
			require.Error(t, sleepErr)
			return
		}
	}

	t.Fatal("expected canceled sleep before reconnect")
}

func TestListenReconnectingStreamGenerationChangeFastPath(t *testing.T) {
	disableStreamBackoffForTest(t)

	releaseReconnect := make(chan struct{})
	initialClient := &testListenClient{
		recvFn: func() (testListenEvent, error) {
			return testListenEvent{}, status.Error(codes.Unavailable, "broken")
		},
	}
	replacementClient := &testListenClient{
		recvFn: func() (testListenEvent, error) {
			return testListenEvent{}, io.EOF
		},
	}

	constructorCalls := atomic.Int32{}
	stream := newTestListenStream(t, initialClient, func(ctx context.Context) (*testListenClient, error) {
		if constructorCalls.Add(1) == 1 {
			<-releaseReconnect
		}
		return replacementClient, nil
	})

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- listenReconnectingStream(context.Background(), stream, testListenConfig(stream, context.Background(), func(context.Context) bool {
			return false
		}))
	}()

	require.Eventually(t, func() bool {
		return constructorCalls.Load() == 1
	}, time.Second, 10*time.Millisecond)

	require.NoError(t, stream.installClient(replacementClient))
	close(releaseReconnect)

	require.NoError(t, <-listenErr)
	assert.Equal(t, int32(1), constructorCalls.Load())
}

func TestListenReconnectingStreamClosesListenedClientOnExit(t *testing.T) {
	initialClient := &testListenClient{
		recvFn: func() (testListenEvent, error) {
			return testListenEvent{}, io.EOF
		},
	}
	replacementClient := &testListenClient{
		recvFn: func() (testListenEvent, error) {
			return testListenEvent{}, io.EOF
		},
	}

	stream := newTestListenStream(t, initialClient, func(ctx context.Context) (*testListenClient, error) {
		return replacementClient, nil
	})

	require.NoError(t, listenReconnectingStream(context.Background(), stream, testListenConfig(stream, context.Background(), func(context.Context) bool {
		return false
	})))
	assert.True(t, initialClient.closeCalled.Load())
	assert.False(t, replacementClient.closeCalled.Load())
}
