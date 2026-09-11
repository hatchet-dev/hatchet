package serverlessoperator

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/lease"
)

// tenantReleaser is implemented by hosts that hold per-tenant state outside sessions, such
// as the gRPC host's client cache. The runner calls ReleaseTenant when it owns no more units
// of the tenant, after every session for the tenant is closed. The in-process host holds no
// such state and does not implement it.
type tenantReleaser interface {
	ReleaseTenant(tenantId uuid.UUID)
}

// endpointPageSize is the keyset page size for loading a gained unit's endpoints.
const endpointPageSize int64 = 500

// tenantState is everything the process keeps for a served tenant: the routing cache, the
// one registration the tenant's owned units share, the pollers of owned endpoints and whether
// the host could authenticate as the tenant.
//
// opMu serializes the operations on the tenant (gaining and losing units, maintenance,
// shutdown), which include network work and poller shutdown waits; it is never held together
// with the runner's lock. mu guards the fields read from outside those operations: reg,
// units and loaded.
type tenantState struct {
	cache    *routingCache
	reg      *registration
	pollers  map[uuid.UUID]*endpointPoller
	units    map[int32]struct{}
	hcSem    chan struct{}
	tenantId uuid.UUID
	opMu     sync.Mutex
	mu       sync.Mutex
	noToken  atomic.Bool
	loaded   bool
	removed  bool
}

func (ts *tenantState) registration() *registration {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	return ts.reg
}

// setRegistration installs reg and returns the one it replaced.
func (ts *tenantState) setRegistration(reg *registration) *registration {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	prev := ts.reg
	ts.reg = reg

	return prev
}

// detachRegistration removes reg if it is the current one and reports whether it was.
func (ts *tenantState) detachRegistration(reg *registration) bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.reg != reg {
		return false
	}

	ts.reg = nil

	return true
}

// ownedShards snapshots the owned shard set.
func (ts *tenantState) ownedShards() map[int32]struct{} {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	out := make(map[int32]struct{}, len(ts.units))

	for shard := range ts.units {
		out[shard] = struct{}{}
	}

	return out
}

func (ts *tenantState) unitCount() int {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	return len(ts.units)
}

// runner turns owned units into registrations and endpoint pollers and routes assigned
// actions to endpoints. It implements lease.Reconciler.
//
// mu guards tenants and pending only and is held for map operations, never across network
// work: each tenant's work runs under its own opMu, so a hung engine or a slow poller
// shutdown on one tenant does not stall the others or the lease tick.
type runner struct {
	repo         repository.ServerlessRepository
	host         operator.Host
	kind         sqlcv1.V1OperatorKind
	workerName   string
	enc          encryption.EncryptionService
	sender       RequestSender
	l            *zerolog.Logger
	m            *metrics
	tenants      map[uuid.UUID]*tenantState
	pending      map[lease.Unit]struct{}
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
	// Pollers live on loopCtx and are stopped first at shutdown; deliveries live on
	// deliveryCtx, which is cancelled only after the drain timeout.
	loopCtx, stopLoops := context.WithCancel(context.Background())
	deliveryCtx, cancel := context.WithCancel(context.Background())

	return &runner{
		repo:         deps.Repo,
		host:         deps.Host,
		kind:         deps.OperatorKind,
		workerName:   deps.WorkerName,
		enc:          deps.Encryption,
		sender:       deps.Sender,
		l:            deps.Logger,
		m:            m,
		cfg:          cfg,
		processId:    deps.ProcessId,
		tenants:      map[uuid.UUID]*tenantState{},
		pending:      map[lease.Unit]struct{}{},
		hcSem:        make(chan struct{}, cfg.HealthcheckConcurrency),
		loopCtx:      loopCtx,
		stopLoops:    stopLoops,
		deliveryCtx:  deliveryCtx,
		stopDelivery: cancel,
	}
}

// groupUnits splits units by tenant.
func groupUnits(units []lease.Unit) map[uuid.UUID][]int32 {
	out := map[uuid.UUID][]int32{}

	for _, unit := range units {
		out[unit.TenantId] = append(out[unit.TenantId], unit.Shard)
	}

	return out
}

