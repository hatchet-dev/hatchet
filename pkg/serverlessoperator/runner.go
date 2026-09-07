package serverlessoperator

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/lease"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/link"
)

// endpointPageSize is the keyset page size for loading a gained unit's endpoints.
const endpointPageSize int64 = 500

// tenantState is everything the process keeps for a served tenant: the routing cache, the
// registration per owned shard, the pollers of owned endpoints and whether the link could
// authenticate. mu guards regs; the rest is only touched from the runner's serialized
// reconcile paths (lease tick and maintenance loop share runner.mu).
type tenantState struct {
	cache    *routingCache
	regs     map[int32]*registration
	pollers  map[uuid.UUID]*endpointPoller
	units    map[int32]struct{}
	tenantId uuid.UUID
	mu       sync.Mutex
	noToken  bool
}

// runner turns owned units into registrations and endpoint pollers and routes assigned
// actions to endpoints. It implements lease.Reconciler.
type runner struct {
	repo         repository.ServerlessRepository
	link         link.Link
	enc          encryption.EncryptionService
	sender       RequestSender
	l            *zerolog.Logger
	m            *metrics
	tenants      map[uuid.UUID]*tenantState
	hcSem        chan struct{}
	loopCtx      context.Context
	stopLoops    context.CancelFunc
	deliveryCtx  context.Context
	stopDelivery context.CancelFunc
	cfg          Config
	processId    uuid.UUID
	mu           sync.Mutex
	wg           sync.WaitGroup
}

func newRunner(deps Deps, cfg Config, m *metrics) *runner {
	// Pollers and action loops live on loopCtx and are stopped first at shutdown;
	// deliveries live on deliveryCtx, which is cancelled only after the drain timeout.
	loopCtx, stopLoops := context.WithCancel(context.Background())
	deliveryCtx, cancel := context.WithCancel(context.Background())

	return &runner{
		repo:         deps.Repo,
		link:         deps.Link,
		enc:          deps.Encryption,
		sender:       deps.Sender,
		l:            deps.Logger,
		m:            m,
		cfg:          cfg,
		processId:    deps.ProcessId,
		tenants:      map[uuid.UUID]*tenantState{},
		hcSem:        make(chan struct{}, cfg.HealthcheckConcurrency),
		loopCtx:      loopCtx,
		stopLoops:    stopLoops,
		deliveryCtx:  deliveryCtx,
		stopDelivery: cancel,
	}
}

// UnitsGained implements lease.Reconciler: load the units' endpoints, ensure the tenant's
// routing cache, open the registration and start pollers. Failures are logged; the
// maintenance loop retries registrations still missing.
func (r *runner) UnitsGained(ctx context.Context, units []lease.Unit) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.loadUnitEndpoints(ctx, units); err != nil {
		r.l.Error().Err(err).Msg("could not load endpoints for gained units")
	}

	for _, unit := range units {
		ts, err := r.ensureTenant(ctx, unit.TenantId)

		if err != nil {
			r.l.Error().Err(err).Str("tenant_id", unit.TenantId.String()).Msg("could not load tenant routing cache")
			continue
		}

		ts.units[unit.Shard] = struct{}{}

		if err := r.openRegistration(ctx, ts, unit.Shard); err != nil {
			r.l.Warn().Err(err).Str("tenant_id", unit.TenantId.String()).Int32("shard", unit.Shard).Msg("registration not opened; will retry")
		}

		r.reconcilePollers(ts)
	}

	r.updateGaugesLocked()
}

// loadUnitEndpoints pages through the gained units' endpoints and seeds their tenants'
// caches, so a tenant with many endpoints outside these units is not reloaded in full for
// every gained shard.
func (r *runner) loadUnitEndpoints(ctx context.Context, units []lease.Unit) error {
	after := uuid.Nil

	for {
		rows, err := r.repo.Endpoints().ListForUnits(ctx, units, after, endpointPageSize)

		if err != nil {
			return err
		}

		for _, row := range rows {
			ts, err := r.ensureTenant(ctx, row.TenantID)

			if err != nil {
				return err
			}

			ts.cache.mu.Lock()
			ts.cache.upsertLocked(row)
			ts.cache.recomputeUnionLocked()
			ts.cache.mu.Unlock()
		}

		if int64(len(rows)) < endpointPageSize {
			return nil
		}

		after = rows[len(rows)-1].ID
	}
}

// ensureTenant returns the tenant's state, loading the routing cache on first use.
func (r *runner) ensureTenant(ctx context.Context, tenantId uuid.UUID) (*tenantState, error) {
	if ts, ok := r.tenants[tenantId]; ok {
		return ts, nil
	}

	ts := &tenantState{
		tenantId: tenantId,
		cache:    newRoutingCache(tenantId, r.repo.Endpoints(), r.enc, r.l),
		regs:     map[int32]*registration{},
		pollers:  map[uuid.UUID]*endpointPoller{},
		units:    map[int32]struct{}{},
	}

	if err := ts.cache.Load(ctx); err != nil {
		return nil, err
	}

	r.tenants[tenantId] = ts

	return ts, nil
}

