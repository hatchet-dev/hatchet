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
		db.StepRun.ChildWorkflowRuns.Fetch(),
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

func (s *stepRunEngineRepository) ListRunningStepRunsForTicker(ctx context.Context, tickerId string) ([]*dbsqlc.GetStepRunForEngineRow, error) {
	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer deferRollback(ctx, s.l, tx.Rollback)

	srs, err := s.queries.ListStepRuns(ctx, tx, dbsqlc.ListStepRunsParams{
		Status: dbsqlc.NullStepRunStatus{
			StepRunStatus: dbsqlc.StepRunStatusRUNNING,
			Valid:         true,
		},
		TickerId: sqlchelpers.UUIDFromStr(tickerId),
	})

	if err != nil {
		return nil, err
	}

	res, err := s.queries.GetStepRunForEngine(ctx, tx, dbsqlc.GetStepRunForEngineParams{
		Ids: srs,
	})

	if err != nil {
		return nil, err
	}

	err = tx.Commit(ctx)

	return res, err
}

func (s *stepRunEngineRepository) ListStepRuns(ctx context.Context, tenantId string, opts *repository.ListStepRunsOpts) ([]*dbsqlc.GetStepRunForEngineRow, error) {
	if err := s.v.Validate(opts); err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer deferRollback(ctx, s.l, tx.Rollback)

	listOpts := dbsqlc.ListStepRunsParams{
		TenantId: sqlchelpers.UUIDFromStr(tenantId),
	}

	if opts.Status != nil {
		listOpts.Status = dbsqlc.NullStepRunStatus{
			StepRunStatus: *opts.Status,
			Valid:         true,
		}
	}

	if opts.WorkflowRunIds != nil {
		listOpts.WorkflowRunIds = make([]pgtype.UUID, len(opts.WorkflowRunIds))

		for i, id := range opts.WorkflowRunIds {
			listOpts.WorkflowRunIds[i] = sqlchelpers.UUIDFromStr(id)
		}
	}

	if opts.JobRunId != nil {
		listOpts.JobRunId = sqlchelpers.UUIDFromStr(*opts.JobRunId)
	}

	srs, err := s.queries.ListStepRuns(ctx, tx, listOpts)

	if err != nil {
		return nil, err
	}

	res, err := s.queries.GetStepRunForEngine(ctx, tx, dbsqlc.GetStepRunForEngineParams{
		Ids: srs,
	})

	if err != nil {
		return nil, err
	}

	err = tx.Commit(ctx)

	return res, err
}

func (s *stepRunEngineRepository) ListStepRunsToRequeue(ctx context.Context, tenantId string) ([]*dbsqlc.GetStepRunForEngineRow, error) {
	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer deferRollback(ctx, s.l, tx.Rollback)

	// get the step run and make sure it's still in pending
	stepRunIds, err := s.queries.ListStepRunsToRequeue(ctx, tx, pgTenantId)

	if err != nil {
		return nil, err
	}

	stepRuns, err := s.queries.GetStepRunForEngine(ctx, tx, dbsqlc.GetStepRunForEngineParams{
		Ids:      stepRunIds,
		TenantId: pgTenantId,
	})

	if err != nil {
		return nil, err
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, err
	}

	return stepRuns, nil
}

func (s *stepRunEngineRepository) ListStepRunsToReassign(ctx context.Context, tenantId string) ([]*dbsqlc.GetStepRunForEngineRow, error) {
	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer deferRollback(ctx, s.l, tx.Rollback)

	// get the step run and make sure it's still in pending
	stepRunIds, err := s.queries.ListStepRunsToReassign(ctx, tx, pgTenantId)

	if err != nil {
		return nil, err
	}

	stepRuns, err := s.queries.GetStepRunForEngine(ctx, tx, dbsqlc.GetStepRunForEngineParams{
		Ids:      stepRunIds,
		TenantId: pgTenantId,
	})

	if err != nil {
		return nil, err
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, err
	}

	return stepRuns, nil
}

var deadlockRetry = func(l *zerolog.Logger, f func() error) error {
	return genericRetry(l.Warn(), 3, f, "deadlock", func(err error) (bool, error) {
		return strings.Contains(err.Error(), "deadlock detected"), err
	})
}

var unassignedRetry = func(l *zerolog.Logger, f func() error) error {
	return genericRetry(l.Debug(), 5, f, "unassigned", func(err error) (bool, error) {
		var target *errNoWorkerWithSlots

		if errors.As(err, &target) {
			// if there are no slots available at all, don't retry
			if target.totalSlots != 0 {
				return true, err
			}

			return false, repository.ErrNoWorkerAvailable
		}

		return errors.Is(err, repository.ErrNoWorkerAvailable), err
	})
}

