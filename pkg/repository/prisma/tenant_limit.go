package prisma

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type tenantLimitRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
	config  *server.ConfigFileRuntime
}

type Limits struct {
	resource         dbsqlc.LimitResource
	limit            int32
	alarm            int32
	window           *time.Duration
	customValueMeter bool
}

type PlanLimitMap map[dbsqlc.TenantSubscriptionPlanCodes][]Limits

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

func (t *tenantLimitRepository) planLimitMap(plan dbsqlc.TenantSubscriptionPlanCodes) []Limits {
	daily := time.Hour * 24
	// monthly := daily * 30

	planLimitMap := PlanLimitMap{
		dbsqlc.TenantSubscriptionPlanCodesFree: {
			{
				resource:         dbsqlc.LimitResourceWORKFLOWRUN,
				limit:            int32(t.config.Limits.DefaultWorkflowRunLimit),
				alarm:            int32(t.config.Limits.DefaultWorkflowRunAlarmLimit),
				window:           &t.config.Limits.DefaultWorkflowRunWindow,
				customValueMeter: false,
			},
			{
				resource:         dbsqlc.LimitResourceEVENT,
				limit:            int32(t.config.Limits.DefaultEventLimit),
				alarm:            int32(t.config.Limits.DefaultEventAlarmLimit),
				window:           &t.config.Limits.DefaultEventWindow,
				customValueMeter: false,
			},
			{
				resource:         dbsqlc.LimitResourceWORKER,
				limit:            int32(t.config.Limits.DefaultWorkerLimit),
				alarm:            int32(t.config.Limits.DefaultWorkerAlarmLimit),
				window:           nil,
				customValueMeter: true,
			},
		},
		dbsqlc.TenantSubscriptionPlanCodesStarter: {
			{
				resource:         dbsqlc.LimitResourceWORKFLOWRUN,
				limit:            10_000,
				alarm:            7_500,
				window:           &daily,
				customValueMeter: false,
			},
			{
				resource:         dbsqlc.LimitResourceEVENT,
				limit:            10_000,
				alarm:            7_500,
				window:           &daily,
				customValueMeter: false,
			},
			{
				resource:         dbsqlc.LimitResourceWORKER,
				limit:            8,
				alarm:            4,
				window:           nil,
				customValueMeter: true,
			},
		},
		dbsqlc.TenantSubscriptionPlanCodesGrowth: {
			{
				resource:         dbsqlc.LimitResourceWORKFLOWRUN,
				limit:            50_000,
				alarm:            37_500,
				window:           &daily,
				customValueMeter: false,
			},
			{
				resource:         dbsqlc.LimitResourceEVENT,
				limit:            50_000,
				alarm:            37_500,
				window:           &daily,
				customValueMeter: false,
			},
			{
				resource:         dbsqlc.LimitResourceWORKER,
				limit:            40,
				alarm:            20,
				window:           nil,
				customValueMeter: true,
			},
		},
	}

	if _, ok := planLimitMap[plan]; !ok {
		plan = dbsqlc.TenantSubscriptionPlanCodesFree
	}

	return planLimitMap[plan]
}

func (t *tenantLimitRepository) SelectOrInsertTenantLimits(ctx context.Context, tenantId string, plan *dbsqlc.TenantSubscriptionPlanCodes) error {
	if plan == nil {
		_plan := dbsqlc.TenantSubscriptionPlanCodesFree
		plan = &_plan
	}

	planLimits := t.planLimitMap(*plan)

	for _, limits := range planLimits {
		err := t.patchTenantResourceLimit(ctx, tenantId, limits, false)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *tenantLimitRepository) UpsertTenantLimits(ctx context.Context, tenantId string, plan *dbsqlc.TenantSubscriptionPlanCodes) error {
	if plan == nil {
		_plan := dbsqlc.TenantSubscriptionPlanCodesFree
		plan = &_plan
	}

	planLimits := t.planLimitMap(*plan)

	for _, limits := range planLimits {
		err := t.patchTenantResourceLimit(ctx, tenantId, limits, true)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *tenantLimitRepository) patchTenantResourceLimit(ctx context.Context, tenantId string, limits Limits, upsert bool) error {

	limit := pgtype.Int4{}

	if limits.limit >= 0 {
		limit.Int32 = limits.limit
		limit.Valid = true
	}

	alarm := pgtype.Int4{}

	if limits.alarm >= 0 {
		alarm.Int32 = limits.alarm
		alarm.Valid = true
	}

	window := pgtype.Text{}

	if limits.window != nil {
		window.String = limits.window.String()
		window.Valid = true
	}

	cvm := pgtype.Bool{Bool: false, Valid: true}

	if limits.customValueMeter {
		cvm.Bool = true
	}

	if upsert {
		_, err := t.queries.UpsertTenantResourceLimit(ctx, t.pool, dbsqlc.UpsertTenantResourceLimitParams{
			Tenantid: sqlchelpers.UUIDFromStr(tenantId),
			Resource: dbsqlc.NullLimitResource{
				LimitResource: limits.resource,
				Valid:         true,
			},
			LimitValue:       limit,
			AlarmValue:       alarm,
			Window:           window,
			CustomValueMeter: cvm,
		})

		return err
	}

	_, err := t.queries.SelectOrInsertTenantResourceLimit(ctx, t.pool, dbsqlc.SelectOrInsertTenantResourceLimitParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Resource: dbsqlc.NullLimitResource{
			LimitResource: limits.resource,
			Valid:         true,
		},
		LimitValue:       limit,
		AlarmValue:       alarm,
		Window:           window,
		CustomValueMeter: cvm,
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

		// TODO get the correct plan
		err = t.SelectOrInsertTenantLimits(ctx, tenantId, nil)

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
