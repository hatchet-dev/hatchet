package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	promclient "github.com/prometheus/client_golang/prometheus"

	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// tenantControlRefreshInterval is how often each process re-reads v1_tenant_control.
const tenantControlRefreshInterval = 2 * time.Second

// TenantControlChangeTimeout is the statement timeout of Set and Clear. Ending a pause moves the
// schedule deadline of every queued item of the tenant in the same statement (a trigger on
// v1_tenant_control does it), which takes seconds for a backlog of a few hundred thousand items.
const TenantControlChangeTimeout = 10 * time.Minute

// ErrTenantPaused is returned by TenantLimit().CanCreate and TenantControl().Admit for new runs and
// events of a tenant that an operator paused. It wraps ErrResourceExhausted. Unlike a rate limit it
// says nothing about when to retry: the pause lasts until an operator ends it.
var ErrTenantPaused = fmt.Errorf("%w: tenant is paused by an operator", ErrResourceExhausted)

// IsTenantControlRefusal reports whether err is a refusal by an operator control: ErrTenantPaused
// or a *TenantRateLimitedError.
func IsTenantControlRefusal(err error) bool {
	var rateLimited *TenantRateLimitedError

	return errors.Is(err, ErrTenantPaused) || errors.As(err, &rateLimited)
}

// the values of the control label of the tenant control metrics
const (
	tenantControlPaused = "paused"
	tenantControlIngest = "ingest"
)

// TenantControlRepository writes operator controls for single tenants (v1_tenant_control) and
// applies this process's snapshot of them, which RefreshLoop keeps current.
type TenantControlRepository interface {
	// Set replaces the tenant's control row. Unpausing a paused tenant moves its schedule deadlines
	// (see TenantControlChangeTimeout).
	Set(ctx context.Context, opts SetTenantControlOpts) (*sqlcv1.V1TenantControl, error)

	// Clear deletes the tenant's control row and reports whether there was one. Clearing a paused
	// tenant moves its schedule deadlines, like Set.
	Clear(ctx context.Context, tenantId uuid.UUID) (bool, error)

	List(ctx context.Context) ([]*sqlcv1.V1TenantControl, error)

	// Admit applies the snapshot's controls to a request for items new runs or events of the
	// tenant. It returns ErrTenantPaused for a paused tenant, and a *TenantRateLimitedError, taking
	// nothing, when the tenant is over its ingest rate; otherwise it takes the request's tokens and
	// returns nil. It does not touch the database. TenantLimit().CanCreate calls it for runs and
	// events.
	Admit(tenantId uuid.UUID, items int) error

	// IsPaused reports whether the snapshot has the tenant paused. It does not touch the database.
	// The scheduler does not use it: it reads the pause with its tenant list
	// (TenantRepository.ListSchedulableTenantsBySchedulerPartition), so a stale or empty snapshot
	// cannot make it schedule a paused tenant.
	IsPaused(tenantId uuid.UUID) bool

	// RefreshLoop reads every control row now and then every 2 seconds until ctx ends, swaps the
	// snapshot and logs each change it sees. A process that never runs it treats every tenant as
	// uncontrolled.
	RefreshLoop(ctx context.Context)
}

type SetTenantControlOpts struct {
	TenantId uuid.UUID

	// IngestPerSecond is the most new runs and events each process admits for the tenant per
	// second, at least 1, or nil for no limit.
	IngestPerSecond *int32

	// Paused stops the engine from scheduling the tenant's queued work and refuses its new runs and
	// events. A row sets at least one control.
	Paused bool

	Reason string
}

type tenantControlRepository struct {
	*sharedRepository

	// snapshot is this process's copy of v1_tenant_control, swapped whole by the refresher. It
	// starts empty, so a process that has not read the table yet controls no tenant.
	snapshot atomic.Pointer[map[uuid.UUID]*tenantControl]
}

// tenantControl is one tenant's entry in the snapshot. A field of a control the row does not set is
// nil.
type tenantControl struct {
	row           *sqlcv1.V1TenantControl
	ingest        *ingestBucket
	refusedPaused promclient.Counter
	refusedIngest promclient.Counter
}

func newTenantControlRepository(shared *sharedRepository) *tenantControlRepository {
	r := &tenantControlRepository{sharedRepository: shared}
	r.snapshot.Store(&map[uuid.UUID]*tenantControl{})

	return r
}

func (r *tenantControlRepository) Set(ctx context.Context, opts SetTenantControlOpts) (*sqlcv1.V1TenantControl, error) {
	params := sqlcv1.SetTenantControlParams{
		Tenantid: opts.TenantId,
		Paused:   opts.Paused,
		Reason:   opts.Reason,
	}

	if opts.IngestPerSecond != nil {
		params.IngestPerSecond = pgtype.Int4{Int32: *opts.IngestPerSecond, Valid: true}
	}

	var row *sqlcv1.V1TenantControl

	err := r.change(ctx, func(tx sqlcv1.DBTX) (err error) {
		row, err = r.queries.SetTenantControl(ctx, tx, params)

		return err
	})

	return row, err
}

