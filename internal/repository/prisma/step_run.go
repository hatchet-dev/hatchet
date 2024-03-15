package prisma

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
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
	"github.com/hatchet-dev/hatchet/internal/telemetry"
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
					return err
				}

				l.Err(err).Msgf("deadlock detected, retry %d", retries)

				// sleep with jitter
				sleepWithJitter(100*time.Millisecond, 300*time.Millisecond)
			} else {
				return err
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

func (s *stepRunRepository) AssignStepRunToWorker(tenantId, stepRunId string) (string, string, error) {
	// var assigned
	var assigned *dbsqlc.AssignStepRunToWorkerRow

	err := retrier(s.l, func() (err error) {
		assigned, err = s.queries.AssignStepRunToWorker(context.Background(), s.pool, dbsqlc.AssignStepRunToWorkerParams{
			Steprunid: sqlchelpers.UUIDFromStr(stepRunId),
			Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
		})

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return repository.ErrNoWorkerAvailable
			}

			return err
		}

		return nil
	})

	if err != nil {
		return "", "", err
	}

	return sqlchelpers.UUIDToStr(assigned.WorkerId), sqlchelpers.UUIDToStr(assigned.DispatcherId), nil
}

func (s *stepRunRepository) AssignStepRunToTicker(tenantId, stepRunId string) (tickerId string, err error) {
	err = retrier(s.l, func() error {
		assigned, err := s.queries.AssignStepRunToTicker(context.Background(), s.pool, dbsqlc.AssignStepRunToTickerParams{
			Steprunid: sqlchelpers.UUIDFromStr(stepRunId),
			Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
		})

		if err != nil {
			return err
		}

		tickerId = sqlchelpers.UUIDToStr(assigned.TickerId)

		return nil
	})

	return tickerId, err
}

