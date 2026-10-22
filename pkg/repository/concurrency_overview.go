package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// strategy summaries are read on every dashboard poll, so they are cached briefly per tenant
const concurrencyStrategySummaryCacheDuration = 5 * time.Second

// ConcurrencyScope is where a concurrency strategy is defined: on a task, on a workflow, or
// shared across tasks at the tenant level.
type ConcurrencyScope string

const (
	ConcurrencyScopeShared   ConcurrencyScope = "SHARED"
	ConcurrencyScopeWorkflow ConcurrencyScope = "WORKFLOW"
	ConcurrencyScopeTask     ConcurrencyScope = "TASK"
)

// ErrConcurrencyStrategyNotFound is returned when a strategy does not exist in the given scope.
var ErrConcurrencyStrategyNotFound = errors.New("concurrency strategy not found")

// ConcurrencyStrategyUsage is a task that uses a shared strategy.
type ConcurrencyStrategyUsage struct {
	WorkflowID   uuid.UUID
	WorkflowName string
	TaskName     string
}

// ConcurrencyStrategySummary is a concurrency strategy with its live slot counts.
type ConcurrencyStrategySummary struct {
	ID    int64
	Scope ConcurrencyScope

	Name string

	// Unset for shared strategies.
	WorkflowID        *uuid.UUID
	WorkflowVersionID *uuid.UUID
	WorkflowName      string
	WorkflowVersion   string

	// Only set for task strategies.
	TaskName string

	// Only set for shared strategies.
	UsedBy []ConcurrencyStrategyUsage

	Expression        string
	Strategy          sqlcv1.V1ConcurrencyStrategy
	MaxRuns           int32
	MaxRunsExpression *string

	KeyCount        int64
	QueuedKeyCount  int64
	RunningCount    int64
	QueuedCount     int64
	OldestQueuedAt  *time.Time
	OldestQueuedKey string
}

// ListConcurrencyKeysOpts filters and pages the keys returned by ListKeys.
type ListConcurrencyKeysOpts struct {
	Search     *string
	OnlyQueued bool
	Limit      int64 `validate:"min=1,max=1000"`
	Offset     int64 `validate:"min=0"`
}

// ConcurrencyKey is a key of a strategy with its running and queued run counts.
type ConcurrencyKey struct {
	Key            string
	MaxRuns        int32
	RunningCount   int64
	QueuedCount    int64
	OldestQueuedAt *time.Time
}

// ConcurrencyOverviewRepository reads concurrency strategies and their keys for the dashboard.
type ConcurrencyOverviewRepository interface {
	// ListStrategySummaries returns the strategies on the tenant's latest workflow versions,
	// and the shared strategies their tasks use, with live slot counts, most queued first.
	ListStrategySummaries(ctx context.Context, tenantId uuid.UUID) ([]*ConcurrencyStrategySummary, error)

	// ListKeys returns a page of a strategy's keys and the total number of matching keys.
	// Returns ErrConcurrencyStrategyNotFound if the strategy does not exist in the given scope.
	ListKeys(ctx context.Context, tenantId uuid.UUID, scope ConcurrencyScope, strategyId int64, opts ListConcurrencyKeysOpts) ([]*ConcurrencyKey, int64, error)
}

type concurrencyOverviewRepository struct {
	*sharedRepository

	summaryCache *cache.Cache
}

func newConcurrencyOverviewRepository(shared *sharedRepository) (ConcurrencyOverviewRepository, func()) {
	summaryCache := cache.New(concurrencyStrategySummaryCacheDuration)

	return &concurrencyOverviewRepository{
		sharedRepository: shared,
		summaryCache:     summaryCache,
	}, summaryCache.Stop
}

type slotStats struct {
	keyCount, queuedKeyCount, runningCount, queuedCount int64
	oldestQueuedAt                                      pgtype.Timestamptz
	oldestQueuedKey                                     string
}

func (s slotStats) apply(sum *ConcurrencyStrategySummary) {
	sum.KeyCount = s.keyCount
	sum.QueuedKeyCount = s.queuedKeyCount
	sum.RunningCount = s.runningCount
	sum.QueuedCount = s.queuedCount
	sum.OldestQueuedAt = timePtr(s.oldestQueuedAt)
	sum.OldestQueuedKey = s.oldestQueuedKey
}

func (r *concurrencyOverviewRepository) ListStrategySummaries(ctx context.Context, tenantId uuid.UUID) ([]*ConcurrencyStrategySummary, error) {
	res, err := cache.MakeCacheable(r.summaryCache, tenantId.String(), func() (*[]*ConcurrencyStrategySummary, error) {
		summaries, err := r.listStrategySummaries(ctx, tenantId)
		if err != nil {
			return nil, err
		}

		return &summaries, nil
	})

	if err != nil {
		return nil, err
	}

	return *res, nil
}

