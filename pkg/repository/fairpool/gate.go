package fairpool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/sync/semaphore"

	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

// sharedKey is the gate bucket for engine work that is not tied to one tenant.
// It is gated by its own limit, separate from the per-tenant one.
const sharedKey = "shared"

// RetryAfter is how long callers are told to wait before retrying a request that a LimitError
// rejected. It is a hint: the cap clears as soon as an in-flight checkout is released.
const RetryAfter = time.Second

// LimitError is returned when a tenant or shared work waits MaxWait without a free slot.
type LimitError struct {
	// Key is the gate bucket. Tenants use the tenant id string; shared work uses "shared".
	Key      string
	TenantID uuid.UUID
	Limit    int64
	Waited   time.Duration
}

func (e *LimitError) Error() string {
	if e.Key == sharedKey {
		return fmt.Sprintf("fairpool-exhausted: shared work had the maximum of %d database checkouts in flight for %s", e.Limit, e.Waited)
	}

	return fmt.Sprintf("fairpool-exhausted: tenant %s had the maximum of %d database checkouts in flight for %s", e.TenantID, e.Limit, e.Waited)
}

// bucketLabel is the bounded metrics label for key: "shared" or "tenant".
func bucketLabel(key string) string {
	if key == sharedKey {
		return sharedKey
	}

	return "tenant"
}

const (
	// logInterval is the minimum time between two log lines of one kind for one bucket.
	logInterval = time.Minute

	// logBurst caps the lines of one kind that a gate emits per logInterval across all of its
	// buckets, so a database-wide slowdown that puts many tenants at their cap stays bounded.
	logBurst = 20
)

type gate struct {
	limit    int64
	maxWait  time.Duration
	poolName string

	// Waits and rejections are sampled separately so frequent waits cannot use up the burst that
	// rejections need. Within a gate, the per-bucket interval keeps one noisy bucket visible
	// (rejections carry a count) and the sampled burst bounds the total volume.
	waitLog   zerolog.Logger
	rejectLog zerolog.Logger

	mu      sync.Mutex
	tenants map[string]*tenantSlots
}

type tenantSlots struct {
	sem  *semaphore.Weighted
	held int64
	refs int

	lastWaitLog   time.Time
	lastRejectLog time.Time

	// rejected counts rejections since the last rejection line was logged. A bucket is only
	// evicted once it is idle, so the count survives for as long as the bucket keeps being rejected.
	rejected int64
}

// slotHold is one checked-out connection counted against a bucket.
type slotHold struct {
	g     *gate
	key   string
	slots *tenantSlots
	once  sync.Once
}

func newGate(limit int64, maxWait time.Duration, poolName string, l *zerolog.Logger) *gate {
	if l == nil {
		nop := zerolog.Nop()
		l = &nop
	}

	return &gate{
		limit:     limit,
		maxWait:   maxWait,
		poolName:  poolName,
		waitLog:   l.Sample(&zerolog.BurstSampler{Burst: logBurst, Period: logInterval}),
		rejectLog: l.Sample(&zerolog.BurstSampler{Burst: logBurst, Period: logInterval}),
		tenants:   make(map[string]*tenantSlots),
	}
}

// pin reserves a slot for key (waiting up to MaxWait), then calls open to check
// out the connection. The slot is held until the returned hold is released.
//
// The cap bounds checkouts in flight, so the slot is taken before open on purpose.
// A waiter therefore never holds a pool connection, and an over-cap bucket cannot
// fill the pool with parked checkouts that other buckets need.
func (g *gate) pin(ctx context.Context, key string, tenantID uuid.UUID, open func() error) (*slotHold, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	slots := g.borrow(key)
	defer g.releaseBorrow(key, slots)

	waited, err := g.reserve(ctx, key, slots, tenantID)
	if err != nil {
		return nil, err
	}

	if err := open(); err != nil {
		slots.sem.Release(1)
		return nil, err
	}

	if waited > 0 {
		prometheus.FairpoolWait.WithLabelValues(g.poolName, bucketLabel(key), "acquired").Observe(waited.Seconds())
		g.logWaited(key, slots, waited)
	}

	return g.take(key, slots), nil
}

