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
// cannot starve another tenant's subscription. Handlers were written against
// the rabbitmq backend, which runs every delivery on its own goroutine, and
// rely on concurrency: scheduler handlers can block on database reads, and the
// dispatcher's buffered tenant reader only batches messages that arrive while
// earlier ones wait for a flush.
//
// The bounds are a safety limit rather than a throttle: about 2-3x the peak
// concurrency the rabbitmq backend reached under synthetic load with 50ms
// handlers (3000 msgs/s on a scheduler partition, 1000 msgs/s on a tenant
// stream). A scheduler-partition subscription multiplexes every tenant of the
// partition, so it gets the larger bound; its handlers' database work is
// already limited to one replenish per tenant at a time.
var maxConcurrentHandlers = map[msgqueue.TopicKind]int{
	msgqueue.TopicKindSchedulerPartition: 512,
	msgqueue.TopicKindTenantStream:       128,
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

// acquire blocks until a slot is free. It reports false, holding no slot, once
// the pool is closed.
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

// release returns a slot taken by acquire without running anything on it.
func (h *handlerPool) release() {
	<-h.slots
}

// run calls fn on a new goroutine that owns a slot taken by acquire, and
// releases the slot when fn returns.
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

// close makes acquire fail and waits for every in-flight call to return, so no
// call starts after close returns. It must not be called from a call running
// on the pool, which would wait for itself.
func (h *handlerPool) close() {
	h.closeOnce.Do(func() {
		close(h.closed)

		for range cap(h.slots) {
			h.slots <- struct{}{}
		}
	})
}
