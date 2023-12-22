package prisma

import (
	"context"
	"encoding/json"
	"time"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/validator"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type stepRunRepository struct {
	client  *db.PrismaClient
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
}

func NewStepRunRepository(client *db.PrismaClient, pool *pgxpool.Pool, v validator.Validator) repository.StepRunRepository {
	return &stepRunRepository{
		client: client,
		pool:   pool,
		v:      v,
	}
}

func (j *stepRunRepository) ListAllStepRuns(opts *repository.ListAllStepRunsOpts) ([]db.StepRunModel, error) {
	if err := j.v.Validate(opts); err != nil {
		return nil, err
	}

	params := []db.StepRunWhereParam{}

	if opts.TickerId != nil {
		params = append(params, db.StepRun.TickerID.Equals(*opts.TickerId))
	}

	if opts.Status != nil {
		params = append(params, db.StepRun.Status.Equals(*opts.Status))
	}

	if opts.NoTickerId != nil && *opts.NoTickerId {
		params = append(params, db.StepRun.TickerID.IsNull())
	}

	return j.client.StepRun.FindMany(
		params...,
	).With(
		db.StepRun.Step.Fetch().With(
			db.Step.Action.Fetch(),
		),
		db.StepRun.Next.Fetch(),
		db.StepRun.Prev.Fetch(),
		db.StepRun.JobRun.Fetch().With(
			db.JobRun.Job.Fetch(),
		),
		db.StepRun.Ticker.Fetch(),
	).Exec(context.Background())
}

func (j *stepRunRepository) ListStepRuns(tenantId string, opts *repository.ListStepRunsOpts) ([]db.StepRunModel, error) {
	if err := j.v.Validate(opts); err != nil {
		return nil, err
	}

	params := []db.StepRunWhereParam{
		db.StepRun.TenantID.Equals(tenantId),
	}

	if opts.Requeuable != nil {
		// job runs are requeuable if they are past their requeue after time, don't have a worker assigned, have a pending status,
		// and their previous step is completed
		params = append(
			params,
			db.StepRun.RequeueAfter.Before(time.Now()),
			db.StepRun.WorkerID.IsNull(),
			db.StepRun.Status.Equals(db.StepRunStatusPendingAssignment),
			// db.StepRun.Or(
			// 	db.StepRun.Prev
			// 	db.StepRun.Step.Where(
			// 		db.Step.Prev.Where(
			// 			db.Step.StepRuns.Some(
			// 				db.StepRun.Status.Equals(db.StepRunStatusSUCCEEDED),
			// 			),
			// 		),
			// 	),
			// ),
		)
	}

	if opts.Status != nil {
		params = append(params, db.StepRun.Status.Equals(*opts.Status))
	}

	if opts.JobRunId != nil {
		params = append(params, db.StepRun.JobRunID.Equals(*opts.JobRunId))
	}

	return j.client.StepRun.FindMany(
		params...,
	).With(
		db.StepRun.Step.Fetch().With(
			db.Step.Action.Fetch(),
		),
		db.StepRun.Next.Fetch(),
		db.StepRun.Prev.Fetch(),
		db.StepRun.JobRun.Fetch().With(
			db.JobRun.Job.Fetch(),
		),
		db.StepRun.Ticker.Fetch(),
	).Exec(context.Background())
}

