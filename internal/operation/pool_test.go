package operation

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTenantOperationPool_RunOrContinueIfOwned(t *testing.T) {
	owned, other := uuid.New(), uuid.New()

	var mu sync.Mutex
	ran := map[string]int{}

	pool := &TenantOperationPool{
		ql:      &l,
		timeout: time.Second,
		cancel:  func() {},
		method: func(_ context.Context, id string) (bool, error) {
			mu.Lock()
			defer mu.Unlock()
			ran[id]++
			return false, nil
		},
	}
	defer pool.Cleanup()

	pool.setTenants([]uuid.UUID{owned})

	assert.True(t, pool.RunOrContinueIfOwned(owned))
	assert.False(t, pool.RunOrContinueIfOwned(other))

	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return ran[owned.String()] == 1
	}, time.Second, 10*time.Millisecond)

	mu.Lock()
	assert.Zero(t, ran[other.String()])
	mu.Unlock()

	_, tracked := pool.ops.Load(other)
	assert.False(t, tracked)
}
