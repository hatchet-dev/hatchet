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
	// minStreamBudgetBytes lets a stream run about 1,000 small requests at once (at
	// handlerCostBytes each) when the max message size is under 4 MiB.
	minStreamBudgetBytes int64 = 16 * 1024 * 1024

	// handlerCostBytes is added to each request's size for the goroutine that handles it. A
	// handler blocked sending its response uses about 6 KiB (4 KiB stack, 1.6 KiB heap, measured).
	// The higher charge is deliberate: it makes many small requests use up the budget too, capping
	// a 16 MiB stream budget at about 1,000 concurrent handlers.
	handlerCostBytes int64 = 16 * 1024
)

type tenantBudget struct {
	sem  *semaphore.Weighted
	refs int
}

// durableTaskBudgets caps the bytes of DurableTask requests being handled at once, per stream
// and per tenant. A request holds its cost until its handler returns, which is after its
// response is sent. A stream's budget is 4× the max gRPC message size, at least
// minStreamBudgetBytes; a tenant's is 8× that, shared by all its streams on this engine replica.
// With the default 4 MiB max message size: 16 MiB per stream, 128 MiB per tenant per replica.
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

// cost is capped at the stream budget because semaphore.Acquire blocks until ctx ends when n
// exceeds the semaphore's size. The engine never reaches the cap: its gRPC server rejects
// messages over GRPCMaxMsgSize, and the stream budget is at least 4× that.
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
