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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/pkg/client/retry"
)

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

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- listenStream(context.Background(), stream,
			func(c *testListenClient) (testListenEvent, error) { return c.Recv() },
			func(event testListenEvent) error {
				handled.Store(event.value)
				return nil
			},
			newStreamClassifier(func(context.Context) bool { return false }),
		)
	}()

	recvChan <- testListenEvent{value: "event-1"}
	require.Eventually(t, func() bool {
		return handled.Load() == "event-1"
	}, time.Second, 10*time.Millisecond)

	close(recvChan)
	require.NoError(t, <-listenErr)
	assert.True(t, client.closeCalled.Load())
}

func TestListenReconnectingStreamContextCancellationReturnsNil(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	client := &testListenClient{
		recvFn: func() (testListenEvent, error) {
			cancel()
			return testListenEvent{}, status.Error(codes.Canceled, "canceled")
		},
	}

	stream := newTestListenStream(t, client, func(ctx context.Context) (*testListenClient, error) {
		return client, nil
	})

	err := listenStream(ctx, stream,
		func(c *testListenClient) (testListenEvent, error) { return c.Recv() },
		func(testListenEvent) error { return nil },
		newStreamClassifier(func(context.Context) bool { return true }),
	)
	require.NoError(t, err)
}

func TestListenReconnectingStreamPermanentRecvErrorReturnsError(t *testing.T) {
	recvErr := status.Error(codes.PermissionDenied, "permission denied")
	client := &testListenClient{
		recvFn: func() (testListenEvent, error) {
			return testListenEvent{}, recvErr
		},
	}

	stream := newTestListenStream(t, client, func(ctx context.Context) (*testListenClient, error) {
		t.Fatal("constructor should not run after permanent receive error")
		return nil, nil
	})

	err := listenStream(context.Background(), stream,
		func(c *testListenClient) (testListenEvent, error) { return c.Recv() },
		func(testListenEvent) error { return nil },
		newStreamClassifier(func(context.Context) bool { return true }),
	)
	require.ErrorIs(t, err, recvErr)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestListenReconnectingStreamReconnectsOnEOFWhenPolicyAllows(t *testing.T) {
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
	listenErr := make(chan error, 1)
	go func() {
		listenErr <- listenStream(ctx, stream,
			func(c *testListenClient) (testListenEvent, error) { return c.Recv() },
			func(event testListenEvent) error {
				handled.Store(event.value)
				cancel()
				return nil
			},
			newStreamClassifier(func(context.Context) bool { return ctx.Err() == nil }),
		)
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

	err := listenStream(context.Background(), stream,
		func(c *testListenClient) (testListenEvent, error) { return c.Recv() },
		func(testListenEvent) error { return nil },
		newStreamClassifier(func(context.Context) bool {
			return policyCalls.Add(1) == 1
		}),
	)
	require.NoError(t, err)
	assert.Equal(t, int32(2), policyCalls.Load())
	assert.Equal(t, int32(2), recvCalls.Load())
	assert.Equal(t, int32(1), constructorCalls.Load())
}

func TestListenReconnectingStreamNoProgressReconnectsBeforeCap(t *testing.T) {
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

	err := listenStream(context.Background(), stream,
		func(c *testListenClient) (testListenEvent, error) { return c.Recv() },
		func(testListenEvent) error { return nil },
		newStreamClassifier(func(context.Context) bool { return false }),
	)
	require.NoError(t, err)
	assert.Equal(t, int32(1), constructorCalls.Load())
	assert.Equal(t, int32(2), recvCalls.Load())
}

func TestListenStreamNoProgressStopsAtCap(t *testing.T) {
	recvCalls := atomic.Int32{}
	constructorCalls := atomic.Int32{}
	client := &testListenClient{
		recvFn: func() (testListenEvent, error) {
			recvCalls.Add(1)
			return testListenEvent{}, fmt.Errorf("plain recv error")
		},
	}

	stream := newTestListenStream(t, client, func(ctx context.Context) (*testListenClient, error) {
		constructorCalls.Add(1)
		return nil, fmt.Errorf("plain connect error")
	})

	err := listenStream(context.Background(), stream,
		func(c *testListenClient) (testListenEvent, error) { return c.Recv() },
		func(testListenEvent) error { return nil },
		newStreamClassifier(func(context.Context) bool { return true }),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "made no progress")
	assert.Equal(t, int32(1), recvCalls.Load())
	assert.GreaterOrEqual(t, constructorCalls.Load(), int32(maxConsecutiveStreamNoProgress-2))
}

func TestListenStreamNoProgressFatalClassifierStopsImmediately(t *testing.T) {
	recvErr := fmt.Errorf("plain recv error")
	recvCalls := atomic.Int32{}
	client := &testListenClient{
		recvFn: func() (testListenEvent, error) {
			recvCalls.Add(1)
			return testListenEvent{}, recvErr
		},
	}

	stream := newTestListenStream(t, client, func(ctx context.Context) (*testListenClient, error) {
		t.Fatal("constructor should not run when no-progress policy stops immediately")
		return nil, nil
	})

	base := newStreamClassifier(func(context.Context) bool { return false })
	classify := func(ctx context.Context, err error) streamVerdict {
		if v := base(ctx, err); v != verdictNoProgress {
			return v
		}
		return verdictStopError
	}

	err := listenStream(context.Background(), stream,
		func(c *testListenClient) (testListenEvent, error) { return c.Recv() },
		func(testListenEvent) error { return nil },
		classify,
	)
	require.ErrorIs(t, err, recvErr)
	assert.Equal(t, int32(1), recvCalls.Load())
}

func TestListenStreamConnectsUseLifecycleContext(t *testing.T) {
	listenCtx := context.Background()

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

	err := listenStream(listenCtx, stream,
		func(c *testListenClient) (testListenEvent, error) { return c.Recv() },
		func(testListenEvent) error { return nil },
		newStreamClassifier(func(context.Context) bool {
			return recvCalls.Load() == 1
		}),
	)
	require.NoError(t, err)

	storedCtx := constructorCtx.Load().(context.Context)
	assert.Equal(t, stream.lifecycleContext(), storedCtx)
}

func TestListenReconnectingStreamGenerationChangeFastPath(t *testing.T) {
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
		listenErr <- listenStream(context.Background(), stream,
			func(c *testListenClient) (testListenEvent, error) { return c.Recv() },
			func(testListenEvent) error { return nil },
			newStreamClassifier(func(context.Context) bool { return false }),
		)
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

	require.NoError(t, listenStream(context.Background(), stream,
		func(c *testListenClient) (testListenEvent, error) { return c.Recv() },
		func(testListenEvent) error { return nil },
		newStreamClassifier(func(context.Context) bool { return false }),
	))
	assert.True(t, initialClient.closeCalled.Load())
	assert.False(t, replacementClient.closeCalled.Load())
}

func TestListenStreamReconnectsUnboundedlyOnTransientErrors(t *testing.T) {
	constructorCalls := atomic.Int32{}
	ctx, cancel := context.WithCancel(context.Background())

	stream := newTestListenStream(t, nil, func(ctx context.Context) (*testListenClient, error) {
		if constructorCalls.Add(1) >= retry.StreamSyncMaxAttempts+1 {
			cancel()
		}
		return nil, status.Error(codes.Unavailable, "still down")
	})

	err := listenStream(ctx, stream,
		func(c *testListenClient) (testListenEvent, error) { return c.Recv() },
		func(testListenEvent) error { return nil },
		newStreamClassifier(func(context.Context) bool { return true }),
	)
	require.NoError(t, err)
	assert.Greater(t, constructorCalls.Load(), int32(retry.StreamSyncMaxAttempts))
}

func TestListenStreamRetryableRecvFailureAppliesReconnectBackoff(t *testing.T) {
	var sleepAttempts []int
	recvCalls := atomic.Int32{}

	client := &testListenClient{
		recvFn: func() (testListenEvent, error) {
			if recvCalls.Add(1) == 1 {
				return testListenEvent{}, status.Error(codes.Unavailable, "transient")
			}
			return testListenEvent{}, io.EOF
		},
	}

	stream := newTestListenStream(t, client, func(ctx context.Context) (*testListenClient, error) {
		return client, nil
	})
	stream.sleep = func(_ context.Context, attempt int) error {
		sleepAttempts = append(sleepAttempts, attempt)
		return nil
	}

	err := listenStream(context.Background(), stream,
		func(c *testListenClient) (testListenEvent, error) { return c.Recv() },
		func(testListenEvent) error { return nil },
		newStreamClassifier(func(context.Context) bool { return false }),
	)
	require.NoError(t, err)
	assert.Equal(t, []int{0}, sleepAttempts)
}

func TestListenStreamSleepCancellationReturnsNil(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	stream := newTestListenStream(t, &testListenClient{
		recvFn: func() (testListenEvent, error) {
			return testListenEvent{}, status.Error(codes.Unavailable, "stream broken")
		},
	}, func(ctx context.Context) (*testListenClient, error) {
		return nil, status.Error(codes.Unavailable, "still down")
	})
	stream.sleep = func(sleepCtx context.Context, _ int) error {
		cancel()
		return sleepCtx.Err()
	}

	err := listenStream(ctx, stream,
		func(c *testListenClient) (testListenEvent, error) { return c.Recv() },
		func(testListenEvent) error { return nil },
		newStreamClassifier(func(context.Context) bool { return true }),
	)
	require.NoError(t, err)
}
