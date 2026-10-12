package nats

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	prommetrics "github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
)

func metricValue(t *testing.T, m prometheus.Metric) *dto.Metric {
	t.Helper()

	out := &dto.Metric{}
	require.NoError(t, m.Write(out))

	return out
}

func returnsWithin(d time.Duration, fn func()) bool {
	done := make(chan struct{})

	go func() {
		fn()
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-time.After(d):
		return false
	}
}

func TestHandlerPoolBoundsConcurrency(t *testing.T) {
	const limit = 3

	// metrics are process-global, so each run gets its own label
	kind := msgqueue.TopicKind("test-bounds-" + uuid.NewString())
	pool := newHandlerPool(limit, kind)

	const calls = 10

	unblock := make(chan struct{})
	var mu sync.Mutex
	var running, maxRunning int
	var finished atomic.Int64

	go func() {
		for range calls {
			if !pool.acquire() {
				return
			}

			pool.run(func() {
				mu.Lock()
				running++
				maxRunning = max(maxRunning, running)
				mu.Unlock()

				<-unblock

				mu.Lock()
				running--
				mu.Unlock()
				finished.Add(1)
			})
		}
	}()

	runningNow := func() int {
		mu.Lock()
		defer mu.Unlock()
		return running
	}

	require.Eventually(t, func() bool { return runningNow() == limit }, time.Second, time.Millisecond)

	// the producer is now blocked in acquire; nothing beyond the limit starts
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, limit, runningNow())
	assert.EqualValues(t, limit, metricValue(t, prommetrics.PubSubHandlersInFlight.WithLabelValues("nats", string(kind))).GetGauge().GetValue())
	assert.EqualValues(t, 1, metricValue(t, prommetrics.PubSubHandlerPoolFull.WithLabelValues("nats", string(kind))).GetCounter().GetValue())

	close(unblock)
	require.Eventually(t, func() bool { return finished.Load() == calls }, time.Second, time.Millisecond)

	pool.close()

	mu.Lock()
	assert.Equal(t, limit, maxRunning)
	mu.Unlock()
	assert.EqualValues(t, 0, metricValue(t, prommetrics.PubSubHandlersInFlight.WithLabelValues("nats", string(kind))).GetGauge().GetValue())

	wait := metricValue(t, prommetrics.PubSubHandlerSlotWait.WithLabelValues("nats", string(kind)).(prometheus.Metric)).GetHistogram()
	assert.EqualValues(t, calls, wait.GetSampleCount(), "every acquire observes its wait")
	assert.Greater(t, wait.GetSampleSum(), 0.0)
}

func TestHandlerPoolCloseWaitsForRunningCalls(t *testing.T) {
	pool := newHandlerPool(2, msgqueue.TopicKind("test-close-waits"))

	unblock := make(chan struct{})
	finished := make(chan struct{})

	require.True(t, pool.acquire())
	pool.run(func() {
		<-unblock
		close(finished)
	})

	assert.False(t, returnsWithin(100*time.Millisecond, pool.close), "close must wait for the running call")

	close(unblock)

	assert.True(t, returnsWithin(time.Second, pool.close))

	select {
	case <-finished:
	default:
		t.Fatal("close returned before the running call finished")
	}
}

func TestHandlerPoolCloseUnblocksWaitingAcquire(t *testing.T) {
	pool := newHandlerPool(1, msgqueue.TopicKind("test-close-unblocks"))

	unblock := make(chan struct{})

	require.True(t, pool.acquire())
	pool.run(func() { <-unblock })

	acquired := make(chan bool, 1)
	go func() { acquired <- pool.acquire() }()

	// the pool is full, so the acquire waits
	select {
	case <-acquired:
		t.Fatal("acquire returned while the pool was full")
	case <-time.After(50 * time.Millisecond):
	}

	closeDone := make(chan struct{})
	go func() {
		pool.close()
		close(closeDone)
	}()

	select {
	case ok := <-acquired:
		assert.False(t, ok, "a waiting acquire fails once the pool is closed")
	case <-time.After(time.Second):
		t.Fatal("close did not unblock the waiting acquire")
	}

	close(unblock)
	<-closeDone
}

func TestHandlerPoolAcquireAfterClose(t *testing.T) {
	pool := newHandlerPool(4, msgqueue.TopicKind("test-after-close"))

	pool.close()
	pool.close()

	assert.False(t, pool.acquire())
}

func TestHandlerPoolReleaseFreesSlot(t *testing.T) {
	pool := newHandlerPool(1, msgqueue.TopicKind("test-release"))

	require.True(t, pool.acquire())
	pool.release()

	var acquired atomic.Bool
	require.True(t, returnsWithin(time.Second, func() { acquired.Store(pool.acquire()) }))
	require.True(t, acquired.Load())

	pool.release()
	pool.close()
}

func TestHandlerLimit(t *testing.T) {
	assert.Equal(t, maxConcurrentHandlers[msgqueue.TopicKindSchedulerPartition], handlerLimit(msgqueue.TopicKindSchedulerPartition))
	assert.Equal(t, maxConcurrentHandlers[msgqueue.TopicKindTenantStream], handlerLimit(msgqueue.TopicKindTenantStream))
	assert.Equal(t, defaultMaxConcurrentHandlers, handlerLimit(msgqueue.TopicKind("other")))
}
