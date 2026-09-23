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

// maxQueryLabels bounds how many distinct statement names one bucket remembers.
// Further names fold into "other" so raw SQL cannot grow the map without limit.
const maxQueryLabels = 64

// LimitError is returned when a tenant or shared work waits MaxWait without getting a connection slot.
type LimitError struct {
	// Key is the gate bucket. Tenants use the tenant id string; shared work uses "shared".
	Key      string
	TenantID uuid.UUID
	Limit    int64
	Waited   time.Duration
	// Queries counts statements currently holding a slot in this bucket, keyed by sqlc name.
	Queries map[string]int
}

func (e *LimitError) Error() string {
	var msg string
	if e.Key == sharedKey {
		msg = fmt.Sprintf("fairpool-exhausted: shared work held the maximum of %d database connections for %s", e.Limit, e.Waited)
	} else {
		msg = fmt.Sprintf("fairpool-exhausted: tenant %s held the maximum of %d database connections for %s", e.TenantID, e.Limit, e.Waited)
	}

	if counts := formatQueryCounts(e.Queries); counts != "" {
		msg += " " + counts
	}

	return msg
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
	queries  map[string]int
	lastWarn time.Time
}

// slotHold is one checked-out connection. retag moves its count when a transaction
// runs a statement, so an exhaustion error names the statement rather than "begin".
type slotHold struct {
	g     *gate
	key   string
	slots *tenantSlots
	label string
	done  bool
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

// enter takes one slot for key and counts it under label. tenantID is recorded on
// LimitError for tenant buckets and is uuid.Nil for shared work. The timeout used
// while waiting is not returned; callers must run the query with the context they were given.
func (g *gate) enter(ctx context.Context, key, label string, tenantID uuid.UUID) (*slotHold, error) {
	slots := g.slots(key)

	if slots.sem.TryAcquire(1) {
		return g.take(key, label, slots), nil
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

		return nil, &LimitError{
			Key:      key,
			TenantID: tenantID,
			Limit:    g.limit,
			Waited:   waited,
			Queries:  g.snapshotQueries(slots),
		}
	}

	prometheus.TenantPoolGateWait.WithLabelValues(g.poolName, "acquired").Observe(waited.Seconds())
	hold := g.take(key, label, slots)
	g.warnLimited(key, slots, waited)

	telemetry.WithAttributes(span,
		telemetry.AttributeKV{Key: "tenant_id", Value: key},
		telemetry.AttributeKV{Key: "limit", Value: g.limit},
		telemetry.AttributeKV{Key: "waited_ms", Value: waited.Milliseconds()},
		telemetry.AttributeKV{Key: "outcome", Value: "acquired"},
	)

	return hold, nil
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

func (g *gate) take(key, label string, slots *tenantSlots) *slotHold {
	g.mu.Lock()
	defer g.mu.Unlock()

	slots.held++
	g.noteHeldLocked(key, slots)

	hold := &slotHold{g: g, key: key, slots: slots}
	hold.label = slots.addQueryLocked(label)

	return hold
}

func (h *slotHold) release() {
	if h == nil {
		return
	}

	h.once.Do(func() {
		h.g.mu.Lock()
		h.done = true
		h.slots.dropQueryLocked(h.label)
		h.slots.held--
		h.g.noteHeldLocked(h.key, h.slots)
		h.g.mu.Unlock()

		h.slots.sem.Release(1)
	})
}

func (h *slotHold) retag(label string) {
	if h == nil || label == "" {
		return
	}

	h.g.mu.Lock()
	defer h.g.mu.Unlock()

	if h.done || h.label == label {
		return
	}

	h.slots.dropQueryLocked(h.label)
	h.label = h.slots.addQueryLocked(label)
}

func (s *tenantSlots) addQueryLocked(label string) string {
	if s.queries == nil {
		s.queries = make(map[string]int)
	}
	if _, ok := s.queries[label]; !ok && len(s.queries) >= maxQueryLabels {
		label = "other"
	}
	s.queries[label]++

	return label
}

func (s *tenantSlots) dropQueryLocked(label string) {
	s.queries[label]--
	if s.queries[label] <= 0 {
		delete(s.queries, label)
	}
}

func (g *gate) snapshotQueries(slots *tenantSlots) map[string]int {
	g.mu.Lock()
	defer g.mu.Unlock()

	if len(slots.queries) == 0 {
		return nil
	}

	out := make(map[string]int, len(slots.queries))
	for name, n := range slots.queries {
		out[name] = n
	}

	return out
}

func (g *gate) noteHeldLocked(key string, slots *tenantSlots) {
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