// tenantFor returns the tenant's state, creating an unloaded one on first use.
func (r *runner) tenantFor(tenantId uuid.UUID) *tenantState {
	r.mu.Lock()
	defer r.mu.Unlock()

	if ts, ok := r.tenants[tenantId]; ok {
		return ts
	}

	ts := &tenantState{
		tenantId: tenantId,
		cache:    newRoutingCache(tenantId, r.repo.Endpoints(), r.enc, r.l),
		pollers:  map[uuid.UUID]*endpointPoller{},
		units:    map[int32]struct{}{},
		hcSem:    make(chan struct{}, r.cfg.HealthcheckTenantConcurrency),
	}

	r.tenants[tenantId] = ts

	return ts
}

func (r *runner) tenant(tenantId uuid.UUID) *tenantState {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.tenants[tenantId]
}

// forgetTenant drops ts from the served tenants if it is still the registered state.
func (r *runner) forgetTenant(ts *tenantState) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.tenants[ts.tenantId] == ts {
		delete(r.tenants, ts.tenantId)
	}
}

// deferUnits records gained units whose tenant could not be loaded; maintenance retries
// them, so a claimed unit is never left owned but unserved.
func (r *runner) deferUnits(tenantId uuid.UUID, shards []int32) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, shard := range shards {
		r.pending[lease.Unit{TenantId: tenantId, Shard: shard}] = struct{}{}
	}
}

func (r *runner) clearPending(tenantId uuid.UUID, shards []int32) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, shard := range shards {
		delete(r.pending, lease.Unit{TenantId: tenantId, Shard: shard})
	}
}

func (r *runner) pendingUnits() []lease.Unit {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]lease.Unit, 0, len(r.pending))

	for unit := range r.pending {
		out = append(out, unit)
	}

	return out
}

// UnitsGained implements lease.Reconciler: for each tenant, load its routing cache (or the
// gained units' endpoints when it is already served), open its registration if it has none
// and start pollers. A tenant whose load fails keeps its units pending for maintenance to
// retry; a registration that fails to open is retried by maintenance as well. One batch
// gains its tenants concurrently, MaintenanceConcurrency at a time: a startup or a takeover
// gains many tenants at once, and each tenant's gain is a load and a Host.Open, so in series
// the lease tick would wait through every one of them.
func (r *runner) UnitsGained(ctx context.Context, units []lease.Unit) {
	r.gainGroups(ctx, groupUnits(units))
	r.updateGauges()
}

// gainGroups gains every tenant's units on a bounded pool and returns once all are done. A
// tenant's gain runs under its own opMu, so the pool never runs two gains of one tenant.
func (r *runner) gainGroups(ctx context.Context, groups map[uuid.UUID][]int32) {
	sem := make(chan struct{}, r.cfg.MaintenanceConcurrency)

	var wg sync.WaitGroup

	for tenantId, shards := range groups {
		wg.Add(1)

		go func(tenantId uuid.UUID, shards []int32) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			r.gainUnits(ctx, tenantId, shards)
		}(tenantId, shards)
	}

	wg.Wait()
}

func (r *runner) gainUnits(ctx context.Context, tenantId uuid.UUID, shards []int32) {
	for {
		ts := r.tenantFor(tenantId)

		ts.opMu.Lock()

		if ts.removed {
			// The tenant's last unit was lost while this gain waited; start over with a
			// fresh state.
			ts.opMu.Unlock()
			continue
		}

		r.gainUnitsLocked(ctx, ts, shards)
		ts.opMu.Unlock()

		return
	}
}

// gainUnitsLocked runs under ts.opMu.
func (r *runner) gainUnitsLocked(ctx context.Context, ts *tenantState, shards []int32) {
	if err := r.loadTenant(ctx, ts, shards); err != nil {
		r.l.Error().Err(err).Str("tenant_id", ts.tenantId.String()).Msg("could not load tenant routing cache; the gained units are retried by maintenance")
		r.deferUnits(ts.tenantId, shards)

		if !ts.loaded && ts.unitCount() == 0 {
			r.forgetTenant(ts)
		}

		return
	}

	r.clearPending(ts.tenantId, shards)

	ts.mu.Lock()

	for _, shard := range shards {
		ts.units[shard] = struct{}{}
	}

	ts.mu.Unlock()

	if ts.registration() == nil {
		if err := r.openRegistration(ctx, ts); err != nil {
			r.l.Warn().Err(err).Str("tenant_id", ts.tenantId.String()).Msg("registration not opened; will retry")
		}
	}

	r.reconcilePollers(ts)
}

