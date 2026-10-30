package dispatcher

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/sync/semaphore"

	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
)

const (
	minDurableTaskStreamBudgetBytes int64 = 16 << 20

	// approximate cost of a handler goroutine, so small requests are bounded by count too
	durableTaskRequestOverheadBytes int64 = 16 << 10
)

type tenantBudget struct {
	sem  *semaphore.Weighted
	refs int
}

type durableTaskBudgets struct {
	streamBytes int64
	waitLog     zerolog.Logger

	mu      sync.Mutex
	tenants map[uuid.UUID]*tenantBudget
}

// A stream fits four of the largest requests the server accepts, and a tenant eight streams.
func newDurableTaskBudgets(l *zerolog.Logger, maxMsgSize int) *durableTaskBudgets {
	return &durableTaskBudgets{
		streamBytes: max(minDurableTaskStreamBudgetBytes, 4*int64(maxMsgSize)),
		waitLog:     l.Sample(&zerolog.BurstSampler{Burst: 20, Period: time.Minute}),
		tenants:     make(map[uuid.UUID]*tenantBudget),
	}
}

func (b *durableTaskBudgets) tenant(tenantId uuid.UUID) (*semaphore.Weighted, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()

	t, ok := b.tenants[tenantId]
	if !ok {
		t = &tenantBudget{sem: semaphore.NewWeighted(8 * b.streamBytes)}
		b.tenants[tenantId] = t
	}

	t.refs++

	return t.sem, func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		t.refs--
		if t.refs == 0 {
			delete(b.tenants, tenantId)
		}
	}
}

func (b *durableTaskBudgets) acquire(ctx context.Context, tenantId uuid.UUID, budget string, sem *semaphore.Weighted, n int64) error {
	if sem.TryAcquire(n) {
		return nil
	}

	b.waitLog.Warn().Str("tenant_id", tenantId.String()).Str("budget", budget).Msg("durable task stream paused reading until its in-flight budget frees up")

	start := time.Now()
	err := sem.Acquire(ctx, n)

	outcome := "acquired"
	if err != nil {
		outcome = "canceled"
	}

	prometheus.DurableTaskBudgetWait.WithLabelValues(budget, outcome).Observe(time.Since(start).Seconds())

	return err
}
