package serverlessoperator

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	// unhealthyAfterFailures is how many consecutive healthcheck failures flip an endpoint
	// unhealthy. One success flips it back.
	unhealthyAfterFailures = 3

	// noTokenStatusError is written once on the endpoints of a tenant the host cannot
	// authenticate as.
	noTokenStatusError = "no token for tenant"
)

// endpointPoller healthchecks one owned endpoint on its own schedule and, when the response
// changes, registers the workflows through the unit's registration and writes
// registered_actions. Only the owning process runs a poller for an endpoint.
type endpointPoller struct {
	r      *runner
	ts     *tenantState
	ep     *cachedEndpoint
	cancel context.CancelFunc
	done   chan struct{}

	// putHashes remembers the canonical hash of every workflow this poller put, by
	// namespaced name, so a response change re-puts only what changed. It is pruned to the
	// names of the last catalog applied, accepted or not, so a stream of new names cannot
	// grow it past the catalog cap.
	putHashes map[string]string

	// registered is what this poller knows the endpoint row's registered_actions to be: the
	// cached value when the first change is applied, then every set it wrote. The write is
	// decided against it rather than the cache, since SetHealthcheck moves the cache ahead of
	// the row before the write and a failed attempt must not leave the write skipped.
	registered      []string
	registeredKnown bool
	lastHash        string

	// rejectedHash is the response hash the engine or the host last refused, with the
	// backoff before the same catalog is tried again; an unchanged rejected catalog is not
	// re-put on every poll.
	rejectedHash    string
	rejectedErr     string
	rejectedRetryAt time.Time
	rejectedBackoff time.Duration
	failures        int
}

// rejectedBackoffMax caps how long an unchanged rejected catalog waits before a retry.
const rejectedBackoffMax = time.Hour

// catalogRejected marks an error the engine gave for the catalog itself, which the same
// catalog will get again: it backs off. Transport and engine failures are retried on the next
// poll.
type catalogRejected struct{ err error }

func (e catalogRejected) Error() string { return e.err.Error() }
func (e catalogRejected) Unwrap() error { return e.err }

func newEndpointPoller(r *runner, ts *tenantState, ep *cachedEndpoint) *endpointPoller {
	return &endpointPoller{r: r, ts: ts, ep: ep, done: make(chan struct{}), putHashes: map[string]string{}}
}

func (p *endpointPoller) start(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	p.cancel = cancel

	p.r.wg.Add(1)

	go func() {
		defer p.r.wg.Done()
		defer close(p.done)

		p.run(ctx)
	}()
}

// stop cancels the poller and waits for its current poll to finish.
func (p *endpointPoller) stop() {
	p.cancel()
	<-p.done
}

func (p *endpointPoller) run(ctx context.Context) {
	// The first poll is spread out too: a gained unit starts every poller at once.
	select {
	case <-ctx.Done():
		return
	case <-time.After(firstPollDelay(p.ts.cache.Config(p.ep).pollIntervalSeconds)):
	}

	for {
		p.pollOnce(ctx)

		cfg := p.ts.cache.Config(p.ep)

		select {
		case <-ctx.Done():
			return
		case <-time.After(pollInterval(cfg.pollIntervalSeconds)):
		}
	}
}

// pollStatus is the outcome of one poll: the health the endpoint row should show.
type pollStatus struct {
	err     string
	healthy bool
}

// pollOnce runs one healthcheck and applies its result, then writes the endpoint's status
// once if the outcome differs from what the row shows. An endpoint whose catalog the engine
// keeps rejecting is unhealthy with the engine's error and stays so without further writes.
func (p *endpointPoller) pollOnce(ctx context.Context) {
	cfg := p.ts.cache.Config(p.ep)

	if !cfg.enabled {
		return
	}

	if err := p.acquireSlots(ctx); err != nil {
		return
	}

	hctx, cancel := context.WithTimeout(ctx, p.r.cfg.HealthcheckTimeout)
	start := time.Now()

	res, err := pollHealthcheck(hctx, p.r.sender, p.ep, cfg, p.r.catalogLimits())

	cancel()
	p.releaseSlots()

	if ctx.Err() != nil {
		return
	}

	if err != nil {
		p.r.m.healthcheck("error", time.Since(start))
		p.failures++

		if p.failures == unhealthyAfterFailures {
			p.r.l.Warn().Err(err).Str("endpoint_id", p.ep.id.String()).Msg("serverless endpoint unhealthy")
		}

		if p.failures >= unhealthyAfterFailures {
			p.writeStatusIfChanged(ctx, cfg, pollStatus{healthy: false, err: err.Error()})
		}

		return
	}

	p.r.m.healthcheck("ok", time.Since(start))
	p.failures = 0

	p.writeStatusIfChanged(ctx, cfg, p.apply(ctx, res))
}

