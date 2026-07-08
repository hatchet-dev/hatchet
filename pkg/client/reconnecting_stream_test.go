package client

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/client/retry"
)

func TestRetrySend_ResubscribesOnSendFailure(t *testing.T) {
	constructorCalls := atomic.Int32{}

	failingClient := &mockSubscribeClient{
		sendErr:  fmt.Errorf("stream broken"),
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}

	workingClient := &mockSubscribeClient{
		sendErr:  nil,
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}

	stream := newTestWorkflowStream(t, failingClient, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		calls := constructorCalls.Add(1)
		if calls == 1 {
			return failingClient, nil
		}
		return workingClient, nil
	})

	err := stream.retrySend(context.Background(), func(c dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient) error {
		return c.Send(&dispatchercontracts.SubscribeToWorkflowRunsRequest{WorkflowRunId: "test-workflow-run-id"})
	})

	require.NoError(t, err)
	assert.GreaterOrEqual(t, constructorCalls.Load(), int32(1))
	assert.GreaterOrEqual(t, failingClient.sendCount.Load(), int32(1))
	assert.Equal(t, int32(1), workingClient.sendCount.Load())
}

func TestReconnectingStreamSnapshotNotBlockedByConnect(t *testing.T) {
	constructorEntered := make(chan struct{})
	releaseConstructor := make(chan struct{})
	var closeConstructorEntered sync.Once

	oldClient := &mockSubscribeClient{
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}
	newClient := &mockSubscribeClient{
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}

	stream := newTestWorkflowStream(t, oldClient, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		closeConstructorEntered.Do(func() {
			close(constructorEntered)
		})
		<-releaseConstructor
		return newClient, nil
	})

	retryErr := make(chan error, 1)
	go func() {
		retryErr <- stream.connectOnce(context.Background())
	}()

	select {
	case <-constructorEntered:
	case <-time.After(time.Second):
		t.Fatal("reconnect did not reach constructor")
	}

	snapshotRead := make(chan struct {
		client     dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient
		generation uint64
	}, 1)
	go func() {
		client, generation, _ := stream.snapshot()
		snapshotRead <- struct {
			client     dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient
			generation uint64
		}{client: client, generation: generation}
	}()

	select {
	case snapshot := <-snapshotRead:
		require.Same(t, oldClient, snapshot.client)
		require.Equal(t, uint64(0), snapshot.generation)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("snapshot blocked behind reconnect")
	}

	close(releaseConstructor)

	require.NoError(t, <-retryErr)
	require.NoError(t, stream.Close())
}

func TestRetrySendConnectAttemptsAreFlat(t *testing.T) {
	failingClient := &mockSubscribeClient{
		sendErr:  fmt.Errorf("stream broken"),
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}

	constructorCalls := atomic.Int32{}
	sleepCalls := atomic.Int32{}

	stream := newTestWorkflowStream(t, failingClient, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		constructorCalls.Add(1)
		return nil, status.Error(codes.Unavailable, "engine down")
	})
	stream.sleep = func(context.Context, int) error {
		sleepCalls.Add(1)
		return nil
	}

	err := stream.retrySend(context.Background(), func(c dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient) error {
		return c.Send(&dispatchercontracts.SubscribeToWorkflowRunsRequest{WorkflowRunId: "test-workflow-run-id"})
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("could not send to workflow run listener after %d attempts", retry.StreamSyncMaxAttempts))
	assert.Equal(t, int32(retry.StreamSyncMaxAttempts), constructorCalls.Load())
	assert.Equal(t, int32(retry.StreamSyncMaxAttempts-1), sleepCalls.Load())
	assert.Equal(t, int32(retry.StreamSyncMaxAttempts), failingClient.sendCount.Load())
}