// reserve acquires one slot, waiting up to MaxWait. The caller must take or release it.
func (g *gate) reserve(ctx context.Context, key string, slots *tenantSlots, tenantID uuid.UUID) (time.Duration, error) {
	if slots.sem.TryAcquire(1) {
		return 0, nil
	}

	start := time.Now()
	waitCtx, cancel := context.WithTimeout(ctx, g.maxWait)
	defer cancel()

	_, span := telemetry.NewSpan(ctx, "db.fairpool.wait")
	err := slots.sem.Acquire(waitCtx, 1)
	waited := time.Since(start)
	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant_id", Value: key},
		telemetry.AttributeKV{Key: "limit", Value: g.limit},
		telemetry.AttributeKV{Key: "waited_ms", Value: waited.Milliseconds()},
	)
	span.End()

	if err == nil {
		return waited, nil
	}

	if ctx.Err() != nil {
		prometheus.FairpoolWait.WithLabelValues(g.poolName, bucketLabel(key), "canceled").Observe(waited.Seconds())
		return waited, ctx.Err()
	}

	return waited, g.reject(key, slots, tenantID, waited)
}

func (g *gate) reject(key string, slots *tenantSlots, tenantID uuid.UUID, waited time.Duration) error {
	prometheus.FairpoolWait.WithLabelValues(g.poolName, bucketLabel(key), "rejected").Observe(waited.Seconds())
	prometheus.FairpoolRejections.WithLabelValues(g.poolName, bucketLabel(key)).Inc()
	g.logRejected(key, slots, waited)

	return &LimitError{
		Key:      key,
		TenantID: tenantID,
		Limit:    g.limit,
		Waited:   waited,
	}
}

func (g *gate) borrow(key string) *tenantSlots {
	g.mu.Lock()
	defer g.mu.Unlock()

	slots, ok := g.tenants[key]
	if !ok {
		slots = &tenantSlots{sem: semaphore.NewWeighted(g.limit)}
		g.tenants[key] = slots
	}

	slots.refs++

	return slots
}

func (g *gate) releaseBorrow(key string, slots *tenantSlots) {
	g.mu.Lock()
	defer g.mu.Unlock()

	slots.refs--
	g.evictLocked(key, slots)
}

func (g *gate) take(key string, slots *tenantSlots) *slotHold {
	g.mu.Lock()
	defer g.mu.Unlock()

	slots.held++
	slots.refs++
	g.noteHeldLocked(key, slots)

	return &slotHold{g: g, key: key, slots: slots}
}

func (h *slotHold) release() {
	if h == nil {
		return
	}

	h.once.Do(func() {
		h.g.mu.Lock()
		h.slots.held--
		h.slots.refs--
		h.g.noteHeldLocked(h.key, h.slots)
		h.g.evictLocked(h.key, h.slots)
		h.g.mu.Unlock()

		h.slots.sem.Release(1)
	})
}

func (g *gate) evictLocked(key string, slots *tenantSlots) {
	if slots.held <= 0 && slots.refs <= 0 {
		delete(g.tenants, key)
	}
}

func (g *gate) noteHeldLocked(key string, slots *tenantSlots) {
	if slots.held <= 0 {
		slots.held = 0
		prometheus.FairpoolHeldConns.DeleteLabelValues(g.poolName, key)
		return
	}

	prometheus.FairpoolHeldConns.WithLabelValues(g.poolName, key).Set(float64(slots.held))
}

// logWaited logs, at most once per interval per bucket, that a request got its slot only after
// waiting. It is INFO: the request succeeded, so the line is a signal that a bucket is near its cap.
func (g *gate) logWaited(key string, slots *tenantSlots, waited time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if time.Since(slots.lastWaitLog) < logInterval {
		return
	}

	// a nil event means the gate-wide burst is spent; leave the timestamp so the next wait retries
	e := g.waitLog.Info()
	if e == nil {
		return
	}

	slots.lastWaitLog = time.Now()

	g.fields(e, key, waited).Msg(subject(key) + " waited for a connection slot")
}

// logRejected counts a rejection and logs, at most once per interval per bucket, how many
// rejections the bucket has had since the previous line. The count is cleared only when a line is
// emitted, so rejections the rate limit or the gate-wide burst suppressed roll into the next line.
func (g *gate) logRejected(key string, slots *tenantSlots, waited time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()

	slots.rejected++

	if time.Since(slots.lastRejectLog) < logInterval {
		return
	}

	e := g.rejectLog.Warn()
	if e == nil {
		return
	}

	slots.lastRejectLog = time.Now()

	g.fields(e, key, waited).Int64("rejected", slots.rejected).Msg(subject(key) + " checkout rejected at cap")

	slots.rejected = 0
}

func (g *gate) fields(e *zerolog.Event, key string, waited time.Duration) *zerolog.Event {
	return e.
		Str("tenant_id", key).
		Str("pool", g.poolName).
		Str("bucket", bucketLabel(key)).
		Int64("limit", g.limit).
		Dur("waited", waited)
}

func subject(key string) string {
	if key == sharedKey {
		return "fairpool: shared work"
	}

	return "fairpool: tenant"
}
