package repository

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type ListRateLimitOpts struct {
	// (optional) a search query for the key
	Search *string

	// (optional) number of events to skip
	Offset *int

	// (optional) number of events to return
	Limit *int

	// (optional) the order by field
	OrderBy *string `validate:"omitempty,oneof=key value limitValue"`

	// (optional) the order direction
	OrderDirection *string `validate:"omitempty,oneof=ASC DESC"`
}

type ListRateLimitsResult struct {
	Rows  []*sqlcv1.ListRateLimitsForTenantNoMutateRow
	Count int
}

type UpsertRateLimitOpts struct {
	// The rate limit max value
	Limit int

	// The rate limit duration
	Duration *string `validate:"omitnil,oneof=SECOND MINUTE HOUR DAY WEEK MONTH YEAR"`
}

type RateLimitDefinition struct {
	LimitValue int32
	Window     string
}

type RateLimitRepository interface {
	// definitions are upserted under the same advisory lock as the usage charge, so dynamic rate limit rows have a single multi-row writer
	FlushRateLimits(ctx context.Context, tenantId uuid.UUID, usage map[string]int, definitions map[string]RateLimitDefinition) ([]*sqlcv1.ListRateLimitsForTenantWithMutateRow, *time.Time, error)

	UpsertRateLimit(ctx context.Context, tenantId uuid.UUID, key string, opts *UpsertRateLimitOpts) (*sqlcv1.RateLimit, error)

	ListRateLimits(ctx context.Context, tenantId uuid.UUID, opts *ListRateLimitOpts) (*ListRateLimitsResult, error)

	DeleteRateLimits(ctx context.Context, tenantId uuid.UUID, key string) error

	// PauseRateLimit holds back queued runs that consume the key until the given time; an earlier time never shortens an active pause
	PauseRateLimit(ctx context.Context, tenantId uuid.UUID, key string, until time.Time) (*sqlcv1.V1RateLimitPause, error)
}

const MAX_TENANT_RATE_LIMITS = 10000

type rateLimitRepository struct {
	*sharedRepository
}

func newRateLimitRepository(shared *sharedRepository) *rateLimitRepository {
	return &rateLimitRepository{
		sharedRepository: shared,
	}
}

func (r *rateLimitRepository) FlushRateLimits(ctx context.Context, tenantId uuid.UUID, usage map[string]int, definitions map[string]RateLimitDefinition) ([]*sqlcv1.ListRateLimitsForTenantWithMutateRow, *time.Time, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

	if err != nil {
		return nil, nil, err
	}

	defer rollback()

	params := sqlcv1.ChargeRateLimitUsageParams{
		Tenantid: tenantId,
		Keys:     make([]string, 0, len(usage)),
		Units:    make([]int32, 0, len(usage)),
	}

	for k, v := range usage {
		params.Keys = append(params.Keys, k)
		params.Units = append(params.Units, int32(v)) // nolint: gosec
	}

	tenantInt := tenantAdvisoryInt(tenantId)

	err = r.queries.AdvisoryLock(ctx, tx, tenantInt)

	if err != nil {
		return nil, nil, err
	}

	if len(definitions) > 0 {
		upsertParams := sqlcv1.UpsertDynamicRateLimitDefinitionsParams{
			Tenantid:    tenantId,
			Keys:        make([]string, 0, len(definitions)),
			Limitvalues: make([]int32, 0, len(definitions)),
			Windows:     make([]string, 0, len(definitions)),
		}

		for k, def := range definitions {
			upsertParams.Keys = append(upsertParams.Keys, k)
			upsertParams.Limitvalues = append(upsertParams.Limitvalues, def.LimitValue)
			upsertParams.Windows = append(upsertParams.Windows, def.Window)
		}

		if err := r.queries.UpsertDynamicRateLimitDefinitions(ctx, tx, upsertParams); err != nil {
			return nil, nil, fmt.Errorf("could not bulk upsert dynamic rate limits: %w", err)
		}
	}

	_, err = r.queries.ChargeRateLimitUsage(ctx, tx, params)

	if err != nil {
		return nil, nil, err
	}

	if err := r.queries.SettleExpiredRateLimitPauses(ctx, tx, tenantId); err != nil {
		return nil, nil, fmt.Errorf("could not settle expired rate limit pauses: %w", err)
	}

	newRls, err := r.queries.ListRateLimitsForTenantWithMutate(ctx, tx, tenantId)

	if err != nil {
		return nil, nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, nil, err
	}

	res := make(map[string]int, len(newRls))

	for _, rl := range newRls {
		res[rl.Key] = int(rl.Value)
	}

	nextRefillAt := time.Now().Add(time.Second * 2)

	if len(newRls) > 0 {
		// get min of all next refill times
		for _, rl := range newRls {
			next := rl.NextRefillAt.Time

			// a paused key doesn't refill, and its window resumes when the pause ends
			if rl.PausedUntil.Valid {
				next = rl.PausedUntil.Time
			}

			if next.Before(nextRefillAt) {
				nextRefillAt = next
			}
		}
	}

	return newRls, &nextRefillAt, err
}

