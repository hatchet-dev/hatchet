package prisma

import (
	"context"

	"github.com/jackc/pgx/v5"
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

func (t *tenantLimitRepository) CreateTenantDefaultLimits(tenantId string) error {
	err := t.createDefaultWorkflowRunLimit(tenantId)

	if err != nil {
		return err
	}

	err = t.createDefaultEventLimit(tenantId)

	if err != nil {
		return err
	}

	err = t.createDefaultWorkerLimit(tenantId)

	return err
}

var WORKFLOW_RESOURCE = dbsqlc.NullLimitResource{
	LimitResource: dbsqlc.LimitResourceWORKFLOWRUN,
	Valid:         true,
}

func (t *tenantLimitRepository) GetLimits(tenantId string) ([]*dbsqlc.TenantResourceLimit, error) {
	if !t.config.EnforceLimits {
		return []*dbsqlc.TenantResourceLimit{}, nil
	}

	limits, err := t.queries.ListTenantResourceLimits(context.Background(), t.pool, sqlchelpers.UUIDFromStr(tenantId))

	if err != nil {
		return nil, err
	}

	// patch custom worker limits
	for _, limit := range limits {

		if limit.Resource == dbsqlc.LimitResourceWORKER {
			workerCount, err := t.queries.CountTenantWorkers(context.Background(), t.pool, sqlchelpers.UUIDFromStr(tenantId))
			if err != nil {
				return nil, err
			}
			limit.Value = int32(workerCount)
		}

	}

	return limits, nil
}

func (t *tenantLimitRepository) createDefaultWorkflowRunLimit(tenantId string) error {
	_, err := t.queries.CreateTenantResourceLimit(context.Background(), t.pool, dbsqlc.CreateTenantResourceLimitParams{
		Tenantid:   sqlchelpers.UUIDFromStr(tenantId),
		Resource:   WORKFLOW_RESOURCE,
		LimitValue: sqlchelpers.ToInt(int32(t.config.Limits.DefaultWorkflowRunLimit)),
		AlarmValue: sqlchelpers.ToInt(int32(t.config.Limits.DefaultWorkflowRunAlarmLimit)),
		Window:     sqlchelpers.TextFromStr(t.config.Limits.DefaultWorkflowRunWindow.String()),
	})

	return err
}

// CanCreateWorkflowRun implements repository.TenantLimitRepository.
func (t *tenantLimitRepository) CanCreateWorkflowRun(tenantId string) (bool, error) {

	if !t.config.EnforceLimits {
		return true, nil
	}

	limit, err := t.queries.GetTenantResourceLimit(context.Background(), t.pool, dbsqlc.GetTenantResourceLimitParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Resource: WORKFLOW_RESOURCE,
	})

	if err == pgx.ErrNoRows {
		t.l.Warn().Msg("no workflow run tenant limit found, creating default limit")

		err = t.createDefaultWorkflowRunLimit(tenantId)

		if err != nil {
			return false, err
		}

		return true, nil
	}

	if err != nil {
		return false, err
	}

	if limit.Value >= limit.LimitValue {
		return false, nil
	}

	return true, nil
}

func (t *tenantLimitRepository) MeterWorkflowRun(tenantId string) error {
	if !t.config.EnforceLimits {
		return nil
	}

	_, err := t.queries.MeterTenantResource(context.Background(), t.pool, dbsqlc.MeterTenantResourceParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Resource: WORKFLOW_RESOURCE,
	})

	if err != nil {
		return err
	}

	return nil
}

var EVENT_RESOURCE = dbsqlc.NullLimitResource{
	LimitResource: dbsqlc.LimitResourceEVENT,
	Valid:         true,
}

func (t *tenantLimitRepository) createDefaultEventLimit(tenantId string) error {

	_, err := t.queries.CreateTenantResourceLimit(context.Background(), t.pool, dbsqlc.CreateTenantResourceLimitParams{
		Tenantid:   sqlchelpers.UUIDFromStr(tenantId),
		Resource:   EVENT_RESOURCE,
		LimitValue: sqlchelpers.ToInt(int32(t.config.Limits.DefaultEventLimit)),
		AlarmValue: sqlchelpers.ToInt(int32(t.config.Limits.DefaultEventAlarmLimit)),
		Window:     sqlchelpers.TextFromStr(t.config.Limits.DefaultEventWindow.String()),
	})

	return err
}

func (t *tenantLimitRepository) CanCreateEvent(tenantId string) (bool, error) {
	if !t.config.EnforceLimits {
		return true, nil
	}

	limit, err := t.queries.GetTenantResourceLimit(context.Background(), t.pool, dbsqlc.GetTenantResourceLimitParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Resource: EVENT_RESOURCE,
	})

	if err == pgx.ErrNoRows {
		t.l.Warn().Msg("no event tenant limit found, creating default limit")

		err = t.createDefaultEventLimit(tenantId)

		if err != nil {
			return false, err
		}

		return true, nil
	}

	if err != nil {
		return false, err
	}

	if limit.Value >= limit.LimitValue {
		return false, nil
	}

	return true, nil
}

func (t *tenantLimitRepository) MeterEvent(tenantId string) error {
	if !t.config.EnforceLimits {
		return nil
	}

	_, err := t.queries.MeterTenantResource(context.Background(), t.pool, dbsqlc.MeterTenantResourceParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Resource: EVENT_RESOURCE,
	})

	if err != nil {
		return err
	}

	return nil
}

var WORKER_RESOURCE = dbsqlc.NullLimitResource{
	LimitResource: dbsqlc.LimitResourceWORKER,
	Valid:         true,
}

func (t *tenantLimitRepository) createDefaultWorkerLimit(tenantId string) error {

	_, err := t.queries.CreateTenantResourceLimit(context.Background(), t.pool, dbsqlc.CreateTenantResourceLimitParams{
		Tenantid:   sqlchelpers.UUIDFromStr(tenantId),
		Resource:   WORKER_RESOURCE,
		LimitValue: sqlchelpers.ToInt(int32(t.config.Limits.DefaultWorkerLimit)),
		AlarmValue: sqlchelpers.ToInt(int32(t.config.Limits.DefaultWorkerAlarmLimit)),
	})

	return err
}

func (t *tenantLimitRepository) CanCreateWorker(tenantId string) (bool, error) {
	if !t.config.EnforceLimits {
		return true, nil
	}

	limit, err := t.queries.GetTenantResourceLimit(context.Background(), t.pool, dbsqlc.GetTenantResourceLimitParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Resource: WORKER_RESOURCE,
	})

	if err == pgx.ErrNoRows {
		t.l.Warn().Msg("no event tenant limit found, creating default limit")

		err = t.createDefaultWorkerLimit(tenantId)

		if err != nil {
			return false, err
		}

		return true, nil
	}

	if err != nil {
		return false, err
	}

	count, err := t.queries.CountTenantWorkers(context.Background(), t.pool, sqlchelpers.UUIDFromStr(tenantId))

	if err != nil {
		return false, err
	}

	t.l.Debug().Int64("count", count).Int64("limit", int64(limit.LimitValue)).Msg("worker count")

	if count >= int64(limit.LimitValue) {
		return false, nil
	}

	return true, nil
}
