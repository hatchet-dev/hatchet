package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/loadtest/eventkeys"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1" //nolint:staticcheck // SA1019: used only for REST timing queries in --externalWorker mode
)

// timingPendingTTL bounds how long a run stays in TimingCollector.pending:
// a pending run is given up on (with a warning) after this long of repeated
// fetch failures, so a permanently-broken run id doesn't retry forever.
const timingPendingTTL = 5 * time.Minute

// timingFetchConcurrency bounds how many V1WorkflowRunGetTimings calls the
// collector has in flight at once. Fetching one run's timings per REST
// round-trip is inherently an N+1 query pattern (there's no batch timings
// endpoint), so at load-test throughputs a single sequential fetch loop
// falls further behind every sweep and never catches up. Fetching
// concurrently, bounded by this limit, lets the collector actually drain
// the backlog instead of piling it up until it's abandoned at shutdown.
const timingFetchConcurrency = 50

// timingPollInterval is how often the collector re-lists for newly
// completed workflow runs.
const timingPollInterval = 2 * time.Second

// timingPageLimit is the page size used when listing workflow runs.
const timingPageLimit int64 = 100

// PhaseSample is one observation of the three latency phases for a single
// completed task, as derived from the engine's V1TaskTiming timestamps.
type PhaseSample struct {
	EventKey   eventkeys.EventKey
	Queued     time.Duration
	Scheduling time.Duration
	Execution  time.Duration
}

func applyNamespace(name, namespace string) string {
	if namespace == "" || strings.HasPrefix(name, namespace) {
		return name
	}
	return namespace + name
}

func ResolveWorkflowIDs(ctx context.Context, api *rest.ClientWithResponses, tenantId uuid.UUID, names []string, waitTimeout time.Duration) ([]uuid.UUID, error) {
	deadline := time.Now().Add(waitTimeout)

	for {
		ids, missing, err := tryResolveWorkflowIDs(ctx, api, tenantId, names)

		if err != nil {
			return nil, err
		}

		if len(missing) == 0 {
			return ids, nil
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out after %s waiting for workflow(s) %v to be registered - make sure the external SDK worker is running and has registered these tasks", waitTimeout, missing)
		}

		l.Info().Msgf("externalWorker: waiting for workflow(s) %v to be registered...", missing)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func tryResolveWorkflowIDs(ctx context.Context, api *rest.ClientWithResponses, tenantId uuid.UUID, names []string) (ids []uuid.UUID, missing []string, err error) {
	for _, name := range names {
		name := name

		resp, reqErr := api.WorkflowListWithResponse(ctx, tenantId, &rest.WorkflowListParams{Name: &name})
		if reqErr != nil {
			l.Info().Msgf("error listing workflows for %q, will retry: %v", name, reqErr)
			missing = append(missing, name)
			continue
		}

		found := false

		if resp.JSON200 != nil && resp.JSON200.Rows != nil {
			for _, wf := range *resp.JSON200.Rows {
				if wf.Name == name {
					id, err := uuid.Parse(wf.Metadata.Id)
					if err != nil {
						return nil, nil, fmt.Errorf("invalid workflow id %q for workflow %q: %w", wf.Metadata.Id, name, err)
					}
					ids = append(ids, id)
					found = true
					break
				}
			}
		}

		if !found {
			missing = append(missing, name)
		}
	}

	return ids, missing, nil
}

// TimingCollector discovers completed workflow runs for a set of already-
// resolved workflow ids and turns their V1TaskTiming rows into PhaseSample
// values, via the engine's REST API (V1WorkflowRunList +
// V1WorkflowRunGetTimings) - language agnostic, since discovery never
// touches the worker process at all.
type TimingCollector struct {
	windowStart  time.Time
	api          *rest.ClientWithResponses
	seen         map[uuid.UUID]time.Time          // successfully fetched, or deliberately skipped by sampling; value is decision time
	pending      map[uuid.UUID]time.Time          // sampled-in and awaiting a successful fetch; value is first-discovered time
	pendingKeys  map[uuid.UUID]eventkeys.EventKey // event key per pending run, so the concurrent fetch can attribute its samples
	workflowIds  []uuid.UUID
	workflowKeys map[uuid.UUID]eventkeys.EventKey
	pollInterval time.Duration
	sampleRate   float64 // proportion (0, 1] of discovered runs to fetch full timings for; 1 = every run
	sampleAcc    float64 // accumulator for deterministic proportional sampling, see sweep()
	discovered   int64   // count of runs discovered so far
	mu           sync.Mutex
	tenantId     uuid.UUID
}

// NewTimingCollector builds a collector for already-resolved workflow ids.
// sampleRate is the proportion of discovered runs to fetch full timings for
// - e.g. 0.3 fetches roughly 30% of runs, chosen deterministically so the
// long-run proportion converges to sampleRate exactly. sampleRate <= 0 or >
// 1 is invalid and clamps to 1 (every run), the safe direction since it
// never drops data. Sampling only a proportion of runs still lets the
// average converge, at a fraction of the REST load on the engine.
func NewTimingCollector(hatchet v1.HatchetClient, workflowIds []uuid.UUID, workflowKeys map[uuid.UUID]eventkeys.EventKey, pollInterval time.Duration, sampleRate float64) *TimingCollector { //nolint:staticcheck // SA1019
	if sampleRate <= 0 || sampleRate > 1 {
		sampleRate = 1
	}

	return &TimingCollector{
		api:          hatchet.V0().API(),
		tenantId:     uuid.MustParse(hatchet.V0().TenantId()),
		workflowIds:  workflowIds,
		workflowKeys: workflowKeys,
		pollInterval: pollInterval,
		sampleRate:   sampleRate,
		// Start the window slightly in the past so the first sweep can pick
		// up runs that were created just before the collector started.
		windowStart: time.Now().Add(-pollInterval),
		seen:        make(map[uuid.UUID]time.Time),
		pending:     make(map[uuid.UUID]time.Time),
		pendingKeys: make(map[uuid.UUID]eventkeys.EventKey),
	}
}

// Pending reports how many discovered runs are still awaiting a successful
// fetch (in flight or queued for retry). Callers that need every result -
// rather than whatever happened to be fetched before some fixed deadline -
// can poll this and only tear the collector down once it's been 0 for a
// full poll cycle (see do.go).
func (c *TimingCollector) Pending() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.pending)
}

