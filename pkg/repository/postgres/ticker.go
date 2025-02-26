package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type tickerRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
}

func NewTickerRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.TickerEngineRepository {
	queries := dbsqlc.New()

	return &tickerRepository{
		pool:    pool,
		v:       v,
		queries: queries,
		l:       l,
	}
}

func (t *tickerRepository) CreateNewTicker(ctx context.Context, opts *repository.CreateTickerOpts) (*dbsqlc.Ticker, error) {
	if err := t.v.Validate(opts); err != nil {
		return nil, err
	}

	return t.queries.CreateTicker(ctx, t.pool, sqlchelpers.UUIDFromStr(opts.ID))
}

func (t *tickerRepository) UpdateTicker(ctx context.Context, tickerId string, opts *repository.UpdateTickerOpts) (*dbsqlc.Ticker, error) {
	if err := t.v.Validate(opts); err != nil {
		return nil, err
	}

	return t.queries.UpdateTicker(
		ctx,
		t.pool,
		dbsqlc.UpdateTickerParams{
			ID:              sqlchelpers.UUIDFromStr(tickerId),
			LastHeartbeatAt: sqlchelpers.TimestampFromTime(opts.LastHeartbeatAt.UTC()),
		},
	)
}

func (t *tickerRepository) ListTickers(ctx context.Context, opts *repository.ListTickerOpts) ([]*dbsqlc.Ticker, error) {
	if err := t.v.Validate(opts); err != nil {
		return nil, err
	}

	params := dbsqlc.ListTickersParams{}

	if opts.LatestHeartbeatAfter != nil {
		params.LastHeartbeatAfter = sqlchelpers.TimestampFromTime(opts.LatestHeartbeatAfter.UTC())
	}

	if opts.Active != nil {
		params.IsActive = *opts.Active
	}

	return t.queries.ListTickers(
		ctx,
		t.pool,
		params,
	)
}

func (t *tickerRepository) DeactivateTicker(ctx context.Context, tickerId string) error {
	_, err := t.queries.DeactivateTicker(
		ctx,
		t.pool,
		sqlchelpers.UUIDFromStr(tickerId),
	)

	return err
}

func (t *tickerRepository) PollGetGroupKeyRuns(ctx context.Context, tickerId string) ([]*dbsqlc.GetGroupKeyRun, error) {
	return t.queries.PollGetGroupKeyRuns(ctx, t.pool, sqlchelpers.UUIDFromStr(tickerId))
}

func (t *tickerRepository) PollCronSchedules(ctx context.Context, tickerId string) ([]*dbsqlc.PollCronSchedulesRow, error) {
	return t.queries.PollCronSchedules(ctx, t.pool, sqlchelpers.UUIDFromStr(tickerId))
}

func (t *tickerRepository) PollScheduledWorkflows(ctx context.Context, tickerId string) ([]*dbsqlc.PollScheduledWorkflowsRow, error) {
	return t.queries.PollScheduledWorkflows(ctx, t.pool, sqlchelpers.UUIDFromStr(tickerId))
}

func (t *tickerRepository) PollTenantAlerts(ctx context.Context, tickerId string) ([]*dbsqlc.PollTenantAlertsRow, error) {
	return t.queries.PollTenantAlerts(ctx, t.pool, sqlchelpers.UUIDFromStr(tickerId))
}

func (t *tickerRepository) PollExpiringTokens(ctx context.Context) ([]*dbsqlc.PollExpiringTokensRow, error) {
	return t.queries.PollExpiringTokens(ctx, t.pool)
}

func (t *tickerRepository) PollTenantResourceLimitAlerts(ctx context.Context) ([]*dbsqlc.TenantResourceLimitAlert, error) {
	return t.queries.PollTenantResourceLimitAlerts(ctx, t.pool)
}

func (t *tickerRepository) PollUnresolvedFailedStepRuns(ctx context.Context) ([]*dbsqlc.PollUnresolvedFailedStepRunsRow, error) {
	return t.queries.PollUnresolvedFailedStepRuns(ctx, t.pool)
}
