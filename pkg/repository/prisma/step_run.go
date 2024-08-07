package prisma

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/validator"
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
	return s.client.StepRun.FindFirst(
		db.StepRun.ID.Equals(stepRunId),
		db.StepRun.DeletedAt.IsNull(),
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

func (s *stepRunAPIRepository) ListStepRunArchives(tenantId string, stepRunId string, opts *repository.ListStepRunArchivesOpts) (*repository.ListStepRunArchivesResult, error) {
	if err := s.v.Validate(opts); err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer deferRollback(context.Background(), s.l, tx.Rollback)

	pgStepRunId := sqlchelpers.UUIDFromStr(stepRunId)
	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	listParams := dbsqlc.ListStepRunArchivesParams{
		Steprunid: pgStepRunId,
		Tenantid:  pgTenantId,
	}

	if opts.Offset != nil {
		listParams.Offset = *opts.Offset
	}

	if opts.Limit != nil {
		listParams.Limit = *opts.Limit
	}

	archives, err := s.queries.ListStepRunArchives(context.Background(), tx, listParams)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			archives = make([]*dbsqlc.StepRunResultArchive, 0)
		} else {
			return nil, fmt.Errorf("could not list step run archives: %w", err)
		}
	}

	count, err := s.queries.CountStepRunArchives(context.Background(), tx, pgStepRunId)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			count = 0
		} else {
			return nil, fmt.Errorf("could not count step run archives: %w", err)
		}
	}

	err = tx.Commit(context.Background())

	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	return &repository.ListStepRunArchivesResult{
		Rows:  archives,
		Count: int(count),
	}, nil
}

type stepRunEngineRepository struct {
	pool    *pgxpool.Pool
	v       validator.Validator
	l       *zerolog.Logger
	queries *dbsqlc.Queries
	cf      *server.ConfigFileRuntime
}

func NewStepRunEngineRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger, cf *server.ConfigFileRuntime) repository.StepRunEngineRepository {
	queries := dbsqlc.New()

	return &stepRunEngineRepository{
		pool:    pool,
		v:       v,
		l:       l,
		queries: queries,
		cf:      cf,
	}
}

