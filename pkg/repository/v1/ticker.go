package v1

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type TickerRepository interface {
	IsTenantAlertActive(ctx context.Context, tenantId string) (bool, time.Time, error)
}

type tickerRepository struct {
	*sharedRepository
}

func newTickerRepository(shared *sharedRepository) TickerRepository {
	return &tickerRepository{
		sharedRepository: shared,
	}
}

func (t *tickerRepository) IsTenantAlertActive(ctx context.Context, tenantId string) (bool, time.Time, error) {
	res, err := t.queries.IsTenantAlertActive(ctx, t.pool, uuid.MustParse(tenantId))

	if err != nil {
		return false, time.Now(), err
	}

	return res.IsActive, res.LastAlertedAt.Time, nil
}
