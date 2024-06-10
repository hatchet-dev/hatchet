package prisma

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/config/server"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/validator"
)

type tenantLimitRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
	config  *server.ConfigFileRuntime
}

func NewTenantLimitRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger, s *server.ConfigFileRuntime) repository.TenantLimitRepository {
	queries := dbsqlc.New()

	return &tenantLimitRepository{
		v:       v,
		queries: queries,
		pool:    pool,
		l:       l,
		config:  s,
	}
}

func (t *tenantLimitRepository) ResolveAllTenantResourceLimits(ctx context.Context) error {
	_, err := t.queries.ResolveAllLimitsIfWindowPassed(ctx, t.pool)
	return err
}

func (t *tenantLimitRepository) CreateTenantDefaultLimits(ctx context.Context, tenantId string) error {
	err := t.createDefaultWorkflowRunLimit(ctx, tenantId)

	if err != nil {
		return err
	}

	err = t.createDefaultEventLimit(ctx, tenantId)

	if err != nil {
		return err
	}

	// TODO: implement cron limits
	// err = t.createDefaultCronLimit(ctx, tenantId)

	// if err != nil {
	// 	return err
	// }

	// TODO: implement schedule limits
	// err = t.createDefaultScheduleLimit(ctx, tenantId)

	// if err != nil {
	// 	return err
	// }

	err = t.createDefaultWorkerLimit(ctx, tenantId)

	return err
}

func (t *tenantLimitRepository) createDefaultWorkflowRunLimit(ctx context.Context, tenantId string) error {
	_, err := t.queries.SelectOrInsertTenantResourceLimit(ctx, t.pool, dbsqlc.SelectOrInsertTenantResourceLimitParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Resource: dbsqlc.NullLimitResource{
			LimitResource: dbsqlc.LimitResourceWORKFLOWRUN,
			Valid:         true,
		},
		LimitValue: sqlchelpers.ToInt(int32(t.config.Limits.DefaultWorkflowRunLimit)),
		AlarmValue: sqlchelpers.ToInt(int32(t.config.Limits.DefaultWorkflowRunAlarmLimit)),
		Window:     sqlchelpers.TextFromStr(t.config.Limits.DefaultWorkflowRunWindow.String()),
	})

	return err
}

func (t *tenantLimitRepository) createDefaultEventLimit(ctx context.Context, tenantId string) error {

	_, err := t.queries.SelectOrInsertTenantResourceLimit(ctx, t.pool, dbsqlc.SelectOrInsertTenantResourceLimitParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Resource: dbsqlc.NullLimitResource{
			LimitResource: dbsqlc.LimitResourceEVENT,
			Valid:         true,
		},
		LimitValue: sqlchelpers.ToInt(int32(t.config.Limits.DefaultEventLimit)),
		AlarmValue: sqlchelpers.ToInt(int32(t.config.Limits.DefaultEventAlarmLimit)),
		Window:     sqlchelpers.TextFromStr(t.config.Limits.DefaultEventWindow.String()),
	})

	return err
}

// func (t *tenantLimitRepository) createDefaultCronLimit(ctx context.Context, tenantId string) error {

// 	_, err := t.queries.SelectOrInsertTenantResourceLimit(ctx, t.pool, dbsqlc.SelectOrInsertTenantResourceLimitParams{
// 		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
// 		Resource: dbsqlc.NullLimitResource{
// 			LimitResource: dbsqlc.LimitResourceCRON,
// 			Valid:         true,
// 		},
// 		LimitValue: sqlchelpers.ToInt(int32(t.config.Limits.DefaultCronLimit)),
// 		AlarmValue: sqlchelpers.ToInt(int32(t.config.Limits.DefaultCronAlarmLimit)),
// 	})

// 	return err
// }

// func (t *tenantLimitRepository) createDefaultScheduleLimit(ctx context.Context, tenantId string) error {