func (s *stepRunEngineRepository) GetStepRunMetaForEngine(ctx context.Context, tenantId, stepRunId string) (*dbsqlc.GetStepRunMetaRow, error) {
	return s.queries.GetStepRunMeta(ctx, s.pool, dbsqlc.GetStepRunMetaParams{
		Steprunid: sqlchelpers.UUIDFromStr(stepRunId),
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
	})
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

	if err != nil {
		return nil, err
	}

	return res, nil

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

	// get the limits for the step runs
	limit, err := s.queries.GetMaxRunsLimit(ctx, tx, pgTenantId)

	if err != nil {
		return nil, err
	}

	if limit > int32(s.cf.RequeueLimit) {
		limit = int32(s.cf.RequeueLimit)
	}

	// get the step run and make sure it's still in pending
	stepRunIds, err := s.queries.ListStepRunsToRequeue(ctx, tx, dbsqlc.ListStepRunsToRequeueParams{
		Tenantid: pgTenantId,
		Limit:    limit,
	})

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

func (s *stepRunEngineRepository) ListStepRunsToTimeout(ctx context.Context, tenantId string) ([]*dbsqlc.GetStepRunForEngineRow, error) {
	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer deferRollback(ctx, s.l, tx.Rollback)

	// get the step run and make sure it's still in pending
	stepRunIds, err := s.queries.ListStepRunsToTimeout(ctx, tx, pgTenantId)

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

		if stepRun.SRSemaphoreReleased {
			return nil
		}

		data := map[string]interface{}{"worker_id": sqlchelpers.UUIDToStr(stepRun.SRWorkerId)}

		dataBytes, err := json.Marshal(data)

		if err != nil {
			return fmt.Errorf("could not marshal data: %w", err)
		}

		err = s.queries.CreateStepRunEvent(ctx, tx, dbsqlc.CreateStepRunEventParams{
			Steprunid: stepRun.SRID,
			Reason:    dbsqlc.StepRunEventReasonSLOTRELEASED,
			Severity:  dbsqlc.StepRunEventSeverityINFO,
			Message:   "Slot released",
			Data:      dataBytes,
		})

		if err != nil {
			return fmt.Errorf("could not create step run event: %w", err)
		}

		_, err = s.queries.ReleaseWorkerSemaphoreSlot(ctx, tx, dbsqlc.ReleaseWorkerSemaphoreSlotParams{
			Steprunid: stepRun.SRID,
			Tenantid:  stepRun.SRTenantId,
		})

		if err != nil {
			return fmt.Errorf("could not release worker semaphore slot: %w", err)
		}

		_, err = s.queries.UnlinkStepRunFromWorker(ctx, tx, dbsqlc.UnlinkStepRunFromWorkerParams{
			Steprunid: stepRun.SRID,
			Tenantid:  stepRun.SRTenantId,
		})

		if err != nil {
			return fmt.Errorf("could not unlink step run from worker: %w", err)
		}

		// Update the Step Run to release the semaphore
		_, err = s.queries.UpdateStepRun(ctx, tx, dbsqlc.UpdateStepRunParams{
			ID:       stepRun.SRID,
			Tenantid: stepRun.SRTenantId,
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
			Steprunid: stepRun.SRID,
			Tenantid:  stepRun.SRTenantId,
		})

		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("could not release previous worker semaphore: %w", err)
		}

		// this means that a worker is assigned: unlink the existing worker from the step run,
		// so that we don't re-increment the old worker semaphore on each retry
		if err == nil {
			_, err = s.queries.UnlinkStepRunFromWorker(ctx, tx, dbsqlc.UnlinkStepRunFromWorkerParams{
				Steprunid: stepRun.SRID,
				Tenantid:  stepRun.SRTenantId,
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

func (s *stepRunEngineRepository) assignStepRunToWorkerAttempt(ctx context.Context, stepRun *dbsqlc.GetStepRunForEngineRow) (*dbsqlc.AcquireWorkerSemaphoreSlotAndAssignRow, error) {
	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer deferRollback(ctx, s.l, tx.Rollback)

	// if stepRun.WorkflowRunStickyState {
	// 	fmt
	// }

	var DesiredWorkerId pgtype.UUID

	if stepRun.StickyStrategy.Valid {
		lockedStickyState, err := s.queries.GetWorkflowRunStickyStateForUpdate(ctx, tx, dbsqlc.GetWorkflowRunStickyStateForUpdateParams{
			Workflowrunid: stepRun.WorkflowRunId,
			Tenantid:      stepRun.SRTenantId,
		})

		if err != nil {
			return nil, fmt.Errorf("could not get workflow run sticky state: %w", err)
		}

		// confirm the worker is still available
		if lockedStickyState.DesiredWorkerId.Valid {

			checkedWorker, err := s.queries.CheckWorker(ctx, tx, dbsqlc.CheckWorkerParams{
				Workerid: lockedStickyState.DesiredWorkerId,
				Tenantid: stepRun.SRTenantId,
			})

			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("could not check worker: %w", err)
			}

			// if the worker is no longer available, desired worker will be nil and we'll reassign
			// for soft strategy
			DesiredWorkerId = lockedStickyState.DesiredWorkerId

			// if the strategy is hard, but the worker is not available, we can break early
			if stepRun.StickyStrategy.StickyStrategy == dbsqlc.StickyStrategyHARD && !checkedWorker.Valid {
				return nil, repository.ErrNoWorkerAvailable
			}
		}
	}

	// acquire a semaphore slot
	assigned, err := s.queries.AcquireWorkerSemaphoreSlotAndAssign(ctx, tx, dbsqlc.AcquireWorkerSemaphoreSlotAndAssignParams{
		Steprunid:   stepRun.SRID,
		Actionid:    stepRun.ActionId,
		StepTimeout: stepRun.StepTimeout,
		Tenantid:    stepRun.SRTenantId,
		WorkerId:    DesiredWorkerId,
	})

	if err != nil {
		return nil, fmt.Errorf("could not acquire worker semaphore slot: %w", err)
	}

	if assigned.RemainingSlots == 0 {
		return nil, &errNoWorkerWithSlots{totalSlots: int(0)}
	}

	if assigned.ExhaustedRateLimitCount > 0 {
		return nil, repository.ErrRateLimitExceeded
	}

	if !assigned.WorkerId.Valid || !assigned.DispatcherId.Valid {
		// this likely means that the step run was skip locked by another assign attempt
		return nil, repository.ErrStepRunIsNotAssigned
	}

	if stepRun.StickyStrategy.Valid {
		// check if the worker is the same as the previous worker
		workerId := sqlchelpers.UUIDToStr(assigned.WorkerId)
		previousWorkerId := sqlchelpers.UUIDToStr(DesiredWorkerId)

		if workerId != previousWorkerId {
			err = s.queries.UpdateWorkflowRunStickyState(ctx, tx, dbsqlc.UpdateWorkflowRunStickyStateParams{
				Workflowrunid:   stepRun.WorkflowRunId,
				DesiredWorkerId: assigned.WorkerId,
				Tenantid:        stepRun.SRTenantId,
			})

			if err != nil {
				return nil, fmt.Errorf("could not update sticky state: %w", err)
			}
		}
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, err
	}

	return assigned, nil
}

func (s *stepRunEngineRepository) DeferredStepRunEvent(
	stepRunId pgtype.UUID,
	reason dbsqlc.StepRunEventReason,
	severity dbsqlc.StepRunEventSeverity,
	message string,
	data map[string]interface{},
) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	deferredStepRunEvent(
		ctx,
		s.l,
		s.pool,
		s.queries,
		stepRunId,
		reason,
		severity,
		message,
		data,
	)
}

func deferredStepRunEvent(
	ctx context.Context,
	l *zerolog.Logger,
	dbtx dbsqlc.DBTX,
	queries *dbsqlc.Queries,
	stepRunId pgtype.UUID,
	reason dbsqlc.StepRunEventReason,
	severity dbsqlc.StepRunEventSeverity,
	message string,
	data map[string]interface{},
) {
	dataBytes, err := json.Marshal(data)

	if err != nil {
		l.Err(err).Msg("could not marshal deferred step run event data")
		return
	}

	err = queries.CreateStepRunEvent(ctx, dbtx, dbsqlc.CreateStepRunEventParams{
		Steprunid: stepRunId,
		Message:   message,
		Reason:    reason,
		Severity:  severity,
		Data:      dataBytes,
	})

	if err != nil {
		l.Err(err).Msg("could not create deferred step run event")
	}
}

func (s *stepRunEngineRepository) bulkStepRunsAssigned(
	stepRunIds []pgtype.UUID,
	workerIds []pgtype.UUID,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	messages := make([]string, len(stepRunIds))
	reasons := make([]dbsqlc.StepRunEventReason, len(stepRunIds))
	severities := make([]dbsqlc.StepRunEventSeverity, len(stepRunIds))
	data := make([]map[string]interface{}, len(stepRunIds))

	for i := range stepRunIds {
		workerId := sqlchelpers.UUIDToStr(workerIds[i])
		messages[i] = fmt.Sprintf("Assigned to worker %s", workerId)
		reasons[i] = dbsqlc.StepRunEventReasonASSIGNED
		severities[i] = dbsqlc.StepRunEventSeverityINFO
		data[i] = map[string]interface{}{"worker_id": workerId}
	}

	deferredBulkStepRunEvents(
		ctx,
		s.l,
		s.pool,
		s.queries,
		stepRunIds,
		reasons,
		severities,
		messages,
		data,
	)
}

func (s *stepRunEngineRepository) bulkStepRunsUnassigned(
	stepRunIds []pgtype.UUID,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	messages := make([]string, len(stepRunIds))
	reasons := make([]dbsqlc.StepRunEventReason, len(stepRunIds))
	severities := make([]dbsqlc.StepRunEventSeverity, len(stepRunIds))
	data := make([]map[string]interface{}, len(stepRunIds))

	for i := range stepRunIds {
		messages[i] = "No worker available"
		reasons[i] = dbsqlc.StepRunEventReasonREQUEUEDNOWORKER
		severities[i] = dbsqlc.StepRunEventSeverityWARNING
		// TODO: semaphore extra data
		data[i] = map[string]interface{}{}
	}

	deferredBulkStepRunEvents(
		ctx,
		s.l,
		s.pool,
		s.queries,
		stepRunIds,
		reasons,
		severities,
		messages,
		data,
	)
}

func deferredBulkStepRunEvents(
	ctx context.Context,
	l *zerolog.Logger,
	dbtx dbsqlc.DBTX,
	queries *dbsqlc.Queries,
	stepRunIds []pgtype.UUID,
	reasons []dbsqlc.StepRunEventReason,
	severities []dbsqlc.StepRunEventSeverity,
	messages []string,
	data []map[string]interface{},
) {
	inputData := [][]byte{}
	inputReasons := []string{}
	inputSeverities := []string{}

	for _, d := range data {
		dataBytes, err := json.Marshal(d)

		if err != nil {
			l.Err(err).Msg("could not marshal deferred step run event data")
			return
		}

		inputData = append(inputData, dataBytes)
	}

	for _, r := range reasons {
		inputReasons = append(inputReasons, string(r))
	}

	for _, s := range severities {
		inputSeverities = append(inputSeverities, string(s))
	}

	err := queries.BulkCreateStepRunEvent(ctx, dbtx, dbsqlc.BulkCreateStepRunEventParams{
		Steprunids: stepRunIds,
		Reasons:    inputReasons,
		Severities: inputSeverities,
		Messages:   messages,
		Data:       inputData,
	})

	if err != nil {
		l.Err(err).Msg("could not create deferred step run event")
	}
}

func (s *stepRunEngineRepository) UnassignStepRunFromWorker(ctx context.Context, tenantId, stepRunId string) error {
	return deadlockRetry(s.l, func() error {
		tx, err := s.pool.Begin(ctx)

		if err != nil {
			return err
		}

		pgStepRunId := sqlchelpers.UUIDFromStr(stepRunId)
		pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

		defer deferRollback(ctx, s.l, tx.Rollback)

		_, err = s.queries.ReleaseWorkerSemaphoreSlot(ctx, tx, dbsqlc.ReleaseWorkerSemaphoreSlotParams{
			Steprunid: pgStepRunId,
			Tenantid:  pgTenantId,
		})

		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("could not release previous worker semaphore: %w", err)
		}

		_, err = s.queries.UnlinkStepRunFromWorker(ctx, tx, dbsqlc.UnlinkStepRunFromWorkerParams{
			Steprunid: pgStepRunId,
			Tenantid:  pgTenantId,
		})

		if err != nil {
			return fmt.Errorf("could not unlink step run from worker: %w", err)
		}
		_, err = s.queries.UpdateStepRun(ctx, tx, dbsqlc.UpdateStepRunParams{
			ID:       pgStepRunId,
			Tenantid: pgTenantId,
			Status: dbsqlc.NullStepRunStatus{
				StepRunStatus: dbsqlc.StepRunStatusPENDINGASSIGNMENT,
				Valid:         true,
			},
		})

		if err != nil {
			return fmt.Errorf("could not update step run status: %w", err)
		}

		return tx.Commit(ctx)
	})
}

type debugInfo struct {
	UniqueActions          []string       `json:"unique_actions"`
	TotalStepRuns          int            `json:"total_step_runs"`
	TotalStepRunsAssigned  int            `json:"total_step_runs_assigned"`
	TotalSlots             int            `json:"total_slots"`
	StartingSlotsPerAction map[string]int `json:"starting_slots"`
	EndingSlotsPerAction   map[string]int `json:"ending_slots"`
}

type queueItemWithOrder struct {
	*dbsqlc.QueueItem

	order int
}

func (s *stepRunEngineRepository) QueueStepRuns(ctx context.Context, tenantId string) (repository.QueueStepRunsResult, error) {
	emptyRes := repository.QueueStepRunsResult{
		Queued:             []repository.QueuedStepRun{},
		SchedulingTimedOut: []string{},
		Continue:           false,
	}

	if ctx.Err() != nil {
		return emptyRes, ctx.Err()
	}

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return emptyRes, err
	}

	defer deferRollback(ctx, s.l, tx.Rollback)

	// list queues
	queues, err := s.queries.ListQueues(ctx, tx, pgTenantId)

	if err != nil {
		return emptyRes, fmt.Errorf("could not list queues: %w", err)
	}

	if len(queues) == 0 {
		return emptyRes, nil
	}

	// construct params for list queue items
	query := []dbsqlc.ListQueueItemsParams{}

	for _, queue := range queues {
		query = append(query, dbsqlc.ListQueueItemsParams{
			Tenantid: pgTenantId,
			Queue:    queue.Name,
		})
	}

	queueItems := make([]queueItemWithOrder, 0)

	results := s.queries.ListQueueItems(ctx, tx, query)

	// TODO: verify whether this is multithreaded and if it is, whether thread safe
	results.Query(func(i int, qi []*dbsqlc.QueueItem, err error) {
		if err != nil {
			return
		}

		for i := range qi {
			queueItems = append(queueItems, queueItemWithOrder{
				QueueItem: qi[i],
				order:     i,
			})
		}
	})

	if len(queueItems) == 0 {
		return emptyRes, nil
	}

	// sort the queue items by order from least to greatest, then by queue id
	sort.Slice(queueItems, func(i, j int) bool {
		if queueItems[i].order == queueItems[j].order {
			return queueItems[i].QueueItem.ID < queueItems[j].QueueItem.ID
		}

		return queueItems[i].order < queueItems[j].order
	})

	queuedItems := make([]int64, 0)
	queuedStepRuns := make([]repository.QueuedStepRun, 0)
	timedOutStepRuns := make([]pgtype.UUID, 0)

	// get a list of unique actions
	uniqueActions := make(map[string]bool)

	for _, row := range queueItems {
		uniqueActions[row.ActionId.String] = true
	}

	uniqueActionsArr := make([]string, 0, len(uniqueActions))

	for action := range uniqueActions {
		uniqueActionsArr = append(uniqueActionsArr, action)
	}

	slots, err := s.queries.ListSemaphoreSlotsToAssign(ctx, tx, dbsqlc.ListSemaphoreSlotsToAssignParams{
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
		Actionids: uniqueActionsArr,
	})

	if err != nil {
		return emptyRes, fmt.Errorf("could not list semaphore slots to assign: %w", err)
	}

	// NOTE(abelanger5): this is a version of the assignment problem. There is a more optimal solution i.e. optimal
	// matching which can run in polynomial time. This is a naive approach which assigns the first steps which were
	// queued to the first slots which are seen.
	actionsToSlots := make(map[string]map[string]*dbsqlc.ListSemaphoreSlotsToAssignRow)
	slotsToActions := make(map[string][]string)

	for _, slot := range slots {
		slotId := sqlchelpers.UUIDToStr(slot.ID)

		if _, ok := actionsToSlots[slot.ActionId]; !ok {
			actionsToSlots[slot.ActionId] = make(map[string]*dbsqlc.ListSemaphoreSlotsToAssignRow)
		}

		actionsToSlots[slot.ActionId][slotId] = slot

		if _, ok := slotsToActions[slotId]; !ok {
			slotsToActions[slotId] = make([]string, 0)
		}

		slotsToActions[slotId] = append(slotsToActions[slotId], slot.ActionId)
	}

	// assemble debug information
	startingSlotsPerAction := make(map[string]int)

	for action, slots := range actionsToSlots {
		startingSlotsPerAction[action] = len(slots)
	}

	// match slots to step runs in the order the step runs were returned
	stepRunIds := make([]pgtype.UUID, 0)
	slotIds := make([]pgtype.UUID, 0)
	workerIds := make([]pgtype.UUID, 0)
	stepRunTimeouts := make([]string, 0)
	unassignedStepRunIds := make([]pgtype.UUID, 0)

	allStepRunsWithActionAssigned := make(map[string]bool)

	for _, uniqueAction := range uniqueActionsArr {
		allStepRunsWithActionAssigned[uniqueAction] = true
	}

	for _, qi := range queueItems {
		if len(actionsToSlots[qi.ActionId.String]) == 0 {
			allStepRunsWithActionAssigned[qi.ActionId.String] = false
			unassignedStepRunIds = append(unassignedStepRunIds, qi.StepRunId)
			continue
		}

		// if the current time is after the scheduleTimeoutAt, then mark this as timed out
		now := time.Now().UTC().UTC()
		scheduleTimeoutAt := qi.ScheduleTimeoutAt.Time

		// timed out if the scheduleTimeoutAt is set and the current time is after the scheduleTimeoutAt
		isTimedOut := !scheduleTimeoutAt.IsZero() && scheduleTimeoutAt.Before(now)

		if isTimedOut {
			timedOutStepRuns = append(timedOutStepRuns, qi.StepRunId)
			continue
		}

		slot := popRandMapValue(actionsToSlots[qi.ActionId.String])

		// delete from all other actions
		for _, action := range slotsToActions[sqlchelpers.UUIDToStr(slot.ID)] {
			delete(actionsToSlots[action], sqlchelpers.UUIDToStr(slot.ID))
		}

		stepRunIds = append(stepRunIds, qi.StepRunId)
		stepRunTimeouts = append(stepRunTimeouts, qi.StepTimeout.String)
		slotIds = append(slotIds, slot.ID)
		workerIds = append(workerIds, slot.WorkerId)

		queuedStepRuns = append(queuedStepRuns, repository.QueuedStepRun{
			StepRunId:    sqlchelpers.UUIDToStr(qi.StepRunId),
			WorkerId:     sqlchelpers.UUIDToStr(slot.WorkerId),
			DispatcherId: sqlchelpers.UUIDToStr(slot.DispatcherId),
		})

		queuedItems = append(queuedItems, qi.ID)
	}

	_, err = s.queries.BulkAssignStepRunsToWorkers(ctx, tx, dbsqlc.BulkAssignStepRunsToWorkersParams{
		Steprunids:      stepRunIds,
		Stepruntimeouts: stepRunTimeouts,
		Slotids:         slotIds,
		Workerids:       workerIds,
	})

	if err != nil {
		return emptyRes, fmt.Errorf("could not bulk assign step runs to workers: %w", err)
	}

	err = s.queries.BulkQueueItems(ctx, tx, queuedItems)

	if err != nil {
		return emptyRes, fmt.Errorf("could not bulk queue items: %w", err)
	}

	// if there are step runs to place in a cancelling state, do so
	if len(timedOutStepRuns) > 0 {
		_, err = s.queries.BulkMarkStepRunsAsCancelling(ctx, tx, timedOutStepRuns)

		if err != nil {
			return emptyRes, fmt.Errorf("could not bulk mark step runs as cancelling: %w", err)
		}
	}

	err = tx.Commit(ctx)

	defer s.bulkStepRunsAssigned(stepRunIds, workerIds)
	defer s.bulkStepRunsUnassigned(unassignedStepRunIds)

	if err != nil {
		return emptyRes, fmt.Errorf("could not commit transaction: %w", err)
	}

	// print debug information
	endingSlotsPerAction := make(map[string]int)
	for action, slots := range actionsToSlots {
		endingSlotsPerAction[action] = len(slots)
	}

	defer func() {
		// pretty-print json with 2 spaces
		debugInfo := debugInfo{
			UniqueActions:          uniqueActionsArr,
			TotalStepRuns:          len(queueItems),
			TotalStepRunsAssigned:  len(stepRunIds),
			TotalSlots:             len(slots),
			StartingSlotsPerAction: startingSlotsPerAction,
			EndingSlotsPerAction:   endingSlotsPerAction,
		}

		debugInfoBytes, err := json.MarshalIndent(debugInfo, "", "  ")

		if err != nil {
			s.l.Warn().Err(err).Msg("could not marshal debug info")
			return
		}

		s.l.Warn().Msg(string(debugInfoBytes))
	}()

	shouldContinue := false

	// if at least one of the actions got all step runs assigned, and there are slots remaining, return true
	for action := range uniqueActions {
		if _, ok := allStepRunsWithActionAssigned[action]; ok {
			// check if there are slots remaining
			if len(actionsToSlots[action]) > 0 {
				shouldContinue = true
				break
			}
		}
	}

	timedOutStepRunsStr := make([]string, len(timedOutStepRuns))

	for i, id := range timedOutStepRuns {
		timedOutStepRunsStr[i] = sqlchelpers.UUIDToStr(id)
	}

	return repository.QueueStepRunsResult{
		Queued:             queuedStepRuns,
		SchedulingTimedOut: timedOutStepRunsStr,
		Continue:           shouldContinue,
	}, nil
}

