package dispatcher

import (
	"sync"

	"github.com/google/uuid"
	"golang.org/x/sync/semaphore"
)

const (
	durableTaskStreamBudgetBytes int64 = 16 << 20
	durableTaskTenantBudgetBytes int64 = 64 << 20

	// approximate cost of a handler goroutine, so small requests are bounded by count too
	durableTaskRequestOverheadBytes int64 = 16 << 10
)

type tenantBudget struct {
	sem  *semaphore.Weighted
	refs int
}

type tenantBudgets struct {
	budgets map[uuid.UUID]*tenantBudget
	mu      sync.Mutex
}

func (t *tenantBudgets) acquire(tenantId uuid.UUID) (*semaphore.Weighted, func()) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.budgets == nil {
		t.budgets = make(map[uuid.UUID]*tenantBudget)
	}

	b, ok := t.budgets[tenantId]
	if !ok {
		b = &tenantBudget{sem: semaphore.NewWeighted(durableTaskTenantBudgetBytes)}
		t.budgets[tenantId] = b
	}

	b.refs++

	return b.sem, func() {
		t.mu.Lock()
		defer t.mu.Unlock()

		b.refs--
		if b.refs == 0 {
			delete(t.budgets, tenantId)
		}
	}
}
