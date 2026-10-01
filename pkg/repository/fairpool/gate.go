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
// It uses the same connection limit as any single tenant.
const sharedKey = "shared"

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
		return fmt.Sprintf("fairpool-exhausted: shared work held the maximum of %d database connections for %s", e.Limit, e.Waited)
	}

	return fmt.Sprintf("fairpool-exhausted: tenant %s held the maximum of %d database connections for %s", e.TenantID, e.Limit, e.Waited)
}

type gate struct {
	limit    int64
	maxWait  time.Duration
	poolName string
	l        *zerolog.Logger

	mu      sync.Mutex
	tenants map[string]*tenantSlots
}

type tenantSlots struct {
	sem      *semaphore.Weighted
	held     int64
	refs     int
	lastWarn time.Time
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
		limit:    limit,
		maxWait:  maxWait,
		poolName: poolName,
		l:        l,
		tenants:  make(map[string]*tenantSlots),
	}
}

// pin reserves a slot for key (waiting up to MaxWait), then calls open to check
// out the connection. The slot is held until the returned hold is released.
//
// The slot is taken before open on purpose. Checking out the connection first
// would park a pool connection while waiting on the tenant cap.
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
		prometheus.FairpoolWait.WithLabelValues(g.poolName, "acquired").Observe(waited.Seconds())
		g.warnLimited(key, slots, waited)
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
		prometheus.FairpoolWait.WithLabelValues(g.poolName, "canceled").Observe(waited.Seconds())
		return waited, ctx.Err()
	}

	return waited, g.reject(key, slots, tenantID, waited)
}

func (g *gate) reject(key string, slots *tenantSlots, tenantID uuid.UUID, waited time.Duration) error {
	prometheus.FairpoolWait.WithLabelValues(g.poolName, "rejected").Observe(waited.Seconds())
	prometheus.FairpoolRejections.WithLabelValues(g.poolName).Inc()
	g.warnLimited(key, slots, waited)

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

func (g *gate) warnLimited(key string, slots *tenantSlots, waited time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if time.Since(slots.lastWarn) < time.Minute {
		return
	}

	slots.lastWarn = time.Now()

	msg := "tenant waited for a database connection slot"
	if key == sharedKey {
		msg = "shared work waited for a database connection slot"
	}

	g.l.Warn().
		Str("tenant_id", key).
		Str("pool", g.poolName).
		Int64("limit", g.limit).
		Dur("waited", waited).
		Msg(msg)
}