var genericRetry = func(l *zerolog.Event, maxRetries int, f func() error, msg string, condition func(err error) (bool, error)) error {
	retries := 0

	for {
		err := f()

		if err != nil {
			// condition detected, retry
			if ok, overrideErr := condition(err); ok {
				retries++

				if retries > maxRetries {
					return err
				}

				l.Err(err).Msgf("retry (%s) condition met, retry %d", msg, retries)

				// sleep with jitter
				sleepWithJitter(50*time.Millisecond, 200*time.Millisecond)
			} else {
				if overrideErr != nil {
					return overrideErr
				}

				return err
			}
		}

		if err == nil {
			if retries > 0 {
				l.Msgf("retry (%s) condition resolved after %d retries", msg, retries)
			}

			break
		}
	}

	return nil
}

func (s *stepRunEngineRepository) incrementWorkerSemaphore(ctx context.Context, stepRun *dbsqlc.GetStepRunForEngineRow) error {
	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return err
	}

	defer deferRollback(ctx, s.l, tx.Rollback)

	// Update the old worker semaphore. This will only increment if the step run was already assigned to a worker,
	// which means the step run is being retried or rerun.
	_, err = s.queries.UpdateWorkerSemaphore(ctx, tx, dbsqlc.UpdateWorkerSemaphoreParams{
		Inc:       1,
		Steprunid: stepRun.StepRun.ID,
		Tenantid:  stepRun.StepRun.TenantId,
	})

	isNoRowsErr := err != nil && errors.Is(err, pgx.ErrNoRows)

	if err != nil && !isNoRowsErr {
		return fmt.Errorf("could not upsert old worker semaphore: %w", err)
	}

	// this means that a worker is assigned: unlink the existing worker from the step run,
	// so that we don't re-increment the old worker semaphore on each retry
	if err == nil {
		_, err = s.queries.UnlinkStepRunFromWorker(ctx, tx, dbsqlc.UnlinkStepRunFromWorkerParams{
			Steprunid: stepRun.StepRun.ID,
			Tenantid:  stepRun.StepRun.TenantId,
		})

		if err != nil {
			return fmt.Errorf("could not unlink step run from worker: %w", err)
		}
	}

	return tx.Commit(ctx)
}

type errNoWorkerWithSlots struct {
	totalSlots int
}

func (e *errNoWorkerWithSlots) Error() string {
	return fmt.Sprintf("no worker available, slots left: %d", e.totalSlots)
}

func (s *stepRunEngineRepository) assignStepRunToWorkerAttempt(ctx context.Context, stepRun *dbsqlc.GetStepRunForEngineRow) (*dbsqlc.AssignStepRunToWorkerRow, error) {
	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer deferRollback(ctx, s.l, tx.Rollback)

	if err != nil {
		return nil, err
	}

	assigned, err := s.queries.AssignStepRunToWorker(ctx, tx, dbsqlc.AssignStepRunToWorkerParams{
		Steprunid:   stepRun.StepRun.ID,
		Tenantid:    stepRun.StepRun.TenantId,
		Actionid:    stepRun.ActionId,
		StepTimeout: stepRun.StepTimeout,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, repository.ErrNoWorkerAvailable
		}

		return nil, fmt.Errorf("query to assign worker failed: %w", err)
	}

	// if a row was returned, but does not have a valid UUID, we return no worker with the slots
	if assigned != nil && !assigned.ID.Valid {
		return nil, &errNoWorkerWithSlots{totalSlots: int(assigned.TsTotalSlots)}
	}

	semaphore, err := s.queries.UpdateWorkerSemaphore(ctx, tx, dbsqlc.UpdateWorkerSemaphoreParams{
		Inc:       -1,
		Steprunid: stepRun.StepRun.ID,
		Tenantid:  stepRun.StepRun.TenantId,
	})

	isNoRowsErr := err != nil && errors.Is(err, pgx.ErrNoRows)

	if err != nil && !isNoRowsErr {
		return nil, fmt.Errorf("could not upsert new worker semaphore: %w", err)
	}

	if !isNoRowsErr && semaphore.Slots < 0 {
		return nil, repository.ErrNoWorkerAvailable
	}

	rateLimits, err := s.queries.UpdateStepRateLimits(ctx, tx, dbsqlc.UpdateStepRateLimitsParams{
		Stepid:   stepRun.StepId,
		Tenantid: stepRun.StepRun.TenantId,
	})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("could not update rate limit: %w", err)
	}

	if len(rateLimits) > 0 {
		for _, rateLimit := range rateLimits {
			if rateLimit.Value < 0 {
				return nil, repository.ErrRateLimitExceeded
			}
		}
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, err
	}

	return assigned, nil
}