func (s *stepRunRepository) UpdateStepRun(ctx context.Context, tenantId, stepRunId string, opts *repository.UpdateStepRunOpts) (*dbsqlc.GetStepRunForEngineRow, *repository.StepRunUpdateInfo, error) {
	ctx, span := telemetry.NewSpan(ctx, "update-step-run")
	defer span.End()

	if err := s.v.Validate(opts); err != nil {
		return nil, nil, err
	}

	updateParams, updateJobRunLookupDataParams, resolveJobRunParams, resolveLaterStepRunsParams, err := getUpdateParams(tenantId, stepRunId, opts)

	if err != nil {
		return nil, nil, err
	}

	updateInfo := &repository.StepRunUpdateInfo{}

	var stepRun *dbsqlc.GetStepRunForEngineRow

	err = retrier(s.l, func() error {
		tx, err := s.pool.Begin(context.Background())

		if err != nil {
			return err
		}

		defer deferRollback(context.Background(), s.l, tx.Rollback)

		stepRun, err = s.updateStepRunCore(ctx, tx, tenantId, updateParams, updateJobRunLookupDataParams)

		if err != nil {
			return err
		}

		err = tx.Commit(context.Background())

		return err
	})

	if err != nil {
		return nil, nil, err
	}

	err = retrier(s.l, func() error {
		tx, err := s.pool.Begin(context.Background())

		if err != nil {
			return err
		}

		defer deferRollback(context.Background(), s.l, tx.Rollback)

		updateInfo, err = s.updateStepRunExtra(ctx, tx, tenantId, resolveJobRunParams, resolveLaterStepRunsParams)

		if err != nil {
			return err
		}

		err = tx.Commit(context.Background())

		return err
	})

	if err != nil {
		// non-fatal error, log and continue
		s.l.Err(err).Msg("could not update step run extra")
		return nil, nil, nil
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

func (s *stepRunRepository) QueueStepRun(ctx context.Context, tenantId, stepRunId string, opts *repository.UpdateStepRunOpts) (*dbsqlc.GetStepRunForEngineRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "queue-step-run-database")
	defer span.End()

	if err := s.v.Validate(opts); err != nil {
		return nil, err
	}

	updateParams, updateJobRunLookupDataParams, resolveJobRunParams, resolveLaterStepRunsParams, err := getUpdateParams(tenantId, stepRunId, opts)

	if err != nil {
		return nil, err
	}

	var stepRun *dbsqlc.GetStepRunForEngineRow

	err = retrier(s.l, func() error {
		tx, err := s.pool.Begin(context.Background())

		if err != nil {
			return err
		}

		defer deferRollback(context.Background(), s.l, tx.Rollback)

		// get the step run and make sure it's still in pending
		innerStepRun, err := s.queries.GetStepRun(context.Background(), tx, dbsqlc.GetStepRunParams{
			ID:       sqlchelpers.UUIDFromStr(stepRunId),
			Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		})

		if err != nil {
			return err
		}

		if innerStepRun.Status != dbsqlc.StepRunStatusPENDING {
			return repository.ErrStepRunIsNotPending
		}

		stepRun, err = s.updateStepRunCore(ctx, tx, tenantId, updateParams, updateJobRunLookupDataParams)

		if err != nil {
			return err
		}

		if err != nil {
			return err
		}

		err = tx.Commit(context.Background())

		return err
	})

	err = retrier(s.l, func() error {
		tx, err := s.pool.Begin(context.Background())

		if err != nil {
			return err
		}

		defer deferRollback(context.Background(), s.l, tx.Rollback)

		_, err = s.updateStepRunExtra(ctx, tx, tenantId, resolveJobRunParams, resolveLaterStepRunsParams)

		if err != nil {
			return err
		}

		err = tx.Commit(context.Background())

		return err
	})

	if err != nil {
		// non-fatal error, log and continue
		s.l.Err(err).Msg("could not update step run extra")
		return nil, nil
	}

	return stepRun, nil
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

func (s *stepRunRepository) updateStepRunCore(
	ctx context.Context,
	tx pgx.Tx,
	tenantId string,
	updateParams dbsqlc.UpdateStepRunParams,
	updateJobRunLookupDataParams *dbsqlc.UpdateJobRunLookupDataWithStepRunParams,
) (*dbsqlc.GetStepRunForEngineRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "update-step-run-core")
	defer span.End()

	updateStepRun, err := s.queries.UpdateStepRun(ctx, tx, updateParams)

	if err != nil {
		return nil, fmt.Errorf("could not update step run: %w", err)
	}

	stepRuns, err := s.queries.GetStepRunForEngine(ctx, tx, dbsqlc.GetStepRunForEngineParams{
		Ids:      []pgtype.UUID{updateStepRun.ID},
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, fmt.Errorf("could not get step run for engine: %w", err)
	}

	// update the job run lookup data if not nil
	if updateJobRunLookupDataParams != nil {
		err = s.queries.UpdateJobRunLookupDataWithStepRun(ctx, tx, *updateJobRunLookupDataParams)

		if err != nil {
			return nil, fmt.Errorf("could not update job run lookup data: %w", err)
		}
	}

	if len(stepRuns) == 0 {
		return nil, fmt.Errorf("could not find step run for engine")
	}

	return stepRuns[0], nil
}

func (s *stepRunRepository) updateStepRunExtra(
	ctx context.Context,
	tx pgx.Tx,
	tenantId string,
	resolveJobRunParams dbsqlc.ResolveJobRunStatusParams,
	resolveLaterStepRunsParams dbsqlc.ResolveLaterStepRunsParams,
) (*repository.StepRunUpdateInfo, error) {
	ctx, span := telemetry.NewSpan(ctx, "update-step-run-extra")
	defer span.End()

	_, err := s.queries.ResolveLaterStepRuns(context.Background(), tx, resolveLaterStepRunsParams)

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

// performant query for step run id, only returns what the engine needs
func (s *stepRunRepository) GetStepRunForEngine(tenantId, stepRunId string) (*dbsqlc.GetStepRunForEngineRow, error) {
	res, err := s.queries.GetStepRunForEngine(context.Background(), s.pool, dbsqlc.GetStepRunForEngineParams{
		Ids:      []pgtype.UUID{sqlchelpers.UUIDFromStr(stepRunId)},
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, nil
	}

	return res[0], nil
}

func (s *stepRunRepository) ListStartableStepRuns(tenantId, jobRunId string, parentStepRunId *string) ([]*dbsqlc.GetStepRunForEngineRow, error) {
	tx, err := s.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer deferRollback(context.Background(), s.l, tx.Rollback)

	params := dbsqlc.ListStartableStepRunsParams{
		Jobrunid: sqlchelpers.UUIDFromStr(jobRunId),
	}

	if parentStepRunId != nil {
		params.ParentStepRunId = sqlchelpers.UUIDFromStr(*parentStepRunId)
	}

	srs, err := s.queries.ListStartableStepRuns(context.Background(), tx, params)

	if err != nil {
		return nil, err
	}

	res, err := s.queries.GetStepRunForEngine(context.Background(), tx, dbsqlc.GetStepRunForEngineParams{
		Ids:      srs,
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, err
	}

	err = tx.Commit(context.Background())

	return res, err
}

func (s *stepRunRepository) ArchiveStepRunResult(tenantId, stepRunId string) error {
	_, err := s.queries.ArchiveStepRunResultFromStepRun(context.Background(), s.pool, dbsqlc.ArchiveStepRunResultFromStepRunParams{
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
		Steprunid: sqlchelpers.UUIDFromStr(stepRunId),
	})

	return err
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

// sleepWithJitter sleeps for a random duration between min and max duration.
// min and max are time.Duration values, specifying the minimum and maximum sleep times.
func sleepWithJitter(min, max time.Duration) {
	if min > max {
		min, max = max, min // Swap if min is greater than max
	}

	jitter := max - min
	if jitter > 0 {
		sleepDuration := min + time.Duration(rand.Int63n(int64(jitter))) // nolint: gosec
		time.Sleep(sleepDuration)
	} else {
		time.Sleep(min) // Sleep for min duration if jitter is not positive
	}
}
