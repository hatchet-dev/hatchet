package prisma

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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

	wrRunningCallbacks []repository.Callback[pgtype.UUID]
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

func (j *jobRunAPIRepository) RegisterWorkflowRunRunningCallback(callback repository.Callback[pgtype.UUID]) {
	if j.wrRunningCallbacks == nil {
		j.wrRunningCallbacks = make([]repository.Callback[pgtype.UUID], 0)
	}

	j.wrRunningCallbacks = append(j.wrRunningCallbacks, callback)
}

func (j *jobRunAPIRepository) SetJobRunStatusRunning(tenantId, jobRunId string) error {
	wrId, err := setJobRunStatusRunning(context.Background(), j.pool, j.queries, j.l, tenantId, jobRunId)

	if err != nil {
		return err
	}

	for _, cb := range j.wrRunningCallbacks {
		cb.Do(j.l, tenantId, *wrId)
	}

	return nil
}

func (j *jobRunAPIRepository) ListJobRunByWorkflowRunId(ctx context.Context, tenantId, workflowRunId string) ([]*dbsqlc.ListJobRunsForWorkflowRunFullRow, error) {
	return j.queries.ListJobRunsForWorkflowRunFull(ctx, j.pool,
		dbsqlc.ListJobRunsForWorkflowRunFullParams{
			Tenantid:      sqlchelpers.UUIDFromStr(tenantId),
			Workflowrunid: sqlchelpers.UUIDFromStr(workflowRunId),
		},
	)
}

type jobRunEngineRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger

	wrRunningCallbacks []repository.Callback[pgtype.UUID]
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

func (j *jobRunEngineRepository) RegisterWorkflowRunRunningCallback(callback repository.Callback[pgtype.UUID]) {
	if j.wrRunningCallbacks == nil {
		j.wrRunningCallbacks = make([]repository.Callback[pgtype.UUID], 0)
	}

	j.wrRunningCallbacks = append(j.wrRunningCallbacks, callback)
}

func (j *jobRunEngineRepository) SetJobRunStatusRunning(ctx context.Context, tenantId, jobRunId string) error {
	wrId, err := setJobRunStatusRunning(ctx, j.pool, j.queries, j.l, tenantId, jobRunId)

	if err != nil {
		return err
	}

	for _, cb := range j.wrRunningCallbacks {
		cb.Do(j.l, tenantId, *wrId)
	}

	return nil
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

func (j *jobRunEngineRepository) GetJobRunsByWorkflowRunId(ctx context.Context, tenantId string, workflowRunId string) ([]*dbsqlc.GetJobRunsByWorkflowRunIdRow, error) {
	return j.queries.GetJobRunsByWorkflowRunId(ctx, j.pool, dbsqlc.GetJobRunsByWorkflowRunIdParams{
		Workflowrunid: sqlchelpers.UUIDFromStr(workflowRunId),
		Tenantid:      sqlchelpers.UUIDFromStr(tenantId),
	})
}

func setJobRunStatusRunning(ctx context.Context, pool *pgxpool.Pool, queries *dbsqlc.Queries, l *zerolog.Logger, tenantId, jobRunId string) (*pgtype.UUID, error) {
	tx, err := pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer sqlchelpers.DeferRollback(context.Background(), l, tx.Rollback)

	jobRun, err := queries.UpdateJobRunStatus(context.Background(), tx, dbsqlc.UpdateJobRunStatusParams{
		ID:       sqlchelpers.UUIDFromStr(jobRunId),
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Status:   dbsqlc.JobRunStatusRUNNING,
	})

	if err != nil {
		return nil, err
	}

	wr, err := queries.UpdateWorkflowRun(
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
		return nil, err
	}

	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}

	return &wr.ID, nil
}

func (r *jobRunEngineRepository) ClearJobRunPayloadData(ctx context.Context, tenantId string) (bool, error) {
	hasMore, err := r.queries.ClearJobRunLookupData(ctx, r.pool, dbsqlc.ClearJobRunLookupDataParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Limit:    1000,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}

		return false, err
	}

	return hasMore, nil
}