func popRandMapValue(m map[string]*dbsqlc.ListSemaphoreSlotsToAssignRow) *dbsqlc.ListSemaphoreSlotsToAssignRow {
	for k, v := range m {
		delete(m, k)
		return v
	}

	return nil
}

func (s *stepRunEngineRepository) AssignStepRunToWorker(ctx context.Context, stepRun *dbsqlc.GetStepRunForEngineRow) (string, string, error) {

	if ctx.Err() != nil {
		return "", "", ctx.Err()
	}

	err := s.releaseWorkerSemaphore(ctx, stepRun)

	if err != nil {
		return "", "", err
	}

	var assigned *dbsqlc.AcquireWorkerSemaphoreSlotAndAssignRow

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

			return fmt.Errorf("could not assign worker for step run %s (step %s): %w", sqlchelpers.UUIDToStr(stepRun.SRID), stepRun.StepReadableId.String, err)
		}

		return nil
	})

	if err != nil {
		var target *errNoWorkerWithSlots

		labels, labelErr := s.queries.GetStepDesiredWorkerLabels(ctx, s.pool, stepRun.StepId)

		if labelErr != nil {
			return "", "", fmt.Errorf("could not get step desired worker labels: %w", err)
		}

		semaphoreExtra := s.unmarshalSemaphoreExtraData(nil, &labels)

		if errors.As(err, &target) {
			defer s.DeferredStepRunEvent(
				stepRun.SRID,
				dbsqlc.StepRunEventReasonREQUEUEDNOWORKER,
				dbsqlc.StepRunEventSeverityWARNING,
				"No worker available",
				map[string]interface{}{
					"worker_id": nil,
					"semaphore": semaphoreExtra,
				},
			)

			return "", "", repository.ErrNoWorkerAvailable
		}

		if errors.Is(err, repository.ErrNoWorkerAvailable) {
			defer s.DeferredStepRunEvent(
				stepRun.SRID,
				dbsqlc.StepRunEventReasonREQUEUEDNOWORKER,
				dbsqlc.StepRunEventSeverityWARNING,
				"No worker available",
				map[string]interface{}{
					"worker_id": nil,
					"semaphore": semaphoreExtra,
				},
			)
		}

		if errors.Is(err, repository.ErrRateLimitExceeded) {
			defer s.DeferredStepRunEvent(
				stepRun.SRID,
				dbsqlc.StepRunEventReasonREQUEUEDRATELIMIT,
				dbsqlc.StepRunEventSeverityWARNING,
				"Rate limit exceeded",
				nil, // TODO add label data
			)
		}

		return "", "", err
	}

	semaphoreExtra := s.unmarshalSemaphoreExtraData(assigned, nil)

	defer s.DeferredStepRunEvent(
		stepRun.SRID,
		dbsqlc.StepRunEventReasonASSIGNED,
		dbsqlc.StepRunEventSeverityINFO,
		fmt.Sprintf("Assigned to worker %s", sqlchelpers.UUIDToStr(assigned.WorkerId)),
		map[string]interface{}{
			"worker_id": sqlchelpers.UUIDToStr(assigned.WorkerId),
			"semaphore": semaphoreExtra,
		},
	)

	return sqlchelpers.UUIDToStr(assigned.WorkerId), sqlchelpers.UUIDToStr(assigned.DispatcherId), nil
}

