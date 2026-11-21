package scheduler

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	repov1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

// fakeBacklogTasks serves GetTenantBacklogs from a fixed map; any other TaskRepository
// method panics on the nil embedded interface.
type fakeBacklogTasks struct {
	repov1.TaskRepository

	backlogs map[uuid.UUID]repov1.TenantBacklog
	err      error
}

func (f *fakeBacklogTasks) GetTenantBacklogs(_ context.Context, tenantIds []uuid.UUID, ceiling int64) (map[uuid.UUID]repov1.TenantBacklog, error) {
	if f.err != nil {
		return nil, f.err
	}

	res := make(map[uuid.UUID]repov1.TenantBacklog, len(tenantIds))

	for _, tenantId := range tenantIds {
		b := f.backlogs[tenantId]

		res[tenantId] = repov1.TenantBacklog{
			Queued:      min(b.Queued, ceiling),
			Concurrency: min(b.Concurrency, ceiling),
			RateLimited: min(b.RateLimited, ceiling),
			Batched:     min(b.Batched, ceiling),
			Paused:      min(b.Paused, ceiling),
			Retry:       min(b.Retry, ceiling),
		}
	}

	return res, nil
}

func TestTenantBacklogPollerSeries(t *testing.T) {
	l := zerolog.Nop()
	tasks := &fakeBacklogTasks{}
	p := newTenantBacklogPoller(tasks, &l)

	spread := uuid.New()
	atFloor := uuid.New()
	below := uuid.New()
	capped := uuid.New()

	backlogSeries := func(tenantId uuid.UUID) map[string]float64 {
		return collectGaugeSeries(t, prometheus.TenantBacklog, tenantId.String())
	}

	tasks.backlogs = map[uuid.UUID]repov1.TenantBacklog{
		spread:  {Queued: 3_000, Paused: 2_000},
		atFloor: {Concurrency: tenantBacklogFloor - 1, Retry: 1},
		below:   {Queued: tenantBacklogFloor - 1},
		capped:  {RateLimited: 10 * tenantBacklogCeiling},
	}
	p.poll(context.Background(), []uuid.UUID{spread, atFloor, below, capped})

	// series exist only for a total at or above the floor, one per non-empty state, and a
	// capped state reports the ceiling
	assert.Equal(t, map[string]float64{"state=queued": 3_000, "state=paused": 2_000}, backlogSeries(spread))
	assert.Equal(t, map[string]float64{"state=concurrency": tenantBacklogFloor - 1, "state=retry": 1}, backlogSeries(atFloor))
	assert.Empty(t, backlogSeries(below))
	assert.Equal(t, map[string]float64{"state=rate_limited": tenantBacklogCeiling}, backlogSeries(capped))

	// a state that empties loses its series; a tenant that drops below the floor loses all of
	// them; one that rises gains them
	tasks.backlogs[spread] = repov1.TenantBacklog{Queued: 4_000}
	tasks.backlogs[atFloor] = repov1.TenantBacklog{Concurrency: tenantBacklogFloor - 1}
	tasks.backlogs[below] = repov1.TenantBacklog{Batched: 2_000}
	p.poll(context.Background(), []uuid.UUID{spread, atFloor, below, capped})

	assert.Equal(t, map[string]float64{"state=queued": 4_000}, backlogSeries(spread))
	assert.Empty(t, backlogSeries(atFloor))
	assert.Equal(t, map[string]float64{"state=batched": 2_000}, backlogSeries(below))

	// a failed pass leaves the series as they were
	tasks.err = assert.AnError
	p.poll(context.Background(), []uuid.UUID{spread, below, capped})

	assert.Equal(t, map[string]float64{"state=queued": 4_000}, backlogSeries(spread))
	assert.Equal(t, map[string]float64{"state=batched": 2_000}, backlogSeries(below))
	assert.Equal(t, map[string]float64{"state=rate_limited": tenantBacklogCeiling}, backlogSeries(capped))

	// a tenant no longer on the partition loses its series
	tasks.err = nil
	p.poll(context.Background(), []uuid.UUID{spread, below})

	assert.Empty(t, backlogSeries(capped))
	assert.Equal(t, map[string]float64{"state=queued": 4_000}, backlogSeries(spread))

	// no tenants left: every series is gone
	p.poll(context.Background(), nil)

	assert.Empty(t, backlogSeries(spread))
	assert.Empty(t, backlogSeries(below))
	assert.Empty(t, p.reported)
}
