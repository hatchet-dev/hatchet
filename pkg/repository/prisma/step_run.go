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

	"github.com/hashicorp/go-multierror"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/services/shared/defaults"
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
	cachedWorkerCounts *workerCountCache
}

func NewStepRunEngineRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger, cf *server.ConfigFileRuntime) repository.StepRunEngineRepository {
	queries := dbsqlc.New()

	return &stepRunEngineRepository{
		pool:               pool,
		v:                  v,
		l:                  l,
		queries:            queries,
		cf:                 cf,
		exhaustedRLCache:   scheduling.NewExhaustedRateLimitCache(time.Minute),
		cachedWorkerCounts: &workerCountCache{},
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

func (s *stepRunEngineRepository) ListStepRunsToReassign(ctx context.Context, tenantId string) ([]string, error) {
	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer deferRollback(ctx, s.l, tx.Rollback)

	// get the step run and make sure it's still in pending
	stepRunReassign, err := s.queries.ListStepRunsToReassign(ctx, tx, pgTenantId)

	if err != nil {
		return nil, err
	}

	stepRunIds := make([]pgtype.UUID, len(stepRunReassign))
	stepRunIdsStr := make([]string, len(stepRunReassign))
	workerIds := make([]pgtype.UUID, len(stepRunReassign))
	retryCounts := make([]int32, len(stepRunReassign))

	for i, sr := range stepRunReassign {
		stepRunIds[i] = sr.ID
		stepRunIdsStr[i] = sqlchelpers.UUIDToStr(sr.ID)
		workerIds[i] = sr.WorkerId
		retryCounts[i] = sr.RetryCount
	}

	// release the semaphore slot
	err = s.bulkReleaseWorkerSemaphoreQueueItems(ctx, tx, tenantId, workerIds, stepRunIds, retryCounts)

	if err != nil {
		return nil, err
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, err
	}

	messages := make([]string, len(stepRunIds))
	reasons := make([]dbsqlc.StepRunEventReason, len(stepRunIds))
	severities := make([]dbsqlc.StepRunEventSeverity, len(stepRunIds))
	data := make([]map[string]interface{}, len(stepRunIds))

	for i := range stepRunIds {
		workerId := sqlchelpers.UUIDToStr(workerIds[i])
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

	return stepRunIdsStr, nil
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

func (s *stepRunEngineRepository) ReleaseStepRunSemaphore(ctx context.Context, tenantId, stepRunId string, isUserTriggered bool) error {
	return deadlockRetry(s.l, func() error {
		tx, commit, rollback, err := s.prepareTx(ctx, 5000)

		if err != nil {
			return err
		}

		defer rollback()

		err = s.releaseWorkerSemaphoreSlot(ctx, tx, tenantId, stepRunId)

		if err != nil {
			return fmt.Errorf("could not release worker semaphore slot: %w", err)
		}

		if isUserTriggered {

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

			// Update the Step Run to release the semaphore
			err = s.queries.ManualReleaseSemaphore(ctx, tx, dbsqlc.ManualReleaseSemaphoreParams{
				Steprunid: stepRun.SRID,
				Tenantid:  stepRun.SRTenantId,
			})

			if err != nil {
				return fmt.Errorf("could not update step run semaphoreRelease: %w", err)
			}
		}

		return commit(ctx)
	})
}

func (s *stepRunEngineRepository) DeferredStepRunEvent(
	tenantId, stepRunId string,
	opts repository.CreateStepRunEventOpts,
) {
	if err := s.v.Validate(opts); err != nil {
		s.l.Err(err).Msg("could not validate step run event")
		return
	}

	deferredStepRunEvent(
		s.l,
		s.pool,
		s.queries,
		tenantId,
		stepRunId,
		opts,
	)
}

func deferredStepRunEvent(
	l *zerolog.Logger,
	dbtx dbsqlc.DBTX,
	queries *dbsqlc.Queries,
	tenantId, stepRunId string,
	opts repository.CreateStepRunEventOpts,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := insertStepRunQueueItem(ctx, dbtx, queries, tenantId, updateStepRunQueueData{
		StepRunId: stepRunId,
		Event:     &opts,
	})

	if err != nil {
		l.Err(err).Msg("could not create deferred step run event")
		return
	}
}

func (s *stepRunEngineRepository) bulkStepRunsAssigned(
	stepRunIds []pgtype.UUID,
	workerIds []pgtype.UUID,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	workerIdToStepRunIds := make(map[string][]string)
	messages := make([]string, len(stepRunIds))
	reasons := make([]dbsqlc.StepRunEventReason, len(stepRunIds))
	severities := make([]dbsqlc.StepRunEventSeverity, len(stepRunIds))
	data := make([]map[string]interface{}, len(stepRunIds))

	for i := range stepRunIds {
		workerId := sqlchelpers.UUIDToStr(workerIds[i])

		if _, ok := workerIdToStepRunIds[workerId]; !ok {
			workerIdToStepRunIds[workerId] = make([]string, 0)
		}

		workerIdToStepRunIds[workerId] = append(workerIdToStepRunIds[workerId], sqlchelpers.UUIDToStr(stepRunIds[i]))
		messages[i] = fmt.Sprintf("Assigned to worker %s", workerId)
		reasons[i] = dbsqlc.StepRunEventReasonASSIGNED
		severities[i] = dbsqlc.StepRunEventSeverityINFO
		data[i] = map[string]interface{}{"worker_id": workerId}
	}

	orderedWorkerIds := make([]pgtype.UUID, 0)
	assignedStepRuns := make([][]byte, 0)

	for workerId, stepRunIds := range workerIdToStepRunIds {
		orderedWorkerIds = append(orderedWorkerIds, sqlchelpers.UUIDFromStr(workerId))
		assignedStepRunsBytes, _ := json.Marshal(stepRunIds) // nolint: errcheck
		assignedStepRuns = append(assignedStepRuns, assignedStepRunsBytes)
	}

	err := s.queries.CreateWorkerAssignEvents(ctx, s.pool, dbsqlc.CreateWorkerAssignEventsParams{
		Workerids:        orderedWorkerIds,
		Assignedstepruns: assignedStepRuns,
	})

	if err != nil {
		s.l.Err(err).Msg("could not create worker assign events")
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

	listQueuesFinishedAt := time.Now().UTC()

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

	durationListQueueItems := time.Since(startedAt)

	var duplicates []*scheduling.QueueItemWithOrder
	var finalized []*scheduling.QueueItemWithOrder

	queueItems, duplicates = removeDuplicates(queueItems)
	queueItems, finalized, err = s.removeFinalizedStepRuns(ctx, tx, queueItems)

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

	startGetWorkerCounts := time.Now()

	// list workers to assign
	workers, err := s.queries.GetWorkerDispatcherActions(ctx, tx, dbsqlc.GetWorkerDispatcherActionsParams{
		Tenantid:  pgTenantId,
		Actionids: uniqueActionsArr,
	})

	if err != nil {
		return emptyRes, fmt.Errorf("could not get worker dispatcher actions: %w", err)
	}

	workerIds := make([]string, 0, len(workers))

	for _, worker := range workers {
		workerIds = append(workerIds, sqlchelpers.UUIDToStr(worker.ID))
	}

	workersToCounts := s.cachedWorkerCounts.get(tenantId, workerIds)

	slots := make([]*scheduling.Slot, 0)

	for _, worker := range workers {
		workerId := sqlchelpers.UUIDToStr(worker.ID)
		dispatcherId := sqlchelpers.UUIDToStr(worker.DispatcherId)
		actionId := worker.ActionId

		count, ok := workersToCounts[workerId]

		if !ok {
			continue
		}

		for i := 0; i < count; i++ {
			slots = append(slots, &scheduling.Slot{
				ID:           fmt.Sprintf("%s-%d", workerId, i),
				WorkerId:     workerId,
				DispatcherId: dispatcherId,
				ActionId:     actionId,
			})
		}
	}

	finishedGetWorkerCounts := time.Since(startGetWorkerCounts)

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
		workerIdSet := UniqueSet(slots, func(x *scheduling.Slot) string {
			return x.WorkerId
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

	startAssignTime := time.Now()

	numAssigns := make(map[string]int)

	for _, workerId := range plan.WorkerIds {
		numAssigns[sqlchelpers.UUIDToStr(workerId)]++
	}

	// update the assigned worker counts
	s.cachedWorkerCounts.storeUnprocessedAssigns(tenantId, numAssigns)

	err = s.bulkAssignWorkerSemaphoreQueueItems(ctx, tx, tenantId, plan.WorkerIds, plan.StepRunIds)

	if err != nil {
		return emptyRes, fmt.Errorf("could not bulk assign worker semaphore queue items: %w", err)
	}

	_, err = s.queries.UpdateStepRunsToAssigned(ctx, tx, dbsqlc.UpdateStepRunsToAssignedParams{
		Steprunids:      plan.StepRunIds,
		Workerids:       plan.WorkerIds,
		Stepruntimeouts: plan.StepRunTimeouts,
	})

	if err != nil {
		return emptyRes, fmt.Errorf("could not bulk assign step runs to workers: %w", err)
	}

	finishedAssignTime := time.Since(startAssignTime)

	popItems := plan.QueuedItems

	// we'd like to remove duplicates from the queue items as well
	for _, item := range duplicates {
		// print a warning for duplicates
		s.l.Warn().Msgf("duplicate queue item: %d for step run %s", item.QueueItem.ID, sqlchelpers.UUIDToStr(item.QueueItem.StepRunId))

		popItems = append(popItems, item.QueueItem.ID)
	}

	// we'd like to remove finalized step runs from the queue items as well
	for _, item := range finalized {
		popItems = append(popItems, item.QueueItem.ID)
	}

	startQueueTime := time.Now()

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

	finishQueueTime := time.Since(startQueueTime)

	err = tx.Commit(ctx)

	if err != nil {
		return emptyRes, fmt.Errorf("could not commit transaction: %w", err)
	}

	defer s.bulkStepRunsAssigned(plan.StepRunIds, plan.WorkerIds)
	defer s.bulkStepRunsUnassigned(plan.UnassignedStepRunIds)
	defer s.bulkStepRunsRateLimited(plan.RateLimitedStepRuns)

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

	defer printQueueDebugInfo(ql, tenantId, queues, queueItems, duplicates, finalized, plan, slots, startedAt, listQueuesFinishedAt.Sub(startedAt), durationListQueueItems, finishedGetWorkerCounts, finishedAssignTime, finishQueueTime)

	return repository.QueueStepRunsResult{
		Queued:             plan.QueuedStepRuns,
		SchedulingTimedOut: timedOutStepRunsStr,
		Continue:           plan.ShouldContinue,
	}, nil
}

func (s *stepRunEngineRepository) UpdateWorkerSemaphoreCounts(ctx context.Context, qlp *zerolog.Logger, tenantId string) (bool, bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "update-worker-semaphore-counts-database")
	defer span.End()

	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return false, false, err
	}

	defer deferRollback(ctx, s.l, tx.Rollback)

	shouldContinue, didReleaseSlots, err := s.updateWorkerSemaphoreCountsTx(ctx, tx, tenantId)

	if err != nil {
		return false, false, fmt.Errorf("could not update worker semaphore counts: %w", err)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return false, false, fmt.Errorf("could not commit transaction: %w", err)
	}

	return shouldContinue, didReleaseSlots, nil
}

func (s *stepRunEngineRepository) updateWorkerSemaphoreCountsTx(ctx context.Context, tx pgx.Tx, tenantId string) (bool, bool, error) {
	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	limit := 200

	if s.cf.SingleQueueLimit != 0 {
		limit = s.cf.SingleQueueLimit * 2 // we're assigning and releasing slots, so we need to double the limit
	}

	pgLimit := pgtype.Int4{
		Int32: int32(limit),
		Valid: true,
	}

	// list queues
	queueItems, err := s.queries.ListInternalQueueItems(ctx, tx, dbsqlc.ListInternalQueueItemsParams{
		Tenantid: pgTenantId,
		Queue:    dbsqlc.InternalQueueWORKERSEMAPHORECOUNT,
		Limit:    pgLimit,
	})

	if err != nil {
		return false, false, fmt.Errorf("could not list queues: %w", err)
	}

	data, err := toQueueItemData[workerSemaphoreQueueData](queueItems)

	if err != nil {
		return false, false, fmt.Errorf("could not convert internal queue item data to worker semaphore queue data: %w", err)
	}

	uniqueWorkerIds := make(map[string]bool)

	for _, item := range data {
		uniqueWorkerIds[item.WorkerId] = true
	}

	workerIds := make([]pgtype.UUID, 0, len(uniqueWorkerIds))

	for workerId := range uniqueWorkerIds {
		workerIds = append(workerIds, sqlchelpers.UUIDFromStr(workerId))
	}

	workerCounts, err := s.queries.GetWorkerSemaphoreCounts(ctx, tx, dbsqlc.GetWorkerSemaphoreCountsParams{
		Tenantid:  pgTenantId,
		WorkerIds: workerIds,
	})

	if err != nil {
		return false, false, fmt.Errorf("could not get worker semaphore counts: %w", err)
	}

	workersToCounts := make(map[string]int)

	for _, worker := range workerCounts {
		workersToCounts[sqlchelpers.UUIDToStr(worker.WorkerId)] = int(worker.Count)
	}

	processedAssigns := make(map[string]int)
	didReleaseSlots := false

	// append the semaphore queue items to the worker counts
	for _, item := range data {
		workersToCounts[item.WorkerId] += item.Inc

		if item.Inc < 0 {
			processedAssigns[item.WorkerId]++
		}

		if item.Inc > 0 {
			didReleaseSlots = true
		}
	}

	qiIds := make([]int64, 0, len(data))

	for _, item := range queueItems {
		qiIds = append(qiIds, item.ID)
	}

	// update the processed semaphore queue items
	err = s.queries.MarkInternalQueueItemsProcessed(ctx, tx, qiIds)

	if err != nil {
		return false, false, fmt.Errorf("could not mark worker semaphore queue items processed: %w", err)
	}

	updateCountParams := dbsqlc.UpdateWorkerSemaphoreCountsParams{
		Workerids: make([]pgtype.UUID, 0, len(workersToCounts)),
		Counts:    make([]int32, 0, len(workersToCounts)),
	}

	for workerId, count := range workersToCounts {
		updateCountParams.Workerids = append(updateCountParams.Workerids, sqlchelpers.UUIDFromStr(workerId))
		updateCountParams.Counts = append(updateCountParams.Counts, int32(count))

		// if count is negative, print a warning
		if count < 0 {
			s.l.Warn().Msgf("worker %s has a negative count: %d", workerId, count)
		}
	}

	err = s.queries.UpdateWorkerSemaphoreCounts(ctx, tx, updateCountParams)

	if err != nil {
		return false, false, fmt.Errorf("could not update worker semaphore counts: %w", err)
	}

	s.cachedWorkerCounts.store(tenantId, workersToCounts, processedAssigns)

	return len(queueItems) == limit, didReleaseSlots, nil
}

func (s *stepRunEngineRepository) ProcessStepRunUpdates(ctx context.Context, qlp *zerolog.Logger, tenantId string) (repository.ProcessStepRunUpdatesResult, error) {
	ql := qlp.With().Str("tenant_id", tenantId).Logger()
	startedAt := time.Now().UTC()

	emptyRes := repository.ProcessStepRunUpdatesResult{
		Continue: false,
	}

	ctx, span := telemetry.NewSpan(ctx, "process-step-run-updates-database")
	defer span.End()

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	limit := 100

	if s.cf.SingleQueueLimit != 0 {
		limit = s.cf.SingleQueueLimit * 4 // we call update step run 4x
	}

	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return emptyRes, err
	}

	defer deferRollback(ctx, s.l, tx.Rollback)

	// list queues
	queueItems, err := s.queries.ListInternalQueueItems(ctx, tx, dbsqlc.ListInternalQueueItemsParams{
		Tenantid: pgTenantId,
		Queue:    dbsqlc.InternalQueueSTEPRUNUPDATE,
		Limit: pgtype.Int4{
			Int32: int32(limit),
			Valid: true,
		},
	})

	if err != nil {
		return emptyRes, fmt.Errorf("could not list queues: %w", err)
	}

	data, err := toQueueItemData[updateStepRunQueueData](queueItems)

	if err != nil {
		return emptyRes, fmt.Errorf("could not convert internal queue item data to worker semaphore queue data: %w", err)
	}

	// process the data by preparing a batch exec on UpdateStepRun
	params := make([]dbsqlc.UpdateStepRunBatchParams, 0, len(data))
	stepRunIds := make([]pgtype.UUID, 0, len(data))
	eventReasons := make([]dbsqlc.StepRunEventReason, 0, len(data))
	eventStepRunIds := make([]pgtype.UUID, 0, len(data))
	eventSeverities := make([]dbsqlc.StepRunEventSeverity, 0, len(data))
	eventMessages := make([]string, 0, len(data))
	eventData := make([]map[string]interface{}, 0, len(data))
	dedupe := make(map[string]bool)

	for _, item := range data {
		stepRunId := sqlchelpers.UUIDFromStr(item.StepRunId)

		if item.Event != nil {
			if item.Event.EventMessage == nil || item.Event.EventReason == nil {
				continue
			}

			dedupeKey := fmt.Sprintf("EVENT-%s-%s", item.StepRunId, *item.Event.EventReason)

			if _, ok := dedupe[dedupeKey]; ok {
				continue
			}

			dedupe[dedupeKey] = true

			eventStepRunIds = append(eventStepRunIds, stepRunId)
			eventMessages = append(eventMessages, *item.Event.EventMessage)
			eventReasons = append(eventReasons, *item.Event.EventReason)

			if item.Event.EventSeverity != nil {
				eventSeverities = append(eventSeverities, *item.Event.EventSeverity)
			} else {
				eventSeverities = append(eventSeverities, dbsqlc.StepRunEventSeverityINFO)
			}

			if item.Event.EventData != nil {
				eventData = append(eventData, item.Event.EventData)
			} else {
				eventData = append(eventData, map[string]interface{}{})
			}

			continue
		} else if item.Status != nil {
			dedupeKey := fmt.Sprintf("SR-%s-%s", item.StepRunId, *item.Status)

			if _, ok := dedupe[dedupeKey]; ok {
				continue
			}

			dedupe[dedupeKey] = true
		}

		stepRunIds = append(stepRunIds, stepRunId)
		updateParam := dbsqlc.UpdateStepRunBatchParams{
			ID:       stepRunId,
			Tenantid: pgTenantId,
		}

		if item.Status != nil {
			updateParam.Status = dbsqlc.NullStepRunStatus{
				StepRunStatus: dbsqlc.StepRunStatus(*item.Status),
				Valid:         true,
			}

			switch dbsqlc.StepRunStatus(*item.Status) {
			case dbsqlc.StepRunStatusSUCCEEDED:
				eventStepRunIds = append(eventStepRunIds, stepRunId)
				eventReasons = append(eventReasons, dbsqlc.StepRunEventReasonFINISHED)
				eventSeverities = append(eventSeverities, dbsqlc.StepRunEventSeverityINFO)
				eventMessages = append(eventMessages, fmt.Sprintf("Step run finished at %s", item.FinishedAt.Format(time.RFC1123)))
				eventData = append(eventData, map[string]interface{}{})
			case dbsqlc.StepRunStatusRUNNING:
				eventStepRunIds = append(eventStepRunIds, stepRunId)
				eventReasons = append(eventReasons, dbsqlc.StepRunEventReasonSTARTED)
				eventSeverities = append(eventSeverities, dbsqlc.StepRunEventSeverityINFO)
				eventMessages = append(eventMessages, fmt.Sprintf("Step run started at %s", item.StartedAt.Format(time.RFC1123)))
				eventData = append(eventData, map[string]interface{}{})
			case dbsqlc.StepRunStatusFAILED:
				eventStepRunIds = append(eventStepRunIds, stepRunId)
				eventMessage := fmt.Sprintf("Step run failed on %s", item.FinishedAt.Format(time.RFC1123))
				eventReason := dbsqlc.StepRunEventReasonFAILED

				if item.Error != nil && *item.Error == "TIMED_OUT" {
					eventReason = dbsqlc.StepRunEventReasonTIMEDOUT
					eventMessage = "Step exceeded timeout duration"
				}

				eventReasons = append(eventReasons, eventReason)
				eventSeverities = append(eventSeverities, dbsqlc.StepRunEventSeverityCRITICAL)
				eventMessages = append(eventMessages, eventMessage)
				eventData = append(eventData, map[string]interface{}{})
			case dbsqlc.StepRunStatusCANCELLED:
				eventStepRunIds = append(eventStepRunIds, stepRunId)
				eventReasons = append(eventReasons, dbsqlc.StepRunEventReasonCANCELLED)
				eventSeverities = append(eventSeverities, dbsqlc.StepRunEventSeverityWARNING)
				eventMessages = append(eventMessages, fmt.Sprintf("Step run was cancelled on %s for the following reason: %s", item.CancelledAt.Format(time.RFC1123), *item.CancelledReason))
				eventData = append(eventData, map[string]interface{}{})
			}
		}

		if item.StartedAt != nil {
			updateParam.StartedAt = sqlchelpers.TimestampFromTime(*item.StartedAt)
		}

		if item.FinishedAt != nil {
			updateParam.FinishedAt = sqlchelpers.TimestampFromTime(*item.FinishedAt)
		}

		if item.Error != nil {
			updateParam.Error = pgtype.Text{
				String: *item.Error,
				Valid:  true,
			}
		}

		if item.CancelledAt != nil {
			updateParam.CancelledAt = sqlchelpers.TimestampFromTime(*item.CancelledAt)
		}

		if item.CancelledReason != nil {
			updateParam.CancelledReason = pgtype.Text{
				String: *item.CancelledReason,
				Valid:  true,
			}
		}

		if item.Output != nil {
			updateParam.Output = item.Output
		}

		params = append(params, updateParam)
	}

	err = nil

	s.queries.UpdateStepRunBatch(ctx, tx, params).Exec(func(i int, innerErr error) {
		if innerErr != nil {
			err = multierror.Append(err, fmt.Errorf("could not update step run batch: %w", innerErr))
		}
	})

	if err != nil {
		return emptyRes, fmt.Errorf("could not update step runs: %w", err)
	}

	durationUpdateStepRuns := time.Since(startedAt)

	startResolveJobRunStatus := time.Now()

	// update the job runs and workflow runs as well
	jobRunIds, err := s.queries.ResolveJobRunStatus(ctx, tx, dbsqlc.ResolveJobRunStatusParams{
		Steprunids: stepRunIds,
		Tenantid:   pgTenantId,
	})

	if err != nil {
		return emptyRes, fmt.Errorf("could not resolve job run status: %w", err)
	}

	durationResolveJobRunStatus := time.Since(startResolveJobRunStatus)

	startResolveWorkflowRuns := time.Now()

	completedWorkflowRuns, err := s.queries.ResolveWorkflowRunStatus(ctx, tx, dbsqlc.ResolveWorkflowRunStatusParams{
		Jobrunids: jobRunIds,
		Tenantid:  pgTenantId,
	})

	if err != nil {
		return emptyRes, fmt.Errorf("could not resolve workflow run status: %w", err)
	}

	durationResolveWorkflowRuns := time.Since(startResolveWorkflowRuns)

	qiIds := make([]int64, 0, len(data))

	for _, item := range queueItems {
		qiIds = append(qiIds, item.ID)
	}

	startMarkQueueItemsProcessed := time.Now()

	// update the processed semaphore queue items
	err = s.queries.MarkInternalQueueItemsProcessed(ctx, tx, qiIds)

	if err != nil {
		return emptyRes, fmt.Errorf("could not mark worker semaphore queue items processed: %w", err)
	}

	durationMarkQueueItemsProcessed := time.Since(startMarkQueueItemsProcessed)

	startRunEvents := time.Now()

	// NOTE: actually not deferred
	deferredBulkStepRunEvents(ctx, s.l, tx, s.queries, eventStepRunIds, eventReasons, eventSeverities, eventMessages, eventData)

	durationRunEvents := time.Since(startRunEvents)

	err = tx.Commit(ctx)

	if err != nil {
		return emptyRes, fmt.Errorf("could not commit transaction: %w", err)
	}

	defer printProcessStepRunUpdateInfo(ql, tenantId, startedAt, len(stepRunIds), durationUpdateStepRuns, durationResolveJobRunStatus, durationResolveWorkflowRuns, durationMarkQueueItemsProcessed, durationRunEvents)

	return repository.ProcessStepRunUpdatesResult{
		CompletedWorkflowRuns: completedWorkflowRuns,
		Continue:              len(queueItems) == limit,
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

func (s *stepRunEngineRepository) CleanupInternalQueueItems(ctx context.Context, tenantId string) error {
	// setup telemetry
	ctx, span := telemetry.NewSpan(ctx, "cleanup-internal-queue-items-database")
	defer span.End()

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	// get the min and max queue items
	minMax, err := s.queries.GetMinMaxProcessedInternalQueueItems(ctx, s.pool, pgTenantId)

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
		err := s.queries.CleanupInternalQueueItems(ctx, s.pool, dbsqlc.CleanupInternalQueueItemsParams{
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

func (s *stepRunEngineRepository) StepRunStarted(ctx context.Context, tenantId, stepRunId string, startedAt time.Time) error {
	ctx, span := telemetry.NewSpan(ctx, "step-run-started-db")
	defer span.End()

	running := string(dbsqlc.StepRunStatusRUNNING)

	// write a queue item that the step run has started
	err := insertStepRunQueueItem(
		ctx,
		s.pool,
		s.queries,
		tenantId,
		updateStepRunQueueData{
			StepRunId: stepRunId,
			StartedAt: &startedAt,
			Status:    &running,
		},
	)

	if err != nil {
		return fmt.Errorf("could not insert step run queue item: %w", err)
	}

	return nil
}

func (s *stepRunEngineRepository) StepRunSucceeded(ctx context.Context, tenantId, stepRunId string, finishedAt time.Time, output []byte) error {
	ctx, span := telemetry.NewSpan(ctx, "step-run-started-db")
	defer span.End()

	finished := string(dbsqlc.StepRunStatusSUCCEEDED)

	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return err
	}

	defer deferRollback(ctx, s.l, tx.Rollback)

	// write a queue item to release the worker semaphore
	err = s.releaseWorkerSemaphoreSlot(ctx, tx, tenantId, stepRunId)

	if err != nil {
		return fmt.Errorf("could not release worker semaphore queue items: %w", err)
	}

	// write a queue item that the step run has finished
	err = insertStepRunQueueItem(
		ctx,
		tx,
		s.queries,
		tenantId,
		updateStepRunQueueData{
			StepRunId:  stepRunId,
			FinishedAt: &finishedAt,
			Status:     &finished,
			Output:     output,
		},
	)

	if err != nil {
		return fmt.Errorf("could not insert step run queue item: %w", err)
	}

	// update the job run lookup data
	err = s.queries.UpdateJobRunLookupDataWithStepRun(ctx, tx, dbsqlc.UpdateJobRunLookupDataWithStepRunParams{
		Steprunid: sqlchelpers.UUIDFromStr(stepRunId),
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
		Jsondata:  output,
	})

	if err != nil {
		return fmt.Errorf("could not update job run lookup data: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("could not commit transaction: %w", err)
	}

	return nil
}

func (s *stepRunEngineRepository) StepRunCancelled(ctx context.Context, tenantId, stepRunId string, cancelledAt time.Time, cancelledReason string) error {
	ctx, span := telemetry.NewSpan(ctx, "step-run-cancelled-db")
	defer span.End()

	cancelled := string(dbsqlc.StepRunStatusCANCELLED)

	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return err
	}

	defer deferRollback(ctx, s.l, tx.Rollback)

	// release the worker semaphore
	err = s.releaseWorkerSemaphoreSlot(ctx, tx, tenantId, stepRunId)

	if err != nil {
		return fmt.Errorf("could not release worker semaphore queue items: %w", err)
	}

	// check that the step run is not in a final state
	stepRun, err := s.getStepRunForEngineTx(ctx, tx, tenantId, stepRunId)

	if err != nil {
		return fmt.Errorf("could not get step run: %w", err)
	}

	if !repository.IsFinalStepRunStatus(stepRun.SRStatus) {
		// write a queue item that the step run has failed
		err = insertStepRunQueueItem(
			ctx,
			tx,
			s.queries,
			tenantId,
			updateStepRunQueueData{
				StepRunId:       stepRunId,
				CancelledAt:     &cancelledAt,
				CancelledReason: &cancelledReason,
				Status:          &cancelled,
			},
		)

		if err != nil {
			return fmt.Errorf("could not insert step run queue item: %w", err)
		}

		_, err = s.queries.ResolveLaterStepRuns(ctx, tx, dbsqlc.ResolveLaterStepRunsParams{
			Steprunid: sqlchelpers.UUIDFromStr(stepRunId),
			Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
			Status:    dbsqlc.StepRunStatusCANCELLED,
		})

		if err != nil {
			return fmt.Errorf("could not resolve later step runs: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("could not commit transaction: %w", err)
	}

	return nil
}

func (s *stepRunEngineRepository) StepRunFailed(ctx context.Context, tenantId, stepRunId string, failedAt time.Time, errStr string) error {
	ctx, span := telemetry.NewSpan(ctx, "step-run-failed-db")
	defer span.End()

	failed := string(dbsqlc.StepRunStatusFAILED)

	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return err
	}

	defer deferRollback(ctx, s.l, tx.Rollback)

	// release the worker semaphore
	err = s.releaseWorkerSemaphoreSlot(ctx, tx, tenantId, stepRunId)

	if err != nil {
		return fmt.Errorf("could not release worker semaphore queue items: %w", err)
	}

	// check that the step run is not in a final state
	stepRun, err := s.getStepRunForEngineTx(ctx, tx, tenantId, stepRunId)

	if err != nil {
		return fmt.Errorf("could not get step run: %w", err)
	}

	if !repository.IsFinalStepRunStatus(stepRun.SRStatus) {
		// write a queue item that the step run has failed
		err = insertStepRunQueueItem(
			ctx,
			tx,
			s.queries,
			tenantId,
			updateStepRunQueueData{
				StepRunId:  stepRunId,
				FinishedAt: &failedAt,
				Error:      &errStr,
				Status:     &failed,
			},
		)

		if err != nil {
			return fmt.Errorf("could not insert step run queue item: %w", err)
		}

		_, err = s.queries.ResolveLaterStepRuns(ctx, tx, dbsqlc.ResolveLaterStepRunsParams{
			Steprunid: sqlchelpers.UUIDFromStr(stepRunId),
			Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
			Status:    dbsqlc.StepRunStatusFAILED,
		})

		if err != nil {
			return fmt.Errorf("could not resolve later step runs: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("could not commit transaction: %w", err)
	}

	return nil
}

func (s *stepRunEngineRepository) ReplayStepRun(ctx context.Context, tenantId, stepRunId string, input []byte) (*dbsqlc.GetStepRunForEngineRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "replay-step-run")
	defer span.End()

	tx, commit, rollback, err := s.prepareTx(ctx, 5000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	innerStepRun, err := s.getStepRunForEngineTx(ctx, tx, tenantId, stepRunId)

	if err != nil {
		return nil, err
	}

	sev := dbsqlc.StepRunEventSeverityINFO
	reason := dbsqlc.StepRunEventReasonRETRIEDBYUSER

	defer deferredStepRunEvent(
		s.l,
		s.pool,
		s.queries,
		tenantId,
		stepRunId,
		repository.CreateStepRunEventOpts{
			EventMessage:  repository.StringPtr("This step was manually replayed by a user"),
			EventSeverity: &sev,
			EventReason:   &reason,
		},
	)

	// check if the step run is in a final state
	if !repository.IsFinalStepRunStatus(innerStepRun.SRStatus) {
		return nil, fmt.Errorf("step run is not in a final state")
	}

	// reset the job run, workflow run and all fields as part of the core tx
	_, err = s.queries.ReplayStepRunResetWorkflowRun(ctx, tx, innerStepRun.WorkflowRunId)

	if err != nil {
		return nil, err
	}

	_, err = s.queries.ReplayStepRunResetJobRun(ctx, tx, innerStepRun.JobRunId)

	if err != nil {
		return nil, err
	}

	laterStepRuns, err := s.queries.GetLaterStepRuns(ctx, tx, dbsqlc.GetLaterStepRunsParams{
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
		Steprunid: sqlchelpers.UUIDFromStr(stepRunId),
	})

	if err != nil {
		return nil, err
	}

	// archive each of the later step run results
	for _, laterStepRun := range laterStepRuns {
		laterStepRunId := sqlchelpers.UUIDToStr(laterStepRun.ID)

		err = archiveStepRunResult(ctx, s.queries, tx, tenantId, laterStepRunId)

		if err != nil {
			return nil, err
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
			return nil, err
		}

		// create a deferred event for each of these step runs
		sev := dbsqlc.StepRunEventSeverityINFO
		reason := dbsqlc.StepRunEventReasonRETRIEDBYUSER

		defer deferredStepRunEvent(
			s.l,
			s.pool,
			s.queries,
			tenantId,
			laterStepRunId,
			repository.CreateStepRunEventOpts{
				EventMessage:  repository.StringPtr(fmt.Sprintf("Parent step run %s was replayed, resetting step run result", innerStepRun.StepReadableId.String)),
				EventSeverity: &sev,
				EventReason:   &reason,
			},
		)
	}

	// reset all later step runs to a pending state
	_, err = s.queries.ReplayStepRunResetStepRuns(ctx, tx, dbsqlc.ReplayStepRunResetStepRunsParams{
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
		Steprunid: sqlchelpers.UUIDFromStr(stepRunId),
		Input:     input,
	})

	if err != nil {
		return nil, err
	}

	stepRun, err := s.getStepRunForEngineTx(ctx, tx, tenantId, stepRunId)

	if err != nil {
		return nil, err
	}

	err = commit(ctx)

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

func (s *stepRunEngineRepository) QueueStepRun(ctx context.Context, tenantId, stepRunId string, opts *repository.QueueStepRunOpts) (*dbsqlc.GetStepRunForEngineRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "queue-step-run-database")
	defer span.End()

	if err := s.v.Validate(opts); err != nil {
		return nil, err
	}

	tx, commit, rollback, err := s.prepareTx(ctx, 5000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	queueParams := dbsqlc.QueueStepRunParams{
		ID:       sqlchelpers.UUIDFromStr(stepRunId),
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	}

	priority := 1

	if opts.Input != nil {
		queueParams.Input = opts.Input
	}

	if opts.IsRetry {
		queueParams.IsRetry = pgtype.Bool{
			Bool:  true,
			Valid: true,
		}
	}

	if opts.IsRetry || opts.IsInternalRetry {
		// if this is a retry, write a queue item to release the worker semaphore
		err = s.releaseWorkerSemaphoreSlot(ctx, tx, tenantId, stepRunId)

		if err != nil {
			return nil, fmt.Errorf("could not release worker semaphore queue items: %w", err)
		}

		// retries get highest priority to ensure that they're run immediately
		priority = 4
	}

	innerStepRun, err := s.getStepRunForEngineTx(ctx, tx, tenantId, stepRunId)

	if err != nil {
		return nil, err
	}

	err = s.queries.QueueStepRun(ctx, tx, queueParams)

	if err != nil {
		return nil, err
	}

	createQiParams := dbsqlc.CreateQueueItemParams{
		StepRunId:   innerStepRun.SRID,
		StepId:      innerStepRun.StepId,
		ActionId:    sqlchelpers.TextFromStr(innerStepRun.ActionId),
		StepTimeout: innerStepRun.StepTimeout,
		Tenantid:    sqlchelpers.UUIDFromStr(tenantId),
		Queue:       innerStepRun.SRQueue,
		Priority: pgtype.Int4{
			Valid: true,
			Int32: int32(priority),
		},
		Sticky:            innerStepRun.StickyStrategy,
		DesiredWorkerId:   innerStepRun.DesiredWorkerId,
		ScheduleTimeoutAt: getScheduleTimeout(innerStepRun),
	}

	// insert a queue item that the step run has been queued
	err = s.queries.CreateQueueItem(ctx, tx, createQiParams)

	if err != nil {
		return nil, fmt.Errorf("could not create queue item: %w", err)
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return innerStepRun, nil
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

	var srs []pgtype.UUID

	if parentStepRunId != nil {
		srs, err = s.queries.ListStartableStepRuns(ctx, tx, dbsqlc.ListStartableStepRunsParams{
			Jobrunid:                 sqlchelpers.UUIDFromStr(jobRunId),
			SucceededParentStepRunId: sqlchelpers.UUIDFromStr(*parentStepRunId),
		})

		if err != nil {
			return nil, fmt.Errorf("could not list startable step runs: %w", err)
		}
	} else {
		srs, err = s.queries.ListInitialStepRuns(ctx, tx, sqlchelpers.UUIDFromStr(jobRunId))

		if err != nil {
			return nil, fmt.Errorf("could not list initial step runs: %w", err)
		}
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

	sev := dbsqlc.StepRunEventSeverityINFO
	reason := dbsqlc.StepRunEventReasonTIMEOUTREFRESHED

	defer deferredStepRunEvent(
		s.l,
		s.pool,
		s.queries,
		tenantId,
		stepRunId,
		repository.CreateStepRunEventOpts{
			EventMessage:  repository.StringPtr(fmt.Sprintf("Timeout refreshed by %s", incrementTimeoutBy)),
			EventReason:   &reason,
			EventSeverity: &sev,
		},
	)

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

func (s *stepRunEngineRepository) removeFinalizedStepRuns(ctx context.Context, tx pgx.Tx, qis []*scheduling.QueueItemWithOrder) ([]*scheduling.QueueItemWithOrder, []*scheduling.QueueItemWithOrder, error) {
	currStepRunIds := make([]pgtype.UUID, len(qis))

	for i, qi := range qis {
		currStepRunIds[i] = qi.StepRunId
	}

	finalizedStepRuns, err := s.queries.GetFinalizedStepRuns(ctx, tx, currStepRunIds)

	if err != nil {
		return nil, nil, err
	}

	finalizedStepRunsMap := make(map[string]bool, len(finalizedStepRuns))

	for _, sr := range finalizedStepRuns {
		s.l.Warn().Msgf("step run %s is in state %s, skipping queueing", sqlchelpers.UUIDToStr(sr.ID), string(sr.Status))
		finalizedStepRunsMap[sqlchelpers.UUIDToStr(sr.ID)] = true
	}

	// remove cancelled step runs from the queue items
	remaining := make([]*scheduling.QueueItemWithOrder, 0, len(qis))
	cancelled := make([]*scheduling.QueueItemWithOrder, 0, len(qis))

	for _, qi := range qis {
		if _, ok := finalizedStepRunsMap[sqlchelpers.UUIDToStr(qi.StepRunId)]; ok {
			cancelled = append(cancelled, qi)
			continue
		}

		remaining = append(remaining, qi)
	}

	return remaining, cancelled, nil
}

func (s *stepRunEngineRepository) releaseWorkerSemaphoreSlot(ctx context.Context, tx pgx.Tx, tenantId, stepRunId string) error {
	oldWorkerIdAndRetryCount, err := s.queries.UpdateStepRunUnsetWorkerId(ctx, tx, dbsqlc.UpdateStepRunUnsetWorkerIdParams{
		Steprunid: sqlchelpers.UUIDFromStr(stepRunId),
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return err
	}

	if oldWorkerIdAndRetryCount.WorkerId.Valid {
		// add to internal queue
		err = s.bulkReleaseWorkerSemaphoreQueueItems(
			ctx,
			tx,
			tenantId,
			[]pgtype.UUID{oldWorkerIdAndRetryCount.WorkerId},
			[]pgtype.UUID{sqlchelpers.UUIDFromStr(stepRunId)},
			[]int32{oldWorkerIdAndRetryCount.RetryCount},
		)

		if err != nil {
			return err
		}
	}

	return nil
}

type workerSemaphoreQueueData struct {
	WorkerId  string `json:"worker_id"`
	StepRunId string `json:"step_run_id"`

	// Inc is what to increment the semaphore count by (-1 for assignment, 1 for release)
	Inc int `json:"inc"`
}

func (s *stepRunEngineRepository) bulkAssignWorkerSemaphoreQueueItems(
	ctx context.Context,
	tx pgx.Tx,
	tenantId string,
	workerIds []pgtype.UUID,
	stepRunIds []pgtype.UUID,
) error {
	insertData := make([]any, len(stepRunIds))

	for i, stepRunId := range stepRunIds {
		insertData[i] = workerSemaphoreQueueData{
			WorkerId:  sqlchelpers.UUIDToStr(workerIds[i]),
			StepRunId: sqlchelpers.UUIDToStr(stepRunId),
			Inc:       -1,
		}
	}

	return bulkInsertInternalQueueItem(
		ctx,
		tx,
		s.queries,
		tenantId,
		dbsqlc.InternalQueueWORKERSEMAPHORECOUNT,
		insertData,
	)
}

func (s *stepRunEngineRepository) bulkReleaseWorkerSemaphoreQueueItems(
	ctx context.Context,
	tx pgx.Tx,
	tenantId string,
	workerIds []pgtype.UUID,
	stepRunIds []pgtype.UUID,
	retryCounts []int32,
) error {
	// if length of workerIds and stepRunIds is not the same, return an error
	if len(workerIds) != len(stepRunIds) {
		return fmt.Errorf("workerIds and stepRunIds must be the same length")
	}

	insertData := make([]any, len(workerIds))
	uniqueKeys := make([]string, len(workerIds))

	for i, workerId := range workerIds {
		insertData[i] = workerSemaphoreQueueData{
			WorkerId:  sqlchelpers.UUIDToStr(workerId),
			StepRunId: sqlchelpers.UUIDToStr(stepRunIds[i]),
			Inc:       1,
		}

		uniqueKeys[i] = fmt.Sprintf(
			"%s:%d:release",
			sqlchelpers.UUIDToStr(stepRunIds[i]),
			retryCounts[i],
		)
	}

	return s.bulkInsertUniqueInternalQueueItem(
		ctx,
		tx,
		tenantId,
		dbsqlc.InternalQueueWORKERSEMAPHORECOUNT,
		insertData,
		uniqueKeys,
	)
}

func toQueueItemData[d any](items []*dbsqlc.InternalQueueItem) ([]d, error) {
	res := make([]d, len(items))

	for i, item := range items {
		var data d

		err := json.Unmarshal(item.Data, &data)

		if err != nil {
			return nil, err
		}

		res[i] = data
	}

	return res, nil
}

type updateStepRunQueueData struct {
	StepRunId string `json:"step_run_id"`

	Event *repository.CreateStepRunEventOpts `json:"event,omitempty"`

	StartedAt       *time.Time `json:"started_at,omitempty"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
	CancelledAt     *time.Time `json:"cancelled_at,omitempty"`
	Output          []byte     `json:"output"`
	CancelledReason *string    `json:"cancelled_reason,omitempty"`
	Error           *string    `json:"error,omitempty"`
	Status          *string    `json:"status,omitempty"`
}

func insertStepRunQueueItem(
	ctx context.Context,
	dbtx dbsqlc.DBTX,
	queries *dbsqlc.Queries,
	tenantId string,
	data updateStepRunQueueData,
) error {
	insertData := make([]any, 1)
	insertData[0] = data

	return bulkInsertInternalQueueItem(
		ctx,
		dbtx,
		queries,
		tenantId,
		dbsqlc.InternalQueueSTEPRUNUPDATE,
		insertData,
	)
}

func bulkInsertInternalQueueItem(
	ctx context.Context,
	dbtx dbsqlc.DBTX,
	queries *dbsqlc.Queries,
	tenantId string,
	queue dbsqlc.InternalQueue,
	data []any,
) error {
	// construct bytes for the data
	insertData := make([][]byte, len(data))

	for i, d := range data {
		b, err := json.Marshal(d)

		if err != nil {
			return err
		}

		insertData[i] = b
	}

	err := queries.CreateInternalQueueItemsBulk(ctx, dbtx, dbsqlc.CreateInternalQueueItemsBulkParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Queue:    queue,
		Datas:    insertData,
	})

	if err != nil {
		return err
	}

	return nil
}

func (s *stepRunEngineRepository) bulkInsertUniqueInternalQueueItem(
	ctx context.Context,
	tx pgx.Tx,
	tenantId string,
	queue dbsqlc.InternalQueue,
	data []any,
	uniqueKeys []string,
) error {
	// construct bytes for the data
	insertData := make([][]byte, len(data))

	for i, d := range data {
		b, err := json.Marshal(d)

		if err != nil {
			return err
		}

		insertData[i] = b
	}

	err := s.queries.CreateUniqueInternalQueueItemsBulk(ctx, tx, dbsqlc.CreateUniqueInternalQueueItemsBulkParams{
		Tenantid:   sqlchelpers.UUIDFromStr(tenantId),
		Queue:      queue,
		Datas:      insertData,
		Uniquekeys: uniqueKeys,
	})

	if err != nil {
		return err
	}

	return nil
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

func (r *stepRunEngineRepository) prepareTx(ctx context.Context, timeoutMs int) (pgx.Tx, func(context.Context) error, func(), error) {
	tx, err := r.pool.Begin(ctx)

	if err != nil {
		return nil, nil, nil, err
	}

	commit := func(ctx context.Context) error {
		// reset statement timeout
		_, err = tx.Exec(ctx, "SET statement_timeout=0")

		if err != nil {
			return err
		}

		return tx.Commit(ctx)
	}

	rollback := func() {
		deferRollback(ctx, r.l, tx.Rollback)
	}

	// set tx timeout to 5 seconds to avoid deadlocks
	_, err = tx.Exec(ctx, fmt.Sprintf("SET statement_timeout=%d", timeoutMs))

	if err != nil {
		return nil, nil, nil, err
	}

	return tx, commit, rollback, nil
}

func printQueueDebugInfo(
	l zerolog.Logger,
	tenantId string,
	queues []*dbsqlc.Queue,
	queueItems []*scheduling.QueueItemWithOrder,
	duplicates []*scheduling.QueueItemWithOrder,
	cancelled []*scheduling.QueueItemWithOrder,
	plan scheduling.SchedulePlan,
	slots []*scheduling.Slot,
	startedAt time.Time,
	durationListQueues time.Duration,
	durationListQueueItems time.Duration,
	durationGetWorkerCounts time.Duration,
	durationAssignQueueItems time.Duration,
	durationPopQueueItems time.Duration,
) {
	duration := time.Since(startedAt)

	e := l.Debug()
	msg := "queue debug information"

	if duration > 100*time.Millisecond {
		e = l.Warn()
		msg = fmt.Sprintf("queue duration was greater than 100ms (%s) for %d assignments", duration, len(plan.StepRunIds))
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
	).Dur(
		"duration_list_queues", durationListQueues,
	).Dur(
		"duration_list_queue_items", durationListQueueItems,
	).Dur(
		"duration_get_worker_counts", durationGetWorkerCounts,
	).Dur(
		"duration_assign_queue_items", durationAssignQueueItems,
	).Dur(
		"duration_pop_queue_items", durationPopQueueItems,
	).Msg(msg)
}

func printProcessStepRunUpdateInfo(
	l zerolog.Logger,
	tenantId string,
	startedAt time.Time,
	numStepRuns int,
	durationUpdateStepRuns time.Duration,
	durationResolveJobRuns time.Duration,
	durationResolveWorkflowRuns time.Duration,
	durationMarkQueueItemsProcessed time.Duration,
	durationWriteStepRunEvents time.Duration,
) {
	duration := time.Since(startedAt)

	e := l.Debug()
	msg := "process step run updates debug information"

	if duration > 100*time.Millisecond {
		e = l.Warn()
		msg = fmt.Sprintf("process step run updates duration was longer than 100ms (%s) for %d step runs", duration, numStepRuns)
	}

	e.Str(
		"tenant_id", tenantId,
	).Int(
		"num_step_runs", numStepRuns,
	).Dur(
		"total_duration", duration,
	).Dur(
		"duration_update_step_runs", durationUpdateStepRuns,
	).Dur(
		"duration_resolve_job_runs", durationResolveJobRuns,
	).Dur(
		"duration_resolve_workflow_runs", durationResolveWorkflowRuns,
	).Dur(
		"duration_mark_queue_items_processed", durationMarkQueueItemsProcessed,
	).Dur(
		"duration_write_step_run_events", durationWriteStepRunEvents,
	).Msg(msg)
}

func getScheduleTimeout(stepRun *dbsqlc.GetStepRunForEngineRow) pgtype.Timestamp {
	var timeoutDuration time.Duration

	// get the schedule timeout from the step
	stepScheduleTimeout := stepRun.StepScheduleTimeout

	if stepScheduleTimeout != "" {
		timeoutDuration, _ = time.ParseDuration(stepScheduleTimeout)
	} else {
		timeoutDuration = defaults.DefaultScheduleTimeout
	}

	timeout := time.Now().UTC().Add(timeoutDuration)

	return sqlchelpers.TimestampFromTime(timeout)
}
