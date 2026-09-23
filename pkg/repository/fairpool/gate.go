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

// LimitError is returned when a tenant or shared work waits MaxWait without getting a connection slot.
type LimitError struct {
	// Key is the gate bucket. Tenants use the tenant id string; shared work uses "shared".
	Key      string
	TenantID uuid.UUID
	Limit    int64
	Waited   time.Duration
}

func (e *LimitError) Error() string {
	if e.Key == sharedKey {
		return fmt.Sprintf("shared work held the maximum of %d database connections for %s", e.Limit, e.Waited)
	}

	return fmt.Sprintf("tenant %s held the maximum of %d database connections for %s", e.TenantID, e.Limit, e.Waited)
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
		tenants:  make(map[string]*tenantSlots),
	}
}

// enter takes one slot for key. tenantID is recorded on LimitError for tenant buckets
// and is uuid.Nil for shared work. The timeout used while waiting is not returned;
// callers must run the query with the context they were given.
func (g *gate) enter(ctx context.Context, key string, tenantID uuid.UUID) (func(), error) {
	slots := g.slots(key)

	if slots.sem.TryAcquire(1) {
		g.addHeld(key, slots, 1)
		return g.release(key, slots), nil
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
		prometheus.TenantPoolGateRejections.WithLabelValues(g.poolName, key).Inc()
		g.warnLimited(key, slots, waited)

		telemetry.WithAttributes(span,
			telemetry.AttributeKV{Key: "tenant_id", Value: key},
			telemetry.AttributeKV{Key: "limit", Value: g.limit},
			telemetry.AttributeKV{Key: "waited_ms", Value: waited.Milliseconds()},
			telemetry.AttributeKV{Key: "outcome", Value: "rejected"},
		)

		return nil, &LimitError{Key: key, TenantID: tenantID, Limit: g.limit, Waited: waited}
	}

	prometheus.TenantPoolGateWait.WithLabelValues(g.poolName, "acquired").Observe(waited.Seconds())
	g.addHeld(key, slots, 1)
	g.warnLimited(key, slots, waited)

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant_id", Value: key},
		telemetry.AttributeKV{Key: "limit", Value: g.limit},
		telemetry.AttributeKV{Key: "waited_ms", Value: waited.Milliseconds()},
		telemetry.AttributeKV{Key: "outcome", Value: "acquired"},
	)

	return g.release(key, slots), nil
}

func (g *gate) slots(key string) *tenantSlots {
	g.mu.Lock()
	defer g.mu.Unlock()

	slots, ok := g.tenants[key]
	if !ok {
		slots = &tenantSlots{sem: semaphore.NewWeighted(g.limit)}
		g.tenants[key] = slots
	}

	return slots
}

func (g *gate) release(key string, slots *tenantSlots) func() {
	var once sync.Once

	return func() {
		once.Do(func() {
			slots.sem.Release(1)
			g.addHeld(key, slots, -1)
		})
	}
}

func (g *gate) addHeld(key string, slots *tenantSlots, delta int64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	slots.held += delta

	if slots.held <= 0 {
		slots.held = 0
		prometheus.TenantPoolHeldConns.DeleteLabelValues(g.poolName, key)
		return
	}

	prometheus.TenantPoolHeldConns.WithLabelValues(g.poolName, key).Set(float64(slots.held))
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