// Run polls until ctx is done, sending a PhaseSample on out for every task
// timing row with a full queued/scheduling/execution triple.
func (c *TimingCollector) Run(ctx context.Context, out chan<- PhaseSample) {
	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()

	// Don't wait a full interval before the first check.
	c.sweep(ctx, out)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.sweep(ctx, out)
		}
	}
}

func (c *TimingCollector) sweep(ctx context.Context, out chan<- PhaseSample) {
	now := time.Now()

	c.mu.Lock()
	// Anchored at the collector's start (see windowStart) so late-finishing
	// runs stay in range for the whole test rather than sliding out of a
	// trailing window before they go terminal.
	since := c.windowStart
	c.mu.Unlock()

	statuses := []rest.V1TaskStatus{rest.V1TaskStatusCOMPLETED, rest.V1TaskStatusFAILED}

	var offset int64

	for {
		limit := timingPageLimit

		params := &rest.V1WorkflowRunListParams{
			Since:       since,
			Until:       &now,
			WorkflowIds: &c.workflowIds,
			Statuses:    &statuses,
			Offset:      &offset,
			Limit:       &limit,
		}

		resp, err := c.api.V1WorkflowRunListWithResponse(ctx, c.tenantId, params)
		if err != nil {
			l.Warn().Err(err).Msg("timing collector: error listing workflow runs")
			break
		}

		if resp.JSON200 == nil {
			break
		}

		rows := resp.JSON200.Rows

		c.mu.Lock()
		for _, row := range rows {
			runId := row.WorkflowRunExternalId

			// Already fetched (or skipped by sampling), or already
			// queued/in-flight from an earlier sweep (including one still
			// being retried after a prior failure) - don't reconsider it.
			if _, ok := c.seen[runId]; ok {
				continue
			}
			if _, ok := c.pending[runId]; ok {
				continue
			}

			c.discovered++

			// Deterministic proportional sampling: accumulate sampleRate per
			// discovered run and select one whenever the accumulator crosses
			// 1, then subtract 1 - like a Bresenham line, this keeps the
			// selected fraction converging to sampleRate exactly rather than
			// drifting with random variance. The first run is always
			// selected (regardless of sampleRate) so a short test still
			// gets at least one sample instead of a spurious "no timing
			// samples observed" error. Runs that aren't selected are marked
			// seen immediately (not fetched, never retried) so they don't
			// keep costing a List-window reconsideration every sweep.
			c.sampleAcc += c.sampleRate
			selected := c.sampleAcc >= 1
			if selected {
				c.sampleAcc--
			}
			if c.discovered == 1 {
				selected = true
			}

			if selected {
				c.pending[runId] = now
				c.pendingKeys[runId] = c.workflowKeys[row.WorkflowId]
			} else {
				c.seen[runId] = now
			}
		}
		c.mu.Unlock()

		if int64(len(rows)) < timingPageLimit {
			break
		}

		offset += timingPageLimit
	}

	// Fetch every currently-pending run - newly discovered above, plus any
	// carried over from a previous sweep whose fetch failed - concurrently,
	// bounded by timingFetchConcurrency. A run is only removed from
	// `pending` (and added to `seen`) once its fetch actually succeeds, so a
	// failed fetch (including one aborted by ctx cancellation on shutdown)
	// just gets retried on the next sweep instead of silently and
	// permanently dropping that run's samples.
	c.mu.Lock()
	toFetch := make([]uuid.UUID, 0, len(c.pending))
	for id := range c.pending {
		toFetch = append(toFetch, id)
	}
	c.mu.Unlock()

	var wg errgroup.Group
	wg.SetLimit(timingFetchConcurrency)

	for _, runId := range toFetch {
		runId := runId

		wg.Go(func() error {
			c.mu.Lock()
			key := c.pendingKeys[runId]
			c.mu.Unlock()

			if err := c.fetchTimings(ctx, runId, key, out); err != nil {
				l.Warn().Err(err).Str("workflow_run_id", runId.String()).Msg("timing collector: error fetching task timings")
				return nil
			}

			c.mu.Lock()
			delete(c.pending, runId)
			delete(c.pendingKeys, runId)
			c.seen[runId] = time.Now()
			c.mu.Unlock()

			return nil
		})
	}

	_ = wg.Wait() // fetchTimings never returns a non-nil error to the group; failures are handled (and logged) above

	c.mu.Lock()
	for id, firstSeen := range c.pending {
		if now.Sub(firstSeen) > timingPendingTTL {
			l.Warn().Str("workflow_run_id", id.String()).Msg("timing collector: giving up on workflow run after repeated fetch failures")
			delete(c.pending, id)
			delete(c.pendingKeys, id)
		}
	}
	c.mu.Unlock()
}

func (c *TimingCollector) fetchTimings(ctx context.Context, runId uuid.UUID, key eventkeys.EventKey, out chan<- PhaseSample) error {
	var depth int64

	resp, err := c.api.V1WorkflowRunGetTimingsWithResponse(ctx, runId, &rest.V1WorkflowRunGetTimingsParams{Depth: &depth})
	if err != nil {
		return err
	}

	if resp.JSON200 == nil {
		return fmt.Errorf("unexpected response: %s", resp.Status())
	}

	for _, row := range resp.JSON200.Rows {
		if row.QueuedAt == nil || row.StartedAt == nil || row.FinishedAt == nil {
			// Not fully timed (e.g. failed before being queued/started) -
			// skip rather than error the whole run's worth of samples.
			continue
		}

		sample := PhaseSample{
			EventKey:   key,
			Queued:     row.QueuedAt.Sub(row.TaskInsertedAt),
			Scheduling: row.StartedAt.Sub(*row.QueuedAt),
			Execution:  row.FinishedAt.Sub(*row.StartedAt),
		}

		select {
		case out <- sample:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}
