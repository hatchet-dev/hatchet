package prisma

import (
	"context"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/validator"
	"github.com/jackc/pgx/v5/pgxpool"
)

type jobRunRepository struct {
	client  *db.PrismaClient
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
}

func NewJobRunRepository(client *db.PrismaClient, pool *pgxpool.Pool, v validator.Validator) repository.JobRunRepository {
	queries := dbsqlc.New()

	return &jobRunRepository{
		client:  client,
		v:       v,
		pool:    pool,
		queries: queries,
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

func (j *jobRunRepository) UpdateJobRun(tenantId, jobRunId string, opts *repository.UpdateJobRunOpts) (*db.JobRunModel, error) {
	if err := j.v.Validate(opts); err != nil {
		return nil, err
	}

	var params []db.JobRunSetParam

	if opts.Status != nil {
		params = append(params, db.JobRun.Status.Set(*opts.Status))
	}

	return j.client.JobRun.FindUnique(
		db.JobRun.ID.Equals(jobRunId),
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
		db.JobRun.Ticker.Fetch(),
	).Update(
		params...,
	).Exec(context.Background())
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

	defer tx.Rollback(context.Background())

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