func (s *stepRunEngineRepository) unmarshalSemaphoreExtraData(semaphore *dbsqlc.AcquireWorkerSemaphoreSlotAndAssignRow, desiredLabelBytes *[]byte) map[string]interface{} {
	// Initialize the result maps
	var desiredLabels []map[string]interface{}
	var workerLabels []map[string]interface{}

	// Check if desiredLabelBytes is not nil and unmarshal it
	switch {
	case desiredLabelBytes != nil:
		err := json.Unmarshal(*desiredLabelBytes, &desiredLabels)
		if err != nil && err.Error() != "unexpected end of JSON input" {
			s.l.Warn().Err(err).Msg("failed to unmarshal desiredLabelBytes")
		}
	case semaphore != nil:
		// If desiredLabelBytes is nil, use semaphore.DesiredLabels
		err := json.Unmarshal(semaphore.DesiredLabels, &desiredLabels)
		if err != nil {
			s.l.Warn().Err(err).Msg("failed to unmarshal semaphore.DesiredLabels")
		}

	default:
		s.l.Warn().Msg("semaphore is nil, cannot unmarshal DesiredLabels")
	}

	// Unmarshal WorkerLabels from semaphore if it's not nil
	if semaphore != nil && semaphore.WorkerLabels != nil {
		err := json.Unmarshal(semaphore.WorkerLabels, &workerLabels)
		if err != nil {
			s.l.Warn().Err(err).Msg("failed to unmarshal semaphore.WorkerLabels")
		}
	}

	// Filter values of desiredLabels where desiredLabels.key is empty
	// HACK this is a workaround for the fact that the sqlc query sometimes returns null rows
	filteredDesiredLabels := make([]map[string]interface{}, 0)
	for _, label := range desiredLabels {
		if label["key"] != "" && label["key"] != nil {
			filteredDesiredLabels = append(filteredDesiredLabels, label)
		}
	}

	return map[string]interface{}{
		"desired_worker_labels": filteredDesiredLabels,
		"actual_worker_labels":  workerLabels,
	}
}

