package prisma

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/metered"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type workflowRunEventEngineRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
	m       *metered.Metered
}

func NewWorkflowRunEventEngineRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger, m *metered.Metered) repository.WorkflowRunEventEngineRepository {
	queries := dbsqlc.New()

	return &workflowRunEventEngineRepository{
		v:       v,
		queries: queries,
		pool:    pool,
		l:       l,
		m:       m,
	}
}

func (r *workflowRunEventEngineRepository) CreateSucceededWorkflowRunEvent(ctx context.Context, tenantId string, workflowRunId string) error {

	_, err := r.queries.CreateWorkflowRunEvent(ctx, r.pool, dbsqlc.CreateWorkflowRunEventParams{
		TenantId:      sqlchelpers.UUIDFromStr(tenantId),
		WorkflowRunId: sqlchelpers.UUIDFromStr(workflowRunId),
		EventType:     dbsqlc.WorkflowRunEventTypeSUCCEEDED,
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *workflowRunEventEngineRepository) CreateQueuedWorkflowRunEvent(ctx context.Context, tenantId string, workflowRunId string) error {

	_, err := r.queries.CreateWorkflowRunEvent(ctx, r.pool, dbsqlc.CreateWorkflowRunEventParams{
		TenantId:      sqlchelpers.UUIDFromStr(tenantId),
		WorkflowRunId: sqlchelpers.UUIDFromStr(workflowRunId),
		EventType:     dbsqlc.WorkflowRunEventTypeQUEUED,
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *workflowRunEventEngineRepository) GetWorkflowRunEventMetrics(ctx context.Context, tenantId string, startTimestamp *time.Time, endTimestamp *time.Time) ([]*dbsqlc.WorkflowRunEventsMetricsRow, error) {

	rows, err := r.queries.WorkflowRunEventsMetrics(ctx, r.pool, dbsqlc.WorkflowRunEventsMetricsParams{
		Column1: sqlchelpers.UUIDFromStr(tenantId),
		Column2: sqlchelpers.TimestampFromTime(*startTimestamp),
		Column3: sqlchelpers.TimestampFromTime(*endTimestamp),
	})
	if err != nil {
		return nil, err
	}

	return rows, nil
}
