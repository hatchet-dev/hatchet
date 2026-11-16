package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// tenantControlRefreshInterval is how often each process re-reads v1_tenant_control.
const tenantControlRefreshInterval = 2 * time.Second

// TenantControlResumeTimeout is the statement timeout of the transaction that ends a pause, in place
// of the pool's: moving the deadlines of a large backlog is one long statement. Callers of Set and
// Clear give their context at least this long.
const TenantControlResumeTimeout = 10 * time.Minute

// ErrTenantPaused is returned by TenantLimit().CanCreate for new runs and events of a tenant that
// an operator paused. It wraps ErrResourceExhausted, so callers that map a quota error map it too.
var ErrTenantPaused = fmt.Errorf("%w: tenant is paused by an operator", ErrResourceExhausted)

// TenantControlRepository writes operator controls for single tenants (v1_tenant_control) and
// answers from this process's snapshot of them, which is refreshed every few seconds.
type TenantControlRepository interface {
	// Set replaces the tenant's control row. When it ends a pause, it first moves the schedule
	// deadlines of the tenant's queued items forward by the length of the pause.
	Set(ctx context.Context, opts SetTenantControlOpts) (*sqlcv1.V1TenantControl, error)

	// Clear deletes the tenant's control row and reports whether there was one. When the row was
	// paused, it moves the deadlines like Set.
	Clear(ctx context.Context, tenantId uuid.UUID) (bool, error)

	List(ctx context.Context) ([]*sqlcv1.V1TenantControl, error)

	// IsPaused reports whether the snapshot has the tenant paused. It does not touch the database.
	IsPaused(tenantId uuid.UUID) bool

	// Admit applies the snapshot's controls to a request for items new runs or events of the
	// tenant. It returns ErrTenantPaused for a paused tenant, a *TenantRateLimitedError for one over
	// its ingest rate (the request takes no tokens), else nil, and takes the request's tokens. It
	// does not touch the database. TenantLimit().CanCreate calls it for runs and events.
	Admit(tenantId uuid.UUID, items int) error

	// RefreshLoop keeps this process's snapshot current until ctx ends: it reads every control row
	// now and then every 2 seconds, pushes the connection and budget overrides into the pools, and
	// logs each change it sees. Processes that serve requests or schedule run it; one that never
	// does treats every tenant as uncontrolled.
	RefreshLoop(ctx context.Context)

	// AwaitBudget is the yield point of the tenant's deferrable work: a per-tenant loop calls it at
	// the top of an iteration, before it takes any lock or connection. While the tenant is over its
	// database time budget on the main pool it waits, for at most 10 seconds, or until ctx ends and
	// then returns ctx's error. See fairpool.Pool.AwaitBudget.
	AwaitBudget(ctx context.Context, tenantId uuid.UUID) error
}

type SetTenantControlOpts struct {
	TenantId uuid.UUID
	Paused   bool
	// DBConnPercent is the tenant's share of each connection pool, from 1 to 100. Nil means the
	// process default.
	DBConnPercent *int32
	// DBTimePercent is the tenant's database time budget, from 1 to 100. Nil means the process
	// default.
	DBTimePercent *int32
	// IngestPerSecond is the most new runs and events each process admits for the tenant per
	// second, at least 1. Nil means unlimited.
	IngestPerSecond *int32
	Reason          string
}

type tenantControlRepository struct {
	*sharedRepository

	// pools receive the tenants' connection and budget overrides on each refresh.
	pools []*fairpool.Pool
}

func newTenantControlRepository(shared *sharedRepository, pools ...*fairpool.Pool) *tenantControlRepository {
	return &tenantControlRepository{sharedRepository: shared, pools: pools}
}