func TestRetrySend_FailsAfterMaxRetries(t *testing.T) {
	failingClient := &mockSubscribeClient{
		sendErr:  fmt.Errorf("stream broken"),
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}

	stream := newTestWorkflowStream(t, failingClient, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		return failingClient, nil
	})

	err := stream.retrySend(context.Background(), func(c dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient) error {
		return c.Send(&dispatchercontracts.SubscribeToWorkflowRunsRequest{WorkflowRunId: "test-workflow-run-id"})
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not send to")
	assert.Equal(t, int32(retry.StreamSyncMaxAttempts), failingClient.sendCount.Load())
}

func TestRetrySend_SucceedsOnFirstAttempt(t *testing.T) {
	workingClient := &mockSubscribeClient{
		sendErr:  nil,
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}

	constructorCalls := atomic.Int32{}

	stream := newTestWorkflowStream(t, workingClient, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		constructorCalls.Add(1)
		return workingClient, nil
	})

	err := stream.retrySend(context.Background(), func(c dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient) error {
		return c.Send(&dispatchercontracts.SubscribeToWorkflowRunsRequest{WorkflowRunId: "test-workflow-run-id"})
	})

	require.NoError(t, err)
	assert.Equal(t, int32(1), workingClient.sendCount.Load())
	assert.Equal(t, int32(0), constructorCalls.Load())
}

func TestRetrySend_HandlesNilClient(t *testing.T) {
	stream := newTestWorkflowStream(t, nil, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		return nil, fmt.Errorf("connection failed")
	})

	err := stream.retrySend(context.Background(), func(c dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient) error {
		return c.Send(&dispatchercontracts.SubscribeToWorkflowRunsRequest{WorkflowRunId: "test-workflow-run-id"})
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, errStreamNotConnected)
}

func TestRetrySend_ConcurrentSafety(t *testing.T) {
	workingClient := &mockSubscribeClient{
		sendErr:  nil,
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
		sendFn: func(req *dispatchercontracts.SubscribeToWorkflowRunsRequest) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		},
	}

	stream := newTestWorkflowStream(t, workingClient, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		return workingClient, nil
	})

	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			err := stream.retrySend(context.Background(), func(c dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient) error {
				return c.Send(&dispatchercontracts.SubscribeToWorkflowRunsRequest{WorkflowRunId: fmt.Sprintf("workflow-run-%d", id)})
			})
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()
	assert.Equal(t, int32(numGoroutines), workingClient.sendCount.Load())
}

func TestRetrySubscribe_SingleflightCoalescesConcurrentCalls(t *testing.T) {
	constructorCalls := atomic.Int32{}

	client := &mockSubscribeClient{
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}

	stream := newTestWorkflowStream(t, client, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
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
			err := stream.connectSync(context.Background())
			assert.NoError(t, err)
		}()
	}

	wg.Wait()
	assert.Equal(t, int32(1), constructorCalls.Load())
}

func TestRetrySubscribe_GenerationIncrements(t *testing.T) {
	client := &mockSubscribeClient{
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}

	stream := newTestWorkflowStream(t, client, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		return client, nil
	})

	_, gen0, _ := stream.snapshot()
	assert.Equal(t, uint64(0), gen0)

	require.NoError(t, stream.connectSync(context.Background()))

	_, gen1, _ := stream.snapshot()
	assert.Equal(t, uint64(1), gen1)

	require.NoError(t, stream.connectSync(context.Background()))

	_, gen2, _ := stream.snapshot()
	assert.Equal(t, uint64(2), gen2)
}

func TestGetClientSnapshot_ReturnsCurrentClient(t *testing.T) {
	client1 := &mockSubscribeClient{recvChan: make(chan *dispatchercontracts.WorkflowRunEvent)}

	stream := newTestWorkflowStream(t, client1, nil)

	got, gen, ok := stream.snapshot()
	assert.True(t, ok)
	assert.Equal(t, client1, got)
	assert.Equal(t, uint64(0), gen)
}

