package repository

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const (
	// tenantBacklogHoldRefreshInterval is how often a process that admits new runs re-reads
	// v1_tenant_backlog_hold.
	tenantBacklogHoldRefreshInterval = 2 * time.Second

	// tenantBacklogHoldRefreshTimeout bounds one read. It is longer than the interval so that a
	// read waiting for a pooled connection under load still lands; a hold lasts two minutes, so
	// a late snapshot changes little.
	tenantBacklogHoldRefreshTimeout = 10 * time.Second

	// TenantBacklogRetryAfter is how long a refused caller is told to wait. It is the longest
	// delay the SDKs wait before retrying a throttled call.
	TenantBacklogRetryAfter = 5 * time.Second
)

// TenantBacklogHoldRepository closes a tenant's intake while its backlog is at or over the
// shard's limit. Schedulers write the holds (v1_tenant_backlog_hold) after each measurement;
// processes that admit new runs keep a snapshot of them, which RefreshLoop keeps current.
//
// A hold lasts two minutes unless the next measurement renews it, so a hold that nothing
// renews (its scheduler stopped, its tenant moved partition or was deleted, or the limit was
// set to 0) ends by itself. Readers ignore an expired hold, and the tenant's next hold
// overwrites it.
type TenantBacklogHoldRepository interface {
	// Update writes one measurement pass with a limit above 0: each tenant in measured whose
	// total is at or over the limit is held for two more minutes, and each other tenant in
	// measured is released. It logs each hold that starts or ends.
	Update(ctx context.Context, limit int64, measured map[uuid.UUID]TenantBacklog) error

	// Admit returns a *TenantBacklogOverLimitError, and counts items as refused, when the
	// tenant is held in this process's snapshot. It does no database work.
	Admit(tenantId uuid.UUID, items int) error

	// RefreshLoop reads the holds now and then every 2 seconds until ctx ends, and swaps the
	// snapshot. Each read may take up to 10 seconds; a failed read keeps the last snapshot. A
	// process that never runs it admits every tenant.
	RefreshLoop(ctx context.Context)
}

// TenantBacklogOverLimitError refuses a new run of a tenant whose backlog is at or over the
// limit. The caller may retry after TenantBacklogRetryAfter.
type TenantBacklogOverLimitError struct {
	Backlog      int64
	Limit        int64
	LargestState string
}

func (e *TenantBacklogOverLimitError) Error() string {
	description := e.LargestState

	// a state name this process does not know (from a scheduler of another version) is shown as is
	for _, s := range (TenantBacklog{}).States() {
		if s.State == e.LargestState {
			description = s.Description
		}
	}

	return fmt.Sprintf(
		"tenant has %d tasks accepted but not yet running, at or over the limit of %d; new runs are refused until fewer are waiting. The largest share is %s",
		e.Backlog, e.Limit, description,
	)
}

// Largest returns the state with the most tasks; a tie goes to the earlier state in States.
func (b TenantBacklog) Largest() TenantBacklogState {
	states := b.States()
	largest := states[0]

	for _, s := range states[1:] {
		if s.Count > largest.Count {
			largest = s
		}
	}

	return largest
}

type tenantBacklogHoldRepository struct {
	*sharedRepository

	// snapshot is this process's copy of the unexpired holds, swapped whole by the refresher.
	// It starts empty, so a process that has not read the table yet admits every tenant.
	snapshot atomic.Pointer[map[uuid.UUID]*sqlcv1.V1TenantBacklogHold]
}

func newTenantBacklogHoldRepository(shared *sharedRepository) *tenantBacklogHoldRepository {
	r := &tenantBacklogHoldRepository{sharedRepository: shared}
	r.snapshot.Store(&map[uuid.UUID]*sqlcv1.V1TenantBacklogHold{})

	return r
}

func (r *tenantBacklogHoldRepository) Update(ctx context.Context, limit int64, measured map[uuid.UUID]TenantBacklog) error {
	params := sqlcv1.UpdateTenantBacklogHoldsParams{
		Heldtenantids:     []uuid.UUID{},
		Backlogs:          []int64{},
		Largeststates:     []string{},
		Backloglimit:      limit,
		Releasedtenantids: []uuid.UUID{},
	}

	for tenantId, backlog := range measured {
		if total := backlog.Total(); total >= limit {
			params.Heldtenantids = append(params.Heldtenantids, tenantId)
			params.Backlogs = append(params.Backlogs, total)
			params.Largeststates = append(params.Largeststates, backlog.Largest().State)
		} else {
			params.Releasedtenantids = append(params.Releasedtenantids, tenantId)
		}
	}

	changes, err := r.queries.UpdateTenantBacklogHolds(ctx, r.pool.ForShared(), params)

	if err != nil {
		return fmt.Errorf("could not update tenant backlog holds: %w", err)
	}

	for _, c := range changes {
		if c.Held {
			r.l.Warn().
				Str("tenant_id", c.TenantID.String()).
				Int64("backlog", c.Backlog).
				Int64("backlog_limit", c.BacklogLimit).
				Str("largest_state", c.LargestState).
				Msg("tenant backlog is at or over the limit, refusing its new runs")
		} else {
			r.l.Info().
				Str("tenant_id", c.TenantID.String()).
				Msg("tenant backlog hold ended, admitting its new runs")
		}
	}

	return nil
}

func (r *tenantBacklogHoldRepository) Admit(tenantId uuid.UUID, items int) error {
	hold := (*r.snapshot.Load())[tenantId]

	if hold == nil || !time.Now().Before(hold.HeldUntil.Time) {
		return nil
	}

	prometheus.TenantBacklogRefused.WithLabelValues(tenantId.String()).Add(float64(items))

	return &TenantBacklogOverLimitError{
		Backlog:      hold.Backlog,
		Limit:        hold.BacklogLimit,
		LargestState: hold.LargestState,
	}
}

func (r *tenantBacklogHoldRepository) RefreshLoop(ctx context.Context) {
	ticker := time.NewTicker(tenantBacklogHoldRefreshInterval)
	defer ticker.Stop()

	for {
		refreshCtx, cancel := context.WithTimeout(ctx, tenantBacklogHoldRefreshTimeout)
		err := r.refresh(refreshCtx)
		cancel()

		if err != nil && ctx.Err() == nil {
			r.l.Error().Err(err).Msg("could not refresh tenant backlog holds, keeping the last snapshot")
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// refresh reads the unexpired holds and swaps the snapshot. Only the refresher writes it.
func (r *tenantBacklogHoldRepository) refresh(ctx context.Context) error {
	holds, err := r.queries.ListTenantBacklogHolds(ctx, r.pool.ForShared())

	if err != nil {
		return err
	}

	next := make(map[uuid.UUID]*sqlcv1.V1TenantBacklogHold, len(holds))

	for _, hold := range holds {
		next[hold.TenantID] = hold
	}

	r.snapshot.Store(&next)

	return nil
}
