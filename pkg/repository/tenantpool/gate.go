package tenantpool

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

// LimitError is returned when a tenant waits MaxWait without getting a connection slot.
type LimitError struct {
	TenantID uuid.UUID
	Limit    int64
	Waited   time.Duration
}

func (e *LimitError) Error() string {
	return fmt.Sprintf("tenant %s held the maximum of %d database connections for %s", e.TenantID, e.Limit, e.Waited)
}

type gate struct {
	limit    int64
	maxWait  time.Duration
	poolName string
	l        *zerolog.Logger

	mu      sync.Mutex
	tenants map[uuid.UUID]*tenantSlots
}

type tenantSlots struct {
	sem      *semaphore.Weighted
	held     int64
	lastWarn time.Time
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
		tenants:  make(map[uuid.UUID]*tenantSlots),
	}
}

// enter takes one slot for tenantID. The timeout used while waiting is not returned;
// callers must run the query with the context they were given.
func (g *gate) enter(ctx context.Context, tenantID uuid.UUID) (func(), error) {
	slots := g.slots(tenantID)

	if slots.sem.TryAcquire(1) {
		g.addHeld(tenantID, slots, 1)
		return g.release(tenantID, slots), nil
	}

	start := time.Now()
	_, span := telemetry.NewSpan(ctx, "db.tenant-gate.wait")
	defer span.End()

	waitCtx, cancel := context.WithTimeout(ctx, g.maxWait)
	defer cancel()

	err := slots.sem.Acquire(waitCtx, 1)
	waited := time.Since(start)

	if err != nil {
		if ctx.Err() != nil {
			prometheus.TenantPoolGateWait.WithLabelValues(g.poolName, "canceled").Observe(waited.Seconds())
			return nil, ctx.Err()
		}

		prometheus.TenantPoolGateWait.WithLabelValues(g.poolName, "rejected").Observe(waited.Seconds())
		prometheus.TenantPoolGateRejections.WithLabelValues(g.poolName, tenantID.String()).Inc()
		g.warnLimited(tenantID, slots, waited)

		telemetry.WithAttributes(span,
			telemetry.AttributeKV{Key: "tenant_id", Value: tenantID},
			telemetry.AttributeKV{Key: "limit", Value: g.limit},
			telemetry.AttributeKV{Key: "waited_ms", Value: waited.Milliseconds()},
			telemetry.AttributeKV{Key: "outcome", Value: "rejected"},
		)

		return nil, &LimitError{TenantID: tenantID, Limit: g.limit, Waited: waited}
	}

	prometheus.TenantPoolGateWait.WithLabelValues(g.poolName, "acquired").Observe(waited.Seconds())
	g.addHeld(tenantID, slots, 1)
	g.warnLimited(tenantID, slots, waited)

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant_id", Value: tenantID},
		telemetry.AttributeKV{Key: "limit", Value: g.limit},
		telemetry.AttributeKV{Key: "waited_ms", Value: waited.Milliseconds()},
		telemetry.AttributeKV{Key: "outcome", Value: "acquired"},
	)

	return g.release(tenantID, slots), nil
}

func (g *gate) slots(tenantID uuid.UUID) *tenantSlots {
	g.mu.Lock()
	defer g.mu.Unlock()

	slots, ok := g.tenants[tenantID]
	if !ok {
		slots = &tenantSlots{sem: semaphore.NewWeighted(g.limit)}
		g.tenants[tenantID] = slots
	}

	return slots
}

func (g *gate) release(tenantID uuid.UUID, slots *tenantSlots) func() {
	var once sync.Once

	return func() {
		once.Do(func() {
			slots.sem.Release(1)
			g.addHeld(tenantID, slots, -1)
		})
	}
}

func (g *gate) addHeld(tenantID uuid.UUID, slots *tenantSlots, delta int64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	slots.held += delta

	if slots.held <= 0 {
		slots.held = 0
		prometheus.TenantPoolHeldConns.DeleteLabelValues(g.poolName, tenantID.String())
		return
	}

	prometheus.TenantPoolHeldConns.WithLabelValues(g.poolName, tenantID.String()).Set(float64(slots.held))
}

func (g *gate) warnLimited(tenantID uuid.UUID, slots *tenantSlots, waited time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if time.Since(slots.lastWarn) < time.Minute {
		return
	}

	slots.lastWarn = time.Now()

	g.l.Warn().
		Str("tenant_id", tenantID.String()).
		Str("pool", g.poolName).
		Int64("limit", g.limit).
		Dur("waited", waited).
		Msg("tenant waited for a database connection slot")
}
