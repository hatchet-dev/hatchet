package prisma

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
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
	"github.com/hatchet-dev/hatchet/pkg/scheduling"
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
	pool               *pgxpool.Pool
	v                  validator.Validator
	l                  *zerolog.Logger
	queries            *dbsqlc.Queries
	cf                 *server.ConfigFileRuntime
	cachedMinQueuedIds sync.Map
	exhaustedRLCache   *scheduling.ExhaustedRateLimitCache
}

func NewStepRunEngineRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger, cf *server.ConfigFileRuntime) repository.StepRunEngineRepository {
	queries := dbsqlc.New()

	return &stepRunEngineRepository{
		pool:             pool,
		v:                v,
		l:                l,
		queries:          queries,
		cf:               cf,
		exhaustedRLCache: scheduling.NewExhaustedRateLimitCache(time.Minute),
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

	messages := make([]string, len(stepRuns))
	reasons := make([]dbsqlc.StepRunEventReason, len(stepRuns))
	severities := make([]dbsqlc.StepRunEventSeverity, len(stepRuns))
	data := make([]map[string]interface{}, len(stepRuns))

	for i := range stepRuns {
		workerId := sqlchelpers.UUIDToStr(stepRuns[i].SRWorkerId)
		messages[i] = "Worker has become inactive"
		reasons[i] = dbsqlc.StepRunEventReasonREASSIGNED
		severities[i] = dbsqlc.StepRunEventSeverityCRITICAL
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
		tx, rollback, err := s.prepareTx(ctx, 5000)

		if err != nil {
			return err
		}

		defer rollback()

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

func (s *stepRunEngineRepository) bulkStepRunsRateLimited(
	stepRunIds []pgtype.UUID,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	messages := make([]string, len(stepRunIds))
	reasons := make([]dbsqlc.StepRunEventReason, len(stepRunIds))
	severities := make([]dbsqlc.StepRunEventSeverity, len(stepRunIds))
	data := make([]map[string]interface{}, len(stepRunIds))

	for i := range stepRunIds {
		messages[i] = "Rate limit exceeded"
		reasons[i] = dbsqlc.StepRunEventReasonREQUEUEDRATELIMIT
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

func UniqueSet[T any](i []T, keyFunc func(T) string) map[string]struct{} {
	set := make(map[string]struct{})

	for _, item := range i {
		key := keyFunc(item)
		set[key] = struct{}{}
	}

	return set
}

func (s *stepRunEngineRepository) QueueStepRuns(ctx context.Context, qlp *zerolog.Logger, tenantId string) (repository.QueueStepRunsResult, error) {
	ql := qlp.With().Str("tenant_id", tenantId).Logger()

	ctx, span := telemetry.NewSpan(ctx, "queue-step-runs-database")
	defer span.End()

	startedAt := time.Now().UTC()

	emptyRes := repository.QueueStepRunsResult{
		Queued:             []repository.QueuedStepRun{},
		SchedulingTimedOut: []string{},
		Continue:           false,
	}

	if ctx.Err() != nil {
		return emptyRes, ctx.Err()
	}

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	limit := 100

	if s.cf.SingleQueueLimit != 0 {
		limit = s.cf.SingleQueueLimit
	}

	pgLimit := pgtype.Int4{
		Int32: int32(limit),
		Valid: true,
	}

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
		ql.Debug().Msg("no queues found")
		return emptyRes, nil
	}

	// construct params for list queue items
	query := []dbsqlc.ListQueueItemsParams{}

	// randomly order queues
	rand.New(rand.NewSource(time.Now().UnixNano())).Shuffle(len(queues), func(i, j int) { queues[i], queues[j] = queues[j], queues[i] }) // nolint:gosec

	for _, queue := range queues {
		// check whether we have exhausted rate limits for this queue
		if s.exhaustedRLCache.IsExhausted(tenantId, queue.Name) {
			ql.Debug().Msgf("queue %s is rate limited, skipping queueing", queue.Name)
			continue
		}

		name := queue.Name

		q := dbsqlc.ListQueueItemsParams{
			Tenantid: pgTenantId,
			Queue:    name,
			Limit:    pgLimit,
		}

		// lookup to see if we have a min queued id cached
		minQueuedId, ok := s.cachedMinQueuedIds.Load(getCacheName(tenantId, name))

		if ok {
			if minQueuedIdInt, ok := minQueuedId.(int64); ok {
				q.GtId = pgtype.Int8{
					Int64: minQueuedIdInt,
					Valid: true,
				}
			}
		}

		query = append(query, q)
	}

	results := s.queries.ListQueueItems(ctx, tx, query)
	defer results.Close()

	durationsOfQueueListResults := make([]string, 0)

	queueItems := make([]*scheduling.QueueItemWithOrder, 0)

	// TODO: verify whether this is multithreaded and if it is, whether thread safe
	results.Query(func(i int, qi []*dbsqlc.QueueItem, err error) {
		if err != nil {
			ql.Err(err).Msg("could not list queue items")
			return
		}

		queueName := ""

		for i := range qi {
			queueItems = append(queueItems, &scheduling.QueueItemWithOrder{
				QueueItem: qi[i],
				Order:     i,
			})

			queueName = qi[i].Queue
		}

		durationsOfQueueListResults = append(durationsOfQueueListResults, fmt.Sprintf("%s:%s:%s", queues[i].Name, queueName, time.Since(startedAt).String()))
	})

	err = results.Close()

	if err != nil {
		return emptyRes, fmt.Errorf("could not close queue items result: %w", err)
	}

	if len(queueItems) == 0 {
		ql.Debug().Msg("no queue items found")
		return emptyRes, nil
	}

	var duplicates []*scheduling.QueueItemWithOrder
	var cancelled []*scheduling.QueueItemWithOrder

	queueItems, duplicates = removeDuplicates(queueItems)
	queueItems, cancelled, err = s.removeCancelledStepRuns(ctx, tx, queueItems)

	if err != nil {
		return emptyRes, fmt.Errorf("could not remove cancelled step runs: %w", err)
	}

	// sort the queue items by Order from least to greatest, then by queue id
	sort.Slice(queueItems, func(i, j int) bool {
		// sort by priority, then by order, then by id
		if queueItems[i].Priority == queueItems[j].Priority {
			if queueItems[i].Order == queueItems[j].Order {
				return queueItems[i].QueueItem.ID < queueItems[j].QueueItem.ID
			}

			return queueItems[i].Order < queueItems[j].Order
		}

		return queueItems[i].Priority > queueItems[j].Priority
	})

	// get a list of unique actions
	uniqueActions := make(map[string]bool)

	for _, row := range queueItems {
		uniqueActions[row.ActionId.String] = true
	}

	uniqueActionsArr := make([]string, 0, len(uniqueActions))

	for action := range uniqueActions {
		uniqueActionsArr = append(uniqueActionsArr, action)
	}

	// list rate limits for the tenant
	rateLimits, err := s.queries.ListRateLimitsForTenant(ctx, tx, pgTenantId)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return emptyRes, fmt.Errorf("could not list rate limits for tenant: %w", err)
	}

	currRateLimitValues := make(map[string]*dbsqlc.ListRateLimitsForTenantRow)
	stepRateUnits := make(map[string]map[string]int32)

	if len(rateLimits) > 0 {
		for i := range rateLimits {
			key := rateLimits[i].Key
			currRateLimitValues[key] = rateLimits[i]
		}

		// get a list of unique step ids
		uniqueStepIds := make(map[string]bool)

		for _, row := range queueItems {
			uniqueStepIds[sqlchelpers.UUIDToStr(row.StepId)] = true
		}

		uniqueStepIdsArr := make([]pgtype.UUID, 0, len(uniqueStepIds))

		for step := range uniqueStepIds {
			uniqueStepIdsArr = append(uniqueStepIdsArr, sqlchelpers.UUIDFromStr(step))
		}

		// get the rate limits for the steps
		stepRateLimits, err := s.queries.ListRateLimitsForSteps(ctx, tx, dbsqlc.ListRateLimitsForStepsParams{
			Tenantid: pgTenantId,
			Stepids:  uniqueStepIdsArr,
		})

		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return emptyRes, fmt.Errorf("could not list rate limits for steps: %w", err)
		}

		for _, row := range stepRateLimits {
			stepId := sqlchelpers.UUIDToStr(row.StepId)

			if _, ok := stepRateUnits[stepId]; !ok {
				stepRateUnits[stepId] = make(map[string]int32)
			}

			stepRateUnits[stepId][row.RateLimitKey] = row.Units
		}
	}

	slots, err := s.queries.ListSemaphoreSlotsToAssign(ctx, tx, dbsqlc.ListSemaphoreSlotsToAssignParams{
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
		Actionids: uniqueActionsArr,
	})

	if err != nil {
		return emptyRes, fmt.Errorf("could not list semaphore slots to assign: %w", err)
	}

	// GET UNIQUE STEP IDS
	stepIdSet := UniqueSet(queueItems, func(x *scheduling.QueueItemWithOrder) string {
		return sqlchelpers.UUIDToStr(x.StepId)
	})

	desiredLabels := make(map[string][]*dbsqlc.GetDesiredLabelsRow)
	hasDesired := false

	// GET DESIRED LABELS
	// OPTIMIZATION: CACHEABLE
	for stepId := range stepIdSet {
		labels, err := s.queries.GetDesiredLabels(ctx, tx, sqlchelpers.UUIDFromStr(stepId))
		if err != nil {
			return emptyRes, fmt.Errorf("could not get desired labels: %w", err)
		}
		desiredLabels[stepId] = labels
		hasDesired = true
	}

	var workerLabels = make(map[string][]*dbsqlc.GetWorkerLabelsRow)

	if hasDesired {
		// GET UNIQUE WORKER LABELS
		workerIdSet := UniqueSet(slots, func(x *dbsqlc.ListSemaphoreSlotsToAssignRow) string {
			return sqlchelpers.UUIDToStr(x.WorkerId)
		})

		for workerId := range workerIdSet {
			labels, err := s.queries.GetWorkerLabels(ctx, tx, sqlchelpers.UUIDFromStr(workerId))
			if err != nil {
				return emptyRes, fmt.Errorf("could not get worker labels: %w", err)
			}
			workerLabels[workerId] = labels
		}
	}

	plan, err := scheduling.GeneratePlan(
		ctx,
		slots,
		uniqueActionsArr,
		queueItems,
		stepRateUnits,
		currRateLimitValues,
		workerLabels,
		desiredLabels,
	)

	if err != nil {
		return emptyRes, fmt.Errorf("could not generate scheduling: %w", err)
	}

	// save rate limits as a subtransaction, but don't throw an error if it fails
	func() {
		subtx, err := tx.Begin(ctx)

		if err != nil {
			s.l.Err(err).Msg("could not start subtransaction")
			return
		}

		defer deferRollback(ctx, s.l, subtx.Rollback)

		updateKeys := []string{}
		updateUnits := []int32{}

		for key, value := range plan.RateLimitUnitsConsumed {
			if value == 0 {
				continue
			}

			updateKeys = append(updateKeys, key)
			updateUnits = append(updateUnits, value)
		}

		params := dbsqlc.BulkUpdateRateLimitsParams{
			Tenantid: pgTenantId,
			Keys:     updateKeys,
			Units:    updateUnits,
		}

		_, err = s.queries.BulkUpdateRateLimits(ctx, subtx, params)

		if err != nil {
			s.l.Err(err).Msg("could not bulk update rate limits")
			return
		}

		// throw a warning if any rate limits are below 0
		for key, value := range plan.RateLimitUnitsConsumed {
			if value < 0 {
				s.l.Warn().Msgf("rate limit %s is below 0: %d", key, value)
			}
		}

		err = subtx.Commit(ctx)

		if err != nil {
			s.l.Err(err).Msg("could not commit subtransaction")
			return
		}
	}()

	_, err = s.queries.BulkAssignStepRunsToWorkers(ctx, tx, dbsqlc.BulkAssignStepRunsToWorkersParams{
		Steprunids:      plan.StepRunIds,
		Stepruntimeouts: plan.StepRunTimeouts,
		Slotids:         plan.SlotIds,
		Workerids:       plan.WorkerIds,
	})

	if err != nil {
		return emptyRes, fmt.Errorf("could not bulk assign step runs to workers: %w", err)
	}

	popItems := plan.QueuedItems

	// we'd like to remove duplicates from the queue items as well
	for _, item := range duplicates {
		// print a warning for duplicates
		s.l.Warn().Msgf("duplicate queue item: %d for step run %s", item.QueueItem.ID, sqlchelpers.UUIDToStr(item.QueueItem.StepRunId))

		popItems = append(popItems, item.QueueItem.ID)
	}

	// we'd like to remove cancelled step runs from the queue items as well
	for _, item := range cancelled {
		popItems = append(popItems, item.QueueItem.ID)
	}

	err = s.queries.BulkQueueItems(ctx, tx, popItems)

	if err != nil {
		return emptyRes, fmt.Errorf("could not bulk queue items: %w", err)
	}

	// if there are step runs to place in a cancelling state, do so
	if len(plan.TimedOutStepRuns) > 0 {
		_, err = s.queries.BulkMarkStepRunsAsCancelling(ctx, tx, plan.TimedOutStepRuns)

		if err != nil {
			return emptyRes, fmt.Errorf("could not bulk mark step runs as cancelling: %w", err)
		}
	}

	err = tx.Commit(ctx)

	defer s.bulkStepRunsAssigned(plan.StepRunIds, plan.WorkerIds)
	defer s.bulkStepRunsUnassigned(plan.UnassignedStepRunIds)
	defer s.bulkStepRunsRateLimited(plan.RateLimitedStepRuns)

	if err != nil {
		return emptyRes, fmt.Errorf("could not commit transaction: %w", err)
	}

	// update the cache with the min queued id
	for name, qiId := range plan.MinQueuedIds {
		s.cachedMinQueuedIds.Store(getCacheName(tenantId, name), qiId)
	}

	// update the rate limit cache
	for queue, times := range plan.RateLimitedQueues {
		s.exhaustedRLCache.Set(tenantId, queue, times)
	}

	timedOutStepRunsStr := make([]string, len(plan.TimedOutStepRuns))

	for i, id := range plan.TimedOutStepRuns {
		timedOutStepRunsStr[i] = sqlchelpers.UUIDToStr(id)
	}

	defer printQueueDebugInfo(ql, tenantId, queues, queueItems, duplicates, cancelled, plan, slots, startedAt)

	return repository.QueueStepRunsResult{
		Queued:             plan.QueuedStepRuns,
		SchedulingTimedOut: timedOutStepRunsStr,
		Continue:           plan.ShouldContinue,
	}, nil
}

func (s *stepRunEngineRepository) CleanupQueueItems(ctx context.Context, tenantId string) error {
	// setup telemetry
	ctx, span := telemetry.NewSpan(ctx, "cleanup-queue-items-database")
	defer span.End()

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	// get the min and max queue items
	minMax, err := s.queries.GetMinMaxProcessedQueueItems(ctx, s.pool, pgTenantId)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}

		return fmt.Errorf("could not get min max processed queue items: %w", err)
	}

	if minMax == nil {
		return nil
	}

	minId := minMax.MinId
	maxId := minMax.MaxId

	if maxId == 0 {
		return nil
	}

	// iterate until we have no more queue items to process
	var batchSize int64 = 1000
	var currBatch int64

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		currBatch++

		currMax := minId + batchSize*currBatch

		if currMax > maxId {
			currMax = maxId
		}

		// get the next batch of queue items
		err := s.queries.CleanupQueueItems(ctx, s.pool, dbsqlc.CleanupQueueItemsParams{
			Minid:    minId,
			Maxid:    minId + batchSize*currBatch,
			Tenantid: pgTenantId,
		})

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}

			return fmt.Errorf("could not cleanup queue items: %w", err)
		}

		if currMax == maxId {
			break
		}
	}

	return nil
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
		tx, rollback, err := s.prepareTx(ctx, 5000)

		if err != nil {
			return err
		}

		defer rollback()

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
		tx, rollback, err := s.prepareTx(ctx, 5000)

		if err != nil {
			return err
		}

		defer rollback()

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
		priority := 1

		if innerStepRun.SRPriority.Valid {
			priority = int(innerStepRun.SRPriority.Int32)
		}

		// if the step run is a retry, set the priority to 4
		if innerStepRun.SRRetryCount > 0 || (updateParams.RetryCount.Valid && updateParams.RetryCount.Int32 > 0) {
			priority = 4
		}

		err := s.queries.CreateQueueItem(ctx, tx, dbsqlc.CreateQueueItemParams{
			StepRunId:         innerStepRun.SRID,
			StepId:            innerStepRun.StepId,
			ActionId:          sqlchelpers.TextFromStr(innerStepRun.ActionId),
			StepTimeout:       innerStepRun.StepTimeout,
			ScheduleTimeoutAt: updateParams.ScheduleTimeoutAt,
			Tenantid:          updateParams.Tenantid,
			Queue:             innerStepRun.SRQueue,
			Priority: pgtype.Int4{
				Valid: true,
				Int32: int32(priority),
			},
			Sticky:          innerStepRun.StickyStrategy,
			DesiredWorkerId: innerStepRun.DesiredWorkerId,
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
	tx, rollback, err := s.prepareTx(ctx, 5000)

	if err != nil {
		return nil, err
	}

	defer rollback()

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

func getCacheName(tenantId, queue string) string {
	return fmt.Sprintf("%s:%s", tenantId, queue)
}

func (s *stepRunEngineRepository) removeCancelledStepRuns(ctx context.Context, tx pgx.Tx, qis []*scheduling.QueueItemWithOrder) ([]*scheduling.QueueItemWithOrder, []*scheduling.QueueItemWithOrder, error) {
	currStepRunIds := make([]pgtype.UUID, len(qis))

	for i, qi := range qis {
		currStepRunIds[i] = qi.StepRunId
	}

	cancelledStepRuns, err := s.queries.GetCancelledStepRuns(ctx, tx, currStepRunIds)

	if err != nil {
		return nil, nil, err
	}

	cancelledStepRunsMap := make(map[string]bool, len(cancelledStepRuns))

	for _, sr := range cancelledStepRuns {
		cancelledStepRunsMap[sqlchelpers.UUIDToStr(sr)] = true
	}

	// remove cancelled step runs from the queue items
	remaining := make([]*scheduling.QueueItemWithOrder, 0, len(qis))
	cancelled := make([]*scheduling.QueueItemWithOrder, 0, len(qis))

	for _, qi := range qis {
		if _, ok := cancelledStepRunsMap[sqlchelpers.UUIDToStr(qi.StepRunId)]; ok {
			cancelled = append(cancelled, qi)
			continue
		}

		remaining = append(remaining, qi)
	}

	return remaining, cancelled, nil
}

// removes duplicates from a slice of queue items by step run id
func removeDuplicates(qis []*scheduling.QueueItemWithOrder) ([]*scheduling.QueueItemWithOrder, []*scheduling.QueueItemWithOrder) {
	encountered := map[string]bool{}
	result := []*scheduling.QueueItemWithOrder{}
	duplicates := []*scheduling.QueueItemWithOrder{}

	for _, v := range qis {
		stepRunId := sqlchelpers.UUIDToStr(v.StepRunId)
		if encountered[stepRunId] {
			duplicates = append(duplicates, v)
			continue
		}

		encountered[stepRunId] = true
		result = append(result, v)
	}

	return result, duplicates
}

func (r *stepRunEngineRepository) prepareTx(ctx context.Context, timeoutMs int) (pgx.Tx, func(), error) {
	tx, err := r.pool.Begin(ctx)

	if err != nil {
		return nil, nil, err
	}

	defer deferRollback(ctx, r.l, tx.Rollback)

	// set tx timeout to 5 seconds to avoid deadlocks
	_, err = tx.Exec(ctx, fmt.Sprintf("SET statement_timeout=%d", timeoutMs))

	if err != nil {
		return nil, nil, err
	}

	return tx, func() {
		deferRollback(ctx, r.l, tx.Rollback)
	}, nil
}

func printQueueDebugInfo(
	l zerolog.Logger,
	tenantId string,
	queues []*dbsqlc.Queue,
	queueItems []*scheduling.QueueItemWithOrder,
	duplicates []*scheduling.QueueItemWithOrder,
	cancelled []*scheduling.QueueItemWithOrder,
	plan scheduling.SchedulePlan,
	slots []*dbsqlc.ListSemaphoreSlotsToAssignRow,
	startedAt time.Time,
) {
	duration := time.Since(startedAt)

	e := l.Debug()
	msg := "queue debug information"

	if duration > 100*time.Millisecond {
		e = l.Warn()
		msg = "queue duration was greater than 100ms"
	}

	e.Str(
		"tenant_id", tenantId,
	).Int(
		"num_queues", len(queues),
	).Int(
		"total_step_runs", len(queueItems),
	).Int(
		"total_step_runs_assigned", len(plan.StepRunIds),
	).Int(
		"total_slots", len(slots),
	).Int(
		"num_duplicates", len(duplicates),
	).Int(
		"num_cancelled", len(cancelled),
	).Dur(
		"total_duration", duration,
	).Msg(msg)
}