// 	_, err := t.queries.SelectOrInsertTenantResourceLimit(ctx, t.pool, dbsqlc.SelectOrInsertTenantResourceLimitParams{
// 		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
// 		Resource: dbsqlc.NullLimitResource{
// 			LimitResource: dbsqlc.LimitResourceSCHEDULE,
// 			Valid:         true,
// 		},
// 		LimitValue: sqlchelpers.ToInt(int32(t.config.Limits.DefaultScheduleLimit)),
// 		AlarmValue: sqlchelpers.ToInt(int32(t.config.Limits.DefaultScheduleAlarmLimit)),
// 	})

// 	return err
// }

func (t *tenantLimitRepository) createDefaultWorkerLimit(ctx context.Context, tenantId string) error {

	_, err := t.queries.SelectOrInsertTenantResourceLimit(ctx, t.pool, dbsqlc.SelectOrInsertTenantResourceLimitParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Resource: dbsqlc.NullLimitResource{
			LimitResource: dbsqlc.LimitResourceWORKER,
			Valid:         true,
		},
		LimitValue: sqlchelpers.ToInt(int32(t.config.Limits.DefaultWorkerLimit)),
		AlarmValue: sqlchelpers.ToInt(int32(t.config.Limits.DefaultWorkerAlarmLimit)),
		CustomValueMeter: pgtype.Bool{
			Bool:  true,
			Valid: true,
		},
	})

	return err
}

func (t *tenantLimitRepository) GetLimits(ctx context.Context, tenantId string) ([]*dbsqlc.TenantResourceLimit, error) {
	if !t.config.EnforceLimits {
		return []*dbsqlc.TenantResourceLimit{}, nil
	}

	limits, err := t.queries.ListTenantResourceLimits(ctx, t.pool, sqlchelpers.UUIDFromStr(tenantId))

	if err != nil {
		return nil, err
	}

	// patch custom worker limits
	for _, limit := range limits {

		if limit.Resource == dbsqlc.LimitResourceWORKER {
			workerCount, err := t.queries.CountTenantWorkers(ctx, t.pool, sqlchelpers.UUIDFromStr(tenantId))
			if err != nil {
				return nil, err
			}
			limit.Value = int32(workerCount)
		}

	}

	return limits, nil
}

func (t *tenantLimitRepository) CanCreate(ctx context.Context, resource dbsqlc.LimitResource, tenantId string) (bool, int, error) {

	if !t.config.EnforceLimits {
		return true, 0, nil
	}

	limit, err := t.queries.GetTenantResourceLimit(ctx, t.pool, dbsqlc.GetTenantResourceLimitParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Resource: dbsqlc.NullLimitResource{
			LimitResource: resource,
			Valid:         true,
		},
	})

	if err == pgx.ErrNoRows {
		t.l.Warn().Msgf("no %s tenant limit found, creating default limit", string(resource))

		err = t.CreateTenantDefaultLimits(ctx, tenantId)

		if err != nil {
			return false, 0, err
		}

		return true, 0, nil
	}

	if err != nil {
		return false, 0, err
	}

	var value = limit.Value

	// patch custom worker limits aggregate methods
	if resource == dbsqlc.LimitResourceWORKER {
		count, err := t.queries.CountTenantWorkers(ctx, t.pool, sqlchelpers.UUIDFromStr(tenantId))
		value = int32(count)

		if err != nil {
			return false, 0, err
		}

	}

	if value >= limit.LimitValue {
		return false, 100, nil
	}

	return true, calcPercent(value, limit.LimitValue), nil
}

func calcPercent(value int32, limit int32) int {
	return int((float64(value) / float64(limit)) * 100)
}

func (t *tenantLimitRepository) Meter(ctx context.Context, resource dbsqlc.LimitResource, tenantId string) (*dbsqlc.TenantResourceLimit, error) {
	if !t.config.EnforceLimits {
		return nil, nil
	}

	r, err := t.queries.MeterTenantResource(ctx, t.pool, dbsqlc.MeterTenantResourceParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Resource: dbsqlc.NullLimitResource{
			LimitResource: resource,
			Valid:         true,
		},
	})

	if err != nil {
		return nil, err
	}

	return r, nil
}
