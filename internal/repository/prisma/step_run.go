package prisma

import (
	"context"
	"encoding/json"
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

func (s *stepRunAPIRepository) ListStepRunEvents(stepRunId string, opts *repository.ListStepRunEventOpts) (*repository.ListStepRunEventResult, error) {
	if err := s.v.Validate(opts); err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer deferRollback(context.Background(), s.l, tx.Rollback)

	pgStepRunId := sqlchelpers.UUIDFromStr(stepRunId)

	listParams := dbsqlc.ListStepRunEventsParams{
		Steprunid: pgStepRunId,
	}

	if opts.Offset != nil {
		listParams.Offset = *opts.Offset
	}

	if opts.Limit != nil {
		listParams.Limit = *opts.Limit
	}

	events, err := s.queries.ListStepRunEvents(context.Background(), tx, listParams)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			events = make([]*dbsqlc.StepRunEvent, 0)
		} else {
			return nil, fmt.Errorf("could not list step run events: %w", err)
		}
	}

	count, err := s.queries.CountStepRunEvents(context.Background(), tx, pgStepRunId)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			count = 0
		} else {
			return nil, fmt.Errorf("could not count step run events: %w", err)
		}
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	return &repository.ListStepRunEventResult{
		Rows:  events,
		Count: int(count),
	}, nil
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
	}, 50*time.Millisecond, 200*time.Millisecond)
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
	}, 50*time.Millisecond, 100*time.Millisecond)
}