func (r *tenantControlRepository) Set(ctx context.Context, opts SetTenantControlOpts) (*sqlcv1.V1TenantControl, error) {

	var row *sqlcv1.V1TenantControl

	err := r.changeControl(ctx, opts.TenantId, !opts.Paused, func(tx sqlcv1.DBTX) error {
		var err error

		row, err = r.queries.SetTenantControl(ctx, tx, sqlcv1.SetTenantControlParams{
			Tenantid:        opts.TenantId,
			Paused:          opts.Paused,
			DbConnPercent:   optionalInt(opts.DBConnPercent),
			DbTimePercent:   optionalInt(opts.DBTimePercent),
			IngestPerSecond: optionalInt(opts.IngestPerSecond),
			Reason:          opts.Reason,
		})

		return err
	})

	return row, err
}

func (r *tenantControlRepository) Clear(ctx context.Context, tenantId uuid.UUID) (bool, error) {
	var deleted int64

	err := r.changeControl(ctx, tenantId, true, func(tx sqlcv1.DBTX) error {
		var err error
		deleted, err = r.queries.ClearTenantControl(ctx, tx, tenantId)

		return err
	})

	return deleted > 0, err
}

func optionalInt(value *int32) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{}
	}

	return pgtype.Int4{Int32: *value, Valid: true}
}

// changeControl runs change in a transaction that holds the tenant's control row. If the row is
// paused and resumes is true, the deadlines move first: a process sees the resume only after the
// commit, when every deadline has moved.
func (r *tenantControlRepository) changeControl(ctx context.Context, tenantId uuid.UUID, resumes bool, change func(tx sqlcv1.DBTX) error) error {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool.ForShared(), r.l)
	if err != nil {
		return err
	}

	defer rollback()

	current, err := r.queries.LockTenantControl(ctx, tx, tenantId)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	// a row paused with plain SQL has no paused_at, so its deadlines do not move
	if resumes && current != nil && current.Paused && current.PausedAt.Valid {
		if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL statement_timeout = %d", TenantControlResumeTimeout.Milliseconds())); err != nil {
			return err
		}

		err = r.queries.ShiftScheduleTimeouts(ctx, tx, sqlcv1.ShiftScheduleTimeoutsParams{
			Tenantid: tenantId,
			Pausedat: current.PausedAt,
		})
		if err != nil {
			return fmt.Errorf("could not move the schedule deadlines of tenant %s: %w", tenantId, err)
		}
	}

	if err := change(tx); err != nil {
		return err
	}

	return commit(ctx)
}

func (r *tenantControlRepository) List(ctx context.Context) ([]*sqlcv1.V1TenantControl, error) {
	return r.queries.ListTenantControls(ctx, r.pool.ForShared())
}

func (r *tenantControlRepository) IsPaused(tenantId uuid.UUID) bool {
	return r.tenantPaused(tenantId)
}

func (r *tenantControlRepository) Admit(tenantId uuid.UUID, items int) error {
	return r.admitTenant(tenantId, items)
}

func (r *tenantControlRepository) AwaitBudget(ctx context.Context, tenantId uuid.UUID) error {
	return r.pool.AwaitBudget(ctx, tenantId)
}

