package fairpool

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
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
	// limit is the default per-bucket cap. Zero means a bucket without an override is not capped:
	// its limit is poolSize, which binds only when the pool itself has no free connection.
	limit    int64
	poolSize int64

	// overrides replaces limit for the tenants it lists. Checks read it without locking.
	overrides atomic.Pointer[map[uuid.UUID]int64]

	// budgetRate is the default budget in held-seconds per second; zero means no budget. budgets
	// replaces it for the tenants it lists.
	budgetRate float64
	budgets    atomic.Pointer[map[uuid.UUID]float64]

	maxWait  time.Duration
	poolName string

	// Waits and rejections are sampled separately so frequent waits cannot use up the burst that
	// rejections need. Within a gate, the per-bucket interval keeps one noisy bucket visible
	// (rejections carry a count) and the sampled burst bounds the total volume.
	l             *zerolog.Logger
	waitSampler   *zerolog.BurstSampler
	rejectSampler *zerolog.BurstSampler
	leakSampler   *zerolog.BurstSampler

	mu      sync.Mutex
	tenants map[string]*tenantSlots

	// accounts outlive buckets: a key's account counts its checkouts across all of its buckets
	// (including a replaced bucket that is still draining), and keeps its held-time series and its
	// budget until the key has been idle for accountIdleTimeout.
	accounts  map[string]*account
	lastSweep time.Time
}

// tenantSlots is one bucket. A bucket keeps the limit it was created with: when a tenant's limit
// changes, the next checkout replaces the bucket in the gate's map, and holds on the old bucket
// release into it.
type tenantSlots struct {
	sem      *semaphore.Weighted
	limit    int64
	tenantID uuid.UUID
	held     int64
	refs     int

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
	start time.Time
	once  sync.Once
}

func newGate(limit, poolSize int64, budgetRate float64, maxWait time.Duration, poolName string, l *zerolog.Logger) *gate {
	if l == nil {
		nop := zerolog.Nop()
		l = &nop
	}

	g := &gate{
		limit:         limit,
		poolSize:      poolSize,
		budgetRate:    budgetRate,
		maxWait:       maxWait,
		poolName:      poolName,
		l:             l,
		waitSampler:   &zerolog.BurstSampler{Burst: logBurst, Period: logInterval},
		rejectSampler: &zerolog.BurstSampler{Burst: logBurst, Period: logInterval},
		leakSampler:   &zerolog.BurstSampler{Burst: logBurst, Period: logInterval},
		tenants:       make(map[string]*tenantSlots),
		accounts:      make(map[string]*account),
	}

	g.overrides.Store(&map[uuid.UUID]int64{})
	g.budgets.Store(&map[uuid.UUID]float64{})

	return g
}

// limitFor is the limit for tenantID's bucket.
func (g *gate) limitFor(tenantID uuid.UUID) int64 {
	if limit, ok := (*g.overrides.Load())[tenantID]; ok {
		return limit
	}

	if g.limit == 0 {
		return g.poolSize
	}

	return g.limit
}

// budgetFor is tenantID's budget in held-seconds per second. Zero means no budget.
func (g *gate) budgetFor(tenantID uuid.UUID) float64 {
	if rate, ok := (*g.budgets.Load())[tenantID]; ok {
		return rate
	}

	return g.budgetRate
}

// pin reserves a slot for key, then calls open to check out the connection. The slot is held
// until the returned hold is released. A nil gate (an Ungated pool) only calls open.
//
// A checkout never waits for the tenant's budget; it only debits it on release (see AwaitBudget).
// A capped bucket waits up to MaxWait for a slot. An uncapped
// bucket's limit is the pool size, so it waits for a slot only when the pool has no free
// connection either, and then for as long as the pool would: until ctx ends.
//
// The cap bounds checkouts in flight, so the slot is taken before open on purpose.
// A waiter therefore never holds a pool connection, and an over-cap bucket cannot
// fill the pool with parked checkouts that other buckets need.
func (g *gate) pin(ctx context.Context, key string, tenantID uuid.UUID, open func() error) (*slotHold, error) {
	if g == nil {
		return nil, open()
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	slots := g.borrow(key, tenantID, g.limitFor(tenantID))
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

	waitCtx, cancel := ctx, context.CancelFunc(func() {})
	if slots.limit < g.poolSize {
		waitCtx, cancel = context.WithTimeout(ctx, g.maxWait)
	}
	defer cancel()

	_, span := telemetry.NewSpan(ctx, "db.fairpool.wait")
	err := slots.sem.Acquire(waitCtx, 1)
	waited := time.Since(start)
	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant_id", Value: key},
		telemetry.AttributeKV{Key: "limit", Value: slots.limit},
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
		Limit:    slots.limit,
		Waited:   waited,
	}
}