func TestRetrySubscribeSyncStopsAtStreamSyncMaxAttempts(t *testing.T) {
	constructorCalls := atomic.Int32{}

	stream := newTestWorkflowStream(t, nil, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		constructorCalls.Add(1)
		return nil, status.Error(codes.Unavailable, "still down")
	})

	err := stream.connectSync(context.Background())
	require.Error(t, err)
	assert.Equal(t, int32(retry.StreamSyncMaxAttempts), constructorCalls.Load())
}

func TestRetrySendStaleGenerationSkipsReconnect(t *testing.T) {
	constructorCalls := atomic.Int32{}

	workingClient := &mockSubscribeClient{recvChan: make(chan *dispatchercontracts.WorkflowRunEvent)}
	failingClient := &mockSubscribeClient{
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}

	stream := newTestWorkflowStream(t, failingClient, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		constructorCalls.Add(1)
		return workingClient, nil
	})

	failingClient.sendFn = func(req *dispatchercontracts.SubscribeToWorkflowRunsRequest) error {
		require.NoError(t, stream.installClient(workingClient))
		return status.Error(codes.Unavailable, "send failed")
	}

	require.NoError(t, stream.retrySend(context.Background(), func(c dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient) error {
		return c.Send(&dispatchercontracts.SubscribeToWorkflowRunsRequest{WorkflowRunId: "run-1"})
	}))
	assert.Equal(t, int32(0), constructorCalls.Load())
}

func TestReconnectingStreamConnectsAreCoalesced(t *testing.T) {
	releaseConstructor := make(chan struct{})
	var concurrentConnect atomic.Int32
	var maxConcurrent atomic.Int32

	client := &mockSubscribeClient{
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}

	stream := newTestWorkflowStream(t, client, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		current := concurrentConnect.Add(1)
		for {
			prev := maxConcurrent.Load()
			if current <= prev {
				break
			}
			if maxConcurrent.CompareAndSwap(prev, current) {
				break
			}
		}
		defer concurrentConnect.Add(-1)

		<-releaseConstructor
		return client, nil
	})

	syncDone := make(chan error, 1)
	backgroundDone := make(chan error, 1)

	go func() {
		syncDone <- stream.connectSync(context.Background())
	}()
	go func() {
		backgroundDone <- stream.connectOnce(context.Background())
	}()

	require.Eventually(t, func() bool {
		return concurrentConnect.Load() >= 1
	}, time.Second, 10*time.Millisecond)

	assert.Equal(t, int32(1), concurrentConnect.Load())
	assert.Equal(t, int32(1), maxConcurrent.Load())

	close(releaseConstructor)

	require.NoError(t, <-syncDone)
	require.NoError(t, <-backgroundDone)
}

func TestRetrySendShortCircuitsOnPermanentReconnectError(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		failingClient := &mockSubscribeClient{
			sendErr:  status.Error(codes.Unavailable, "send failed"),
			recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
		}

		constructorCalls := atomic.Int32{}
		stream := newTestWorkflowStream(t, failingClient, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			constructorCalls.Add(1)
			return nil, status.Error(codes.Unauthenticated, "auth failed")
		})

		err := stream.retrySend(context.Background(), func(c dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient) error {
			return c.Send(&dispatchercontracts.SubscribeToWorkflowRunsRequest{WorkflowRunId: "run-1"})
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not reconnect")
		assert.Less(t, failingClient.sendCount.Load(), int32(retry.StreamSyncMaxAttempts))
	})

	t.Run("closed listener", func(t *testing.T) {
		failingClient := &mockSubscribeClient{
			sendErr:  status.Error(codes.Unavailable, "send failed"),
			recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
		}

		stream := newTestWorkflowStream(t, failingClient, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
			return nil, fmt.Errorf("should not be called")
		})

		require.NoError(t, stream.Close())

		err := stream.retrySend(context.Background(), func(c dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient) error {
			return c.Send(&dispatchercontracts.SubscribeToWorkflowRunsRequest{WorkflowRunId: "run-1"})
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, errListenerClosed)
	})
}