// UnitsLost implements lease.Reconciler: stop the unit's pollers, then drain and close its
// registration in the background so the lease tick is not held for DrainTimeout. Action
// sets are untouched. A tenant with no owned units left is forgotten and released on the
// link once its last registration closes.
func (r *runner) UnitsLost(ctx context.Context, units []lease.Unit) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, unit := range units {
		ts, ok := r.tenants[unit.TenantId]

		if !ok {
			continue
		}

		delete(ts.units, unit.Shard)

		ts.mu.Lock()
		reg := ts.regs[unit.Shard]
		delete(ts.regs, unit.Shard)
		ts.mu.Unlock()

		r.reconcilePollers(ts)

		var onClosed func()

		if len(ts.units) == 0 {
			delete(r.tenants, unit.TenantId)
			onClosed = func() { r.releaseTenant(unit.TenantId) }
		}

		if reg == nil {
			if onClosed != nil {
				onClosed()
			}

			continue
		}

		r.wg.Add(1)

		go func() {
			defer r.wg.Done()

			reg.drain(r.cfg.DrainTimeout)
			reg.close()

			if onClosed != nil {
				onClosed()
			}
		}()
	}

	r.updateGaugesLocked()
}

func (r *runner) releaseTenant(tenantId uuid.UUID) {
	if releaser, ok := r.link.(link.TenantReleaser); ok {
		releaser.ReleaseTenant(tenantId)
	}
}

// InFlight implements lease.Reconciler.
func (r *runner) InFlight(unit lease.Unit) int {
	r.mu.Lock()
	ts, ok := r.tenants[unit.TenantId]
	r.mu.Unlock()

	if !ok {
		return 0
	}

	reg := ts.registration(unit.Shard)

	if reg == nil {
		return 0
	}

	return reg.inFlight()
}

// maintain runs every RoutingRefreshInterval: refresh each served tenant's cache (a full
// reload every RoutingFullReloadInterval, which drops deleted endpoints), reconcile pollers,
// reopen registrations that are missing (never opened, no token, stream failed) and push a
// changed action union to every registration for the tenant.
func (r *runner) maintain(ctx context.Context) {
	ticker := time.NewTicker(r.cfg.RoutingRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.maintainOnce(ctx)
		}
	}
}

func (r *runner) maintainOnce(ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for tenantId, ts := range r.tenants {
		var err error

		if time.Since(ts.cache.LastLoad()) >= r.cfg.RoutingFullReloadInterval {
			err = ts.cache.Load(ctx)
		} else {
			err = ts.cache.Refresh(ctx)
		}

		if err != nil {
			r.l.Error().Err(err).Str("tenant_id", tenantId.String()).Msg("could not refresh serverless routing cache")
			continue
		}

		for shard := range ts.units {
			if ts.registration(shard) != nil {
				continue
			}

			if err := r.openRegistration(ctx, ts, shard); err != nil {
				r.l.Debug().Err(err).Str("tenant_id", tenantId.String()).Int32("shard", shard).Msg("registration still not open")
			}
		}

		r.reconcilePollers(ts)
		r.syncTenantActions(ctx, ts)
	}

	r.updateGaugesLocked()
}

// syncTenantActions pushes the cached union to every registration for the tenant whose
// advertised set differs.
func (r *runner) syncTenantActions(ctx context.Context, ts *tenantState) {
	union := ts.cache.ActionUnion()

	ts.mu.Lock()
	regs := make([]*registration, 0, len(ts.regs))

	for _, reg := range ts.regs {
		regs = append(regs, reg)
	}

	ts.mu.Unlock()

	for _, reg := range regs {
		if err := reg.syncActions(ctx, union); err != nil {
			r.l.Error().Err(err).Str("tenant_id", ts.tenantId.String()).Int32("shard", reg.shard).Msg("could not update registration actions")
		}
	}
}

// Shutdown is the graceful stop after leases were released: stop pollers and action loops,
// drain deliveries up to DrainTimeout, close registrations, then wait for every goroutine.
func (r *runner) Shutdown() {
	r.mu.Lock()
	tenants := r.tenants
	r.tenants = map[uuid.UUID]*tenantState{}
	r.mu.Unlock()

	regs := make([]*registration, 0)

	for _, ts := range tenants {
		for _, poller := range ts.pollers {
			poller.stop()
		}

		ts.mu.Lock()

		for _, reg := range ts.regs {
			regs = append(regs, reg)
		}

		ts.regs = map[int32]*registration{}
		ts.mu.Unlock()
	}

	var wg sync.WaitGroup

	for _, reg := range regs {
		wg.Add(1)

		go func(reg *registration) {
			defer wg.Done()

			reg.drain(r.cfg.DrainTimeout)
			reg.close()
		}(reg)
	}

	wg.Wait()

	for tenantId := range tenants {
		r.releaseTenant(tenantId)
	}

	r.stopLoops()
	r.stopDelivery()
	r.wg.Wait()

	r.mu.Lock()
	r.updateGaugesLocked()
	r.mu.Unlock()
}

// updateGaugesLocked recomputes the registration, health and token gauges. Callers hold
// r.mu; paths that cannot (pollers, action loops) leave it to the next maintenance pass.
func (r *runner) updateGaugesLocked() {
	registrations := 0
	unhealthy := 0
	noToken := 0

	for _, ts := range r.tenants {
		ts.mu.Lock()
		registrations += len(ts.regs)
		ts.mu.Unlock()

		if ts.noToken {
			noToken++
		}

		for _, ep := range ts.ownedEndpoints() {
			cfg := ts.cache.Config(ep)

			if cfg.healthKnown && !cfg.healthy {
				unhealthy++
			}
		}
	}

	r.m.setRegistrationsOpen(registrations)
	r.m.setEndpointsUnhealthy(unhealthy)
	r.m.setTenantsWithoutToken(noToken)
}