// loadTenant loads the tenant's routing cache on first use, and afterwards refreshes the
// gained units' endpoints so their pollers start from current rows. Runs under ts.opMu.
func (r *runner) loadTenant(ctx context.Context, ts *tenantState, shards []int32) error {
	if !ts.loaded {
		if err := ts.cache.Load(ctx); err != nil {
			return err
		}

		ts.mu.Lock()
		ts.loaded = true
		ts.mu.Unlock()

		return nil
	}

	units := make([]lease.Unit, 0, len(shards))

	for _, shard := range shards {
		units = append(units, lease.Unit{TenantId: ts.tenantId, Shard: shard})
	}

	return r.loadUnitEndpoints(ctx, ts, units)
}

// loadUnitEndpoints pages through the units' endpoints; a page is applied under one cache
// lock and publishes at most one union revision.
func (r *runner) loadUnitEndpoints(ctx context.Context, ts *tenantState, units []lease.Unit) error {
	after := uuid.Nil

	for {
		rows, err := r.repo.Endpoints().ListForUnits(ctx, units, after, endpointPageSize)

		if err != nil {
			return err
		}

		ts.cache.mu.Lock()
		ts.cache.applyRowsLocked(rows)
		ts.cache.mu.Unlock()

		if int64(len(rows)) < endpointPageSize {
			return nil
		}

		after = rows[len(rows)-1].ID
	}
}

// UnitsLost implements lease.Reconciler: stop the lost units' pollers; the tenant's
// registration stays while it still owns units. A tenant with no owned units left is
// forgotten, its registration paused, drained and closed in the background so the lease tick
// is not held for DrainTimeout, and released on the host once that is done. Action sets are
// untouched.
func (r *runner) UnitsLost(ctx context.Context, units []lease.Unit) {
	for tenantId, shards := range groupUnits(units) {
		r.clearPending(tenantId, shards)

		ts := r.tenant(tenantId)

		if ts == nil {
			continue
		}

		ts.opMu.Lock()

		if ts.removed {
			ts.opMu.Unlock()
			continue
		}

		ts.mu.Lock()

		for _, shard := range shards {
			delete(ts.units, shard)
		}

		remaining := len(ts.units)
		ts.mu.Unlock()

		r.reconcilePollers(ts)

		if remaining > 0 {
			ts.opMu.Unlock()
			continue
		}

		ts.removed = true
		r.forgetTenant(ts)
		reg := ts.setRegistration(nil)
		ts.opMu.Unlock()

		r.wg.Add(1)

		go func() {
			defer r.wg.Done()

			if reg != nil {
				reg.teardown(r.cfg.DrainTimeout)
			}

			r.releaseTenant(tenantId)
		}()
	}

	r.updateGauges()
}

func (r *runner) releaseTenant(tenantId uuid.UUID) {
	if releaser, ok := r.host.(tenantReleaser); ok {
		releaser.ReleaseTenant(tenantId)
	}
}

// InFlight implements lease.Reconciler. Deliveries belong to the tenant's registration, not
// to a unit: losing one of several units stops that unit's pollers and leaves the
// registration, and every delivery on it, where it is. So a unit reports no deliveries while
// the tenant owns others, and shedding it costs nothing in flight. Only the tenant's last
// unit reports the registration's count, since losing it closes the registration, which
// drains for at most DrainTimeout and then aborts what is left; the leaser sheds such a unit
// after the idle ones.
func (r *runner) InFlight(unit lease.Unit) int {
	ts := r.tenant(unit.TenantId)

	if ts == nil {
		return 0
	}

	if ts.unitCount() > 1 {
		return 0
	}

	reg := ts.registration()

	if reg == nil {
		return 0
	}

	return reg.inFlight()
}