func (r *tenantControlRepository) Clear(ctx context.Context, tenantId uuid.UUID) (bool, error) {
	var deleted int64

	err := r.change(ctx, func(tx sqlcv1.DBTX) (err error) {
		deleted, err = r.queries.ClearTenantControl(ctx, tx, tenantId)

		return err
	})

	return deleted > 0, err
}

// change runs fn in a transaction whose statement timeout is TenantControlChangeTimeout.
func (r *tenantControlRepository) change(ctx context.Context, fn func(tx sqlcv1.DBTX) error) error {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool.ForShared(), r.l)
	if err != nil {
		return err
	}

	defer rollback()

	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL statement_timeout = %d", TenantControlChangeTimeout.Milliseconds())); err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		return err
	}

	return commit(ctx)
}

func (r *tenantControlRepository) List(ctx context.Context) ([]*sqlcv1.V1TenantControl, error) {
	return r.queries.ListTenantControls(ctx, r.pool.ForShared())
}

func (r *tenantControlRepository) Admit(tenantId uuid.UUID, items int) error {
	control := (*r.snapshot.Load())[tenantId]
	if control == nil {
		return nil
	}

	if control.row.Paused {
		control.refusedPaused.Add(float64(items))

		return ErrTenantPaused
	}

	if control.ingest == nil {
		return nil
	}

	if wait := control.ingest.take(items, time.Now()); wait > 0 {
		control.refusedIngest.Add(float64(items))

		return &TenantRateLimitedError{RetryAfter: wait}
	}

	return nil
}

func (r *tenantControlRepository) IsPaused(tenantId uuid.UUID) bool {
	control := (*r.snapshot.Load())[tenantId]

	return control != nil && control.row.Paused
}

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

// refresh reads every control row and swaps the snapshot. Only the refresher writes it.
func (r *tenantControlRepository) refresh(ctx context.Context) error {
	rows, err := r.List(ctx)
	if err != nil {
		return err
	}

	prev := *r.snapshot.Load()
	next := make(map[uuid.UUID]*tenantControl, len(rows))

	for _, row := range rows {
		control := &tenantControl{row: row}
		id := row.TenantID.String()

		if row.Paused {
			control.refusedPaused = prometheus.TenantControlRefused.WithLabelValues(id, tenantControlPaused)
		}

		if row.IngestPerSecond.Valid {
			control.refusedIngest = prometheus.TenantControlRefused.WithLabelValues(id, tenantControlIngest)

			// A tenant keeps its bucket, and any debt in it, while its limit stays the same. A new
			// or changed limit starts with a full bucket.
			if old := prev[row.TenantID]; old != nil && old.row.IngestPerSecond == row.IngestPerSecond {
				control.ingest = old.ingest
			} else {
				control.ingest = newIngestBucket(row.IngestPerSecond.Int32)
			}
		}

		next[row.TenantID] = control
	}

	r.snapshot.Store(&next)
	r.noteChanges(prev, next)

	return nil
}

// noteChanges logs each tenant whose controls changed and adds or removes its metric series. The
// line is a warning so that it shows at the default log level.
func (r *tenantControlRepository) noteChanges(prev, next map[uuid.UUID]*tenantControl) {
	for id, control := range next {
		old := prev[id]

		if old != nil && old.row.Paused == control.row.Paused && old.row.IngestPerSecond == control.row.IngestPerSecond {
			continue
		}

		r.l.Warn().
			Str("tenant_id", id.String()).
			Str("from", describeTenantControl(old)).
			Str("to", describeTenantControl(control)).
			Str("reason", control.row.Reason).
			Msg("tenant control changed")

		noteControlSeries(id, old, control)
	}

	for id, old := range prev {
		if next[id] != nil {
			continue
		}

		r.l.Warn().
			Str("tenant_id", id.String()).
			Str("from", describeTenantControl(old)).
			Str("to", describeTenantControl(nil)).
			Msg("tenant control changed")

		noteControlSeries(id, old, nil)
	}
}

// noteControlSeries sets the gauge of each control the tenant has now and removes both series of
// each control it no longer has; either entry may be nil. refresh creates a refusal counter's series
// with its control.
func noteControlSeries(id uuid.UUID, old, control *tenantControl) {
	note := func(name string, had, has bool) {
		if has {
			prometheus.TenantControl.WithLabelValues(id.String(), name).Set(1)
		} else if had {
			prometheus.TenantControl.DeleteLabelValues(id.String(), name)
			prometheus.TenantControlRefused.DeleteLabelValues(id.String(), name)
		}
	}

	note(tenantControlPaused, old != nil && old.row.Paused, control != nil && control.row.Paused)
	note(tenantControlIngest, old != nil && old.row.IngestPerSecond.Valid, control != nil && control.row.IngestPerSecond.Valid)
}

func describeTenantControl(control *tenantControl) string {
	if control == nil {
		return "none"
	}

	var parts []string

	if control.row.Paused {
		parts = append(parts, "paused since "+control.row.PausedAt.Time.UTC().Format(time.RFC3339))
	}

	if control.row.IngestPerSecond.Valid {
		parts = append(parts, fmt.Sprintf("ingest_per_second=%d", control.row.IngestPerSecond.Int32))
	}

	return strings.Join(parts, ", ")
}