func (g *gate) borrow(key string, tenantID uuid.UUID, limit int64) *tenantSlots {
	g.mu.Lock()
	defer g.mu.Unlock()

	slots, ok := g.tenants[key]
	if !ok || slots.limit != limit {
		slots = &tenantSlots{sem: semaphore.NewWeighted(limit), limit: limit, tenantID: tenantID}
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

	now := time.Now()

	slots.held++
	slots.refs++
	g.noteHeldLocked(g.accountLocked(key, now), key, 1)

	return &slotHold{g: g, key: key, slots: slots, start: now}
}

// release returns the slot and records how long it was held, on the key's held-seconds series and
// as a debit of its budget, whatever kind of work the checkout was.
func (h *slotHold) release() {
	if h == nil {
		return
	}

	h.once.Do(func() {
		now := time.Now()
		held := now.Sub(h.start).Seconds()
		rate := h.g.budgetFor(h.slots.tenantID)

		h.g.mu.Lock()
		h.slots.held--
		h.slots.refs--
		h.g.evictLocked(h.key, h.slots)
		acct := h.g.accountLocked(h.key, now)
		h.g.noteHeldLocked(acct, h.key, -1)
		acct.debit(held, rate, now)
		h.g.sweepLocked(now)
		h.g.mu.Unlock()

		acct.heldSeconds.Add(held)
		h.slots.sem.Release(1)
	})
}

func (g *gate) evictLocked(key string, slots *tenantSlots) {
	if slots.held <= 0 && slots.refs <= 0 && g.tenants[key] == slots {
		delete(g.tenants, key)
	}
}

// noteHeldLocked adds delta to the key's checkouts in flight and exports the count.
func (g *gate) noteHeldLocked(acct *account, key string, delta int64) {
	acct.inFlight += delta

	if acct.inFlight <= 0 {
		acct.inFlight = 0
		prometheus.FairpoolHeldConns.DeleteLabelValues(g.poolName, key)
		return
	}

	prometheus.FairpoolHeldConns.WithLabelValues(g.poolName, key).Set(float64(acct.inFlight))
}

// logWaited logs, at most once per interval per bucket, that a request got its slot only after
// waiting. It is INFO: the request succeeded, so the line is a signal that a bucket is near its cap.
func (g *gate) logWaited(key string, slots *tenantSlots, waited time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if time.Since(slots.lastWaitLog) < logInterval {
		return
	}

	wl := g.l.Sample(g.waitSampler)

	// a nil event means the gate-wide burst is spent; leave the timestamp so the next wait retries
	e := wl.Info()
	if e == nil {
		return
	}

	slots.lastWaitLog = time.Now()

	g.fields(e, key, slots, waited).Msg(subject(key) + " waited for a connection slot")
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

	rl := g.l.Sample(g.rejectSampler)

	e := rl.Warn()
	if e == nil {
		return
	}

	slots.lastRejectLog = time.Now()

	g.fields(e, key, slots, waited).Int64("rejected", slots.rejected).Msg(subject(key) + " checkout rejected at cap")

	slots.rejected = 0
}

func (g *gate) fields(e *zerolog.Event, key string, slots *tenantSlots, waited time.Duration) *zerolog.Event {
	return e.
		Str("tenant_id", key).
		Str("pool", g.poolName).
		Str("bucket", bucketLabel(key)).
		Int64("limit", slots.limit).
		Dur("waited_ms", waited) // zerolog writes durations in milliseconds
}

func subject(key string) string {
	if key == sharedKey {
		return "fairpool: shared work"
	}

	return "fairpool: tenant"
}

// leaked logs a wrapper that was garbage-collected without being closed, which is a caller bug.
func (h *slotHold) leaked(kind string) {
	if h == nil {
		return
	}

	ll := h.g.l.Sample(h.g.leakSampler)

	ll.Warn().
		Str("tenant_id", h.key).
		Str("pool", h.g.poolName).
		Str("bucket", bucketLabel(h.key)).
		Str("handle", kind).
		Msg(subject(h.key) + " " + kind + " was garbage-collected without being closed")
}
