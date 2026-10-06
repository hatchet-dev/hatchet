package nats

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	prommetrics "github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
)

// maxConcurrentHandlers bounds how many handler calls one subscription runs at
// once. The bound is per subscription so that one tenant's slow handlers
// cannot starve another tenant's subscription. Handlers need concurrent
// delivery: scheduler handlers can block on database reads, which would
// otherwise stall every later delivery on the subscription, and the
// dispatcher's buffered tenant reader only batches messages that arrive while
// earlier ones wait for a flush.
var maxConcurrentHandlers = map[msgqueue.TopicKind]int{
	msgqueue.TopicKindSchedulerPartition: 128,
	msgqueue.TopicKindTenantStream:       32,
}

const defaultMaxConcurrentHandlers = 32

func handlerLimit(kind msgqueue.TopicKind) int {
	if limit, ok := maxConcurrentHandlers[kind]; ok {
		return limit
	}

	return defaultMaxConcurrentHandlers
}

// handlerPool runs a subscription's handler calls on their own goroutines, at
// most cap(slots) at a time. A goroutine per call rather than long-lived
// workers keeps idle subscriptions free.
type handlerPool struct {
	slots     chan struct{}
	closed    chan struct{}
	closeOnce sync.Once

	inFlight prometheus.Gauge
	full     prometheus.Counter
	wait     prometheus.Observer
}

func newHandlerPool(limit int, kind msgqueue.TopicKind) *handlerPool {
	return &handlerPool{
		slots:    make(chan struct{}, limit),
		closed:   make(chan struct{}),
		inFlight: prommetrics.PubSubHandlersInFlight.WithLabelValues("nats", string(kind)),
		full:     prommetrics.PubSubHandlerPoolFull.WithLabelValues("nats", string(kind)),
		wait:     prommetrics.PubSubHandlerSlotWait.WithLabelValues("nats", string(kind)),
	}
}

// A false result holds no slot, so the caller must not release or run.
func (h *handlerPool) acquire() bool {
	select {
	case <-h.closed:
		return false
	default:
	}

	select {
	case h.slots <- struct{}{}:
		h.wait.Observe(0)
		return true
	default:
	}

	h.full.Inc()
	start := time.Now()

	select {
	case h.slots <- struct{}{}:
		h.wait.Observe(time.Since(start).Seconds())
		return true
	case <-h.closed:
		return false
	}
}

func (h *handlerPool) release() {
	<-h.slots
}

// run takes ownership of the slot from a successful acquire.
func (h *handlerPool) run(fn func()) {
	h.inFlight.Inc()

	go func() {
		defer func() {
			h.inFlight.Dec()
			h.release()
		}()

		fn()
	}()
}

// close waits without a deadline so that no call starts or is still running
// once it returns; a bounded wait would let a call keep running against state
// its caller is tearing down. It must not be called from a call running on the
// pool, which would wait for itself.
func (h *handlerPool) close() {
	h.closeOnce.Do(func() {
		close(h.closed)

		for range cap(h.slots) {
			h.slots <- struct{}{}
		}
	})
}
