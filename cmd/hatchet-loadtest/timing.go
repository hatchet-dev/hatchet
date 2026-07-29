package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1" //nolint:staticcheck // SA1019: used only for REST timing queries in --externalWorker mode
)

// timingSeenTTL bounds the memory used by TimingCollector.seen: entries
// older than this are pruned on each sweep, since a run id is only needed
// long enough to avoid re-fetching its (already-terminal) timings.
const timingSeenTTL = 5 * time.Minute

// timingPageLimit is the page size used when listing workflow runs.
const timingPageLimit int64 = 100

// PhaseSample is one observation of the three latency phases for a single
// completed task, as derived from the engine's V1TaskTiming timestamps.
type PhaseSample struct {
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

		resp, err := api.WorkflowListWithResponse(ctx, tenantId, &rest.WorkflowListParams{Name: &name})
		if err != nil {
			return nil, nil, fmt.Errorf("error listing workflows for %q: %w", name, err)
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
	lastSeen     time.Time
	api          *rest.ClientWithResponses
	seen         map[uuid.UUID]time.Time
	workflowIds  []uuid.UUID
	pollInterval time.Duration
	mu           sync.Mutex
	tenantId     uuid.UUID
}

// NewTimingCollector builds a collector for already-resolved workflow ids.
func NewTimingCollector(hatchet v1.HatchetClient, workflowIds []uuid.UUID, pollInterval time.Duration) *TimingCollector { //nolint:staticcheck // SA1019
	return &TimingCollector{
		api:          hatchet.V0().API(),
		tenantId:     uuid.MustParse(hatchet.V0().TenantId()),
		workflowIds:  workflowIds,
		pollInterval: pollInterval,
		// Start the window slightly in the past so the first sweep can pick
		// up runs that finished just before the collector started.
		lastSeen: time.Now().Add(-pollInterval),
		seen:     make(map[uuid.UUID]time.Time),
	}
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
	// Overlap the window slightly backwards to tolerate clock skew /
	// commit-visibility lag between the engine marking a run terminal and it
	// showing up in a List query.
	since := c.lastSeen.Add(-2 * c.pollInterval)
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
			return
		}

		if resp.JSON200 == nil {
			return
		}

		rows := resp.JSON200.Rows

		for _, row := range rows {
			runId := row.WorkflowRunExternalId

			c.mu.Lock()
			_, alreadySeen := c.seen[runId]
			if !alreadySeen {
				c.seen[runId] = now
			}
			c.mu.Unlock()

			if alreadySeen {
				continue
			}

			c.fetchTimings(ctx, runId, out)
		}

		if int64(len(rows)) < timingPageLimit {
			break
		}

		offset += timingPageLimit
	}

	c.mu.Lock()
	c.lastSeen = now
	for id, seenAt := range c.seen {
		if now.Sub(seenAt) > timingSeenTTL {
			delete(c.seen, id)
		}
	}
	c.mu.Unlock()
}

func (c *TimingCollector) fetchTimings(ctx context.Context, runId uuid.UUID, out chan<- PhaseSample) {
	var depth int64

	resp, err := c.api.V1WorkflowRunGetTimingsWithResponse(ctx, runId, &rest.V1WorkflowRunGetTimingsParams{Depth: &depth})
	if err != nil {
		l.Warn().Err(err).Str("workflow_run_id", runId.String()).Msg("timing collector: error fetching task timings")
		return
	}

	if resp.JSON200 == nil {
		return
	}

	for _, row := range resp.JSON200.Rows {
		if row.QueuedAt == nil || row.StartedAt == nil || row.FinishedAt == nil {
			// Not fully timed (e.g. failed before being queued/started) -
			// skip rather than error the whole run's worth of samples.
			continue
		}

		sample := PhaseSample{
			Queued:     row.QueuedAt.Sub(row.TaskInsertedAt),
			Scheduling: row.StartedAt.Sub(*row.QueuedAt),
			Execution:  row.FinishedAt.Sub(*row.StartedAt),
		}

		select {
		case out <- sample:
		case <-ctx.Done():
			return
		}
	}
}
