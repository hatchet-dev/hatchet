package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

type jobRunAPIRepository struct {
	*sharedRepository
}

func NewJobRunAPIRepository(shared *sharedRepository) repository.JobRunAPIRepository {
	return &jobRunAPIRepository{
		sharedRepository: shared,
	}
}

func (j *jobRunAPIRepository) RegisterWorkflowRunRunningCallback(callback repository.TenantScopedCallback[pgtype.UUID]) {
	if j.wrRunningCallbacks == nil {
		j.wrRunningCallbacks = make([]repository.TenantScopedCallback[pgtype.UUID], 0)
	}

	j.wrRunningCallbacks = append(j.wrRunningCallbacks, callback)
}

func (j *jobRunAPIRepository) SetJobRunStatusRunning(tenantId, jobRunId string) error {
	wrId, err := j.setJobRunStatusRunning(context.Background(), j.pool, tenantId, jobRunId)

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
	*sharedRepository
}

func NewJobRunEngineRepository(shared *sharedRepository) repository.JobRunEngineRepository {

	return &jobRunEngineRepository{
		sharedRepository: shared,
	}
}

func (j *jobRunEngineRepository) RegisterWorkflowRunRunningCallback(callback repository.TenantScopedCallback[pgtype.UUID]) {
	if j.wrRunningCallbacks == nil {
		j.wrRunningCallbacks = make([]repository.TenantScopedCallback[pgtype.UUID], 0)
	}

	j.wrRunningCallbacks = append(j.wrRunningCallbacks, callback)
}

func (j *jobRunEngineRepository) SetJobRunStatusRunning(ctx context.Context, tenantId, jobRunId string) error {

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, j.pool, j.l, 5000)

	if err != nil {
		return err
	}

	defer rollback()

	err = j.setJobRunStatusRunningWithTx(ctx, tx, tenantId, jobRunId)

	if err != nil {
		return err
	}

	err = commit(ctx)
	if err != nil {
		return err

	}

	return nil
}

func (s *sharedRepository) setJobRunStatusRunningWithTx(ctx context.Context, tx dbsqlc.DBTX, tenantId, jobRunId string) error {
	wrId, err := s.setJobRunStatusRunning(ctx, tx, tenantId, jobRunId)

	if err != nil {
		return err
	}

	for _, cb := range s.wrRunningCallbacks {
		cb.Do(s.l, tenantId, *wrId)
	}

	return nil
}

func (j *sharedRepository) ListJobRunsForWorkflowRun(ctx context.Context, tenantId, workflowRunId string) ([]*dbsqlc.ListJobRunsForWorkflowRunRow, error) {
	return j.queries.ListJobRunsForWorkflowRun(ctx, j.pool, sqlchelpers.UUIDFromStr(workflowRunId))
}

func (j *sharedRepository) listJobRunsForWorkflowRunWithTx(ctx context.Context, tx dbsqlc.DBTX, tenantId, workflowRunId string) ([]*dbsqlc.ListJobRunsForWorkflowRunRow, error) {
	return j.queries.ListJobRunsForWorkflowRun(ctx, tx, sqlchelpers.UUIDFromStr(workflowRunId))
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

func (s *sharedRepository) setJobRunStatusRunning(ctx context.Context, tx dbsqlc.DBTX, tenantId, jobRunId string) (*pgtype.UUID, error) {

	jobRun, err := s.queries.UpdateJobRunStatus(context.Background(), tx, dbsqlc.UpdateJobRunStatusParams{
		ID:       sqlchelpers.UUIDFromStr(jobRunId),
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Status:   dbsqlc.JobRunStatusRUNNING,
	})

	if err != nil {
		return nil, err
	}

	wr, err := s.queries.UpdateWorkflowRun(
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
