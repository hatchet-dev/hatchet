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

var WORKFLOW_RESOURCE = dbsqlc.NullLimitResource{
	LimitResource: dbsqlc.LimitResourceWORKFLOWRUN,
	Valid:         true,
}

func (t *tenantLimitRepository) createDefaultWorkflowLimit(tenantId string) error {
	const limitValue = 10000

	_, err := t.queries.CreateTenantResourceLimit(context.Background(), t.pool, dbsqlc.CreateTenantResourceLimitParams{
		Tenantid:   sqlchelpers.UUIDFromStr(tenantId),
		Resource:   WORKFLOW_RESOURCE,
		LimitValue: sqlchelpers.ToInt(limitValue),
		AlarmValue: sqlchelpers.ToInt(limitValue * .75),
		Window:     sqlchelpers.TextFromStr("1 day"),
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

		err = t.createDefaultWorkflowLimit(tenantId)

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
