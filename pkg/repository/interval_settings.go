package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type IntervalSettingsRepository interface {
	ReadAllIntervals(ctx context.Context, operationId string) (map[string]time.Duration, error)
	ReadInterval(ctx context.Context, operationId string, tenantId string) (time.Duration, error)
	SetInterval(ctx context.Context, operationId string, tenantId string, d time.Duration) (time.Duration, error)
}

type NoOpIntervalSettingsRepository struct{}

func NewNoOpIntervalSettingsRepository() IntervalSettingsRepository {
	return &NoOpIntervalSettingsRepository{}
}

func (r *NoOpIntervalSettingsRepository) ReadAllIntervals(ctx context.Context, operationId string) (map[string]time.Duration, error) {
	return make(map[string]time.Duration), nil
}

func (r *NoOpIntervalSettingsRepository) ReadInterval(ctx context.Context, operationId string, tenantId string) (time.Duration, error) {
	return 0, nil
}

func (r *NoOpIntervalSettingsRepository) SetInterval(ctx context.Context, operationId string, tenantId string, d time.Duration) (time.Duration, error) {
	return d, nil
}

type intervalSettingsRepository struct {
	*sharedRepository
}

func newIntervalSettingsRepository(shared *sharedRepository) IntervalSettingsRepository {
	return &intervalSettingsRepository{
		sharedRepository: shared,
	}
}

func (r *intervalSettingsRepository) ReadAllIntervals(ctx context.Context, operationId string) (map[string]time.Duration, error) {
	intervals, err := r.queries.ListIntervalsByOperationId(ctx, r.pool, operationId)

	if err != nil {
		return nil, err
	}

	res := make(map[string]time.Duration)

	for _, interval := range intervals {
		res[interval.TenantID.String()] = time.Duration(interval.IntervalNanoseconds)
	}

	return res, nil
}

func (r *intervalSettingsRepository) ReadInterval(ctx context.Context, operationId string, tenantId string) (time.Duration, error) {
	interval, err := r.queries.ReadInterval(ctx, r.pool, sqlcv1.ReadIntervalParams{
		Operationid: operationId,
		Tenantid:    uuid.MustParse(tenantId),
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}

		return 0, err
	}

	res := time.Duration(interval.IntervalNanoseconds)

	return res, nil
}

func (r *intervalSettingsRepository) SetInterval(ctx context.Context, operationId string, tenantId string, d time.Duration) (time.Duration, error) {
	interval, err := r.queries.UpsertInterval(ctx, r.pool, sqlcv1.UpsertIntervalParams{
		Intervalnanoseconds: int64(d),
		Operationid:         operationId,
		Tenantid:            uuid.MustParse(tenantId),
	})

	if err != nil {
		return 0, err
	}

	res := time.Duration(interval.IntervalNanoseconds)

	return res, nil
}