func (r *concurrencyOverviewRepository) listStrategySummaries(ctx context.Context, tenantId uuid.UUID) ([]*ConcurrencyStrategySummary, error) {
	taskStrategies, err := r.queries.ListTaskConcurrencyStrategiesForTenant(ctx, r.pool, tenantId)
	if err != nil {
		return nil, fmt.Errorf("failed to list task concurrency strategies: %w", err)
	}

	workflowStrategies, err := r.queries.ListWorkflowConcurrencyStrategiesForTenant(ctx, r.pool, tenantId)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow concurrency strategies: %w", err)
	}

	usage, err := r.queries.ListTenantConcurrencyStrategyUsage(ctx, r.pool, tenantId)
	if err != nil {
		return nil, fmt.Errorf("failed to list shared concurrency strategy usage: %w", err)
	}

	usedBy := make(map[int64][]ConcurrencyStrategyUsage)
	sharedIds := make([]int64, 0)

	for _, row := range usage {
		if _, ok := usedBy[row.TenantStrategyID]; !ok {
			sharedIds = append(sharedIds, row.TenantStrategyID)
		}

		usedBy[row.TenantStrategyID] = append(usedBy[row.TenantStrategyID], ConcurrencyStrategyUsage{
			WorkflowID:   row.WorkflowID,
			WorkflowName: row.WorkflowName,
			TaskName:     row.StepReadableID,
		})
	}

	sharedStrategies, err := r.queries.GetTenantConcurrencyStrategiesByIds(ctx, r.pool, sqlcv1.GetTenantConcurrencyStrategiesByIdsParams{
		Tenantid: tenantId,
		Ids:      sharedIds,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get shared concurrency strategies: %w", err)
	}

	slotStatRows, err := r.queries.ListConcurrencySlotStatsForTenant(ctx, r.pool, tenantId)
	if err != nil {
		return nil, fmt.Errorf("failed to list concurrency slot stats: %w", err)
	}

	workflowSlotStatRows, err := r.queries.ListWorkflowConcurrencySlotStatsForTenant(ctx, r.pool, tenantId)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow concurrency slot stats: %w", err)
	}

	// task and shared strategy ids share a sequence; workflow strategy ids have their own
	stats := make(map[int64]slotStats, len(slotStatRows))

	for _, row := range slotStatRows {
		stats[row.StrategyID] = slotStats{row.KeyCount, row.QueuedKeyCount, row.RunningCount, row.QueuedCount, row.OldestQueuedAt, row.OldestQueuedKey}
	}

	workflowStats := make(map[int64]slotStats, len(workflowSlotStatRows))

	for _, row := range workflowSlotStatRows {
		workflowStats[row.StrategyID] = slotStats{row.KeyCount, row.QueuedKeyCount, row.RunningCount, row.QueuedCount, row.OldestQueuedAt, row.OldestQueuedKey}
	}

	res := make([]*ConcurrencyStrategySummary, 0, len(taskStrategies)+len(workflowStrategies)+len(sharedStrategies))

	for _, s := range sharedStrategies {
		sum := &ConcurrencyStrategySummary{
			ID:                s.ID,
			Scope:             ConcurrencyScopeShared,
			Name:              s.Name,
			UsedBy:            usedBy[s.ID],
			Expression:        s.Expression,
			Strategy:          s.Strategy,
			MaxRuns:           s.MaxConcurrency,
			MaxRunsExpression: textPtr(s.MaxRunsExpression),
		}

		stats[s.ID].apply(sum)
		res = append(res, sum)
	}

	for _, s := range workflowStrategies {
		sum := &ConcurrencyStrategySummary{
			ID:                s.ID,
			Scope:             ConcurrencyScopeWorkflow,
			Name:              s.WorkflowName,
			WorkflowID:        &s.WorkflowID,
			WorkflowVersionID: &s.WorkflowVersionID,
			WorkflowName:      s.WorkflowName,
			WorkflowVersion:   s.WorkflowVersion.String,
			Expression:        s.Expression,
			Strategy:          s.Strategy,
			MaxRuns:           s.MaxConcurrency,
		}

		workflowStats[s.ID].apply(sum)
		res = append(res, sum)
	}

	for _, s := range taskStrategies {
		sum := &ConcurrencyStrategySummary{
			ID:                s.ID,
			Scope:             ConcurrencyScopeTask,
			Name:              s.StepReadableID,
			WorkflowID:        &s.WorkflowID,
			WorkflowVersionID: &s.WorkflowVersionID,
			WorkflowName:      s.WorkflowName,
			WorkflowVersion:   s.WorkflowVersion.String,
			TaskName:          s.StepReadableID,
			Expression:        s.Expression,
			Strategy:          s.Strategy,
			MaxRuns:           s.MaxConcurrency,
			MaxRunsExpression: textPtr(s.MaxRunsExpression),
		}

		stats[s.ID].apply(sum)
		res = append(res, sum)
	}

	sort.SliceStable(res, func(i, j int) bool {
		a, b := res[i], res[j]

		if a.QueuedCount != b.QueuedCount {
			return a.QueuedCount > b.QueuedCount
		}

		if (a.OldestQueuedAt == nil) != (b.OldestQueuedAt == nil) {
			return a.OldestQueuedAt != nil
		}

		if a.OldestQueuedAt != nil && !a.OldestQueuedAt.Equal(*b.OldestQueuedAt) {
			return a.OldestQueuedAt.Before(*b.OldestQueuedAt)
		}

		return a.Name < b.Name
	})

	return res, nil
}