func (r *rateLimitRepository) UpsertRateLimit(ctx context.Context, tenantId uuid.UUID, key string, opts *UpsertRateLimitOpts) (*sqlcv1.RateLimit, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	upsertParams := sqlcv1.UpsertRateLimitParams{
		Tenantid: tenantId,
		Key:      key,
		Limit:    int32(opts.Limit), // nolint: gosec
	}

	if opts.Duration != nil {
		upsertParams.Window = sqlchelpers.TextFromStr(getWindowParamFromDurString(*opts.Duration))
	}

	rateLimit, err := r.queries.UpsertRateLimit(ctx, r.pool, upsertParams)

	if err != nil {
		return nil, fmt.Errorf("could not upsert rate limit: %w", err)
	}

	return rateLimit, nil
}

func (r *rateLimitRepository) ListRateLimits(ctx context.Context, tenantId uuid.UUID, opts *ListRateLimitOpts) (*ListRateLimitsResult, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	res := &ListRateLimitsResult{}

	queryParams := sqlcv1.ListRateLimitsForTenantNoMutateParams{
		Tenantid: tenantId,
	}

	countParams := sqlcv1.CountRateLimitsParams{
		Tenantid: tenantId,
	}

	if opts.Search != nil {
		queryParams.Search = sqlchelpers.TextFromStr(*opts.Search)
		countParams.Search = sqlchelpers.TextFromStr(*opts.Search)
	}

	if opts.Offset != nil {
		queryParams.Offset = *opts.Offset
	}

	if opts.Limit != nil {
		queryParams.Limit = *opts.Limit
	}

	orderByField := "key"
	orderByDirection := "ASC"

	if opts.OrderBy != nil {
		orderByField = *opts.OrderBy
	}

	if opts.OrderDirection != nil {
		orderByDirection = *opts.OrderDirection
	}

	queryParams.Orderby = orderByField + " " + orderByDirection
	countParams.Orderby = orderByField + " " + orderByDirection

	tx, err := r.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer sqlchelpers.DeferRollback(context.Background(), r.l, tx.Rollback)

	rls, err := r.queries.ListRateLimitsForTenantNoMutate(ctx, tx, queryParams)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			rls = make([]*sqlcv1.ListRateLimitsForTenantNoMutateRow, 0)
		} else {
			return nil, fmt.Errorf("could not list rate limits: %w", err)
		}
	}

	count, err := r.queries.CountRateLimits(ctx, tx, countParams)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			count = 0
		} else {
			return nil, fmt.Errorf("could not count events: %w", err)
		}
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	res.Rows = rls
	res.Count = int(count)

	return res, nil
}

func (r *rateLimitRepository) DeleteRateLimits(ctx context.Context, tenantId uuid.UUID, key string) error {
	return r.queries.DeleteRateLimitForTenant(ctx, r.pool,
		sqlcv1.DeleteRateLimitForTenantParams{
			Tenantid: tenantId,
			Key:      key,
		})
}

func tenantAdvisoryInt(tenantID uuid.UUID) int64 {
	hasher := fnv.New64a()
	idBytes := []byte(tenantID.String())
	hasher.Write(idBytes)
	return int64(hasher.Sum64()) // nolint: gosec
}

