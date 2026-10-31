package dispatcher

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/proto"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
)

const (
	// keeps room for about a thousand small requests per stream when the max message size is set low
	minStreamBudgetBytes int64 = 16 << 20

	// Added to every request's size so small requests still use up budget: a 16 MiB budget runs at
	// most about 1,000 handlers at once. A handler parked on a send holds about 6 KiB besides its
	// request, so this is conservative.
	handlerCostBytes int64 = 16 << 10
)

type tenantBudget struct {
	sem  *semaphore.Weighted
	refs int
}

// durableTaskBudgets caps the bytes of DurableTask requests being handled at once. Each stream
// may hold 4× the max gRPC message size (at least minStreamBudgetBytes). Each tenant may hold 8×
// that, shared by all of its streams on this engine.
type durableTaskBudgets struct {
	l           *zerolog.Logger
	streamBytes int64

	mu      sync.Mutex
	tenants map[uuid.UUID]*tenantBudget
}

func newDurableTaskBudgets(l *zerolog.Logger, maxMsgSize int) *durableTaskBudgets {
	sampled := l.Sample(&zerolog.BurstSampler{Burst: 20, Period: time.Minute})

	return &durableTaskBudgets{
		l:           &sampled,
		streamBytes: max(minStreamBudgetBytes, 4*int64(maxMsgSize)),
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

// cost is capped at the stream budget so a request larger than the whole budget runs alone
// instead of waiting forever.
func (b *durableTaskBudgets) cost(req *contracts.DurableTaskRequest) int64 {
	return min(int64(proto.Size(req))+handlerCostBytes, b.streamBytes)
}

func (b *durableTaskBudgets) acquire(ctx context.Context, tenantId uuid.UUID, budget string, sem *semaphore.Weighted, n int64) error {
	if sem.TryAcquire(n) {
		return nil
	}

	b.l.Warn().Str("tenant_id", tenantId.String()).Str("budget", budget).Msg("durable task stream paused reading until its in-flight budget frees up")

	start := time.Now()
	err := sem.Acquire(ctx, n)

	outcome := "acquired"
	if err != nil {
		outcome = "canceled"
	}

	prometheus.DurableTaskBudgetWait.WithLabelValues(budget, outcome).Observe(time.Since(start).Seconds())

	return err
}