var genericRetry = func(l *zerolog.Event, maxRetries int, f func() error, msg string, condition func(err error) (bool, error), minSleep, maxSleep time.Duration) error {
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
				sleepWithJitter(minSleep, maxSleep)
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

func (s *stepRunEngineRepository) ReleaseStepRunSemaphore(ctx context.Context, tenantId, stepRunId string) error {
	return deadlockRetry(s.l, func() error {
		tx, err := s.pool.Begin(ctx)

		if err != nil {
			return err
		}

		defer deferRollback(ctx, s.l, tx.Rollback)

		stepRun, err := s.getStepRunForEngineTx(context.Background(), tx, tenantId, stepRunId)

		if err != nil {
			return fmt.Errorf("could not get step run for engine: %w", err)
		}

		if stepRun.StepRun.SemaphoreReleased {
			return nil
		}

		data := map[string]interface{}{"worker_id": sqlchelpers.UUIDToStr(stepRun.StepRun.WorkerId)}

		dataBytes, err := json.Marshal(data)

		if err != nil {
			return fmt.Errorf("could not marshal data: %w", err)
		}

		err = s.queries.CreateStepRunEvent(ctx, tx, dbsqlc.CreateStepRunEventParams{
			Steprunid: stepRun.StepRun.ID,
			Reason:    dbsqlc.StepRunEventReasonSLOTRELEASED,
			Severity:  dbsqlc.StepRunEventSeverityINFO,
			Message:   "Slot released",
			Data:      dataBytes,
		})

		if err != nil {
			return fmt.Errorf("could not create step run event: %w", err)
		}

		_, err = s.queries.ReleaseWorkerSemaphoreSlot(ctx, tx, dbsqlc.ReleaseWorkerSemaphoreSlotParams{
			Steprunid: stepRun.StepRun.ID,
			Tenantid:  stepRun.StepRun.TenantId,
		})

		if err != nil {
			return fmt.Errorf("could not release worker semaphore slot: %w", err)
		}

		_, err = s.queries.UnlinkStepRunFromWorker(ctx, tx, dbsqlc.UnlinkStepRunFromWorkerParams{
			Steprunid: stepRun.StepRun.ID,
			Tenantid:  stepRun.StepRun.TenantId,
		})

		if err != nil {
			return fmt.Errorf("could not unlink step run from worker: %w", err)
		}

		// Update the Step Run to release the semaphore
		_, err = s.queries.UpdateStepRun(ctx, tx, dbsqlc.UpdateStepRunParams{
			ID:       stepRun.StepRun.ID,
			Tenantid: stepRun.StepRun.TenantId,
			SemaphoreReleased: pgtype.Bool{
				Valid: true,
				Bool:  true,
			},
		})

		if err != nil {
			return fmt.Errorf("could not update step run semaphoreRelease: %w", err)
		}

		return tx.Commit(ctx)
	})
}

func (s *stepRunEngineRepository) releaseWorkerSemaphore(ctx context.Context, stepRun *dbsqlc.GetStepRunForEngineRow) error {
	return deadlockRetry(s.l, func() error {
		tx, err := s.pool.Begin(ctx)

		if err != nil {
			return err
		}

		defer deferRollback(ctx, s.l, tx.Rollback)

		_, err = s.queries.ReleaseWorkerSemaphoreSlot(ctx, tx, dbsqlc.ReleaseWorkerSemaphoreSlotParams{
			Steprunid: stepRun.StepRun.ID,
			Tenantid:  stepRun.StepRun.TenantId,
		})

		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("could not release previous worker semaphore: %w", err)
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
	})
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

	// acquire a semaphore slot
	semaphore, err := s.queries.AcquireWorkerSemaphoreSlot(ctx, tx, dbsqlc.AcquireWorkerSemaphoreSlotParams{
		Steprunid: stepRun.StepRun.ID,
		Tenantid:  stepRun.StepRun.TenantId,
		Actionid:  stepRun.ActionId,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &errNoWorkerWithSlots{totalSlots: int(0)}
		}
		return nil, fmt.Errorf("could not acquire worker semaphore slot: %w", err)
	}

	assigned, err := s.queries.AssignStepRunToWorker(ctx, tx, dbsqlc.AssignStepRunToWorkerParams{
		Steprunid:   stepRun.StepRun.ID,
		Tenantid:    stepRun.StepRun.TenantId,
		StepTimeout: stepRun.StepTimeout,
		Workerid:    semaphore.WorkerId,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.l.Warn().Err(err).Msg("no rows returned from worker assign")
		}

		return nil, fmt.Errorf("query to assign worker failed: %w", err)
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

func (s *stepRunEngineRepository) deferredStepRunEvent(
	stepRunId pgtype.UUID,
	reason dbsqlc.StepRunEventReason,
	severity dbsqlc.StepRunEventSeverity,
	message string,
	data map[string]interface{},
) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dataBytes, err := json.Marshal(data)

	if err != nil {
		s.l.Err(err).Msg("could not marshal deferred step run event data")
		return
	}

	err = s.queries.CreateStepRunEvent(ctx, s.pool, dbsqlc.CreateStepRunEventParams{
		Steprunid: stepRunId,
		Message:   message,
		Reason:    reason,
		Severity:  severity,
		Data:      dataBytes,
	})

	if err != nil {
		s.l.Err(err).Msg("could not create deferred step run event")
	}
}

func (s *stepRunEngineRepository) AssignStepRunToWorker(ctx context.Context, stepRun *dbsqlc.GetStepRunForEngineRow) (string, string, error) {
	err := s.releaseWorkerSemaphore(ctx, stepRun)

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

			return fmt.Errorf("could not assign worker for step run %s (step %s): %w", sqlchelpers.UUIDToStr(stepRun.StepRun.ID), stepRun.StepReadableId.String, err)
		}

		return nil
	})

	if err != nil {
		var target *errNoWorkerWithSlots

		if errors.As(err, &target) {
			defer s.deferredStepRunEvent(
				stepRun.StepRun.ID,
				dbsqlc.StepRunEventReasonREQUEUEDNOWORKER,
				dbsqlc.StepRunEventSeverityWARNING,
				"No worker available",
				nil,
			)

			return "", "", repository.ErrNoWorkerAvailable
		}

		if errors.Is(err, repository.ErrNoWorkerAvailable) {
			defer s.deferredStepRunEvent(
				stepRun.StepRun.ID,
				dbsqlc.StepRunEventReasonREQUEUEDNOWORKER,
				dbsqlc.StepRunEventSeverityWARNING,
				"No worker available",
				nil,
			)
		}

		if errors.Is(err, repository.ErrRateLimitExceeded) {
			defer s.deferredStepRunEvent(
				stepRun.StepRun.ID,
				dbsqlc.StepRunEventReasonREQUEUEDRATELIMIT,
				dbsqlc.StepRunEventSeverityWARNING,
				"Rate limit exceeded",
				nil,
			)
		}

		return "", "", err
	}

	defer s.deferredStepRunEvent(
		stepRun.StepRun.ID,
		dbsqlc.StepRunEventReasonASSIGNED,
		dbsqlc.StepRunEventSeverityINFO,
		fmt.Sprintf("Assigned to worker %s", sqlchelpers.UUIDToStr(assigned.WorkerId)),
		map[string]interface{}{
			"worker_id": sqlchelpers.UUIDToStr(assigned.WorkerId),
		},
	)

	return sqlchelpers.UUIDToStr(assigned.WorkerId), sqlchelpers.UUIDToStr(assigned.DispatcherId), nil
}