// RefreshLoop refreshes the snapshot until ctx is done. A failed refresh keeps the last snapshot.
func (r *tenantControlRepository) RefreshLoop(ctx context.Context) {
	ticker := time.NewTicker(tenantControlRefreshInterval)
	defer ticker.Stop()

	for {
		refreshCtx, cancel := context.WithTimeout(ctx, tenantControlRefreshInterval)
		err := r.refresh(refreshCtx)
		cancel()

		if err != nil && ctx.Err() == nil {
			r.l.Error().Err(err).Msg("could not refresh tenant controls, keeping the last snapshot")
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// refresh reads every control row, swaps the snapshot and pushes the connection and budget
// overrides into the pools. Only the refresher writes the snapshot.
func (r *tenantControlRepository) refresh(ctx context.Context) error {
	rows, err := r.List(ctx)
	if err != nil {
		return err
	}

	var prev map[uuid.UUID]*tenantControl
	if loaded := r.tenantControls.Load(); loaded != nil {
		prev = *loaded
	}

	next := make(map[uuid.UUID]*tenantControl, len(rows))
	connPercents := make(map[uuid.UUID]int)
	timePercents := make(map[uuid.UUID]int)

	for _, row := range rows {
		control := &tenantControl{row: row}

		// a tenant keeps its bucket while its limit stays the same; a new or changed limit starts
		// a full bucket, so no debt from the old limit carries over
		if row.IngestPerSecond.Valid {
			if old := prev[row.TenantID]; old != nil && old.ingest != nil && old.ingest.perSecond == row.IngestPerSecond.Int32 {
				control.ingest = old.ingest
			} else {
				control.ingest = newIngestBucket(row.IngestPerSecond.Int32)
			}
		}

		next[row.TenantID] = control

		if row.DbConnPercent.Valid {
			connPercents[row.TenantID] = int(row.DbConnPercent.Int32)
		}

		if row.DbTimePercent.Valid {
			timePercents[row.TenantID] = int(row.DbTimePercent.Int32)
		}
	}

	r.tenantControls.Store(&next)

	for _, pool := range r.pools {
		pool.SetTenantMaxPercents(connPercents)
		pool.SetTenantBudgetPercents(timePercents)
	}

	r.noteChanges(prev, next)

	return nil
}

// noteChanges logs each tenant whose controls changed and updates its gauge series. The line is a
// warning so that it shows at the default log level, like other operator actions on the data.
func (r *tenantControlRepository) noteChanges(prev, next map[uuid.UUID]*tenantControl) {
	tenants := make(map[uuid.UUID]struct{}, len(prev)+len(next))

	for id := range prev {
		tenants[id] = struct{}{}
	}

	for id := range next {
		tenants[id] = struct{}{}
	}

	for id := range tenants {
		var prevRow, row *sqlcv1.V1TenantControl

		if control := prev[id]; control != nil {
			prevRow = control.row
		}

		if control := next[id]; control != nil {
			row = control.row
		}

		from, to := describeTenantControl(prevRow), describeTenantControl(row)
		if from == to {
			continue
		}

		reason := ""
		if row != nil {
			reason = row.Reason
		}

		r.l.Warn().
			Str("tenant_id", id.String()).
			Str("from", from).
			Str("to", to).
			Str("reason", reason).
			Msg("tenant control changed")

		setTenantControlGauge(id, "paused", row != nil && row.Paused)
		setTenantControlGauge(id, "slow", row != nil && row.DbConnPercent.Valid)
		setTenantControlGauge(id, "budget", row != nil && row.DbTimePercent.Valid)
		setTenantControlGauge(id, "ingest", row != nil && row.IngestPerSecond.Valid)
	}
}

func describeTenantControl(row *sqlcv1.V1TenantControl) string {
	if row == nil {
		return "none"
	}

	var parts []string

	if row.Paused {
		parts = append(parts, "paused")
	}

	if row.DbConnPercent.Valid {
		parts = append(parts, fmt.Sprintf("db_conn_percent=%d", row.DbConnPercent.Int32))
	}

	if row.DbTimePercent.Valid {
		parts = append(parts, fmt.Sprintf("db_time_percent=%d", row.DbTimePercent.Int32))
	}

	if row.IngestPerSecond.Valid {
		parts = append(parts, fmt.Sprintf("ingest_per_second=%d", row.IngestPerSecond.Int32))
	}

	if len(parts) == 0 {
		return "none"
	}

	return strings.Join(parts, ", ")
}

func setTenantControlGauge(tenantId uuid.UUID, control string, set bool) {
	if set {
		prometheus.TenantControl.WithLabelValues(tenantId.String(), control).Set(1)
		return
	}

	prometheus.TenantControl.DeleteLabelValues(tenantId.String(), control)
}
