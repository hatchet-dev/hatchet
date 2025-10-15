package v1

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

type IntervalSettingsRepository interface {
	ReadInterval(ctx context.Context, operationId string, tenantId string) (time.Duration, error)
	SetInterval(ctx context.Context, operationId string, tenantId string, d time.Duration) (time.Duration, error)
}

type NoOpIntervalSettingsRepository struct{}

func NewNoOpIntervalSettingsRepository() IntervalSettingsRepository {
	return &NoOpIntervalSettingsRepository{}
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

func (r *intervalSettingsRepository) ReadInterval(ctx context.Context, operationId string, tenantId string) (time.Duration, error) {
	r.l.Error().Str("resource_id", tenantId).Str("operation_id", operationId).Msg("[ReadInterval] reading interval from db")

	interval, err := r.queries.ReadInterval(ctx, r.pool, sqlcv1.ReadIntervalParams{
		Operationid: operationId,
		Tenantid:    sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}

		return 0, err
	}

	res := time.Duration(interval.IntervalNanoseconds)

	r.l.Error().Str("resource_id", tenantId).Str("operation_id", operationId).Dur("interval", res).Msg("[ReadInterval] returning interval from db")

	return res, nil
}

func (r *intervalSettingsRepository) SetInterval(ctx context.Context, operationId string, tenantId string, d time.Duration) (time.Duration, error) {
	r.l.Error().Str("resource_id", tenantId).Str("operation_id", operationId).Dur("interval", d).Msg("[SetInterval] setting interval in db")

	interval, err := r.queries.UpsertInterval(ctx, r.pool, sqlcv1.UpsertIntervalParams{
		Intervalnanoseconds: int64(d),
		Operationid:         operationId,
		Tenantid:            sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return 0, err
	}

	res := time.Duration(interval.IntervalNanoseconds)

	r.l.Error().Str("resource_id", tenantId).Str("operation_id", operationId).Dur("interval", res).Msg("[SetInterval] returning interval in db")

	return res, nil
}
