package task

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/operation"
	"github.com/hatchet-dev/hatchet/internal/services/partition"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

type ownedTenantsRepo struct {
	v1.TenantRepository
	tenants []uuid.UUID
}

func (r ownedTenantsRepo) ListTenantsByControllerPartition(context.Context, string) ([]uuid.UUID, error) {
	return r.tenants, nil
}

func TestHandleWakeRunsPausedItemsSweepForOwnedTenant(t *testing.T) {
	owned, other := uuid.New(), uuid.New()
	l := zerolog.Nop()

	p, err := partition.NewPartition(&l, ownedTenantsRepo{tenants: []uuid.UUID{owned}})
	require.NoError(t, err)

	var mu sync.Mutex
	runs := map[string]int{}

	pool := operation.NewTenantOperationPool(p, &l, "test", time.Second, "test", func(_ context.Context, id string) (bool, error) {
		mu.Lock()
		defer mu.Unlock()
		runs[id]++
		return false, nil
	})
	defer pool.Cleanup()

	runsOf := func(id uuid.UUID) int {
		mu.Lock()
		defer mu.Unlock()
		return runs[id.String()]
	}

	pool.RunOrContinue(owned)
	require.Eventually(t, func() bool { return runsOf(owned) == 1 }, time.Second, 5*time.Millisecond)

	tc := &TasksControllerImpl{l: &l, expirePausedWorkflowQueueItemsOperations: pool}

	for i, id := range []string{msgqueue.MsgIDTogglePauseWorkflow, msgqueue.MsgIDRateLimitPausesChanged} {
		require.NoError(t, tc.handleWake(&msgqueue.Message{ID: id, TenantID: owned}))
		require.Eventually(t, func() bool { return runsOf(owned) == i+2 }, time.Second, 5*time.Millisecond)
	}

	require.NoError(t, tc.handleWake(&msgqueue.Message{ID: msgqueue.MsgIDTogglePauseWorkflow, TenantID: other}))
	require.NoError(t, tc.handleWake(&msgqueue.Message{ID: msgqueue.MsgIDTaskTrigger, TenantID: owned}))

	assert.Never(t, func() bool { return runsOf(owned) > 3 || runsOf(other) > 0 }, 200*time.Millisecond, 10*time.Millisecond)
}