func (s *stepRunEngineRepository) AssignStepRunToWorker(ctx context.Context, stepRun *dbsqlc.GetStepRunForEngineRow) (string, string, error) {

	err := s.incrementWorkerSemaphore(ctx, stepRun)

	if err != nil {
		return "", "", err
	}

	var assigned *dbsqlc.AssignStepRunToWorkerRow

	err = unassignedRetry(s.l, func() (err error) {
		assigned, err = s.assignStepRunToWorkerAttempt(ctx, stepRun)

		if err != nil {
			var target *errNoWorkerWithSlots

			if errors.As(err, &target) {
				return err
			}

			if errors.Is(err, repository.ErrNoWorkerAvailable) {
				return err
			}

			return fmt.Errorf("could not assign worker: %w", err)
		}

		return nil
	})

	if err != nil {
		var target *errNoWorkerWithSlots

		if errors.As(err, &target) {
			return "", "", repository.ErrNoWorkerAvailable
		}

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

	err = deadlockRetry(s.l, func() error {
		tx, err := s.pool.Begin(ctx)

		if err != nil {
			return err
		}

		defer deferRollback(ctx, s.l, tx.Rollback)

		innerStepRun, err := s.queries.GetStepRun(ctx, tx, dbsqlc.GetStepRunParams{
			ID:       sqlchelpers.UUIDFromStr(stepRunId),
			Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		})

		if err != nil {
			return err
		}

		stepRun, err = s.updateStepRunCore(ctx, tx, tenantId, updateParams, updateJobRunLookupDataParams, innerStepRun)

		if err != nil {
			return err
		}

		err = tx.Commit(ctx)

		return err
	})

	if err != nil {
		return nil, nil, err
	}

	err = deadlockRetry(s.l, func() error {
		tx, err := s.pool.Begin(ctx)

		if err != nil {
			return err
		}

		defer deferRollback(ctx, s.l, tx.Rollback)

		updateInfo, err = s.updateStepRunExtra(ctx, tx, tenantId, resolveJobRunParams, resolveLaterStepRunsParams)

		if err != nil {
			return err
		}

		err = tx.Commit(ctx)

		return err
	})

	if err != nil {
		return nil, nil, fmt.Errorf("could not update step run extra: %w", err)
	}

	return stepRun, updateInfo, nil
}

func (s *stepRunEngineRepository) UnlinkStepRunFromWorker(ctx context.Context, tenantId, stepRunId string) error {
	_, err := s.queries.UnlinkStepRunFromWorker(ctx, s.pool, dbsqlc.UnlinkStepRunFromWorkerParams{
		Steprunid: sqlchelpers.UUIDFromStr(stepRunId),
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return fmt.Errorf("could not unlink step run from worker: %w", err)
	}

	return nil
}

func (s *stepRunEngineRepository) UpdateStepRunOverridesData(ctx context.Context, tenantId, stepRunId string, opts *repository.UpdateStepRunOverridesDataOpts) ([]byte, error) {
	if err := s.v.Validate(opts); err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer deferRollback(ctx, s.l, tx.Rollback)

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
		ctx,
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

	err = tx.Commit(ctx)

	if err != nil {
		return nil, err
	}

	return input, nil
}

func (s *stepRunEngineRepository) UpdateStepRunInputSchema(ctx context.Context, tenantId, stepRunId string, schema []byte) ([]byte, error) {
	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer deferRollback(ctx, s.l, tx.Rollback)

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)
	pgStepRunId := sqlchelpers.UUIDFromStr(stepRunId)

	inputSchema, err := s.queries.UpdateStepRunInputSchema(
		ctx,
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

	err = tx.Commit(ctx)

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
	var isNotPending bool

	retrierErr := deadlockRetry(s.l, func() error {
		tx, err := s.pool.Begin(ctx)

		if err != nil {
			return err
		}

		defer deferRollback(ctx, s.l, tx.Rollback)

		// get the step run and make sure it's still in pending
		innerStepRun, err := s.queries.GetStepRun(ctx, tx, dbsqlc.GetStepRunParams{
			ID:       sqlchelpers.UUIDFromStr(stepRunId),
			Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		})

		if err != nil {
			return err
		}

		// if the step run is not pending, we can't queue it, but we still want to update other input params
		if innerStepRun.Status != dbsqlc.StepRunStatusPENDING {
			updateParams.Status = dbsqlc.NullStepRunStatus{}

			isNotPending = true
		}

		sr, err := s.updateStepRunCore(ctx, tx, tenantId, updateParams, updateJobRunLookupDataParams, innerStepRun)
		if err != nil {
			return err
		}

		stepRun = sr

		if err != nil {
			return err
		}

		if err := tx.Commit(ctx); err != nil {
			return err
		}

		return nil
	})

	if retrierErr != nil {
		return nil, fmt.Errorf("could not queue step run: %w", retrierErr)
	}

	if isNotPending {
		return nil, repository.ErrStepRunIsNotPending
	}

	retrierExtraErr := deadlockRetry(s.l, func() error {
		tx, err := s.pool.Begin(ctx)

		if err != nil {
			return err
		}

		defer deferRollback(ctx, s.l, tx.Rollback)

		_, err = s.updateStepRunExtra(ctx, tx, tenantId, resolveJobRunParams, resolveLaterStepRunsParams)

		if err != nil {
			return err
		}

		err = tx.Commit(ctx)

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
	innerStepRun *dbsqlc.StepRun,
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

	if updateParams.Status.Valid &&
		repository.IsFinalStepRunStatus(updateParams.Status.StepRunStatus) &&
		// we must have actually updated the status to a different state
		string(innerStepRun.Status) != string(updateStepRun.Status) {
		_, err := s.queries.UpdateWorkerSemaphore(ctx, tx, dbsqlc.UpdateWorkerSemaphoreParams{
			Inc:       1,
			Steprunid: updateStepRun.ID,
			Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
		})

		// not a fatal error if there's not a semaphore to update
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("could not upsert worker semaphore: %w", err)
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

	_, err := s.queries.ResolveLaterStepRuns(ctx, tx, resolveLaterStepRunsParams)

	if err != nil {
		return nil, fmt.Errorf("could not resolve later step runs: %w", err)
	}

	jobRun, err := s.queries.ResolveJobRunStatus(ctx, tx, resolveJobRunParams)

	if err != nil {
		return nil, fmt.Errorf("could not resolve job run status: %w", err)
	}

	resolveWorkflowRunParams := dbsqlc.ResolveWorkflowRunStatusParams{
		Jobrunid: jobRun.ID,
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	}

	workflowRun, err := s.queries.ResolveWorkflowRunStatus(ctx, tx, resolveWorkflowRunParams)

	if err != nil {
		return nil, fmt.Errorf("could not resolve workflow run status: %w", err)
	}

	return &repository.StepRunUpdateInfo{
		JobRunFinalState:      repository.IsFinalJobRunStatus(jobRun.Status),
		WorkflowRunFinalState: repository.IsFinalWorkflowRunStatus(workflowRun.Status),
		WorkflowRunId:         sqlchelpers.UUIDToStr(workflowRun.ID),
		WorkflowRunStatus:     string(workflowRun.Status),
	}, nil
}

// performant query for step run id, only returns what the engine needs
func (s *stepRunEngineRepository) GetStepRunForEngine(ctx context.Context, tenantId, stepRunId string) (*dbsqlc.GetStepRunForEngineRow, error) {
	res, err := s.queries.GetStepRunForEngine(ctx, s.pool, dbsqlc.GetStepRunForEngineParams{
		Ids:      []pgtype.UUID{sqlchelpers.UUIDFromStr(stepRunId)},
		TenantId: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return nil, fmt.Errorf("could not find step run %s", stepRunId)
	}

	return res[0], nil
}

func (s *stepRunEngineRepository) ListStartableStepRuns(ctx context.Context, tenantId, jobRunId string, parentStepRunId *string) ([]*dbsqlc.GetStepRunForEngineRow, error) {
	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer deferRollback(ctx, s.l, tx.Rollback)

	params := dbsqlc.ListStartableStepRunsParams{
		Jobrunid: sqlchelpers.UUIDFromStr(jobRunId),
	}

	if parentStepRunId != nil {
		params.ParentStepRunId = sqlchelpers.UUIDFromStr(*parentStepRunId)
	}

	srs, err := s.queries.ListStartableStepRuns(ctx, tx, params)

	if err != nil {
		return nil, err
	}

	res, err := s.queries.GetStepRunForEngine(ctx, tx, dbsqlc.GetStepRunForEngineParams{
		Ids:      srs,
		TenantId: sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return nil, err
	}

	err = tx.Commit(ctx)

	return res, err
}

func (s *stepRunEngineRepository) ArchiveStepRunResult(ctx context.Context, tenantId, stepRunId string) error {
	_, err := s.queries.ArchiveStepRunResultFromStepRun(ctx, s.pool, dbsqlc.ArchiveStepRunResultFromStepRunParams{
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