func (r *concurrencyOverviewRepository) ListKeys(ctx context.Context, tenantId uuid.UUID, scope ConcurrencyScope, strategyId int64, opts ListConcurrencyKeysOpts) ([]*ConcurrencyKey, int64, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, 0, err
	}

	// The strategy tables have no index for a (tenant, id) lookup, so the strategy is
	// resolved from the cached summaries.
	summaries, err := r.ListStrategySummaries(ctx, tenantId)
	if err != nil {
		return nil, 0, err
	}

	var strategy *ConcurrencyStrategySummary

	for _, s := range summaries {
		if s.Scope == scope && s.ID == strategyId {
			strategy = s
			break
		}
	}

	if strategy == nil {
		return nil, 0, ErrConcurrencyStrategyNotFound
	}

	search := pgtype.Text{}

	if opts.Search != nil && *opts.Search != "" {
		search = pgtype.Text{String: *opts.Search, Valid: true}
	}

	if scope == ConcurrencyScopeWorkflow {
		rows, err := r.queries.ListWorkflowConcurrencyKeys(ctx, r.pool, sqlcv1.ListWorkflowConcurrencyKeysParams{
			Tenantid:   tenantId,
			Strategyid: strategyId,
			Search:     search,
			Onlyqueued: opts.OnlyQueued,
			Keyoffset:  opts.Offset,
			Keylimit:   opts.Limit,
		})

		if err != nil {
			return nil, 0, fmt.Errorf("failed to list workflow concurrency keys: %w", err)
		}

		total, err := r.queries.CountWorkflowConcurrencyKeys(ctx, r.pool, sqlcv1.CountWorkflowConcurrencyKeysParams{
			Tenantid:   tenantId,
			Strategyid: strategyId,
			Search:     search,
			Onlyqueued: opts.OnlyQueued,
		})

		if err != nil {
			return nil, 0, fmt.Errorf("failed to count workflow concurrency keys: %w", err)
		}

		res := make([]*ConcurrencyKey, len(rows))

		for i, row := range rows {
			res[i] = &ConcurrencyKey{
				Key:            row.Key,
				MaxRuns:        strategy.MaxRuns,
				RunningCount:   row.RunningCount,
				QueuedCount:    row.QueuedCount,
				OldestQueuedAt: timePtr(row.OldestQueuedAt),
			}
		}

		return res, total, nil
	}

	rows, err := r.queries.ListConcurrencyKeys(ctx, r.pool, sqlcv1.ListConcurrencyKeysParams{
		Tenantid:       tenantId,
		Strategyid:     strategyId,
		Defaultmaxruns: strategy.MaxRuns,
		Search:         search,
		Onlyqueued:     opts.OnlyQueued,
		Keyoffset:      opts.Offset,
		Keylimit:       opts.Limit,
	})

	if err != nil {
		return nil, 0, fmt.Errorf("failed to list concurrency keys: %w", err)
	}

	total, err := r.queries.CountConcurrencyKeys(ctx, r.pool, sqlcv1.CountConcurrencyKeysParams{
		Tenantid:   tenantId,
		Strategyid: strategyId,
		Search:     search,
		Onlyqueued: opts.OnlyQueued,
	})

	if err != nil {
		return nil, 0, fmt.Errorf("failed to count concurrency keys: %w", err)
	}

	res := make([]*ConcurrencyKey, len(rows))

	for i, row := range rows {
		res[i] = &ConcurrencyKey{
			Key:            row.Key,
			MaxRuns:        row.MaxRuns,
			RunningCount:   row.RunningCount,
			QueuedCount:    row.QueuedCount,
			OldestQueuedAt: timePtr(row.OldestQueuedAt),
		}
	}

	return res, total, nil
}

func timePtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}

	return &t.Time
}

func textPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}

	return &t.String
}