func (s *stepRunEngineRepository) UpdateStepRun(ctx context.Context, tenantId, stepRunId string, opts *repository.UpdateStepRunOpts) (*dbsqlc.GetStepRunForEngineRow, *repository.StepRunUpdateInfo, error) {
	ctx, span := telemetry.NewSpan(ctx, "update-step-run")
	defer span.End()

	if err := s.v.Validate(opts); err != nil {
		return nil, nil, err
	}

	updateParams, createEventParams, updateJobRunLookupDataParams, err := getUpdateParams(tenantId, stepRunId, opts)

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

		innerStepRun, err := s.getStepRunForEngineTx(ctx, tx, tenantId, stepRunId)

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
		updateInfo, err = s.ResolveRelatedStatuses(ctx, stepRun.SRTenantId, stepRun.SRID)
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

	updateParams, createEventParams, updateJobRunLookupDataParams, err := getUpdateParams(tenantId, stepRunId, opts)

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

		innerStepRun, err := s.getStepRunForEngineTx(ctx, tx, tenantId, stepRunId)

		if err != nil {
			return err
		}

		stepRun, err = s.updateStepRunCore(ctx, tx, tenantId, updateParams, createEventParams, updateJobRunLookupDataParams, innerStepRun)

		if err != nil {
			return err
		}

		// reset the job run, workflow run and all fields as part of the core tx
		_, err = s.queries.ReplayStepRunResetWorkflowRun(ctx, tx, stepRun.WorkflowRunId)

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

			err = archiveStepRunResult(ctx, s.queries, tx, tenantId, laterStepRunId)

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
			defer s.DeferredStepRunEvent(
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

	if !repository.IsFinalStepRunStatus(stepRun.SRStatus) {
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
		innerStepRun, err := s.getStepRunForEngineTx(ctx, tx, tenantId, stepRunId)

		if err != nil {
			return err
		}

		// if the step run is not pending, we can't queue it, but we still want to update other input params
		if innerStepRun.SRStatus != dbsqlc.StepRunStatusPENDING {
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
		_, err = s.ResolveRelatedStatuses(ctx, stepRun.SRTenantId, stepRun.SRID)
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
			return updateParams, nil, nil, err
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
		nil
}

func (s *stepRunEngineRepository) updateStepRunCore(
	ctx context.Context,
	tx pgx.Tx,
	tenantId string,
	updateParams dbsqlc.UpdateStepRunParams,
	createEventParams *dbsqlc.CreateStepRunEventParams,
	updateJobRunLookupDataParams *dbsqlc.UpdateJobRunLookupDataWithStepRunParams,
	innerStepRun *dbsqlc.GetStepRunForEngineRow,
) (*dbsqlc.GetStepRunForEngineRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "update-step-run-core") // nolint:ineffassign
	defer span.End()

	// if the status is set to pending assignment (and was not previously), insert into queue items
	if updateParams.Status.Valid &&
		innerStepRun.SRStatus != dbsqlc.StepRunStatusPENDINGASSIGNMENT &&
		updateParams.Status.StepRunStatus == dbsqlc.StepRunStatusPENDINGASSIGNMENT {
		err := s.queries.CreateQueueItem(ctx, tx, dbsqlc.CreateQueueItemParams{
			StepRunId:         innerStepRun.SRID,
			StepId:            innerStepRun.StepId,
			ActionId:          sqlchelpers.TextFromStr(innerStepRun.ActionId),
			StepTimeout:       innerStepRun.StepTimeout,
			ScheduleTimeoutAt: updateParams.ScheduleTimeoutAt,
			Tenantid:          updateParams.Tenantid,
			Queue:             innerStepRun.SRQueue,
		})

		if err != nil {
			return nil, fmt.Errorf("could not create queue item: %w", err)
		}
	}

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

	// if we're updating the status, and the status update is not updated to RUNNING,
	// release the semaphore slot (all other state transitions should release the semaphore slot)
	if updateParams.Status.Valid &&
		// the semaphore has not already been released manually
		!updateStepRun.SemaphoreReleased &&
		updateStepRun.Status != dbsqlc.StepRunStatusRUNNING &&
		// we must have actually updated the status to a different state
		string(innerStepRun.SRStatus) != string(updateStepRun.Status) {

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

func (s *stepRunEngineRepository) ResolveRelatedStatuses(
	ctx context.Context,
	tenantId pgtype.UUID,
	stepRunId pgtype.UUID,
) (*repository.StepRunUpdateInfo, error) {
	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer deferRollback(ctx, s.l, tx.Rollback)

	ctx, span := telemetry.NewSpan(ctx, "update-step-run-extra") // nolint:ineffassign
	defer span.End()

	_, err = s.queries.ResolveLaterStepRuns(ctx, tx, dbsqlc.ResolveLaterStepRunsParams{
		Steprunid: stepRunId,
		Tenantid:  tenantId,
	})

	if err != nil {
		return nil, fmt.Errorf("could not resolve later step runs: %w", err)
	}

	jobRun, err := s.queries.ResolveJobRunStatus(ctx, tx, dbsqlc.ResolveJobRunStatusParams{
		Steprunid: stepRunId,
		Tenantid:  tenantId,
	})

	if err != nil {
		return nil, fmt.Errorf("could not resolve job run status: %w", err)
	}

	resolveWorkflowRunParams := dbsqlc.ResolveWorkflowRunStatusParams{
		Jobrunid: jobRun.ID,
		Tenantid: tenantId,
	}

	workflowRun, err := s.queries.ResolveWorkflowRunStatus(ctx, tx, resolveWorkflowRunParams)

	if err != nil {
		return nil, fmt.Errorf("could not resolve workflow run status: %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, fmt.Errorf("could commit resolve related statuses tx: %w", err)
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

func (s *stepRunEngineRepository) GetStepRunDataForEngine(ctx context.Context, tenantId, stepRunId string) (*dbsqlc.GetStepRunDataForEngineRow, error) {
	return s.queries.GetStepRunDataForEngine(ctx, s.pool, dbsqlc.GetStepRunDataForEngineParams{
		ID:       sqlchelpers.UUIDFromStr(stepRunId),
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})
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
	return archiveStepRunResult(ctx, s.queries, s.pool, tenantId, stepRunId)
}

func archiveStepRunResult(ctx context.Context, queries *dbsqlc.Queries, db dbsqlc.DBTX, tenantId, stepRunId string) error {
	_, err := queries.ArchiveStepRunResultFromStepRun(ctx, db, dbsqlc.ArchiveStepRunResultFromStepRunParams{
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

	defer s.DeferredStepRunEvent(
		stepRunUUID,
		dbsqlc.StepRunEventReasonTIMEOUTREFRESHED,
		dbsqlc.StepRunEventSeverityINFO,
		fmt.Sprintf("Timeout refreshed by %s", incrementTimeoutBy),
		nil)

	return res, nil
}

func (s *stepRunEngineRepository) ClearStepRunPayloadData(ctx context.Context, tenantId string) (bool, error) {
	hasMore, err := s.queries.ClearStepRunPayloadData(ctx, s.pool, dbsqlc.ClearStepRunPayloadDataParams{
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
