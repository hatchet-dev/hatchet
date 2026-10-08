package concurrency

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Approximate heap cost of each part of the index, measured with BenchmarkKeyMemory and
// TestIndexMemoryFootprint. Used only for the estimated bytes gauge.
const (
	keyBytes         = 145
	runningSlotBytes = 32
	queuedKeyBytes   = 400
	queuedSlotBytes  = 88
)

// residentSize is what one or more sub-queues hold in memory. queuedKeys counts the sub-queues
// with a queued index, which has a fixed cost of its own.
type residentSize struct {
	running    int64
	queued     int64
	queuedKeys int64
}

func (r residentSize) plus(o residentSize) residentSize {
	return residentSize{
		running:    r.running + o.running,
		queued:     r.queued + o.queued,
		queuedKeys: r.queuedKeys + o.queuedKeys,
	}
}

func (r residentSize) minus(o residentSize) residentSize {
	return residentSize{
		running:    r.running - o.running,
		queued:     r.queued - o.queued,
		queuedKeys: r.queuedKeys - o.queuedKeys,
	}
}

// residentCounts is a strategy's residentSize, readable by the gauge callback while the strategy
// runs.
type residentCounts struct {
	running    atomic.Int64
	queued     atomic.Int64
	queuedKeys atomic.Int64
}

func (r *residentCounts) set(size residentSize) {
	r.running.Store(size.running)
	r.queued.Store(size.queued)
	r.queuedKeys.Store(size.queuedKeys)
}

func (r *residentCounts) add(delta residentSize) {
	r.running.Add(delta.running)
	r.queued.Add(delta.queued)
	r.queuedKeys.Add(delta.queuedKeys)
}

func (r *residentCounts) load() residentSize {
	return residentSize{
		running:    r.running.Load(),
		queued:     r.queued.Load(),
		queuedKeys: r.queuedKeys.Load(),
	}
}

// countResident sums what every sub-queue holds.
func (c *ConcurrencyStrategy) countResident() residentSize {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var size residentSize

	for _, sq := range c.subQueues {
		size = size.plus(sq.size())
	}

	return size
}

var (
	trackedMu  sync.Mutex
	tracked    = make(map[*ConcurrencyStrategy]struct{})
	gaugesOnce sync.Once
)

// trackStrategy includes the strategy in the index gauges until ctx is done.
func trackStrategy(ctx context.Context, c *ConcurrencyStrategy) {
	gaugesOnce.Do(func() { registerGauges(c.l) })

	trackedMu.Lock()
	tracked[c] = struct{}{}
	trackedMu.Unlock()

	context.AfterFunc(ctx, func() {
		trackedMu.Lock()
		delete(tracked, c)
		trackedMu.Unlock()
	})
}

// registerGauges reports the size of every in-memory concurrency index in this process.
func registerGauges(l *zerolog.Logger) {
	meter := otel.Meter("hatchet.run/metrics")

	keys, err := meter.Int64ObservableGauge(
		"hatchet.scheduler.concurrency_index.keys",
		metric.WithDescription("Concurrency keys held by the in-memory concurrency indexes"),
		metric.WithUnit("{key}"),
	)
	if err != nil {
		l.Warn().Err(err).Msg("cannot create concurrency index keys gauge")
		return
	}

	slots, err := meter.Int64ObservableGauge(
		"hatchet.scheduler.concurrency_index.slots",
		metric.WithDescription("Concurrency slots held by the in-memory concurrency indexes"),
		metric.WithUnit("{slot}"),
	)
	if err != nil {
		l.Warn().Err(err).Msg("cannot create concurrency index slots gauge")
		return
	}

	bytes, err := meter.Int64ObservableGauge(
		"hatchet.scheduler.concurrency_index.estimated_bytes",
		metric.WithDescription("Estimated heap held by the in-memory concurrency indexes"),
		metric.WithUnit("By"),
	)
	if err != nil {
		l.Warn().Err(err).Msg("cannot create concurrency index bytes gauge")
		return
	}

	running := metric.WithAttributes(attribute.String("state", "running"))
	queued := metric.WithAttributes(attribute.String("state", "queued"))

	_, err = meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		var totalKeys int64
		var total residentSize

		trackedMu.Lock()
		for c := range tracked {
			c.mu.RLock()
			totalKeys += int64(len(c.subQueues))
			c.mu.RUnlock()

			total = total.plus(c.resident.load())
		}
		trackedMu.Unlock()

		o.ObserveInt64(keys, totalKeys)
		o.ObserveInt64(slots, total.running, running)
		o.ObserveInt64(slots, total.queued, queued)
		o.ObserveInt64(bytes, estimatedBytes(totalKeys, total))

		return nil
	}, keys, slots, bytes)
	if err != nil {
		l.Warn().Err(err).Msg("cannot register concurrency index gauge callback")
	}
}

func estimatedBytes(keys int64, size residentSize) int64 {
	return keys*keyBytes +
		size.running*runningSlotBytes +
		size.queuedKeys*queuedKeyBytes +
		size.queued*queuedSlotBytes
}
