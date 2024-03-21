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

type stepRunAPIRepository struct {
	client  *db.PrismaClient
	pool    *pgxpool.Pool
	v       validator.Validator
	l       *zerolog.Logger
	queries *dbsqlc.Queries
}

func NewStepRunAPIRepository(client *db.PrismaClient, pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.StepRunAPIRepository {
	queries := dbsqlc.New()

	return &stepRunAPIRepository{
		client:  client,
		pool:    pool,
		v:       v,
		l:       l,
		queries: queries,
	}
}

func (s *stepRunAPIRepository) GetStepRunById(tenantId, stepRunId string) (*db.StepRunModel, error) {
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

func (s *stepRunAPIRepository) GetFirstArchivedStepRunResult(tenantId, stepRunId string) (*db.StepRunResultArchiveModel, error) {
	return s.client.StepRunResultArchive.FindFirst(
		db.StepRunResultArchive.StepRunID.Equals(stepRunId),
		db.StepRunResultArchive.StepRun.Where(
			db.StepRun.TenantID.Equals(tenantId),
		),
	).OrderBy(
		db.StepRunResultArchive.Order.Order(db.ASC),
	).Exec(context.Background())
}

type stepRunEngineRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	l       *zerolog.Logger
	queries *dbsqlc.Queries
}

func NewStepRunEngineRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.StepRunEngineRepository {
	queries := dbsqlc.New()

	return &stepRunEngineRepository{
		pool:    pool,
		v:       v,
		l:       l,
		queries: queries,
	}
}

func (s *stepRunEngineRepository) ListRunningStepRunsForTicker(tickerId string) ([]*dbsqlc.GetStepRunForEngineRow, error) {
	tx, err := s.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer deferRollback(context.Background(), s.l, tx.Rollback)

	srs, err := s.queries.ListStepRuns(context.Background(), s.pool, dbsqlc.ListStepRunsParams{
		Status: dbsqlc.NullStepRunStatus{
			StepRunStatus: dbsqlc.StepRunStatusRUNNING,
		},
		TickerId: sqlchelpers.UUIDFromStr(tickerId),
	})

	if err != nil {
		return nil, err
	}

	res, err := s.queries.GetStepRunForEngine(context.Background(), tx, dbsqlc.GetStepRunForEngineParams{
		Ids: srs,
	})

	if err != nil {
		return nil, err
	}

	err = tx.Commit(context.Background())

	return res, err
}

func (s *stepRunEngineRepository) ListRunningStepRunsForWorkflowRun(tenantId, workflowRunId string) ([]*dbsqlc.GetStepRunForEngineRow, error) {
	tx, err := s.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer deferRollback(context.Background(), s.l, tx.Rollback)

	srs, err := s.queries.ListStepRuns(context.Background(), s.pool, dbsqlc.ListStepRunsParams{
		Status: dbsqlc.NullStepRunStatus{
			StepRunStatus: dbsqlc.StepRunStatusRUNNING,
		},
		WorkflowRunId: sqlchelpers.UUIDFromStr(workflowRunId),
	})

	if err != nil {
		return nil, err
	}

	res, err := s.queries.GetStepRunForEngine(context.Background(), tx, dbsqlc.GetStepRunForEngineParams{
		Ids: srs,
	})

	if err != nil {
		return nil, err
	}

	err = tx.Commit(context.Background())

	return res, err
}

func (s *stepRunEngineRepository) ListStepRunsToRequeue(tenantId string) ([]*dbsqlc.StepRun, error) {
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

func (s *stepRunEngineRepository) ListStepRunsToReassign(tenantId string) ([]*dbsqlc.StepRun, error) {
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

func (s *stepRunEngineRepository) AssignStepRunToWorker(tenantId, stepRunId string) (string, string, error) {
	tx, err := s.pool.Begin(context.Background())

	if err != nil {
		return "", "", err
	}

	defer deferRollback(context.Background(), s.l, tx.Rollback)

	assigned, err := s.queries.AssignStepRunToWorker(context.Background(), tx, dbsqlc.AssignStepRunToWorkerParams{
		Steprunid: sqlchelpers.UUIDFromStr(stepRunId),
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", repository.ErrNoWorkerAvailable
		}

		return "", "", err
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return "", "", err
	}

	return sqlchelpers.UUIDToStr(assigned.WorkerId), sqlchelpers.UUIDToStr(assigned.DispatcherId), nil
}

func (s *stepRunEngineRepository) UpdateStepRun(ctx context.Context, tenantId, stepRunId string, opts *repository.UpdateStepRunOpts) (*dbsqlc.GetStepRunForEngineRow, *repository.StepRunUpdateInfo, error) {
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
		return nil, nil, fmt.Errorf("could not update step run extra: %w", err)
	}

	return stepRun, updateInfo, nil
}

func (s *stepRunEngineRepository) UpdateStepRunOverridesData(tenantId, stepRunId string, opts *repository.UpdateStepRunOverridesDataOpts) ([]byte, error) {
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

func (s *stepRunEngineRepository) UpdateStepRunInputSchema(tenantId, stepRunId string, schema []byte) ([]byte, error) {
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

func (s *stepRunEngineRepository) QueueStepRun(ctx context.Context, tenantId, stepRunId string, opts *repository.UpdateStepRunOpts) (*dbsqlc.GetStepRunForEngineRow, error) {
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

	retrierErr := retrier(s.l, func() error {
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

		sr, err := s.updateStepRunCore(ctx, tx, tenantId, updateParams, updateJobRunLookupDataParams)
		if err != nil {
			return err
		}

		stepRun = sr

		if err != nil {
			return err
		}

		if err := tx.Commit(context.Background()); err != nil {
			return err
		}

		return nil
	})

	if retrierErr != nil {
		return nil, fmt.Errorf("could not queue step run: %w", retrierErr)
	}

	retrierExtraErr := retrier(s.l, func() error {
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

	if retrierExtraErr != nil {
		return nil, fmt.Errorf("could not update step run extra: %w", retrierExtraErr)
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

func (s *stepRunEngineRepository) updateStepRunCore(
	ctx context.Context,
	tx pgx.Tx,
	tenantId string,
	updateParams dbsqlc.UpdateStepRunParams,
	updateJobRunLookupDataParams *dbsqlc.UpdateJobRunLookupDataWithStepRunParams,
) (*dbsqlc.GetStepRunForEngineRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "update-step-run-core") // nolint:ineffassign
	defer span.End()

	updateStepRun, err := s.queries.UpdateStepRun(ctx, tx, updateParams)

	if err != nil {
		return nil, fmt.Errorf("could not update step run: %w", err)
	}

	stepRuns, err := s.queries.GetStepRunForEngine(ctx, tx, dbsqlc.GetStepRunForEngineParams{
		Ids:      []pgtype.UUID{updateStepRun.ID},
		TenantId: sqlchelpers.UUIDFromStr(tenantId),
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

func (s *stepRunEngineRepository) updateStepRunExtra(
	ctx context.Context,
	tx pgx.Tx,
	tenantId string,
	resolveJobRunParams dbsqlc.ResolveJobRunStatusParams,
	resolveLaterStepRunsParams dbsqlc.ResolveLaterStepRunsParams,
) (*repository.StepRunUpdateInfo, error) {
	ctx, span := telemetry.NewSpan(ctx, "update-step-run-extra") // nolint:ineffassign
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

// performant query for step run id, only returns what the engine needs
func (s *stepRunEngineRepository) GetStepRunForEngine(tenantId, stepRunId string) (*dbsqlc.GetStepRunForEngineRow, error) {
	res, err := s.queries.GetStepRunForEngine(context.Background(), s.pool, dbsqlc.GetStepRunForEngineParams{
		Ids:      []pgtype.UUID{sqlchelpers.UUIDFromStr(stepRunId)},
		TenantId: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, nil
	}

	return res[0], nil
}

func (s *stepRunEngineRepository) ListStartableStepRuns(tenantId, jobRunId string, parentStepRunId *string) ([]*dbsqlc.GetStepRunForEngineRow, error) {
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
		TenantId: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, err
	}

	err = tx.Commit(context.Background())

	return res, err
}

func (s *stepRunEngineRepository) ArchiveStepRunResult(tenantId, stepRunId string) error {
	_, err := s.queries.ArchiveStepRunResultFromStepRun(context.Background(), s.pool, dbsqlc.ArchiveStepRunResultFromStepRunParams{
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
		Steprunid: sqlchelpers.UUIDFromStr(stepRunId),
	})

	return err
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
