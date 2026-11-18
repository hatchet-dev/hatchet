package scheduler

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

// fakeBacklogHolds records the hold updates it is given.
type fakeBacklogHolds struct {
	repov1.TenantBacklogHoldRepository

	updates []backlogHoldUpdate
}

type backlogHoldUpdate struct {
	limit    int64
	measured map[uuid.UUID]repov1.TenantBacklog
}

func (f *fakeBacklogHolds) Update(_ context.Context, limit int64, measured map[uuid.UUID]repov1.TenantBacklog) error {
	f.updates = append(f.updates, backlogHoldUpdate{limit: limit, measured: measured})
	return nil
}

func TestTenantBacklogPollerUpdatesHolds(t *testing.T) {
	l := zerolog.Nop()
	tenantId := uuid.New()

	for name, tc := range map[string]struct {
		limit int64
		want  repov1.TenantBacklog
	}{
		"under the default ceiling":             {limit: 1_000, want: repov1.TenantBacklog{Queued: tenantBacklogCeiling}},
		"above the default ceiling measures up": {limit: 2 * tenantBacklogCeiling, want: repov1.TenantBacklog{Queued: 2 * tenantBacklogCeiling}},
	} {
		t.Run(name, func(t *testing.T) {
			tasks := &fakeBacklogTasks{backlogs: map[uuid.UUID]repov1.TenantBacklog{tenantId: {Queued: 10 * tenantBacklogCeiling}}}
			holds := &fakeBacklogHolds{}
			p := newTenantBacklogPoller(tasks, holds, tc.limit, &l)

			p.poll(context.Background(), []uuid.UUID{tenantId})
			require.Len(t, holds.updates, 1)
			assert.Equal(t, tc.limit, holds.updates[0].limit)
			assert.Equal(t, map[uuid.UUID]repov1.TenantBacklog{tenantId: tc.want}, holds.updates[0].measured)

			// a failed measurement leaves the holds as they are
			tasks.err = assert.AnError
			p.poll(context.Background(), []uuid.UUID{tenantId})
			require.Len(t, holds.updates, 1)
		})
	}

	t.Run("limit 0 makes no call", func(t *testing.T) {
		tasks := &fakeBacklogTasks{backlogs: map[uuid.UUID]repov1.TenantBacklog{tenantId: {Queued: 10 * tenantBacklogCeiling}}}
		holds := &fakeBacklogHolds{}
		p := newTenantBacklogPoller(tasks, holds, 0, &l)

		p.poll(context.Background(), []uuid.UUID{tenantId})
		assert.Empty(t, holds.updates)
		assert.Equal(t, map[string]float64{"state=queued": tenantBacklogCeiling}, collectGaugeSeries(t, prometheus.TenantBacklog, tenantId.String()))

		p.poll(context.Background(), nil)
	})
}

// A limit below the floor lowers the floor to the limit, so every held tenant has series.
func TestTenantBacklogPollerFloorFollowsALowerLimit(t *testing.T) {
	l := zerolog.Nop()
	held := uuid.New()
	under := uuid.New()

	tasks := &fakeBacklogTasks{backlogs: map[uuid.UUID]repov1.TenantBacklog{
		held:  {Paused: 100},
		under: {Paused: 99},
	}}
	p := newTenantBacklogPoller(tasks, &fakeBacklogHolds{}, 100, &l)

	p.poll(context.Background(), []uuid.UUID{held, under})
	assert.Equal(t, map[string]float64{"state=paused": 100}, collectGaugeSeries(t, prometheus.TenantBacklog, held.String()))
	assert.Empty(t, collectGaugeSeries(t, prometheus.TenantBacklog, under.String()))

	// a limit above the floor leaves the floor as it is
	assert.Equal(t, int64(tenantBacklogFloor), newTenantBacklogPoller(tasks, nil, 10*tenantBacklogFloor, &l).floor)

	p.poll(context.Background(), nil)
}

func TestTenantBacklogPollerSeries(t *testing.T) {
	l := zerolog.Nop()
	tasks := &fakeBacklogTasks{}
	p := newTenantBacklogPoller(tasks, nil, 0, &l)

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