func (s *stepRunEngineRepository) UpdateStepRun(ctx context.Context, tenantId, stepRunId string, opts *repository.UpdateStepRunOpts) (*dbsqlc.GetStepRunForEngineRow, *repository.StepRunUpdateInfo, error) {
	ctx, span := telemetry.NewSpan(ctx, "update-step-run")
	defer span.End()

	if err := s.v.Validate(opts); err != nil {
		return nil, nil, err
	}

	updateParams, createEventParams, updateJobRunLookupDataParams, resolveJobRunParams, resolveLaterStepRunsParams, err := getUpdateParams(tenantId, stepRunId, opts)

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

		stepRun, err = s.updateStepRunCore(ctx, tx, tenantId, updateParams, createEventParams, updateJobRunLookupDataParams, innerStepRun)

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

func (s *stepRunEngineRepository) ReplayStepRun(ctx context.Context, tenantId, stepRunId string, opts *repository.UpdateStepRunOpts) (*dbsqlc.GetStepRunForEngineRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "replay-step-run")
	defer span.End()

	if err := s.v.Validate(opts); err != nil {
		return nil, err
	}

	updateParams, createEventParams, updateJobRunLookupDataParams, _, _, err := getUpdateParams(tenantId, stepRunId, opts)

	if err != nil {
		return nil, err
	}

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

		stepRun, err = s.updateStepRunCore(ctx, tx, tenantId, updateParams, createEventParams, updateJobRunLookupDataParams, innerStepRun)

		if err != nil {
			return err
		}

		// reset the job run, workflow run and all fields as part of the core tx
		_, err = s.queries.ReplayStepRunResetWorkflowRun(ctx, tx, stepRun.JobRunId)

		if err != nil {
			return err
		}

		_, err = s.queries.ReplayStepRunResetJobRun(ctx, tx, stepRun.JobRunId)

		if err != nil {
			return err
		}

		laterStepRuns, err := s.queries.GetLaterStepRunsForReplay(ctx, tx, dbsqlc.GetLaterStepRunsForReplayParams{
			Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
			Steprunid: sqlchelpers.UUIDFromStr(stepRunId),
		})

		if err != nil {
			return err
		}

		// archive each of the later step run results
		for _, laterStepRun := range laterStepRuns {
			laterStepRunCp := laterStepRun
			laterStepRunId := sqlchelpers.UUIDToStr(laterStepRun.ID)

			err = s.archiveStepRunResult(ctx, tx, tenantId, laterStepRunId)

			if err != nil {
				return err
			}

			// remove the previous step run result from the job lookup data
			err = s.queries.UpdateJobRunLookupDataWithStepRun(
				ctx,
				tx,
				dbsqlc.UpdateJobRunLookupDataWithStepRunParams{
					Steprunid: sqlchelpers.UUIDFromStr(laterStepRunId),
					Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
				},
			)

			if err != nil {
				return err
			}

			// create a deferred event for each of these step runs
			defer s.deferredStepRunEvent(
				laterStepRunCp.ID,
				dbsqlc.StepRunEventReasonRETRIEDBYUSER,
				dbsqlc.StepRunEventSeverityINFO,
				fmt.Sprintf("Parent step run %s was replayed, resetting step run result", stepRun.StepReadableId.String),
				nil,
			)
		}

		// reset all later step runs to a pending state
		_, err = s.queries.ReplayStepRunResetLaterStepRuns(ctx, tx, dbsqlc.ReplayStepRunResetLaterStepRunsParams{
			Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
			Steprunid: sqlchelpers.UUIDFromStr(stepRunId),
		})

		if err != nil {
			return err
		}

		err = tx.Commit(ctx)

		return err
	})

	if err != nil {
		return nil, err
	}

	return stepRun, nil
}

