package scheduler

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	v1 "github.com/hatchet-dev/hatchet/pkg/scheduling/v1"
)

type countingTaskRepository struct {
	*fakeTaskRepository
	activeRuns int
	calls      int
}

func (c *countingTaskRepository) CountActiveTaskBatchRuns(context.Context, string, string, string) (int, error) {
	c.mu.Lock()
	c.calls++
	c.mu.Unlock()

	return c.activeRuns, nil
}

func TestDecideBatchDefersWhenMaxRunsReached(t *testing.T) {
	t.Parallel()

	logger := newNoopLogger()

	taskRepo := &countingTaskRepository{
		fakeTaskRepository: &fakeTaskRepository{},
		activeRuns:         1,
	}

	stepID := "11111111-2222-3333-4444-555555555555"
	tenantID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	workerID := "99999999-8888-7777-6666-555555555555"

	sched := &Scheduler{
		l: logger,
		repov1: &fakeRepository{
			tasks: taskRepo,
		},
		batchConfigs: cache.NewTTL[string, *batchConfig](),
	}

	t.Cleanup(func() {
		sched.batchConfigs.Stop()
	})

	sched.batchConfigs.Set(stepID, &batchConfig{
		batchSize: 1,
		maxRuns:   1,
	}, batchConfigCacheTTL)

	coord := newSchedulingBatchCoordinator(sched)

	queueItem := &sqlcv1.V1QueueItem{
		TenantID: sqlchelpers.UUIDFromStr(tenantID),
		StepID:   sqlchelpers.UUIDFromStr(stepID),
		BatchKey: pgtype.Text{
			String: "key-1",
			Valid:  true,
		},
	}

	decision := coord.DecideBatch(
		context.Background(),
		queueItem,
		sqlchelpers.UUIDFromStr(workerID),
	)

	require.True(t, decision.Action == v1.BatchActionDefer)
	assert.True(t, decision.ReleaseSlot)
	assert.False(t, decision.Action == v1.BatchActionBuffer)
	assert.Equal(t, 1, taskRepo.calls)
}
