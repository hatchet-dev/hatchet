package prisma

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/validator"
)

type jobRunRepository struct {
	client  *db.PrismaClient
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
}

func NewJobRunRepository(client *db.PrismaClient, pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.JobRunRepository {
	queries := dbsqlc.New()

	return &jobRunRepository{
		client:  client,
		v:       v,
		pool:    pool,
		queries: queries,
		l:       l,
	}
}

func (j *jobRunRepository) ListAllJobRuns(opts *repository.ListAllJobRunsOpts) ([]db.JobRunModel, error) {
	if err := j.v.Validate(opts); err != nil {
		return nil, err
	}

	params := []db.JobRunWhereParam{}

	if opts.TickerId != nil {
		params = append(params, db.JobRun.TickerID.Equals(*opts.TickerId))
	}

	if opts.Status != nil {
		params = append(params, db.JobRun.Status.Equals(*opts.Status))
	}

	if opts.NoTickerId != nil && *opts.NoTickerId {
		params = append(params, db.JobRun.TickerID.IsNull())
	}

	return j.client.JobRun.FindMany(
		params...,
	).With(
		db.JobRun.LookupData.Fetch(),
		db.JobRun.StepRuns.Fetch().With(
			db.StepRun.Step.Fetch().With(
				db.Step.Children.Fetch(),
				db.Step.Parents.Fetch(),
				db.Step.Action.Fetch(),
			),
		),
		db.JobRun.Job.Fetch().With(
			db.Job.Workflow.Fetch(),
		),
	).Exec(context.Background())
}

func (j *jobRunRepository) GetJobRunById(tenantId, jobRunId string) (*db.JobRunModel, error) {
	return j.client.JobRun.FindUnique(
		db.JobRun.ID.Equals(jobRunId),
	).With(
		db.JobRun.LookupData.Fetch(),
		db.JobRun.StepRuns.Fetch().With(
			db.StepRun.Parents.Fetch(),
			db.StepRun.Children.Fetch(),
			db.StepRun.Step.Fetch().With(
				db.Step.Children.Fetch(),
				db.Step.Parents.Fetch(),
				db.Step.Action.Fetch(),
			),
		),
		db.JobRun.Job.Fetch().With(
			db.Job.Workflow.Fetch(),
		),
		db.JobRun.Ticker.Fetch(),
	).Exec(context.Background())
}

func (j *jobRunRepository) SetJobRunStatusRunning(tenantId, jobRunId string) error {
	tx, err := j.pool.Begin(context.Background())

	if err != nil {
		return err
	}

	defer deferRollback(context.Background(), j.l, tx.Rollback)

	jobRun, err := j.queries.UpdateJobRunStatus(context.Background(), tx, dbsqlc.UpdateJobRunStatusParams{
		ID:       sqlchelpers.UUIDFromStr(jobRunId),
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Status:   dbsqlc.JobRunStatusRUNNING,
	})

	if err != nil {
		return err
	}

	_, err = j.queries.UpdateWorkflowRun(
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

func (j *jobRunRepository) GetJobRunLookupData(tenantId, jobRunId string) (*db.JobRunLookupDataModel, error) {
	return j.client.JobRunLookupData.FindUnique(
		db.JobRunLookupData.JobRunIDTenantID(
			db.JobRunLookupData.JobRunID.Equals(jobRunId),
			db.JobRunLookupData.TenantID.Equals(tenantId),
		),
	).Exec(context.Background())
}

func (j *jobRunRepository) UpdateJobRunLookupData(tenantId, jobRunId string, opts *repository.UpdateJobRunLookupDataOpts) error {
	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)
	pgJobRunId := sqlchelpers.UUIDFromStr(jobRunId)

	tx, err := j.pool.Begin(context.Background())

	if err != nil {
		return err
	}

	defer deferRollback(context.Background(), j.l, tx.Rollback)

	err = j.queries.UpsertJobRunLookupData(
		context.Background(),
		tx,
		dbsqlc.UpsertJobRunLookupDataParams{
			Jobrunid:  pgJobRunId,
			Tenantid:  pgTenantId,
			Fieldpath: opts.FieldPath,
			Jsondata:  opts.Data,
		},
	)

	if err != nil {
		return err
	}

	return tx.Commit(context.Background())
}
