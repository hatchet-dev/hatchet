package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type tenantLimitRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
	config  *server.ConfigFileRuntime
	plans   *repository.PlanLimitMap
}

func NewTenantLimitRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger, s *server.ConfigFileRuntime) repository.TenantLimitRepository {
	queries := dbsqlc.New()

	return &tenantLimitRepository{
		v:       v,
		queries: queries,
		pool:    pool,
		l:       l,
		config:  s,
		plans:   nil,
	}
}

func (t *tenantLimitRepository) ResolveAllTenantResourceLimits(ctx context.Context) error {
	_, err := t.queries.ResolveAllLimitsIfWindowPassed(ctx, t.pool)
	return err
}

func (t *tenantLimitRepository) SetPlanLimitMap(planLimitMap repository.PlanLimitMap) error {
	t.plans = &planLimitMap
	return nil
}

func (t *tenantLimitRepository) DefaultLimits() []repository.Limit {
	return []repository.Limit{
		{
			Resource:         dbsqlc.LimitResourceWORKFLOWRUN,
			Limit:            int32(t.config.Limits.DefaultWorkflowRunLimit),      // nolint: gosec
			Alarm:            int32(t.config.Limits.DefaultWorkflowRunAlarmLimit), // nolint: gosec
			Window:           &t.config.Limits.DefaultWorkflowRunWindow,
			CustomValueMeter: false,
		},
		{
			Resource:         dbsqlc.LimitResourceTASKRUN,
			Limit:            int32(t.config.Limits.DefaultTaskRunLimit),      // nolint: gosec
			Alarm:            int32(t.config.Limits.DefaultTaskRunAlarmLimit), // nolint: gosec
			Window:           &t.config.Limits.DefaultTaskRunWindow,
			CustomValueMeter: false,
		},
		{
			Resource:         dbsqlc.LimitResourceEVENT,
			Limit:            int32(t.config.Limits.DefaultEventLimit),      // nolint: gosec
			Alarm:            int32(t.config.Limits.DefaultEventAlarmLimit), // nolint: gosec
			Window:           &t.config.Limits.DefaultEventWindow,
			CustomValueMeter: false,
		},
		{
			Resource:         dbsqlc.LimitResourceWORKER,
			Limit:            int32(t.config.Limits.DefaultWorkerLimit),      // nolint: gosec
			Alarm:            int32(t.config.Limits.DefaultWorkerAlarmLimit), // nolint: gosec
			Window:           nil,
			CustomValueMeter: true,
		},
		{
			Resource:         dbsqlc.LimitResourceWORKERSLOT,
			Limit:            int32(t.config.Limits.DefaultWorkerSlotLimit),      // nolint: gosec
			Alarm:            int32(t.config.Limits.DefaultWorkerSlotAlarmLimit), // nolint: gosec
			Window:           nil,
			CustomValueMeter: true,
		},
	}
}

func (t *tenantLimitRepository) planLimitMap(plan *string) []repository.Limit {

	if t.plans == nil || plan == nil {
		return t.DefaultLimits()
	}

	if _, ok := (*t.plans)[*plan]; !ok {
		t.l.Warn().Msgf("plan %s not found, using default limits", *plan)
		return t.DefaultLimits()
	}

	return (*t.plans)[*plan]
}

func (t *tenantLimitRepository) SelectOrInsertTenantLimits(ctx context.Context, tenantId string, plan *string) error {

	planLimits := t.planLimitMap(plan)

	for _, limits := range planLimits {
		err := t.patchTenantResourceLimit(ctx, tenantId, limits, false)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *tenantLimitRepository) UpsertTenantLimits(ctx context.Context, tenantId string, plan *string) error {
	planLimits := t.planLimitMap(plan)

	for _, limits := range planLimits {
		err := t.patchTenantResourceLimit(ctx, tenantId, limits, true)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *tenantLimitRepository) patchTenantResourceLimit(ctx context.Context, tenantId string, limits repository.Limit, upsert bool) error {

	limit := pgtype.Int4{}

	if limits.Limit >= 0 {
		limit.Int32 = limits.Limit
		limit.Valid = true
	}

	alarm := pgtype.Int4{}

	if limits.Alarm >= 0 {
		alarm.Int32 = limits.Alarm
		alarm.Valid = true
	}

	window := pgtype.Text{}

	if limits.Window != nil {
		window.String = limits.Window.String()
		window.Valid = true
	}

	cvm := pgtype.Bool{Bool: false, Valid: true}

	if limits.CustomValueMeter {
		cvm.Bool = true
	}

	if upsert {
		_, err := t.queries.UpsertTenantResourceLimit(ctx, t.pool, dbsqlc.UpsertTenantResourceLimitParams{
			Tenantid: sqlchelpers.UUIDFromStr(tenantId),
			Resource: dbsqlc.NullLimitResource{
				LimitResource: limits.Resource,
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
			LimitResource: limits.Resource,
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
			limit.Value = int32(workerCount) // nolint: gosec
		}

		if limit.Resource == dbsqlc.LimitResourceWORKERSLOT {
			workerSlotCount, err := t.queries.CountTenantWorkerSlots(ctx, t.pool, sqlchelpers.UUIDFromStr(tenantId))
			if err != nil {
				return nil, err
			}
			limit.Value = int32(workerSlotCount) // nolint: gosec
		}

	}

	return limits, nil
}

func (t *tenantLimitRepository) CanCreate(ctx context.Context, resource dbsqlc.LimitResource, tenantId string, numberOfResources int32) (bool, int, error) {

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

	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		t.l.Warn().Msgf("no %s tenant limit found, creating default limit", string(resource))

		err = t.SelectOrInsertTenantLimits(ctx, tenantId, nil)

		if err != nil {
			return false, 0, err
		}

		return true, 0, nil
	} else if err != nil {
		return false, 0, err
	}

	var value = limit.Value

	// patch custom worker limits aggregate methods
	if resource == dbsqlc.LimitResourceWORKER {
		count, err := t.queries.CountTenantWorkers(ctx, t.pool, sqlchelpers.UUIDFromStr(tenantId))
		value = int32(count) // nolint: gosec

		if err != nil {
			return false, 0, err
		}

	}

	// subtract 1 for backwards compatibility

	if value+numberOfResources-1 >= limit.LimitValue {
		return false, 100, nil
	}

	return true, calcPercent(value+numberOfResources, limit.LimitValue), nil
}

func calcPercent(value int32, limit int32) int {
	return int((float64(value) / float64(limit)) * 100)
}

func (t *tenantLimitRepository) Meter(ctx context.Context, resource dbsqlc.LimitResource, tenantId string, numberOfResources int32) (*dbsqlc.TenantResourceLimit, error) {
	if !t.config.EnforceLimits {
		return nil, nil
	}

	r, err := t.queries.MeterTenantResource(ctx, t.pool, dbsqlc.MeterTenantResourceParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Resource: dbsqlc.NullLimitResource{
			LimitResource: resource,
			Valid:         true,
		},
		Numresources: numberOfResources,
	})

	if err != nil {
		return nil, err
	}

	return r, nil
}