// apply registers a changed catalog and returns the resulting status: healthy when the
// catalog is unchanged or was accepted, unhealthy with the error when it was refused.
func (p *endpointPoller) apply(ctx context.Context, res *healthcheckResult) pollStatus {
	if res.hash == p.lastHash {
		return pollStatus{healthy: true}
	}

	if res.hash == p.rejectedHash && time.Now().Before(p.rejectedRetryAt) {
		return pollStatus{healthy: false, err: p.rejectedErr}
	}

	reg := p.ts.registration()

	if reg == nil {
		// The unit's registration is not open (no token, engine unreachable): the change is
		// retried on the next poll once it is. Not a health transition of the endpoint.
		p.r.l.Debug().Str("endpoint_id", p.ep.id.String()).Msg("healthcheck changed but no registration is open yet")
		return pollStatus{healthy: true}
	}

	// Applying a catalog is bounded on its own: the workflow puts and the action delta must
	// not run on the lifecycle context alone.
	actx, cancel := context.WithTimeout(ctx, p.r.cfg.HealthcheckApplyTimeout)
	defer cancel()

	if err := p.applyChange(actx, reg, res); err != nil {
		p.r.l.Error().Err(err).Str("endpoint_id", p.ep.id.String()).Msg("could not register serverless endpoint workflows")

		var rejection catalogRejected

		if errors.As(err, &rejection) {
			p.rejected(res.hash, err.Error())
		}

		return pollStatus{healthy: false, err: err.Error()}
	}

	p.lastHash = res.hash
	p.rejectedHash = ""

	return pollStatus{healthy: true}
}

// rejected records a refused catalog and doubles the backoff for the same hash.
func (p *endpointPoller) rejected(hash, errMsg string) {
	if p.rejectedHash != hash {
		p.rejectedBackoff = 0
	}

	interval := pollInterval(p.ts.cache.Config(p.ep).pollIntervalSeconds)
	p.rejectedBackoff = max(min(p.rejectedBackoff*2, rejectedBackoffMax), interval)
	p.rejectedHash = hash
	p.rejectedErr = errMsg
	p.rejectedRetryAt = time.Now().Add(p.rejectedBackoff)
}

// writeStatusIfChanged writes the status when it differs from the cached row.
func (p *endpointPoller) writeStatusIfChanged(ctx context.Context, cfg *endpointConfig, status pollStatus) {
	if cfg.healthKnown && cfg.healthy == status.healthy && cfg.statusError == status.err {
		return
	}

	p.r.writeStatus(ctx, p.ts, p.ep, status.healthy, status.err)
}

// applyChange puts the endpoint's changed workflows through the unit's registration, moves
// the cached union to the new action set, pushes the resulting delta to the owner's
// registration, records registered_actions when it changed, and pushes the delta to every
// other registration for the tenant on this process. Other processes pick the union up from
// the database on their next cache refresh. A refused delta restores the endpoint's previous
// contribution, so a rejected catalog never stays in the shared union.
func (p *endpointPoller) applyChange(ctx context.Context, reg *registration, res *healthcheckResult) error {
	// Whatever the outcome, the remembered puts are those of this catalog: an accepted
	// early workflow of a catalog the engine then rejected is still worth not re-putting on
	// the retry, but names from earlier catalogs are not, and a run of rejected catalogs
	// with fresh names must not accumulate.
	defer p.prunePutHashes(res)

	for i, wf := range res.workflows {
		if p.putHashes[wf.Name] == res.workflowHashes[i] {
			continue
		}

		if _, err := reg.session.PutWorkflow(ctx, wf); err != nil {
			return catalogRejected{err: fmt.Errorf("engine rejected workflow %s: %w", wf.Name, err)}
		}

		p.putHashes[wf.Name] = res.workflowHashes[i]
	}

	if !p.registeredKnown {
		p.registered = p.ts.cache.Config(p.ep).registeredActions
		p.registeredKnown = true
	}

	previous := p.ts.cache.Config(p.ep).registeredActions
	unionChanged := p.ts.cache.SetHealthcheck(p.ep.id, res.actions)

	// The owner's registration is the one that runs this endpoint's tasks, so its delta is
	// part of the change: a failure here is retried on the next poll like a rejected put.
	if err := reg.syncActions(ctx, p.ts.cache); err != nil {
		p.ts.cache.SetHealthcheck(p.ep.id, previous)
		return err
	}

	if !stringsEqual(p.registered, res.actions) {
		if err := p.r.repo.Endpoints().UpdateRegisteredActions(ctx, p.ep.id, res.actions); err != nil {
			return fmt.Errorf("could not write registered actions: %w", err)
		}

		p.registered = res.actions
	}

	p.r.l.Info().
		Str("endpoint_id", p.ep.id.String()).
		Int("workflows", len(res.workflows)).
		Int("actions", len(res.actions)).
		Msg("serverless endpoint workflows registered")

	if unionChanged {
		p.r.syncTenantActions(ctx, p.ts)
	}

	return nil
}

