package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type CreateTickerOpts struct {
	ID uuid.UUID `validate:"required"`
}

type UpdateTickerOpts struct {
	LastHeartbeatAt *time.Time
}

type ListTickerOpts struct {
	// Set this to only return tickers whose heartbeat is more recent than this time
	LatestHeartbeatAfter *time.Time

	Active *bool
}

type TickerRepository interface {
	IsTenantAlertActive(ctx context.Context, tenantId uuid.UUID) (bool, time.Time, error)

	// CreateNewTicker creates a new ticker.
	CreateNewTicker(ctx context.Context, opts *CreateTickerOpts) (*sqlcv1.Ticker, error)

	// UpdateTicker updates a ticker.
	UpdateTicker(ctx context.Context, tickerId uuid.UUID, opts *UpdateTickerOpts) (*sqlcv1.Ticker, error)

	// ListTickers lists tickers.
	ListTickers(ctx context.Context, opts *ListTickerOpts) ([]*sqlcv1.Ticker, error)

	// DeactivateTicker deletes a ticker.
	DeactivateTicker(ctx context.Context, tickerId uuid.UUID) error

	// PollCronSchedules returns all cron schedules which should be managed by the ticker
	PollCronSchedules(ctx context.Context, tickerId uuid.UUID) ([]*sqlcv1.PollCronSchedulesRow, error)

	PollScheduledWorkflows(ctx context.Context, tickerId uuid.UUID) ([]*sqlcv1.PollScheduledWorkflowsRow, error)

	PollTenantAlerts(ctx context.Context, tickerId uuid.UUID) ([]*sqlcv1.PollTenantAlertsRow, error)

	PollExpiringTokens(ctx context.Context) ([]*sqlcv1.PollExpiringTokensRow, error)

	PollTenantResourceLimitAlerts(ctx context.Context) ([]*sqlcv1.TenantResourceLimitAlert, error)
}

type tickerRepository struct {
	*sharedRepository
}

func newTickerRepository(shared *sharedRepository) TickerRepository {
	return &tickerRepository{
		sharedRepository: shared,
	}
}

func (t *tickerRepository) IsTenantAlertActive(ctx context.Context, tenantId uuid.UUID) (bool, time.Time, error) {
	res, err := t.queries.IsTenantAlertActive(ctx, t.pool, tenantId)

	if err != nil {
		return false, time.Now(), err
	}

	return res.IsActive, res.LastAlertedAt.Time, nil
}

func (t *tickerRepository) CreateNewTicker(ctx context.Context, opts *CreateTickerOpts) (*sqlcv1.Ticker, error) {
	if err := t.v.Validate(opts); err != nil {
		return nil, err
	}

	return t.queries.CreateTicker(ctx, t.pool, opts.ID)
}

func (t *tickerRepository) UpdateTicker(ctx context.Context, tickerId uuid.UUID, opts *UpdateTickerOpts) (*sqlcv1.Ticker, error) {
	if err := t.v.Validate(opts); err != nil {
		return nil, err
	}

	return t.queries.UpdateTicker(
		ctx,
		t.pool,
		sqlcv1.UpdateTickerParams{
			ID:              tickerId,
			LastHeartbeatAt: sqlchelpers.TimestampFromTime(opts.LastHeartbeatAt.UTC()),
		},
	)
}

func (t *tickerRepository) ListTickers(ctx context.Context, opts *ListTickerOpts) ([]*sqlcv1.Ticker, error) {
	if err := t.v.Validate(opts); err != nil {
		return nil, err
	}

	params := sqlcv1.ListTickersParams{}

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

func (t *tickerRepository) DeactivateTicker(ctx context.Context, tickerId uuid.UUID) error {
	_, err := t.queries.DeactivateTicker(
		ctx,
		t.pool,
		tickerId,
	)

	return err
}

func (t *tickerRepository) PollCronSchedules(ctx context.Context, tickerId uuid.UUID) ([]*sqlcv1.PollCronSchedulesRow, error) {
	return t.queries.PollCronSchedules(ctx, t.pool, tickerId)
}

func (t *tickerRepository) PollScheduledWorkflows(ctx context.Context, tickerId uuid.UUID) ([]*sqlcv1.PollScheduledWorkflowsRow, error) {
	return t.queries.PollScheduledWorkflows(ctx, t.pool, tickerId)
}

func (t *tickerRepository) PollTenantAlerts(ctx context.Context, tickerId uuid.UUID) ([]*sqlcv1.PollTenantAlertsRow, error) {
	return t.queries.PollTenantAlerts(ctx, t.pool, tickerId)
}

func (t *tickerRepository) PollExpiringTokens(ctx context.Context) ([]*sqlcv1.PollExpiringTokensRow, error) {
	return t.queries.PollExpiringTokens(ctx, t.pool)
}

func (t *tickerRepository) PollTenantResourceLimitAlerts(ctx context.Context) ([]*sqlcv1.TenantResourceLimitAlert, error) {
	return t.queries.PollTenantResourceLimitAlerts(ctx, t.pool)
}