func (r *rateLimitRepository) PauseRateLimit(ctx context.Context, tenantId uuid.UUID, key string, until time.Time) (*sqlcv1.V1RateLimitPause, error) {
	if key == "" {
		return nil, errors.New("rate limit key is required")
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

	if err != nil {
		return nil, err
	}

	defer rollback()

	wasPaused, err := r.queries.HasActiveRateLimitPause(ctx, tx, sqlcv1.HasActiveRateLimitPauseParams{
		Tenantid: tenantId,
		Key:      pgtype.Text{String: key, Valid: true},
	})

	if err != nil {
		return nil, fmt.Errorf("could not check rate limit pause: %w", err)
	}

	pause, err := r.queries.PauseRateLimit(ctx, tx, sqlcv1.PauseRateLimitParams{
		Tenantid:    tenantId,
		Key:         key,
		Pauseduntil: sqlchelpers.TimestamptzFromTime(until),
	})

	if err != nil {
		return nil, fmt.Errorf("could not pause rate limit: %w", err)
	}

	// extending an active pause can't add waiting tasks, so only a new pause sweeps the tenant's queue
	if !wasPaused {
		tasks, err := r.queries.ListRateLimitPausedQueuedTasks(ctx, tx, tenantId)

		if err != nil {
			return nil, fmt.Errorf("could not list tasks behind paused rate limit: %w", err)
		}

		toHold := make([]TaskIdInsertedAtRetryCount, 0, len(tasks))

		for _, task := range tasks {
			toHold = append(toHold, TaskIdInsertedAtRetryCount{
				Id:         task.TaskID,
				InsertedAt: task.TaskInsertedAt,
				RetryCount: task.RetryCount,
			})
		}

		if err := r.holdRateLimitPausedTasks(ctx, tx, tenantId, toHold); err != nil {
			return nil, err
		}
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	if r.tenantHasRateLimitPauseCache != nil {
		r.tenantHasRateLimitPauseCache.Add(tenantId, true)
	}

	return pause, nil
}

func (s *sharedRepository) tenantHasRateLimitPause(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID) (bool, error) {
	if s.tenantHasRateLimitPauseCache != nil {
		if paused, ok := s.tenantHasRateLimitPauseCache.Get(tenantId); ok {
			return paused, nil
		}
	}

	paused, err := s.queries.HasActiveRateLimitPause(ctx, tx, sqlcv1.HasActiveRateLimitPauseParams{
		Tenantid: tenantId,
	})

	if err != nil {
		return false, fmt.Errorf("could not check rate limit pauses: %w", err)
	}

	if s.tenantHasRateLimitPauseCache != nil {
		s.tenantHasRateLimitPauseCache.Add(tenantId, paused)
	}

	return paused, nil
}

// holdRateLimitPausedTasks moves the given tasks that use a paused rate limit key into v1_paused_workflow_queue_item,
// releasing their concurrency slots, until ReleaseDueRateLimitPausedQueueItems finds the pause over
func (s *sharedRepository) holdRateLimitPausedTasks(ctx context.Context, tx sqlcv1.DBTX, tenantId uuid.UUID, tasks []TaskIdInsertedAtRetryCount) error {
	if len(tasks) == 0 {
		return nil
	}

	taskIds := make([]int64, 0, len(tasks))
	taskInsertedAts := make([]pgtype.Timestamptz, 0, len(tasks))
	retryCounts := make([]int32, 0, len(tasks))

	for _, task := range tasks {
		taskIds = append(taskIds, task.Id)
		taskInsertedAts = append(taskInsertedAts, task.InsertedAt)
		retryCounts = append(retryCounts, task.RetryCount)
	}

	if err := s.queries.MoveRateLimitPausedQueueItems(ctx, tx, sqlcv1.MoveRateLimitPausedQueueItemsParams{
		Taskids:         taskIds,
		Taskinsertedats: taskInsertedAts,
		Retrycounts:     retryCounts,
		Tenantid:        tenantId,
	}); err != nil {
		return fmt.Errorf("could not hold queue items behind paused rate limit: %w", err)
	}

	if err := s.queries.MoveRateLimitPausedRateLimitedQueueItems(ctx, tx, sqlcv1.MoveRateLimitPausedRateLimitedQueueItemsParams{
		Taskids:         taskIds,
		Taskinsertedats: taskInsertedAts,
		Retrycounts:     retryCounts,
		Tenantid:        tenantId,
	}); err != nil {
		return fmt.Errorf("could not hold rate limited queue items behind paused rate limit: %w", err)
	}

	if err := s.queries.MoveRateLimitPausedConcurrencySlots(ctx, tx, sqlcv1.MoveRateLimitPausedConcurrencySlotsParams{
		Taskids:         taskIds,
		Taskinsertedats: taskInsertedAts,
		Retrycounts:     retryCounts,
		Tenantid:        tenantId,
	}); err != nil {
		return fmt.Errorf("could not hold concurrency slots behind paused rate limit: %w", err)
	}

	return nil
}