// prunePutHashes forgets workflows the catalog does not name.
func (p *endpointPoller) prunePutHashes(res *healthcheckResult) {
	keep := make(map[string]struct{}, len(res.workflows))

	for _, wf := range res.workflows {
		keep[wf.Name] = struct{}{}
	}

	for name := range p.putHashes {
		if _, ok := keep[name]; !ok {
			delete(p.putHashes, name)
		}
	}
}

// writeStatus records a health transition: one row write, mirrored into the cache with the
// write's database timestamp so a refresh returning an older row cannot resurrect the
// previous value.
func (r *runner) writeStatus(ctx context.Context, ts *tenantState, ep *cachedEndpoint, healthy bool, statusError string) {
	var errPtr *string

	if statusError != "" {
		errPtr = &statusError
	}

	changedAt, err := r.repo.Endpoints().UpdateStatus(ctx, ep.id, healthy, errPtr)

	if err != nil {
		r.l.Error().Err(err).Str("endpoint_id", ep.id.String()).Msg("could not write serverless endpoint status")
		return
	}

	ts.cache.SetStatus(ep.id, healthy, statusError, changedAt)
}

// markNoToken writes the no-token status error on the tenant's owned endpoints that do not
// carry it yet. Nothing is polled or registered until the token appears.
func (r *runner) markNoToken(ctx context.Context, ts *tenantState) {
	for _, ep := range ts.ownedEndpoints() {
		cfg := ts.cache.Config(ep)

		if cfg.statusError == noTokenStatusError && cfg.healthKnown && !cfg.healthy {
			continue
		}

		r.writeStatus(ctx, ts, ep, false, noTokenStatusError)
	}
}

// acquireSlots takes the tenant's healthcheck slot, then a process-wide one, so a tenant
// with many slow endpoints holds at most its own share of the process limit.
func (p *endpointPoller) acquireSlots(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case p.ts.hcSem <- struct{}{}:
	}

	select {
	case <-ctx.Done():
		<-p.ts.hcSem
		return ctx.Err()
	case p.r.hcSem <- struct{}{}:
		return nil
	}
}

func (p *endpointPoller) releaseSlots() {
	<-p.r.hcSem
	<-p.ts.hcSem
}

func (r *runner) catalogLimits() catalogLimits {
	return catalogLimits{maxWorkflows: r.cfg.MaxWorkflowsPerEndpoint, maxActions: r.cfg.MaxActionsPerEndpoint}
}

// ownedEndpoints are the tenant's endpoints on shards this process owns, in no particular
// order: a snapshot of the tenant is never sorted just to filter it.
func (ts *tenantState) ownedEndpoints() []*cachedEndpoint {
	return ts.cache.endpointsOnShards(ts.ownedShards())
}

// reconcilePollers starts pollers for owned, enabled endpoints without one and stops pollers
// whose endpoint is no longer owned, enabled or present. With no token nothing is polled.
// Runs under ts.opMu; stopping a poller waits for its current poll.
func (r *runner) reconcilePollers(ts *tenantState) {
	desired := map[uuid.UUID]*cachedEndpoint{}

	if !ts.noToken.Load() {
		for _, ep := range ts.ownedEndpoints() {
			if ts.cache.Config(ep).enabled {
				desired[ep.id] = ep
			}
		}
	}

	for id, poller := range ts.pollers {
		if _, ok := desired[id]; ok {
			continue
		}

		delete(ts.pollers, id)
		poller.stop()
	}

	for id, ep := range desired {
		if _, ok := ts.pollers[id]; ok {
			continue
		}

		poller := newEndpointPoller(r, ts, ep)
		ts.pollers[id] = poller
		poller.start(r.loopCtx)
	}
}