// maintain runs every RoutingRefreshInterval: refresh each served tenant's cache (a full
// reload every RoutingFullReloadInterval, which drops deleted endpoints), reconcile pollers,
// reopen registrations that are missing (never opened, no token), push a changed action
// union to the registration, and retry units whose tenant failed to load.
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
	tenants := make([]*tenantState, 0, len(r.tenants))

	for _, ts := range r.tenants {
		tenants = append(tenants, ts)
	}

	r.mu.Unlock()

	sem := make(chan struct{}, r.cfg.MaintenanceConcurrency)

	var wg sync.WaitGroup

	for _, ts := range tenants {
		wg.Add(1)

		go func(ts *tenantState) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			r.maintainTenant(ctx, ts)
		}(ts)
	}

	wg.Wait()

	r.gainGroups(ctx, groupUnits(r.pendingUnits()))
	r.updateGauges()
}

func (r *runner) maintainTenant(ctx context.Context, ts *tenantState) {
	ts.opMu.Lock()
	defer ts.opMu.Unlock()

	if ts.removed || ts.unitCount() == 0 {
		return
	}

	var err error

	switch {
	case !ts.loaded:
		err = ts.cache.Load(ctx)

		if err == nil {
			ts.mu.Lock()
			ts.loaded = true
			ts.mu.Unlock()
		}
	case time.Since(ts.cache.LastLoad()) >= r.cfg.RoutingFullReloadInterval:
		err = ts.cache.Load(ctx)
	default:
		err = ts.cache.Refresh(ctx)
	}

	if err != nil {
		r.l.Error().Err(err).Str("tenant_id", ts.tenantId.String()).Msg("could not refresh serverless routing cache")
		return
	}

	if ts.registration() == nil {
		if err := r.openRegistration(ctx, ts); err != nil {
			r.l.Debug().Err(err).Str("tenant_id", ts.tenantId.String()).Msg("registration still not open")
		}
	}

	r.reconcilePollers(ts)
	r.syncTenantActions(ctx, ts)
}

// syncTenantActions pushes the cached union to the tenant's registration when its
// advertised revision is behind.
func (r *runner) syncTenantActions(ctx context.Context, ts *tenantState) {
	reg := ts.registration()

	if reg == nil {
		return
	}

	if err := reg.syncActions(ctx, ts.cache); err != nil {
		r.l.Error().Err(err).Str("tenant_id", ts.tenantId.String()).Msg("could not update registration actions")
	}
}

// Shutdown is the graceful stop after leases were released: stop pollers, pause every
// registration's worker, drain deliveries up to DrainTimeout, close the sessions, then wait
// for every goroutine.
func (r *runner) Shutdown() {
	r.mu.Lock()
	tenants := r.tenants
	r.tenants = map[uuid.UUID]*tenantState{}
	r.pending = map[lease.Unit]struct{}{}
	r.mu.Unlock()

	regs := make([]*registration, 0, len(tenants))

	for _, ts := range tenants {
		ts.opMu.Lock()
		ts.removed = true

		for id, poller := range ts.pollers {
			delete(ts.pollers, id)
			poller.stop()
		}

		if reg := ts.setRegistration(nil); reg != nil {
			regs = append(regs, reg)
		}

		ts.opMu.Unlock()
	}

	var wg sync.WaitGroup

	for _, reg := range regs {
		wg.Add(1)

		go func(reg *registration) {
			defer wg.Done()

			reg.teardown(r.cfg.DrainTimeout)
		}(reg)
	}

	wg.Wait()

	for tenantId := range tenants {
		r.releaseTenant(tenantId)
	}

	r.stopLoops()
	r.stopDelivery()
	r.wg.Wait()

	r.updateGauges()
}

// updateGauges recomputes the registration, health and token gauges from a snapshot of the
// served tenants.
func (r *runner) updateGauges() {
	r.mu.Lock()
	tenants := make([]*tenantState, 0, len(r.tenants))

	for _, ts := range r.tenants {
		tenants = append(tenants, ts)
	}

	r.mu.Unlock()

	registrations := 0
	unhealthy := 0
	noToken := 0

	for _, ts := range tenants {
		if ts.registration() != nil {
			registrations++
		}

		if ts.noToken.Load() {
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
