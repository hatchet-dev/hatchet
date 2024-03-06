package prisma

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/internal/validator"
)

type stepRunRepository struct {
	client  *db.PrismaClient
	pool    *pgxpool.Pool
	v       validator.Validator
	l       *zerolog.Logger
	queries *dbsqlc.Queries
}

func NewStepRunRepository(client *db.PrismaClient, pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.StepRunRepository {
	queries := dbsqlc.New()

	return &stepRunRepository{
		client:  client,
		pool:    pool,
		v:       v,
		l:       l,
		queries: queries,
	}
}

func (s *stepRunRepository) ListAllStepRuns(opts *repository.ListAllStepRunsOpts) ([]db.StepRunModel, error) {
	if err := s.v.Validate(opts); err != nil {
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

	return s.client.StepRun.FindMany(
		params...,
	).With(
		db.StepRun.Step.Fetch().With(
			db.Step.Action.Fetch(),
		),
		db.StepRun.Children.Fetch(),
		db.StepRun.Parents.Fetch(),
		db.StepRun.JobRun.Fetch().With(
			db.JobRun.Job.Fetch(),
		),
		db.StepRun.Ticker.Fetch(),
	).Exec(context.Background())
}

func (s *stepRunRepository) ListStepRunsToRequeue(tenantId string) ([]*dbsqlc.StepRun, error) {
	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	tx, err := s.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer deferRollback(context.Background(), s.l, tx.Rollback)

	// get the step run and make sure it's still in pending
	stepRuns, err := s.queries.ListStepRunsToRequeue(context.Background(), tx, pgTenantId)

	if err != nil {
		return nil, err
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return nil, err
	}

	return stepRuns, nil
}

func (s *stepRunRepository) ListStepRunsToReassign(tenantId string) ([]*dbsqlc.StepRun, error) {
	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	tx, err := s.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer deferRollback(context.Background(), s.l, tx.Rollback)

	// get the step run and make sure it's still in pending
	stepRuns, err := s.queries.ListStepRunsToReassign(context.Background(), tx, pgTenantId)

	if err != nil {
		return nil, err
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return nil, err
	}

	return stepRuns, nil
}

func (s *stepRunRepository) ListStepRuns(tenantId string, opts *repository.ListStepRunsOpts) ([]db.StepRunModel, error) {
	if err := s.v.Validate(opts); err != nil {
		return nil, err
	}

	params := []db.StepRunWhereParam{
		db.StepRun.TenantID.Equals(tenantId),
	}

	if opts.Status != nil {
		params = append(params, db.StepRun.Status.Equals(*opts.Status))
	}

	if opts.JobRunId != nil {
		params = append(params, db.StepRun.JobRunID.Equals(*opts.JobRunId))
	}

	if opts.WorkflowRunId != nil {
		params = append(params, db.StepRun.JobRun.Where(
			db.JobRun.WorkflowRunID.Equals(*opts.WorkflowRunId),
		))
	}

	return s.client.StepRun.FindMany(
		params...,
	).With(
		db.StepRun.Step.Fetch().With(
			db.Step.Action.Fetch(),
		),
		db.StepRun.Children.Fetch(),
		db.StepRun.Parents.Fetch(),
		db.StepRun.JobRun.Fetch().With(
			db.JobRun.Job.Fetch(),
		),
		db.StepRun.Ticker.Fetch(),
	).Exec(context.Background())
}

var retrier = func(l *zerolog.Logger, f func() error) error {
	retries := 0

	for {
		err := f()

		if err != nil {
			// deadlock detected, retry
			if strings.Contains(err.Error(), "deadlock detected") {
				retries++

				if retries > 3 {
					return fmt.Errorf("could not update job run lookup data: %w", err)
				}

				l.Err(err).Msgf("deadlock detected, retry %d", retries)
				time.Sleep(100 * time.Millisecond)
			} else {
				return fmt.Errorf("could not update job run lookup data: %w", err)
			}
		}

		if err == nil {
			if retries > 0 {
				l.Info().Msgf("deadlock resolved after %d retries", retries)
			}

			break
		}
	}

	return nil
}

func (s *stepRunRepository) UpdateStepRun(tenantId, stepRunId string, opts *repository.UpdateStepRunOpts) (*db.StepRunModel, *repository.StepRunUpdateInfo, error) {
	if err := s.v.Validate(opts); err != nil {
		return nil, nil, err
	}

	updateParams, updateJobRunLookupDataParams, resolveJobRunParams, resolveLaterStepRunsParams, err := getUpdateParams(tenantId, stepRunId, opts)

	if err != nil {
		return nil, nil, err
	}

	var updateInfo *repository.StepRunUpdateInfo

	err = retrier(s.l, func() error {
		tx, err := s.pool.Begin(context.Background())

		if err != nil {
			return err
		}

		defer deferRollback(context.Background(), s.l, tx.Rollback)

		updateInfo, err = s.updateStepRun(tx, tenantId, updateParams, updateJobRunLookupDataParams, resolveJobRunParams, resolveLaterStepRunsParams)

		if err != nil {
			return err
		}

		err = tx.Commit(context.Background())

		return err
	})

	if err != nil {
		return nil, nil, err
	}

	stepRun, err := s.client.StepRun.FindUnique(
		db.StepRun.ID.Equals(stepRunId),
	).With(
		db.StepRun.Children.Fetch(),
		db.StepRun.Parents.Fetch(),
		db.StepRun.Step.Fetch().With(
			db.Step.Children.Fetch(),
			db.Step.Parents.Fetch(),
			db.Step.Action.Fetch(),
		),
		db.StepRun.JobRun.Fetch().With(
			db.JobRun.Job.Fetch(),
		),
		db.StepRun.Ticker.Fetch(),
	).Exec(context.Background())

	if err != nil {
		return nil, nil, err
	}

	return stepRun, updateInfo, nil
}

func (s *stepRunRepository) UpdateStepRunOverridesData(tenantId, stepRunId string, opts *repository.UpdateStepRunOverridesDataOpts) ([]byte, error) {
	if err := s.v.Validate(opts); err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer deferRollback(context.Background(), s.l, tx.Rollback)

	if err != nil {
		return nil, err
	}

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)
	pgStepRunId := sqlchelpers.UUIDFromStr(stepRunId)

	callerFile := ""

	if opts.CallerFile != nil {
		callerFile = *opts.CallerFile
	}

	input, err := s.queries.UpdateStepRunOverridesData(
		context.Background(),
		tx,
		dbsqlc.UpdateStepRunOverridesDataParams{
			Steprunid: pgStepRunId,
			Tenantid:  pgTenantId,
			Fieldpath: []string{
				"overrides",
				opts.OverrideKey,
			},
			Jsondata: opts.Data,
			Overrideskey: []string{
				opts.OverrideKey,
			},
			Callerfile: callerFile,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("could not update step run overrides data: %w", err)
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return nil, err
	}

	return input, nil
}

func (s *stepRunRepository) UpdateStepRunInputSchema(tenantId, stepRunId string, schema []byte) ([]byte, error) {
	tx, err := s.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer deferRollback(context.Background(), s.l, tx.Rollback)

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)
	pgStepRunId := sqlchelpers.UUIDFromStr(stepRunId)

	inputSchema, err := s.queries.UpdateStepRunInputSchema(
		context.Background(),
		tx,
		dbsqlc.UpdateStepRunInputSchemaParams{
			Steprunid:   pgStepRunId,
			Tenantid:    pgTenantId,
			InputSchema: schema,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("could not update step run input schema: %w", err)
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return nil, err
	}

	return inputSchema, nil
}

func (s *stepRunRepository) QueueStepRun(tenantId, stepRunId string, opts *repository.UpdateStepRunOpts) (*db.StepRunModel, error) {
	if err := s.v.Validate(opts); err != nil {
		return nil, err
	}

	updateParams, updateJobRunLookupDataParams, resolveJobRunParams, resolveLaterStepRunsParams, err := getUpdateParams(tenantId, stepRunId, opts)

	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer deferRollback(context.Background(), s.l, tx.Rollback)

	// get the step run and make sure it's still in pending
	stepRun, err := s.queries.GetStepRun(context.Background(), tx, dbsqlc.GetStepRunParams{
		ID:       sqlchelpers.UUIDFromStr(stepRunId),
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, err
	}

	if stepRun.Status != dbsqlc.StepRunStatusPENDING {
		return nil, repository.ErrStepRunIsNotPending
	}

	_, err = s.updateStepRun(tx, tenantId, updateParams, updateJobRunLookupDataParams, resolveJobRunParams, resolveLaterStepRunsParams)

	if err != nil {
		return nil, err
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return nil, err
	}

	return s.client.StepRun.FindUnique(
		db.StepRun.ID.Equals(stepRunId),
	).With(
		db.StepRun.Children.Fetch(),
		db.StepRun.Parents.Fetch(),
		db.StepRun.Step.Fetch().With(
			db.Step.Children.Fetch(),
			db.Step.Parents.Fetch(),
			db.Step.Action.Fetch(),
		),
		db.StepRun.JobRun.Fetch().With(
			db.JobRun.Job.Fetch(),
		),
		db.StepRun.Ticker.Fetch(),
	).Exec(context.Background())
}

func getUpdateParams(
	tenantId,
	stepRunId string,
	opts *repository.UpdateStepRunOpts,
) (
	updateParams dbsqlc.UpdateStepRunParams,
	updateJobRunLookupDataParams *dbsqlc.UpdateJobRunLookupDataWithStepRunParams,
	resolveJobRunParams dbsqlc.ResolveJobRunStatusParams,
	resolveLaterStepRunsParams dbsqlc.ResolveLaterStepRunsParams,
	err error,
) {
	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)
	pgStepRunId := sqlchelpers.UUIDFromStr(stepRunId)

	updateParams = dbsqlc.UpdateStepRunParams{
		ID:       pgStepRunId,
		Tenantid: pgTenantId,
		Rerun: pgtype.Bool{
			Valid: true,
			Bool:  opts.IsRerun,
		},
	}

	if opts.Output != nil {
		updateJobRunLookupDataParams = &dbsqlc.UpdateJobRunLookupDataWithStepRunParams{
			Steprunid: pgStepRunId,
			Tenantid:  pgTenantId,
			Jsondata:  opts.Output,
		}
	}

	resolveJobRunParams = dbsqlc.ResolveJobRunStatusParams{
		Steprunid: pgStepRunId,
		Tenantid:  pgTenantId,
	}

	resolveLaterStepRunsParams = dbsqlc.ResolveLaterStepRunsParams{
		Steprunid: pgStepRunId,
		Tenantid:  pgTenantId,
	}

	if opts.RequeueAfter != nil {
		updateParams.RequeueAfter = sqlchelpers.TimestampFromTime(*opts.RequeueAfter)
	}

	if opts.ScheduleTimeoutAt != nil {
		updateParams.ScheduleTimeoutAt = sqlchelpers.TimestampFromTime(*opts.ScheduleTimeoutAt)
	}

	if opts.StartedAt != nil {
		updateParams.StartedAt = sqlchelpers.TimestampFromTime(*opts.StartedAt)
	}

	if opts.FinishedAt != nil {
		updateParams.FinishedAt = sqlchelpers.TimestampFromTime(*opts.FinishedAt)
	}

	if opts.Status != nil {
		runStatus := dbsqlc.NullStepRunStatus{}

		if err := runStatus.Scan(string(*opts.Status)); err != nil {
			return updateParams, nil, resolveJobRunParams, resolveLaterStepRunsParams, err
		}

		updateParams.Status = runStatus
	}

	if opts.Input != nil {
		updateParams.Input = opts.Input
	}

	if opts.Output != nil {
		updateParams.Output = opts.Output
	}

	if opts.Error != nil {
		updateParams.Error = sqlchelpers.TextFromStr(*opts.Error)
	}

	if opts.CancelledAt != nil {
		updateParams.CancelledAt = sqlchelpers.TimestampFromTime(*opts.CancelledAt)
	}

	if opts.CancelledReason != nil {
		updateParams.CancelledReason = sqlchelpers.TextFromStr(*opts.CancelledReason)
	}

	if opts.RetryCount != nil {
		updateParams.RetryCount = pgtype.Int4{
			Valid: true,
			Int32: int32(*opts.RetryCount),
		}
	}

	return updateParams, updateJobRunLookupDataParams, resolveJobRunParams, resolveLaterStepRunsParams, nil
}

func (s *stepRunRepository) updateStepRun(
	tx pgx.Tx,
	tenantId string,
	updateParams dbsqlc.UpdateStepRunParams,
	updateJobRunLookupDataParams *dbsqlc.UpdateJobRunLookupDataWithStepRunParams,
	resolveJobRunParams dbsqlc.ResolveJobRunStatusParams,
	resolveLaterStepRunsParams dbsqlc.ResolveLaterStepRunsParams,
) (*repository.StepRunUpdateInfo, error) {
	_, err := s.queries.UpdateStepRun(context.Background(), tx, updateParams)

	if err != nil {
		return nil, fmt.Errorf("could not update step run: %w", err)
	}

	_, err = s.queries.ResolveLaterStepRuns(context.Background(), tx, resolveLaterStepRunsParams)

	if err != nil {
		return nil, fmt.Errorf("could not resolve later step runs: %w", err)
	}

	jobRun, err := s.queries.ResolveJobRunStatus(context.Background(), tx, resolveJobRunParams)

	if err != nil {
		return nil, fmt.Errorf("could not resolve job run status: %w", err)
	}

	resolveWorkflowRunParams := dbsqlc.ResolveWorkflowRunStatusParams{
		Jobrunid: jobRun.ID,
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	}

	workflowRun, err := s.queries.ResolveWorkflowRunStatus(context.Background(), tx, resolveWorkflowRunParams)

	if err != nil {
		return nil, fmt.Errorf("could not resolve workflow run status: %w", err)
	}

	// update the job run lookup data if not nil
	if updateJobRunLookupDataParams != nil {
		err = s.queries.UpdateJobRunLookupDataWithStepRun(context.Background(), tx, *updateJobRunLookupDataParams)

		if err != nil {
			return nil, fmt.Errorf("could not update job run lookup data: %w", err)
		}
	}

	return &repository.StepRunUpdateInfo{
		JobRunFinalState:      isFinalJobRunStatus(jobRun.Status),
		WorkflowRunFinalState: isFinalWorkflowRunStatus(workflowRun.Status),
		WorkflowRunId:         sqlchelpers.UUIDToStr(workflowRun.ID),
		WorkflowRunStatus:     string(workflowRun.Status),
	}, nil
}

func isFinalJobRunStatus(status dbsqlc.JobRunStatus) bool {
	return status != dbsqlc.JobRunStatusPENDING && status != dbsqlc.JobRunStatusRUNNING
}

func isFinalWorkflowRunStatus(status dbsqlc.WorkflowRunStatus) bool {
	return status != dbsqlc.WorkflowRunStatusPENDING && status != dbsqlc.WorkflowRunStatusRUNNING && status != dbsqlc.WorkflowRunStatusQUEUED
}

func (s *stepRunRepository) GetStepRunById(tenantId, stepRunId string) (*db.StepRunModel, error) {
	return s.client.StepRun.FindUnique(
		db.StepRun.ID.Equals(stepRunId),
	).With(
		db.StepRun.Children.Fetch(),
		db.StepRun.Parents.Fetch().With(
			db.StepRun.Step.Fetch(),
		),
		db.StepRun.Step.Fetch().With(
			db.Step.Job.Fetch(),
			db.Step.Action.Fetch(),
		),
		db.StepRun.JobRun.Fetch().With(
			db.JobRun.LookupData.Fetch(),
			db.JobRun.WorkflowRun.Fetch(),
		),
		db.StepRun.Ticker.Fetch(),
	).Exec(context.Background())
}

func (s *stepRunRepository) CancelPendingStepRuns(tenantId, jobRunId, reason string) error {
	_, err := s.client.StepRun.FindMany(
		db.StepRun.JobRunID.Equals(jobRunId),
		db.StepRun.Status.Equals(db.StepRunStatusPending),
	).Update(
		db.StepRun.Status.Set(db.StepRunStatusCancelled),
		db.StepRun.CancelledAt.Set(time.Now().UTC()),
		db.StepRun.CancelledReason.Set(reason),
	).Exec(context.Background())

	return err
}

func (s *stepRunRepository) ListStartableStepRuns(tenantId, jobRunId, parentStepRunId string) ([]*dbsqlc.StepRun, error) {
	tx, err := s.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer deferRollback(context.Background(), s.l, tx.Rollback)

	stepRuns, err := s.queries.ListStartableStepRuns(context.Background(), tx, dbsqlc.ListStartableStepRunsParams{
		Tenantid:        sqlchelpers.UUIDFromStr(tenantId),
		Jobrunid:        sqlchelpers.UUIDFromStr(jobRunId),
		Parentsteprunid: sqlchelpers.UUIDFromStr(parentStepRunId),
	})

	if err != nil {
		return nil, err
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return nil, err
	}

	return stepRuns, nil
}

func (s *stepRunRepository) ArchiveStepRunResult(tenantId, stepRunId string) error {
	tx, err := s.pool.Begin(context.Background())

	if err != nil {
		return err
	}

	defer deferRollback(context.Background(), s.l, tx.Rollback)

	_, err = s.queries.ArchiveStepRunResultFromStepRun(context.Background(), tx, dbsqlc.ArchiveStepRunResultFromStepRunParams{
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
		Steprunid: sqlchelpers.UUIDFromStr(stepRunId),
	})

	if err != nil {
		return err
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return err
	}

	return nil
}

func (s *stepRunRepository) ListArchivedStepRunResults(tenantId, stepRunId string) ([]db.StepRunResultArchiveModel, error) {
	return s.client.StepRunResultArchive.FindMany(
		db.StepRunResultArchive.StepRunID.Equals(stepRunId),
		db.StepRunResultArchive.StepRun.Where(
			db.StepRun.TenantID.Equals(tenantId),
		),
	).OrderBy(
		db.StepRunResultArchive.Order.Order(db.DESC),
	).Exec(context.Background())
}

func (s *stepRunRepository) GetFirstArchivedStepRunResult(tenantId, stepRunId string) (*db.StepRunResultArchiveModel, error) {
	return s.client.StepRunResultArchive.FindFirst(
		db.StepRunResultArchive.StepRunID.Equals(stepRunId),
		db.StepRunResultArchive.StepRun.Where(
			db.StepRun.TenantID.Equals(tenantId),
		),
	).OrderBy(
		db.StepRunResultArchive.Order.Order(db.ASC),
	).Exec(context.Background())
}