func (s *stepRunEngineRepository) PreflightCheckReplayStepRun(ctx context.Context, tenantId, stepRunId string) error {
	// verify that the step run is in a final state
	stepRun, err := s.getStepRunForEngineTx(ctx, s.pool, tenantId, stepRunId)

	if err != nil {
		return err
	}

	if !repository.IsFinalStepRunStatus(stepRun.StepRun.Status) {
		return repository.ErrPreflightReplayStepRunNotInFinalState
	}

	// verify that child step runs are in a final state
	childStepRuns, err := s.queries.ListNonFinalChildStepRuns(ctx, s.pool, dbsqlc.ListNonFinalChildStepRunsParams{
		Steprunid: sqlchelpers.UUIDFromStr(stepRunId),
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("could not list non-final child step runs: %w", err)
	}

	if len(childStepRuns) > 0 {
		return repository.ErrPreflightReplayChildStepRunNotInFinalState
	}

	return nil
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

	updateParams,
		createEventParams,
		updateJobRunLookupDataParams,
		resolveJobRunParams,
		resolveLaterStepRunsParams,
		err := getUpdateParams(tenantId, stepRunId, opts)

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

		sr, err := s.updateStepRunCore(ctx, tx, tenantId, updateParams, createEventParams, updateJobRunLookupDataParams, innerStepRun)

		if err != nil {
			return err
		}

		stepRun = sr

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

func (s *stepRunEngineRepository) CreateStepRunEvent(ctx context.Context, tenantId, stepRunId string, opts repository.CreateStepRunEventOpts) error {

	pgStepRunId := sqlchelpers.UUIDFromStr(stepRunId)

	if opts.EventMessage != nil && opts.EventReason != nil {
		severity := dbsqlc.StepRunEventSeverityINFO

		if opts.EventSeverity != nil {
			severity = *opts.EventSeverity
		}

		var eventData []byte
		var err error

		if opts.EventData != nil {
			eventData, err = json.Marshal(opts.EventData)

			if err != nil {
				return fmt.Errorf("could not marshal step run event data: %w", err)
			}
		}

		createParams := &dbsqlc.CreateStepRunEventParams{
			Steprunid: pgStepRunId,
			Message:   *opts.EventMessage,
			Reason:    *opts.EventReason,
			Severity:  severity,
			Data:      eventData,
		}

		err = s.queries.CreateStepRunEvent(ctx, s.pool, *createParams)

		if err != nil {
			return fmt.Errorf("could not create step run event: %w", err)
		}
	}

	return nil
}

func getUpdateParams(
	tenantId,
	stepRunId string,
	opts *repository.UpdateStepRunOpts,
) (
	updateParams dbsqlc.UpdateStepRunParams,
	createStepRunEventParams *dbsqlc.CreateStepRunEventParams,
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

	// if this is a rerun, we need to reset semaphore released flag
	if opts.IsRerun {
		updateParams.SemaphoreReleased = pgtype.Bool{
			Valid: true,
			Bool:  false,
		}
	}

	var createParams *dbsqlc.CreateStepRunEventParams

	event := opts.Event

	if event != nil && event.EventMessage != nil && event.EventReason != nil {
		severity := dbsqlc.StepRunEventSeverityINFO

		if event.EventSeverity != nil {
			severity = *event.EventSeverity
		}

		createParams = &dbsqlc.CreateStepRunEventParams{
			Steprunid: pgStepRunId,
			Message:   *event.EventMessage,
			Reason:    *event.EventReason,
			Severity:  severity,
		}
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
			return updateParams, nil, nil, resolveJobRunParams, resolveLaterStepRunsParams, err
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

	return updateParams,
		createParams,
		updateJobRunLookupDataParams,
		resolveJobRunParams,
		resolveLaterStepRunsParams,
		nil
}

func (s *stepRunEngineRepository) updateStepRunCore(
	ctx context.Context,
	tx pgx.Tx,
	tenantId string,
	updateParams dbsqlc.UpdateStepRunParams,
	createEventParams *dbsqlc.CreateStepRunEventParams,
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

	// create a step run event if not nil
	if createEventParams != nil {
		err = s.queries.CreateStepRunEvent(ctx, tx, *createEventParams)

		if err != nil {
			return nil, fmt.Errorf("could not create step run event: %w", err)
		}
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
		// the semaphore has not already been released manually
		!updateStepRun.SemaphoreReleased &&
		// we must have actually updated the status to a different state
		string(innerStepRun.Status) != string(updateStepRun.Status) {

		_, err = s.queries.ReleaseWorkerSemaphoreSlot(ctx, tx, dbsqlc.ReleaseWorkerSemaphoreSlotParams{
			Steprunid: updateStepRun.ID,
			Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
		})

		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("could not release worker semaphore: %w", err)
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
	return s.getStepRunForEngineTx(ctx, s.pool, tenantId, stepRunId)
}

func (s *stepRunEngineRepository) getStepRunForEngineTx(ctx context.Context, dbtx dbsqlc.DBTX, tenantId, stepRunId string) (*dbsqlc.GetStepRunForEngineRow, error) {
	res, err := s.queries.GetStepRunForEngine(ctx, dbtx, dbsqlc.GetStepRunForEngineParams{
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
	return s.archiveStepRunResult(ctx, s.pool, tenantId, stepRunId)
}

func (s *stepRunEngineRepository) archiveStepRunResult(ctx context.Context, db dbsqlc.DBTX, tenantId, stepRunId string) error {
	_, err := s.queries.ArchiveStepRunResultFromStepRun(ctx, db, dbsqlc.ArchiveStepRunResultFromStepRunParams{
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

func (s *stepRunEngineRepository) RefreshTimeoutBy(ctx context.Context, tenantId, stepRunId string, opts repository.RefreshTimeoutBy) (*dbsqlc.StepRun, error) {
	stepRunUUID := sqlchelpers.UUIDFromStr(stepRunId)
	tenantUUID := sqlchelpers.UUIDFromStr(tenantId)

	incrementTimeoutBy := opts.IncrementTimeoutBy

	err := s.v.Validate(opts)

	if err != nil {
		return nil, err
	}

	res, err := s.queries.RefreshTimeoutBy(ctx, s.pool, dbsqlc.RefreshTimeoutByParams{
		Steprunid:          stepRunUUID,
		Tenantid:           tenantUUID,
		IncrementTimeoutBy: sqlchelpers.TextFromStr(incrementTimeoutBy),
	})

	if err != nil {
		return nil, err
	}

	defer s.deferredStepRunEvent(
		stepRunUUID,
		dbsqlc.StepRunEventReasonTIMEOUTREFRESHED,
		dbsqlc.StepRunEventSeverityINFO,
		fmt.Sprintf("Timeout refreshed by %s", incrementTimeoutBy),
		nil)

	return res, nil
}
