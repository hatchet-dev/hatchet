package prisma

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type jobRunAPIRepository struct {
	client  *db.PrismaClient
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
}

func NewJobRunAPIRepository(client *db.PrismaClient, pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.JobRunAPIRepository {
	queries := dbsqlc.New()

	return &jobRunAPIRepository{
		client:  client,
		v:       v,
		pool:    pool,
		queries: queries,
		l:       l,
	}
}

func (j *jobRunAPIRepository) SetJobRunStatusRunning(tenantId, jobRunId string) error {
	return setJobRunStatusRunning(context.Background(), j.pool, j.queries, j.l, tenantId, jobRunId)
}

type jobRunEngineRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
}

func NewJobRunEngineRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.JobRunEngineRepository {
	queries := dbsqlc.New()

	return &jobRunEngineRepository{
		v:       v,
		pool:    pool,
		queries: queries,
		l:       l,
	}
}

func (j *jobRunEngineRepository) SetJobRunStatusRunning(ctx context.Context, tenantId, jobRunId string) error {
	return setJobRunStatusRunning(ctx, j.pool, j.queries, j.l, tenantId, jobRunId)
}

func (j *jobRunEngineRepository) ListJobRunsForWorkflowRun(ctx context.Context, tenantId, workflowRunId string) ([]*dbsqlc.ListJobRunsForWorkflowRunRow, error) {
	return j.queries.ListJobRunsForWorkflowRun(ctx, j.pool, sqlchelpers.UUIDFromStr(workflowRunId))
}

func (j *jobRunEngineRepository) GetJobRunByWorkflowRunIdAndJobId(ctx context.Context, tenantId, workflowRunId, jobId string) (*dbsqlc.GetJobRunByWorkflowRunIdAndJobIdRow, error) {
	return j.queries.GetJobRunByWorkflowRunIdAndJobId(ctx, j.pool, dbsqlc.GetJobRunByWorkflowRunIdAndJobIdParams{
		Workflowrunid: sqlchelpers.UUIDFromStr(workflowRunId),
		Jobid:         sqlchelpers.UUIDFromStr(jobId),
		Tenantid:      sqlchelpers.UUIDFromStr(tenantId),
	})
}

func setJobRunStatusRunning(ctx context.Context, pool *pgxpool.Pool, queries *dbsqlc.Queries, l *zerolog.Logger, tenantId, jobRunId string) error {
	tx, err := pool.Begin(ctx)

	if err != nil {
		return err
	}

	defer deferRollback(context.Background(), l, tx.Rollback)

	jobRun, err := queries.UpdateJobRunStatus(context.Background(), tx, dbsqlc.UpdateJobRunStatusParams{
		ID:       sqlchelpers.UUIDFromStr(jobRunId),
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Status:   dbsqlc.JobRunStatusRUNNING,
	})

	if err != nil {
		return err
	}

	_, err = queries.UpdateWorkflowRun(
		context.Background(),
		tx,
		dbsqlc.UpdateWorkflowRunParams{
			ID:       jobRun.WorkflowRunId,
			Tenantid: jobRun.TenantId,
			Status: dbsqlc.NullWorkflowRunStatus{
				WorkflowRunStatus: dbsqlc.WorkflowRunStatusRUNNING,
				Valid:             true,
			},
		},
	)

	if err != nil {
		return err
	}

	return tx.Commit(context.Background())
}
