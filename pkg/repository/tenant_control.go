package repository

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	promclient "github.com/prometheus/client_golang/prometheus"

	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// tenantControlRefreshInterval is how often each process re-reads v1_tenant_control.
const tenantControlRefreshInterval = 2 * time.Second

// TenantControlRepository writes operator controls for single tenants (v1_tenant_control) and
// applies this process's snapshot of them, which RefreshLoop keeps current.
type TenantControlRepository interface {
	// Set replaces the tenant's control row.
	Set(ctx context.Context, opts SetTenantControlOpts) (*sqlcv1.V1TenantControl, error)

	// Clear deletes the tenant's control row and reports whether there was one.
	Clear(ctx context.Context, tenantId uuid.UUID) (bool, error)

	List(ctx context.Context) ([]*sqlcv1.V1TenantControl, error)

	// Admit applies the snapshot's controls to a request for items new runs or events of the
	// tenant. It returns a *TenantRateLimitedError, and takes nothing, when the tenant is over its
	// ingest rate; otherwise it takes the request's tokens and returns nil. It does not touch the
	// database. TenantLimit().CanCreate calls it for runs and events.
	Admit(tenantId uuid.UUID, items int) error

	// RefreshLoop reads every control row now and then every 2 seconds until ctx ends, swaps the
	// snapshot and logs each change it sees. A process that never runs it treats every tenant as
	// uncontrolled.
	RefreshLoop(ctx context.Context)
}

type SetTenantControlOpts struct {
	TenantId uuid.UUID

	// IngestPerSecond is the most new runs and events each process admits for the tenant per
	// second, at least 1.
	IngestPerSecond int32

	Reason string
}

type tenantControlRepository struct {
	*sharedRepository

	// snapshot is this process's copy of v1_tenant_control, swapped whole by the refresher. It
	// starts empty, so a process that has not read the table yet controls no tenant.
	snapshot atomic.Pointer[map[uuid.UUID]*tenantControl]
}

// tenantControl is one tenant's entry in the snapshot.
type tenantControl struct {
	row     *sqlcv1.V1TenantControl
	ingest  *ingestBucket
	refused promclient.Counter
}

func newTenantControlRepository(shared *sharedRepository) *tenantControlRepository {
	r := &tenantControlRepository{sharedRepository: shared}
	r.snapshot.Store(&map[uuid.UUID]*tenantControl{})

	return r
}

func (r *tenantControlRepository) Set(ctx context.Context, opts SetTenantControlOpts) (*sqlcv1.V1TenantControl, error) {
	return r.queries.SetTenantControl(ctx, r.pool.ForShared(), sqlcv1.SetTenantControlParams{
		Tenantid:        opts.TenantId,
		Ingestpersecond: opts.IngestPerSecond,
		Reason:          opts.Reason,
	})
}

func (r *tenantControlRepository) Clear(ctx context.Context, tenantId uuid.UUID) (bool, error) {
	deleted, err := r.queries.ClearTenantControl(ctx, r.pool.ForShared(), tenantId)

	return deleted > 0, err
}

func (r *tenantControlRepository) List(ctx context.Context) ([]*sqlcv1.V1TenantControl, error) {
	return r.queries.ListTenantControls(ctx, r.pool.ForShared())
}

func (r *tenantControlRepository) Admit(tenantId uuid.UUID, items int) error {
	control := (*r.snapshot.Load())[tenantId]
	if control == nil {
		return nil
	}

	if wait := control.ingest.take(items, time.Now()); wait > 0 {
		control.refused.Add(float64(items))

		return &TenantRateLimitedError{RetryAfter: wait}
	}

	return nil
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
		control := &tenantControl{
			row:     row,
			refused: prometheus.TenantControlRefused.WithLabelValues(row.TenantID.String(), "ingest"),
		}

		// A tenant keeps its bucket, and any debt in it, while its limit stays the same. A new or
		// changed limit starts with a full bucket.
		if old := prev[row.TenantID]; old != nil && old.row.IngestPerSecond == row.IngestPerSecond {
			control.ingest = old.ingest
		} else {
			control.ingest = newIngestBucket(row.IngestPerSecond)
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

		if old != nil && old.row.IngestPerSecond == control.row.IngestPerSecond {
			continue
		}

		r.l.Warn().
			Str("tenant_id", id.String()).
			Str("from", describeTenantControl(old)).
			Str("to", describeTenantControl(control)).
			Str("reason", control.row.Reason).
			Msg("tenant control changed")

		prometheus.TenantControl.WithLabelValues(id.String(), "ingest").Set(1)
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

		prometheus.TenantControl.DeleteLabelValues(id.String(), "ingest")
		prometheus.TenantControlRefused.DeleteLabelValues(id.String(), "ingest")
	}
}

func describeTenantControl(control *tenantControl) string {
	if control == nil {
		return "none"
	}

	return fmt.Sprintf("ingest_per_second=%d", control.row.IngestPerSecond)
}
