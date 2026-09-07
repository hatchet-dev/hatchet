package serverlessoperator

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	// unhealthyAfterFailures is how many consecutive healthcheck failures flip an endpoint
	// unhealthy. One success flips it back.
	unhealthyAfterFailures = 3

	// noTokenStatusError is written once on the endpoints of a tenant the link cannot
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

	// putHashes remembers the canonical hash of every workflow put through the current
	// registration, by namespaced name, so a response change re-puts only what changed.
	putHashes map[string]string
	putReg    *registration
	lastHash  string
	failures  int
}

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

func (p *endpointPoller) pollOnce(ctx context.Context) {
	cfg := p.ts.cache.Config(p.ep)

	if !cfg.enabled {
		return
	}

	if err := p.r.acquireHealthcheckSlot(ctx); err != nil {
		return
	}

	defer p.r.releaseHealthcheckSlot()

	hctx, cancel := context.WithTimeout(ctx, p.r.cfg.HealthcheckTimeout)
	defer cancel()

	start := time.Now()

	res, err := pollHealthcheck(hctx, p.r.sender, p.ep, cfg)

	if ctx.Err() != nil {
		return
	}

	if err != nil {
		p.r.m.healthcheck("error", time.Since(start))
		p.failures++

		if p.failures == unhealthyAfterFailures {
			p.r.l.Warn().Err(err).Str("endpoint_id", p.ep.id.String()).Msg("serverless endpoint unhealthy")
		}

		if p.failures >= unhealthyAfterFailures && (cfg.healthy || !cfg.healthKnown) {
			p.r.writeStatus(ctx, p.ts, p.ep, false, err.Error())
		}

		return
	}

	p.r.m.healthcheck("ok", time.Since(start))
	p.failures = 0

	if !cfg.healthy || !cfg.healthKnown {
		p.r.writeStatus(ctx, p.ts, p.ep, true, "")
	}

	if res.hash == p.lastHash {
		return
	}

	reg := p.ts.registration(p.ep.shard)

	if reg == nil {
		// The unit's registration is not open (no token, engine unreachable): the change is
		// retried on the next poll once it is. Not a health transition of the endpoint.
		p.r.l.Debug().Str("endpoint_id", p.ep.id.String()).Msg("healthcheck changed but no registration is open yet")
		return
	}

	if err := p.applyChange(ctx, reg, res); err != nil {
		// lastHash stays unset so the next poll retries the registration.
		p.r.l.Error().Err(err).Str("endpoint_id", p.ep.id.String()).Msg("could not register serverless endpoint workflows")
		p.r.writeStatus(ctx, p.ts, p.ep, false, err.Error())

		return
	}

	p.lastHash = res.hash
}

// applyChange registers the endpoint's workflows through the unit's registration with the
// tenant's full action set, records registered_actions when it changed, and pushes the new
// union to every other registration for the tenant on this process. Other processes pick the
// union up from the database on their next cache refresh.
func (p *endpointPoller) applyChange(ctx context.Context, reg *registration, res *healthcheckResult) error {
	before := p.ts.cache.Config(p.ep).registeredActions
	unionChanged := p.ts.cache.SetHealthcheck(p.ep.id, res.workflows, res.actions)
	union := p.ts.cache.ActionUnion()

	// A registration opened after the last put carries the cached workflows itself, so the
	// put memory is scoped to the registration that received them.
	if p.putReg != reg {
		p.putHashes = map[string]string{}
		p.putReg = reg
	}

	for i, wf := range res.workflows {
		if p.putHashes[wf.Name] == res.workflowHashes[i] {
			continue
		}

		if err := reg.putWorkflow(ctx, wf, union); err != nil {
			return fmt.Errorf("engine rejected workflow %s: %w", wf.Name, err)
		}

		p.putHashes[wf.Name] = res.workflowHashes[i]
	}

	if !stringsEqual(before, res.actions) {
		if err := p.r.repo.Endpoints().UpdateRegisteredActions(ctx, p.ep.id, res.actions); err != nil {
			return fmt.Errorf("could not write registered actions: %w", err)
		}
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

// writeStatus records a health transition: one row write, mirrored into the cache so a
// refresh cannot resurrect the previous value.
func (r *runner) writeStatus(ctx context.Context, ts *tenantState, ep *cachedEndpoint, healthy bool, statusError string) {
	var errPtr *string

	if statusError != "" {
		errPtr = &statusError
	}

	if err := r.repo.Endpoints().UpdateStatus(ctx, ep.id, healthy, errPtr); err != nil {
		r.l.Error().Err(err).Str("endpoint_id", ep.id.String()).Msg("could not write serverless endpoint status")
		return
	}

	ts.cache.SetStatus(ep.id, healthy, statusError)
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

func (r *runner) acquireHealthcheckSlot(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case r.hcSem <- struct{}{}:
		return nil
	}
}

func (r *runner) releaseHealthcheckSlot() {
	<-r.hcSem
}

// ownedEndpoints are the tenant's endpoints on shards this process owns.
func (ts *tenantState) ownedEndpoints() []*cachedEndpoint {
	out := make([]*cachedEndpoint, 0)

	for _, ep := range ts.cache.Endpoints() {
		if _, ok := ts.units[ep.shard]; ok {
			out = append(out, ep)
		}
	}

	return out
}

// reconcilePollers starts pollers for owned, enabled endpoints without one and stops pollers
// whose endpoint is no longer owned, enabled or present. With no token nothing is polled.
func (r *runner) reconcilePollers(ts *tenantState) {
	desired := map[uuid.UUID]*cachedEndpoint{}

	if !ts.noToken {
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