func (j *stepRunRepository) UpdateStepRun(tenantId, stepRunId string, opts *repository.UpdateStepRunOpts) (*db.StepRunModel, error) {
	if err := j.v.Validate(opts); err != nil {
		return nil, err
	}

	pgTenantId := &pgtype.UUID{}

	if err := pgTenantId.Scan(tenantId); err != nil {
		return nil, err
	}

	updateParams := dbsqlc.UpdateStepRunParams{
		ID:       sqlchelpers.UUIDFromStr(stepRunId),
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	}

	resolveJobRunParams := dbsqlc.ResolveJobRunStatusParams{
		Steprunid: sqlchelpers.UUIDFromStr(stepRunId),
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
	}

	if opts.RequeueAfter != nil {
		updateParams.RequeueAfter = sqlchelpers.TimestampFromTime(opts.RequeueAfter)
	}

	if opts.StartedAt != nil {
		updateParams.StartedAt = sqlchelpers.TimestampFromTime(opts.StartedAt)
	}

	if opts.FinishedAt != nil {
		updateParams.FinishedAt = sqlchelpers.TimestampFromTime(opts.FinishedAt)
	}

	if opts.Status != nil {
		runStatus := dbsqlc.NullStepRunStatus{}

		if err := runStatus.Scan(string(*opts.Status)); err != nil {
			return nil, err
		}

		updateParams.Status = runStatus
	}

	if opts.Input != nil {
		updateParams.Input = []byte(json.RawMessage(*opts.Input))
	}

	if opts.Output != nil {
		updateParams.Output = []byte(json.RawMessage(*opts.Output))
	}

	if opts.Error != nil {
		updateParams.Error = sqlchelpers.TextFromStr(*opts.Error)
	}

	if opts.CancelledAt != nil {
		updateParams.CancelledAt = sqlchelpers.TimestampFromTime(opts.CancelledAt)
	}

	if opts.CancelledReason != nil {
		updateParams.CancelledReason = sqlchelpers.TextFromStr(*opts.CancelledReason)
	}

	tx, err := j.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer tx.Rollback(context.Background())

	_, err = j.queries.UpdateStepRun(context.Background(), tx, updateParams)

	if err != nil {
		return nil, err
	}

	resolveLaterStepRunsParams := dbsqlc.ResolveLaterStepRunsParams{
		Steprunid: sqlchelpers.UUIDFromStr(stepRunId),
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
	}

	_, err = j.queries.ResolveLaterStepRuns(context.Background(), tx, resolveLaterStepRunsParams)

	if err != nil {
		return nil, err
	}

	jobRun, err := j.queries.ResolveJobRunStatus(context.Background(), tx, resolveJobRunParams)

	if err != nil {
		return nil, err
	}

	resolveWorkflowRunParams := dbsqlc.ResolveWorkflowRunStatusParams{
		Jobrunid: jobRun.ID,
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	}

	_, err = j.queries.ResolveWorkflowRunStatus(context.Background(), tx, resolveWorkflowRunParams)

	if err != nil {
		return nil, err
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return nil, err
	}

	// updateParams := []db.StepRunSetParam{}

	// if opts.RequeueAfter != nil {
	// 	updateParams = append(updateParams, db.StepRun.RequeueAfter.Set(*opts.RequeueAfter))
	// }

	// if opts.StartedAt != nil {
	// 	updateParams = append(updateParams, db.StepRun.StartedAt.Set(*opts.StartedAt))
	// }

	// if opts.FinishedAt != nil {
	// 	updateParams = append(updateParams, db.StepRun.FinishedAt.Set(*opts.FinishedAt))
	// }

	// if opts.Status != nil {
	// 	updateParams = append(updateParams, db.StepRun.Status.Set(*opts.Status))
	// }

	// if opts.Input != nil {
	// 	updateParams = append(updateParams, db.StepRun.Input.Set(*opts.Input))
	// }

	// if opts.Error != nil {
	// 	updateParams = append(updateParams, db.StepRun.Error.Set(*opts.Error))
	// }

	// if opts.CancelledAt != nil {
	// 	updateParams = append(updateParams, db.StepRun.CancelledAt.Set(*opts.CancelledAt))
	// }

	// if opts.CancelledReason != nil {
	// 	updateParams = append(updateParams, db.StepRun.CancelledReason.Set(*opts.CancelledReason))
	// }

	return j.client.StepRun.FindUnique(
		db.StepRun.ID.Equals(stepRunId),
	).With(
		db.StepRun.Next.Fetch(),
		db.StepRun.Prev.Fetch(),
		db.StepRun.Step.Fetch().With(
			db.Step.Next.Fetch(),
			db.Step.Prev.Fetch(),
			db.Step.Action.Fetch(),
		),
		db.StepRun.JobRun.Fetch().With(
			db.JobRun.Job.Fetch(),
		),
		db.StepRun.Ticker.Fetch(),
	).Exec(context.Background())
}

func (j *stepRunRepository) GetStepRunById(tenantId, stepRunId string) (*db.StepRunModel, error) {
	return j.client.StepRun.FindUnique(
		db.StepRun.ID.Equals(stepRunId),
	).With(
		db.StepRun.Next.Fetch(),
		db.StepRun.Prev.Fetch(),
		db.StepRun.Step.Fetch().With(
			db.Step.Job.Fetch(),
			db.Step.Action.Fetch(),
		),
		db.StepRun.JobRun.Fetch().With(
			db.JobRun.LookupData.Fetch(),
		),
		db.StepRun.Ticker.Fetch(),
	).Exec(context.Background())
}

func (j *stepRunRepository) CancelPendingStepRuns(tenantId, jobRunId, reason string) error {
	_, err := j.client.StepRun.FindMany(
		db.StepRun.JobRunID.Equals(jobRunId),
		db.StepRun.Status.Equals(db.StepRunStatusPending),
	).Update(
		db.StepRun.Status.Set(db.StepRunStatusCancelled),
		db.StepRun.CancelledAt.Set(time.Now()),
		db.StepRun.CancelledReason.Set(reason),
	).Exec(context.Background())

	return err
}
