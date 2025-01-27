package prisma

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
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

func (s *stepRunAPIRepository) GetStepRunById(stepRunId string) (*repository.GetStepRunFull, error) {
	stepRun, err := s.queries.GetStepRun(context.Background(), s.pool, sqlchelpers.UUIDFromStr(stepRunId))

	if err != nil {
		return nil, fmt.Errorf("could not get step run: %w", err)
	}

	childWorkflowRunIds, err := s.queries.ListChildWorkflowRunIds(context.Background(), s.pool, dbsqlc.ListChildWorkflowRunIdsParams{
		Steprun:  sqlchelpers.UUIDFromStr(stepRunId),
		Tenantid: stepRun.TenantId,
	})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("could not get child workflow run ids: %w", err)
	}

	childWorkflowRuns := make([]string, len(childWorkflowRunIds))

	for i, id := range childWorkflowRunIds {
		childWorkflowRuns[i] = sqlchelpers.UUIDToStr(id)
	}

	return &repository.GetStepRunFull{
		StepRun:           stepRun,
		ChildWorkflowRuns: childWorkflowRuns,
	}, nil
}

func (s *stepRunAPIRepository) ListStepRunEvents(stepRunId string, opts *repository.ListStepRunEventOpts) (*repository.ListStepRunEventResult, error) {
	if err := s.v.Validate(opts); err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(context.Background())

	if err != nil {
		return nil, err
	}

	defer sqlchelpers.DeferRollback(context.Background(), s.l, tx.Rollback)

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

func (s *stepRunAPIRepository) ListStepRunEventsByWorkflowRunId(ctx context.Context, tenantId, workflowRunId string, lastId *int32) (*repository.ListStepRunEventResult, error) {
	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer sqlchelpers.DeferRollback(context.Background(), s.l, tx.Rollback)

	listParams := dbsqlc.ListStepRunEventsByWorkflowRunIdParams{
		Workflowrunid: sqlchelpers.UUIDFromStr(workflowRunId),
		Tenantid:      sqlchelpers.UUIDFromStr(tenantId),
	}

	if lastId != nil {
		listParams.LastId = pgtype.Int8{
			Valid: true,
			Int64: int64(*lastId),
		}
	}

	allEvents, err := s.queries.ListWorkflowRunEventsByWorkflowRunId(ctx, tx, sqlchelpers.UUIDFromStr(workflowRunId))

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			allEvents = make([]*dbsqlc.StepRunEvent, 0)
		} else {
			return nil, fmt.Errorf("could not list workflow run events: %w", err)
		}
	}

	srEvents, err := s.queries.ListStepRunEventsByWorkflowRunId(context.Background(), tx, listParams)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			srEvents = make([]*dbsqlc.StepRunEvent, 0)
		} else {
			return nil, fmt.Errorf("could not list step run events: %w", err)
		}
	}

	err = tx.Commit(ctx)

	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	allEvents = append(allEvents, srEvents...)

	// sort all events by id asc
	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].ID > allEvents[j].ID
	})

	return &repository.ListStepRunEventResult{
		Rows: allEvents,
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

	defer sqlchelpers.DeferRollback(context.Background(), s.l, tx.Rollback)

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
	*sharedRepository

	cf        *server.ConfigFileRuntime
	callbacks []repository.TenantScopedCallback[*dbsqlc.ResolveWorkflowRunStatusRow]

	queueActionTenantCache *cache.Cache

	updateConcurrentFactor int
	maxHashFactor          int
}

func NewStepRunEngineRepository(shared *sharedRepository, cf *server.ConfigFileRuntime, rlCache *cache.Cache, queueCache *cache.Cache) *stepRunEngineRepository {
	return &stepRunEngineRepository{
		sharedRepository:       shared,
		cf:                     cf,
		updateConcurrentFactor: cf.UpdateConcurrentFactor,
		maxHashFactor:          cf.UpdateHashFactor,
		queueActionTenantCache: queueCache,
	}
}

func sizeOfUpdateData(item *updateStepRunQueueData) int {
	size := len(item.Output) + len(item.StepRunId)

	if item.Error != nil {
		errorLength := len(*item.Error)
		size += errorLength
	}

	return size
}

func (s *stepRunEngineRepository) RegisterWorkflowRunCompletedCallback(callback repository.TenantScopedCallback[*dbsqlc.ResolveWorkflowRunStatusRow]) {
	if s.callbacks == nil {
		s.callbacks = make([]repository.TenantScopedCallback[*dbsqlc.ResolveWorkflowRunStatusRow], 0)
	}

	s.callbacks = append(s.callbacks, callback)
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

	defer sqlchelpers.DeferRollback(ctx, s.l, tx.Rollback)

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

	defer sqlchelpers.DeferRollback(ctx, s.l, tx.Rollback)

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

func (s *stepRunEngineRepository) ListStepRunsToCancel(ctx context.Context, tenantId, jobRunId string) ([]*dbsqlc.GetStepRunForEngineRow, error) {
	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	srs, err := s.queries.ListStepRunsToCancel(ctx, tx, dbsqlc.ListStepRunsToCancelParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Jobrunid: sqlchelpers.UUIDFromStr(jobRunId),
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

func (s *stepRunEngineRepository) ListStepRunsToReassign(ctx context.Context, tenantId string) ([]string, []*dbsqlc.GetStepRunForEngineRow, error) {
	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, s.pool, s.l, 5000)

	if err != nil {
		return nil, nil, err
	}

	defer rollback()

	// get the step run and make sure it's still in pending
	results, err := s.queries.ListStepRunsToReassign(ctx, tx, dbsqlc.ListStepRunsToReassignParams{
		Maxinternalretrycount: s.cf.MaxInternalRetryCount,
		Tenantid:              pgTenantId,
	})

	if err != nil {
		return nil, nil, err
	}

	stepRunIds := make([]pgtype.UUID, 0, len(results))
	stepRunIdsStr := make([]string, 0, len(results))
	workerIds := make([]pgtype.UUID, 0, len(results))

	failedStepRunIds := make([]pgtype.UUID, 0, len(results))

	for _, sr := range results {
		if sr.Operation == "REASSIGNED" {
			stepRunIds = append(stepRunIds, sr.ID)
			stepRunIdsStr = append(stepRunIdsStr, sqlchelpers.UUIDToStr(sr.ID))
			workerIds = append(workerIds, sr.WorkerId)
		} else if sr.Operation == "FAILED" {
			failedStepRunIds = append(failedStepRunIds, sr.ID)
		}
	}

	failedStepRunResults, err := s.queries.GetStepRunForEngine(ctx, tx, dbsqlc.GetStepRunForEngineParams{
		Ids:      failedStepRunIds,
		TenantId: pgTenantId,
	})

	if err != nil {
		return nil, nil, err
	}

	err = commit(ctx)

	if err != nil {
		return nil, nil, err
	}

	for i, stepRunIdUUID := range stepRunIds {
		workerId := sqlchelpers.UUIDToStr(workerIds[i])
		message := "Worker has become inactive"
		reason := dbsqlc.StepRunEventReasonREASSIGNED
		severity := dbsqlc.StepRunEventSeverityCRITICAL
		timeSeen := time.Now().UTC()

		err := s.bulkEventBuffer.FireForget(tenantId, &repository.CreateStepRunEventOpts{
			StepRunId:     sqlchelpers.UUIDToStr(stepRunIdUUID),
			EventMessage:  &message,
			EventReason:   &reason,
			EventSeverity: &severity,
			Timestamp:     &timeSeen,
			EventData:     map[string]interface{}{"worker_id": workerId},
		})

		if err != nil {
			s.l.Err(err).Msg("could not buffer step run event")
		}
	}

	return stepRunIdsStr, failedStepRunResults, nil
}

func (s *stepRunEngineRepository) InternalRetryStepRuns(ctx context.Context, tenantId string, srIdsIn []string) ([]string, []*dbsqlc.GetStepRunForEngineRow, error) {
	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)
	stepRuns := make([]pgtype.UUID, 0, len(srIdsIn))

	for _, id := range srIdsIn {
		stepRuns = append(stepRuns, sqlchelpers.UUIDFromStr(id))
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, s.pool, s.l, 5000)

	if err != nil {
		return nil, nil, err
	}

	defer rollback()

	// get the step run and make sure it's still in pending
	results, err := s.queries.InternalRetryStepRuns(ctx, tx, dbsqlc.InternalRetryStepRunsParams{
		Maxinternalretrycount: s.cf.MaxInternalRetryCount,
		Tenantid:              pgTenantId,
		Steprunids:            stepRuns,
	})

	if err != nil {
		return nil, nil, err
	}

	stepRunIdsStr := make([]string, 0, len(results))

	failedStepRunIds := make([]pgtype.UUID, 0, len(results))

	for _, sr := range results {
		if sr.Operation == "FAILED" {
			failedStepRunIds = append(failedStepRunIds, sr.ID)
		}
	}

	failedStepRunResults, err := s.queries.GetStepRunForEngine(ctx, tx, dbsqlc.GetStepRunForEngineParams{
		Ids:      failedStepRunIds,
		TenantId: pgTenantId,
	})

	if err != nil {
		return nil, nil, err
	}

	err = commit(ctx)

	if err != nil {
		return nil, nil, err
	}

	return stepRunIdsStr, failedStepRunResults, nil
}

func (s *stepRunEngineRepository) ListStepRunsToTimeout(ctx context.Context, tenantId string) (bool, []*dbsqlc.GetStepRunForEngineRow, error) {
	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return false, nil, err
	}

	defer sqlchelpers.DeferRollback(ctx, s.l, tx.Rollback)

	limit := 100

	if s.cf.SingleQueueLimit != 0 {
		limit = s.cf.SingleQueueLimit
	}

	// get the step run and make sure it's still in pending
	stepRunIds, err := s.queries.PopTimeoutQueueItems(ctx, tx, dbsqlc.PopTimeoutQueueItemsParams{
		Tenantid: pgTenantId,
		Limit: pgtype.Int4{
			Int32: int32(limit), // nolint: gosec
			Valid: true,
		},
	})

	if err != nil {
		return false, nil, err
	}

	// mark the step runs as cancelling
	defer func() {
		_, err = s.queries.BulkMarkStepRunsAsCancelling(ctx, s.pool, stepRunIds)

		if err != nil {
			s.l.Err(err).Msg("could not bulk mark step runs as cancelling")
		}
	}()

	stepRuns, err := s.queries.GetStepRunForEngine(ctx, tx, dbsqlc.GetStepRunForEngineParams{
		Ids:      stepRunIds,
		TenantId: pgTenantId,
	})

	if err != nil {
		return false, nil, err
	}

	err = tx.Commit(ctx)

	if err != nil {
		return false, nil, err
	}

	return len(stepRunIds) == limit, stepRuns, nil
}

func (s *stepRunEngineRepository) ReleaseStepRunSemaphore(ctx context.Context, tenantId, stepRunId string, isUserTriggered bool) error {
	err := s.releaseWorkerSemaphoreSlot(ctx, tenantId, stepRunId)

	if err != nil {
		return fmt.Errorf("could not release worker semaphore slot for step run %s: %w", stepRunId, err)
	}

	if isUserTriggered {
		tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, s.pool, s.l, 5000)

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

		// Update the Step Run to release the semaphore
		err = s.queries.ManualReleaseSemaphore(ctx, tx, dbsqlc.ManualReleaseSemaphoreParams{
			Steprunid: stepRun.SRID,
			Tenantid:  stepRun.SRTenantId,
		})

		if err != nil {
			return fmt.Errorf("could not update step run semaphoreRelease: %w", err)
		}

		return commit(ctx)
	}

	return nil
}

func (s *stepRunEngineRepository) DeferredStepRunEvent(
	tenantId string,
	opts repository.CreateStepRunEventOpts,
) {
	if err := s.v.Validate(opts); err != nil {
		s.l.Err(err).Msg("could not validate step run event")
		return
	}

	s.deferredStepRunEvent(
		tenantId,
		opts,
	)
}

func (s *sharedRepository) deferredStepRunEvent(
	tenantId string,
	opts repository.CreateStepRunEventOpts,
) {
	// fire-and-forget for events
	err := s.bulkEventBuffer.FireForget(tenantId, &opts)

	if err != nil {
		s.l.Error().Err(err).Msg("could not buffer event")
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

func (s *stepRunEngineRepository) GetQueueCounts(ctx context.Context, tenantId string) (map[string]int, error) {
	counts, err := s.queries.GetQueuedCounts(ctx, s.pool, sqlchelpers.UUIDFromStr(tenantId))

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return map[string]int{}, nil
		}

		return nil, err
	}

	res := make(map[string]int)

	for _, count := range counts {
		res[count.Queue] = int(count.Count)
	}

	return res, nil
}

func (s *stepRunEngineRepository) ProcessStepRunUpdatesV2(ctx context.Context, qlp *zerolog.Logger, tenantId string) (repository.ProcessStepRunUpdatesResultV2, error) {
	ql := qlp.With().Str("tenant_id", tenantId).Logger()

	emptyRes := repository.ProcessStepRunUpdatesResultV2{
		Continue: false,
	}

	ctx, span := telemetry.NewSpan(ctx, "process-step-run-updates-database")
	defer span.End()

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	limit := 100 * s.updateConcurrentFactor

	if s.cf.SingleQueueLimit != 0 {
		limit = s.cf.SingleQueueLimit * s.updateConcurrentFactor
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, s.pool, s.l, 25000)

	if err != nil {
		return emptyRes, err
	}

	defer rollback()

	// list queues
	queueItems, err := s.queries.ListInternalQueueItems(ctx, tx, dbsqlc.ListInternalQueueItemsParams{
		Tenantid: pgTenantId,
		Queue:    dbsqlc.InternalQueueSTEPRUNUPDATEV2,
		Limit: pgtype.Int4{
			Int32: int32(limit), // nolint: gosec
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

	var completedWorkflowRuns []*dbsqlc.ResolveWorkflowRunStatusRow

	data = stableSortBatch(data)

	succeededStepRuns, completedWorkflowRunsV1, err := s.processStepRunUpdatesV2(ctx, &ql, tenantId, tx, data)

	if err != nil {
		return emptyRes, fmt.Errorf("could not process step run updates v1: %w", err)
	}

	completedWorkflowRuns = append(completedWorkflowRuns, completedWorkflowRunsV1...)

	qiIds := make([]int64, 0, len(data))

	for _, item := range queueItems {
		qiIds = append(qiIds, item.ID)
	}

	// update the processed semaphore queue items
	err = s.queries.MarkInternalQueueItemsProcessed(ctx, tx, qiIds)

	if err != nil {
		return emptyRes, fmt.Errorf("could not mark worker semaphore queue items processed: %w", err)
	}

	err = commit(ctx)

	if err != nil {
		return emptyRes, fmt.Errorf("could not commit transaction: %w", err)
	}

	for _, cb := range s.callbacks {
		for _, wr := range completedWorkflowRuns {
			wrCp := wr
			cb.Do(s.l, tenantId, wrCp)
		}
	}

	return repository.ProcessStepRunUpdatesResultV2{
		SucceededStepRuns:     succeededStepRuns,
		CompletedWorkflowRuns: completedWorkflowRuns,
		Continue:              len(queueItems) == limit,
	}, nil
}

func stableSortBatch(batch []updateStepRunQueueData) []updateStepRunQueueData {
	sort.SliceStable(batch, func(i, j int) bool {
		return batch[i].StepRunId < batch[j].StepRunId
	})

	return batch
}

func (s *stepRunEngineRepository) processStepRunUpdatesV2(
	ctx context.Context,
	qlp *zerolog.Logger,
	tenantId string,
	outerTx dbsqlc.DBTX,
	data []updateStepRunQueueData,
) (
	succeededStepRuns []*dbsqlc.GetStepRunForEngineRow,
	completedWorkflowRuns []*dbsqlc.ResolveWorkflowRunStatusRow,
	err error,
) {
	// startedAt := time.Now().UTC()
	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	batches := make([][]updateStepRunQueueData, s.updateConcurrentFactor)
	completedStepRunIds := make([]pgtype.UUID, 0, len(data))

	for _, item := range data {
		batch := item.Hash % s.updateConcurrentFactor

		batches[batch] = append(batches[batch], item)
	}

	var wrMu sync.Mutex
	var eg errgroup.Group

	for _, batch := range batches {
		if len(batch) == 0 {
			continue
		}

		fn := func() error {

			startParams := dbsqlc.BulkStartStepRunParams{}
			failParams := dbsqlc.BulkFailStepRunParams{}
			cancelParams := dbsqlc.BulkCancelStepRunParams{}
			finishParams := dbsqlc.BulkFinishStepRunParams{}
			backoffParams := []pgtype.UUID{}
			batchStepRunIds := []pgtype.UUID{}

			for _, item := range batch {
				stepRunId := sqlchelpers.UUIDFromStr(item.StepRunId)
				batchStepRunIds = append(batchStepRunIds, stepRunId)

				switch dbsqlc.StepRunStatus(*item.Status) {
				case dbsqlc.StepRunStatusRUNNING:
					startParams.Steprunids = append(startParams.Steprunids, stepRunId)
					startParams.Startedats = append(startParams.Startedats, sqlchelpers.TimestampFromTime(*item.StartedAt))
				case dbsqlc.StepRunStatusFAILED:
					failParams.Steprunids = append(failParams.Steprunids, stepRunId)
					failParams.Finishedats = append(failParams.Finishedats, sqlchelpers.TimestampFromTime(*item.FinishedAt))
					failParams.Errors = append(failParams.Errors, *item.Error)
				case dbsqlc.StepRunStatusCANCELLED:
					cancelParams.Steprunids = append(cancelParams.Steprunids, stepRunId)
					cancelParams.Cancelledats = append(cancelParams.Cancelledats, sqlchelpers.TimestampFromTime(*item.CancelledAt))
					cancelParams.Finishedats = append(cancelParams.Finishedats, sqlchelpers.TimestampFromTime(*item.CancelledAt))
					cancelParams.Cancelledreasons = append(cancelParams.Cancelledreasons, *item.CancelledReason)
				case dbsqlc.StepRunStatusSUCCEEDED:
					finishParams.Steprunids = append(finishParams.Steprunids, stepRunId)
					finishParams.Finishedats = append(finishParams.Finishedats, sqlchelpers.TimestampFromTime(*item.FinishedAt))
					finishParams.Outputs = append(finishParams.Outputs, item.Output)
				case dbsqlc.StepRunStatusBACKOFF:
					backoffParams = append(backoffParams, stepRunId)
				}
			}

			innerCompletedWorkflowRuns, err := s.bulkProcessStepRunUpdates(ctx, startParams, failParams, cancelParams, finishParams, backoffParams, batchStepRunIds, pgTenantId)

			if err != nil && strings.Contains(err.Error(), "SQLSTATE 22P02") {
				// attempt to validate json for outputs
				finishParams = dbsqlc.BulkFinishStepRunParams{}

				for _, item := range batch {
					stepRunId := sqlchelpers.UUIDFromStr(item.StepRunId)

					if item.Status == nil || dbsqlc.StepRunStatus(*item.Status) != dbsqlc.StepRunStatusSUCCEEDED {
						continue
					}

					validationErr := s.ValidateOutputs(ctx, item.Output)

					if validationErr != nil {
						// put into failed params
						failParams.Steprunids = append(failParams.Steprunids, stepRunId)
						failParams.Finishedats = append(failParams.Finishedats, sqlchelpers.TimestampFromTime(*item.FinishedAt))
						failParams.Errors = append(failParams.Errors, "OUTPUT_NOT_VALID_JSON")
					} else {
						finishParams.Steprunids = append(finishParams.Steprunids, stepRunId)
						finishParams.Finishedats = append(finishParams.Finishedats, sqlchelpers.TimestampFromTime(*item.FinishedAt))
						finishParams.Outputs = append(finishParams.Outputs, item.Output)
					}
				}

				innerCompletedWorkflowRuns, err = s.bulkProcessStepRunUpdates(ctx, startParams, failParams, cancelParams, finishParams, backoffParams, batchStepRunIds, pgTenantId)

				if err != nil {
					return fmt.Errorf("could not process step run updates: %w", err)
				}
			} else if err != nil {
				return fmt.Errorf("could not process step run updates: %w", err)
			}

			wrMu.Lock()
			completedWorkflowRuns = append(completedWorkflowRuns, innerCompletedWorkflowRuns...)
			completedStepRunIds = append(completedStepRunIds, finishParams.Steprunids...)
			wrMu.Unlock()

			return nil
		}

		eg.Go(fn)
	}

	err = eg.Wait()

	if err != nil {
		return nil, nil, fmt.Errorf("could not process step run updates v2: %w", err)
	}

	succeededStepRuns, err = s.queries.GetStepRunForEngine(ctx, outerTx, dbsqlc.GetStepRunForEngineParams{
		Ids:      completedStepRunIds,
		TenantId: pgTenantId,
	})

	if err != nil {
		return nil, nil, fmt.Errorf("could not get succeeded step runs: %w", err)
	}

	return succeededStepRuns, completedWorkflowRuns, nil
}

func (s *stepRunEngineRepository) bulkProcessStepRunUpdates(ctx context.Context,
	startParams dbsqlc.BulkStartStepRunParams,
	failParams dbsqlc.BulkFailStepRunParams,
	cancelParams dbsqlc.BulkCancelStepRunParams,
	finishParams dbsqlc.BulkFinishStepRunParams,
	backoffParams []pgtype.UUID,
	batchStepRunIds []pgtype.UUID,
	pgTenantId pgtype.UUID,
) (completedWorkflowRuns []*dbsqlc.ResolveWorkflowRunStatusRow, err error) {

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, s.pool, s.l, 25000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	if len(finishParams.Steprunids) > 0 {
		err = s.queries.BulkFinishStepRun(ctx, tx, finishParams)

		if err != nil {
			return nil, fmt.Errorf("could not finish step runs: %w", err)
		}
	}

	if len(startParams.Steprunids) > 0 {
		err = s.queries.BulkStartStepRun(ctx, tx, startParams)

		if err != nil {
			return nil, fmt.Errorf("could not start step runs: %w", err)
		}
	}

	if len(failParams.Steprunids) > 0 {
		err = s.queries.BulkFailStepRun(ctx, tx, failParams)

		if err != nil {
			return nil, fmt.Errorf("could not fail step runs: %w", err)
		}
	}

	if len(cancelParams.Steprunids) > 0 {

		err = s.queries.BulkCancelStepRun(ctx, tx, cancelParams)

		if err != nil {
			return nil, fmt.Errorf("could not cancel step runs: %w", err)
		}
	}

	if len(backoffParams) > 0 {
		err = s.queries.BulkBackoffStepRun(ctx, tx, backoffParams)

		if err != nil {
			return nil, fmt.Errorf("could not backoff step runs: %w", err)
		}
	}

	// update the job runs and workflow runs as well
	jobRunIds, err := s.queries.ResolveJobRunStatus(ctx, tx, batchStepRunIds)

	if err != nil {
		return nil, fmt.Errorf("could not resolve job run status: %w", err)
	}

	innerCompletedWorkflowRuns, err := s.queries.ResolveWorkflowRunStatus(ctx, tx, dbsqlc.ResolveWorkflowRunStatusParams{
		Jobrunids: jobRunIds,
		Tenantid:  pgTenantId,
	})

	if err != nil {
		return nil, fmt.Errorf("could not resolve workflow run status: %w", err)
	}

	err = commit(ctx)

	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	return innerCompletedWorkflowRuns, nil
}

func (s *stepRunEngineRepository) ValidateOutputs(ctx context.Context, output []byte) error {
	// new tx for validation
	validationTx, err := s.pool.Begin(ctx)

	if err != nil {
		return err
	}

	return s.queries.ValidatesAsJson(ctx, validationTx, output)
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
	var batchSize int64 = 10000
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
	var batchSize int64 = 10000
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

func (s *stepRunEngineRepository) CleanupRetryQueueItems(ctx context.Context, tenantId string) error {
	// setup telemetry
	ctx, span := telemetry.NewSpan(ctx, "cleanup-retry-queue-items-database")
	defer span.End()

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	// get the min and max queue items
	minMax, err := s.queries.GetMinMaxProcessedRetryQueueItems(ctx, s.pool, pgTenantId)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}

		return fmt.Errorf("could not get min max processed retry queue items: %w", err)
	}

	if minMax == nil {
		return nil
	}

	err = s.queries.CleanupRetryQueueItems(ctx, s.pool, dbsqlc.CleanupRetryQueueItemsParams{
		Minretryafter: minMax.MinRetryAfter,
		Maxretryafter: minMax.MaxRetryAfter,
		Tenantid:      pgTenantId,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}

		return fmt.Errorf("could not cleanup queue items: %w", err)
	}

	return nil
}

func (s *stepRunEngineRepository) StepRunStarted(ctx context.Context, tenantId, workflowRunId, stepRunId string, startedAt time.Time) error {
	ctx, span := telemetry.NewSpan(ctx, "step-run-started-db") // nolint: ineffassign
	defer span.End()

	running := string(dbsqlc.StepRunStatusRUNNING)

	data := &updateStepRunQueueData{
		Hash:      hashToBucket(sqlchelpers.UUIDFromStr(workflowRunId), s.maxHashFactor),
		StepRunId: stepRunId,
		TenantId:  tenantId,
		StartedAt: &startedAt,
		Status:    &running,
	}

	err := s.bulkStatusBuffer.FireForget(tenantId, data)

	if err != nil {
		return fmt.Errorf("could not buffer event: %w", err)
	}

	return nil
}

func (s *stepRunEngineRepository) StepRunAcked(ctx context.Context, tenantId, workflowRunId, stepRunId string, startedAt time.Time) error {
	_, span := telemetry.NewSpan(ctx, "step-run-acked-db")
	defer span.End()

	sev := dbsqlc.StepRunEventSeverityINFO
	ack := dbsqlc.StepRunEventReasonACKNOWLEDGED

	data := &repository.CreateStepRunEventOpts{
		StepRunId:     stepRunId,
		EventMessage:  repository.StringPtr("Step run acknowledged at " + startedAt.Format(time.RFC1123)),
		EventSeverity: &sev,
		EventReason:   &ack,
	}

	err := s.bulkEventBuffer.FireForget(tenantId, data)

	if err != nil {
		return fmt.Errorf("could not buffer step run acked event: %w", err)
	}

	return nil
}

func (s *stepRunEngineRepository) StepRunSucceeded(ctx context.Context, tenantId, workflowRunId, stepRunId string, finishedAt time.Time, output []byte) error {
	ctx, span := telemetry.NewSpan(ctx, "step-run-started-db")
	defer span.End()

	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return err
	}

	defer sqlchelpers.DeferRollback(ctx, s.l, tx.Rollback)

	// update the job run lookup data
	err = s.queries.UpdateJobRunLookupDataWithStepRun(ctx, tx, dbsqlc.UpdateJobRunLookupDataWithStepRunParams{
		Steprunid: sqlchelpers.UUIDFromStr(stepRunId),
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
		Jsondata:  output,
	})

	if err != nil && strings.Contains(err.Error(), "SQLSTATE 22P02") {
		s.l.Err(err).Msg("update job run lookup data with step run failed due to invalid json")

		validationErr := s.ValidateOutputs(ctx, output)

		if validationErr != nil {
			return s.StepRunFailed(ctx, tenantId, workflowRunId, stepRunId, finishedAt, "OUTPUT_NOT_VALID_JSON", 0)
		}
	} else if err != nil {
		s.l.Err(err).Msg("update job run lookup data with step run failed")

		return s.StepRunFailed(ctx, tenantId, workflowRunId, stepRunId, finishedAt, "FAILED_TO_WRITE_DATA", 0)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("could not commit transaction: %w", err)
	}

	finished := string(dbsqlc.StepRunStatusSUCCEEDED)

	data := &updateStepRunQueueData{
		Hash:       hashToBucket(sqlchelpers.UUIDFromStr(workflowRunId), s.maxHashFactor),
		StepRunId:  stepRunId,
		TenantId:   tenantId,
		FinishedAt: &finishedAt,
		Status:     &finished,
		Output:     output,
	}

	// we write to the buffer after updating the job run lookup data so we don't start a step run (which has multiple
	// parents) before the job run lookup data is updated
	_, err = s.bulkStatusBuffer.FireAndWait(ctx, tenantId, data)

	if err != nil {
		return fmt.Errorf("could not buffer step run succeeded: %w", err)
	}

	return nil
}

func (s *stepRunEngineRepository) StepRunCancelled(ctx context.Context, tenantId, workflowRunId, stepRunId string, cancelledAt time.Time, cancelledReason string, propagate bool) error {
	ctx, span := telemetry.NewSpan(ctx, "step-run-cancelled-db")
	defer span.End()

	// write a queue item to release the worker semaphore
	err := s.releaseWorkerSemaphoreSlot(ctx, tenantId, stepRunId)

	if err != nil {
		return fmt.Errorf("could not release worker semaphore queue items: %w", err)
	}

	cancelled := string(dbsqlc.StepRunStatusCANCELLED)

	data := &updateStepRunQueueData{
		Hash:            hashToBucket(sqlchelpers.UUIDFromStr(workflowRunId), s.maxHashFactor),
		StepRunId:       stepRunId,
		TenantId:        tenantId,
		CancelledAt:     &cancelledAt,
		CancelledReason: &cancelledReason,
		Status:          &cancelled,
	}

	err = s.bulkStatusBuffer.FireForget(tenantId, data)

	if err != nil {
		return fmt.Errorf("could not buffer step run cancelled: %w", err)
	}

	if propagate {
		laterStepRuns, err := s.queries.GetLaterStepRuns(ctx, s.pool, sqlchelpers.UUIDFromStr(stepRunId))

		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("could not get later step runs: %w", err)
		}

		var innerErr error

		for _, laterStepRun := range laterStepRuns {
			laterStepRunId := sqlchelpers.UUIDToStr(laterStepRun.ID)
			cancelled := string(dbsqlc.StepRunStatusCANCELLED)
			reason := "PREVIOUS_STEP_CANCELLED"

			err := s.bulkStatusBuffer.FireForget(tenantId, &updateStepRunQueueData{
				Hash:            hashToBucket(sqlchelpers.UUIDFromStr(workflowRunId), s.maxHashFactor),
				StepRunId:       laterStepRunId,
				TenantId:        tenantId,
				CancelledAt:     &cancelledAt,
				CancelledReason: &reason,
				Status:          &cancelled,
			})

			if err != nil {
				innerErr = multierror.Append(innerErr, fmt.Errorf("could not buffer later step run cancelled: %w", err))
			}
		}

		return innerErr
	}

	return nil
}

func (s *stepRunEngineRepository) StepRunFailed(ctx context.Context, tenantId, workflowRunId, stepRunId string, failedAt time.Time, errStr string, retryCount int) error {
	ctx, span := telemetry.NewSpan(ctx, "step-run-failed-db")
	defer span.End()

	// write a queue item to release the worker semaphore
	err := s.releaseWorkerSemaphoreSlot(ctx, tenantId, stepRunId)

	if err != nil {
		return fmt.Errorf("could not release worker semaphore queue items: %w", err)
	}

	failed := string(dbsqlc.StepRunStatusFAILED)

	data := &updateStepRunQueueData{
		Hash:       hashToBucket(sqlchelpers.UUIDFromStr(workflowRunId), s.maxHashFactor),
		StepRunId:  stepRunId,
		TenantId:   tenantId,
		RetryCount: retryCount,
		FinishedAt: &failedAt,
		Error:      &errStr,
		Status:     &failed,
	}

	err = s.bulkStatusBuffer.FireForget(tenantId, data)

	if err != nil {
		return fmt.Errorf("could not buffer step run failed: %w", err)
	}

	laterStepRuns, err := s.queries.GetLaterStepRuns(ctx, s.pool, sqlchelpers.UUIDFromStr(stepRunId))

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("could not get later step runs: %w", err)
	}

	var innerErr error

	for _, laterStepRun := range laterStepRuns {
		laterStepRunId := sqlchelpers.UUIDToStr(laterStepRun.ID)
		cancelled := string(dbsqlc.StepRunStatusCANCELLED)

		reason := "PREVIOUS_STEP_FAILED"

		if errStr == "TIMED_OUT" {
			reason = "PREVIOUS_STEP_TIMED_OUT"
		}

		err := s.bulkStatusBuffer.FireForget(tenantId, &updateStepRunQueueData{
			Hash:            hashToBucket(sqlchelpers.UUIDFromStr(workflowRunId), s.maxHashFactor),
			StepRunId:       laterStepRunId,
			TenantId:        tenantId,
			CancelledAt:     &failedAt,
			CancelledReason: &reason,
			Status:          &cancelled,
		})

		if err != nil {
			innerErr = multierror.Append(innerErr, fmt.Errorf("could not buffer later step run cancelled: %w", err))
		}
	}

	return innerErr
}

func (s *stepRunEngineRepository) StepRunRetryBackoff(ctx context.Context, tenantId, stepRunId string, retryAfter time.Time, retryCount int) error {
	ctx, span := telemetry.NewSpan(ctx, "step-run-retry-backoff-db")
	defer span.End()

	// TODO update backoff after picked up

	backoff := string(dbsqlc.StepRunStatusBACKOFF)

	err := s.bulkStatusBuffer.FireForget(tenantId, &updateStepRunQueueData{
		Hash:       hashToBucket(sqlchelpers.UUIDFromStr(stepRunId), s.maxHashFactor),
		StepRunId:  stepRunId,
		TenantId:   tenantId,
		RetryCount: retryCount,
		FinishedAt: nil,
		Error:      nil,
		Status:     &backoff,
	})

	if err != nil {
		return fmt.Errorf("could not buffer step run backoff: %w", err)
	}

	return s.queries.CreateRetryQueueItem(ctx, s.pool, dbsqlc.CreateRetryQueueItemParams{
		Steprunid:  sqlchelpers.UUIDFromStr(stepRunId),
		Tenantid:   sqlchelpers.UUIDFromStr(tenantId),
		Retryafter: sqlchelpers.TimestampFromTime(retryAfter.UTC()),
	})
}

func (s *stepRunEngineRepository) RetryStepRuns(ctx context.Context, tenantId string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "retry-step-runs-db")
	defer span.End()

	rows, err := s.queries.RetryStepRuns(ctx, s.pool, sqlchelpers.UUIDFromStr(tenantId))

	if err != nil {
		return false, fmt.Errorf("could not list retryable step runs: %w", err)
	}

	// for _, row := range rows {
	// 	status := string(dbsqlc.StepRunStatusPENDINGASSIGNMENT)

	// 	err := s.bulkStatusBuffer.FireForget(tenantId, &updateStepRunQueueData{
	// 		Hash:       hashToBucket(row.StepRunId, s.maxHashFactor),
	// 		StepRunId:  sqlchelpers.UUIDToStr(row.StepRunId),
	// 		TenantId:   tenantId,
	// 		RetryCount: 0,
	// 		FinishedAt: nil,
	// 		Error:      nil,
	// 		Status:     &status,
	// 	})

	// 	if err != nil {
	// 		return false, fmt.Errorf("could not buffer step run backoff: %w", err)
	// 	}
	// }

	return len(rows) == 1000, nil
}

func (s *stepRunEngineRepository) ReplayStepRun(ctx context.Context, tenantId, stepRunId string, input []byte) (*dbsqlc.GetStepRunForEngineRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "replay-step-run")
	defer span.End()

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, s.pool, s.l, 5000)

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

	defer s.deferredStepRunEvent(
		tenantId,
		repository.CreateStepRunEventOpts{
			StepRunId:     stepRunId,
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

	laterStepRuns, err := s.queries.GetLaterStepRuns(ctx, tx, sqlchelpers.UUIDFromStr(stepRunId))

	if err != nil {
		return nil, err
	}

	// archive each of the later step run results
	for _, laterStepRun := range laterStepRuns {
		laterStepRunId := sqlchelpers.UUIDToStr(laterStepRun.ID)

		err = archiveStepRunResult(ctx, s.queries, tx, tenantId, laterStepRunId, nil)

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

		defer s.deferredStepRunEvent(
			tenantId,
			repository.CreateStepRunEventOpts{
				StepRunId:     laterStepRunId,
				EventMessage:  repository.StringPtr(fmt.Sprintf("Parent step run %s was replayed, resetting step run result", innerStepRun.StepReadableId.String)),
				EventSeverity: &sev,
				EventReason:   &reason,
			},
		)
	}

	// reset all later step runs to a pending state
	_, err = s.queries.ReplayStepRunResetStepRuns(ctx, tx, dbsqlc.ReplayStepRunResetStepRunsParams{
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
	childStepRuns, err := s.queries.ListNonFinalChildStepRuns(ctx, s.pool, sqlchelpers.UUIDFromStr(stepRunId))

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("could not list non-final child step runs: %w", err)
	}

	if len(childStepRuns) > 0 {
		return repository.ErrPreflightReplayChildStepRunNotInFinalState
	}

	count, err := s.queries.HasActiveWorkersForActionId(ctx, s.pool, dbsqlc.HasActiveWorkersForActionIdParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Actionid: stepRun.ActionId,
	})

	if err != nil {
		return fmt.Errorf("could not count active workers for action id: %w", err)
	}

	if count == 0 {
		return repository.ErrNoWorkerAvailable
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

	defer sqlchelpers.DeferRollback(ctx, s.l, tx.Rollback)

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

	defer sqlchelpers.DeferRollback(ctx, s.l, tx.Rollback)

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

func (s *stepRunEngineRepository) doCachedUpsertOfQueue(ctx context.Context, tenantId string, innerStepRun *dbsqlc.GetStepRunForEngineRow) error {
	cacheKey := fmt.Sprintf("t-%s-q-%s", tenantId, innerStepRun.SRQueue)

	_, err := cache.MakeCacheable(s.queueActionTenantCache, cacheKey, func() (*bool, error) {
		tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, s.pool, s.l, 5000)

		if err != nil {
			return nil, err
		}

		defer rollback()

		err = s.queries.UpsertQueue(
			ctx,
			tx,
			dbsqlc.UpsertQueueParams{
				Name:     innerStepRun.ActionId,
				Tenantid: sqlchelpers.UUIDFromStr(tenantId),
			},
		)

		if err != nil {
			return nil, err
		}

		err = commit(ctx)

		if err != nil {
			return nil, err
		}

		res := true
		return &res, nil
	})

	return err
}

func (s *stepRunEngineRepository) QueueStepRun(ctx context.Context, tenantId, stepRunId string, opts *repository.QueueStepRunOpts) (*dbsqlc.GetStepRunForEngineRow, error) {
	ctx, span := telemetry.NewSpan(ctx, "queue-step-run-database")
	defer span.End()

	if err := s.v.Validate(opts); err != nil {
		return nil, err
	}

	priority := 1

	if len(opts.ExpressionEvals) > 0 {
		err := s.createExpressionEvals(ctx, s.pool, stepRunId, opts.ExpressionEvals)

		if err != nil {
			return nil, err
		}
	}

	innerStepRun, err := s.getStepRunForEngineTx(ctx, s.pool, tenantId, stepRunId)

	if err != nil {
		return nil, err
	}

	err = s.doCachedUpsertOfQueue(ctx, tenantId, innerStepRun)

	if err != nil {
		return nil, fmt.Errorf("could not upsert queue with actionId: %w", err)
	}

	// if this is an internal retry, and the step run is in a running or final state, this is a no-op. The internal retry
	// may have be delayed (for example, timing out when sending the action to the worker), while the system
	// may have already reassigned the step run to another.
	if opts.IsInternalRetry && (repository.IsFinalStepRunStatus(innerStepRun.SRStatus) || innerStepRun.SRStatus == dbsqlc.StepRunStatusRUNNING) {
		return nil, repository.ErrAlreadyRunning
	}

	// if this is not a retry, and the step run is already in a pending assignment state, this is a no-op
	if !opts.IsRetry && !opts.IsInternalRetry && innerStepRun.SRStatus == dbsqlc.StepRunStatusPENDINGASSIGNMENT {
		return nil, repository.ErrAlreadyQueued
	}

	if opts.IsRetry || opts.IsInternalRetry {
		// if this is a retry, write a queue item to release the worker semaphore
		//
		// FIXME: there is a race condition here where we can delete a worker semaphore slot that has already been reassigned,
		// but the step run was not in a RUNNING state. The fix for this would be to track an total retry count on the step run
		// and use this to identify semaphore slots, but this involves a big refactor of semaphore slots.
		err := s.releaseWorkerSemaphoreSlot(ctx, tenantId, stepRunId)

		if err != nil {
			return nil, fmt.Errorf("could not release worker semaphore queue items: %w", err)
		}

		// retries get highest priority to ensure that they're run immediately
		priority = 4
	}

	_, err = s.bulkQueuer.FireAndWait(ctx, tenantId, bulkQueueStepRunOpts{
		GetStepRunForEngineRow: innerStepRun,
		Priority:               priority,
		IsRetry:                opts.IsRetry,
		Input:                  opts.Input,
	})

	if err != nil {
		return nil, err
	}

	return innerStepRun, nil
}

func (s *stepRunEngineRepository) createExpressionEvals(ctx context.Context, dbtx dbsqlc.DBTX, stepRunId string, opts []repository.CreateExpressionEvalOpt) error {
	if len(opts) == 0 {
		return nil
	}

	pgStepRunId := sqlchelpers.UUIDFromStr(stepRunId)

	strParams := dbsqlc.CreateStepRunExpressionEvalStrsParams{
		Steprunid: pgStepRunId,
	}

	intParams := dbsqlc.CreateStepRunExpressionEvalIntsParams{
		Steprunid: pgStepRunId,
	}

	for _, opt := range opts {
		if opt.ValueStr != nil {
			strParams.Keys = append(strParams.Keys, opt.Key)
			strParams.Kinds = append(strParams.Kinds, string(opt.Kind))
			strParams.Valuesstr = append(strParams.Valuesstr, *opt.ValueStr)
		} else if opt.ValueInt != nil {
			intParams.Keys = append(intParams.Keys, opt.Key)
			intParams.Kinds = append(intParams.Kinds, string(opt.Kind))
			intParams.Valuesint = append(intParams.Valuesint, int32(*opt.ValueInt)) // nolint: gosec
		}
	}

	if len(strParams.Keys) > 0 {
		err := s.queries.CreateStepRunExpressionEvalStrs(ctx, dbtx, strParams)

		if err != nil {
			return fmt.Errorf("could not create step run expression strs: %w", err)
		}
	}

	if len(intParams.Keys) > 0 {
		err := s.queries.CreateStepRunExpressionEvalInts(ctx, dbtx, intParams)

		if err != nil {
			return fmt.Errorf("could not create step run expression ints: %w", err)
		}
	}

	return nil
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

func (s *stepRunEngineRepository) GetStepRunBulkDataForEngine(ctx context.Context, tenantId string, stepRunIds []string) ([]*dbsqlc.GetStepRunBulkDataForEngineRow, error) {
	ids := make([]pgtype.UUID, len(stepRunIds))

	for i, id := range stepRunIds {
		ids[i] = sqlchelpers.UUIDFromStr(id)
	}

	return s.queries.GetStepRunBulkDataForEngine(ctx, s.pool, dbsqlc.GetStepRunBulkDataForEngineParams{
		Ids:      ids,
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	})
}

func (s *sharedRepository) ListInitialStepRunsForJobRun(ctx context.Context, tenantId, jobRunId string) ([]*dbsqlc.GetStepRunForEngineRow, error) {
	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer sqlchelpers.DeferRollback(ctx, s.l, tx.Rollback)

	res, err := s.listInitialStepRunsForJobRunWithTx(ctx, tx, tenantId, jobRunId)

	if err != nil {
		return nil, err
	}

	err = tx.Commit(ctx)

	return res, err
}

func (s *sharedRepository) listInitialStepRunsForJobRunWithTx(ctx context.Context, tx dbsqlc.DBTX, tenantId, jobRunId string) ([]*dbsqlc.GetStepRunForEngineRow, error) {

	srs, err := s.queries.ListInitialStepRuns(ctx, tx, sqlchelpers.UUIDFromStr(jobRunId))

	if err != nil {
		return nil, fmt.Errorf("could not list initial step runs: %w", err)
	}

	res, err := s.queries.GetStepRunForEngine(ctx, tx, dbsqlc.GetStepRunForEngineParams{
		Ids:      srs,
		TenantId: sqlchelpers.UUIDFromStr(tenantId),
	})

	return res, err
}

func (s *stepRunEngineRepository) ListStartableStepRuns(ctx context.Context, tenantId, parentStepRunId string, singleParent bool) ([]*dbsqlc.GetStepRunForEngineRow, error) {
	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer sqlchelpers.DeferRollback(ctx, s.l, tx.Rollback)

	var srs []pgtype.UUID

	if singleParent {
		srs, err = s.queries.ListStartableStepRunsSingleParent(ctx, tx, sqlchelpers.UUIDFromStr(parentStepRunId))

		if err != nil {
			return nil, fmt.Errorf("could not list startable step runs: %w", err)
		}
	} else {
		srs, err = s.queries.ListStartableStepRunsManyParents(ctx, tx, sqlchelpers.UUIDFromStr(parentStepRunId))

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

func (s *stepRunEngineRepository) ArchiveStepRunResult(ctx context.Context, tenantId, stepRunId string, userErr *string) error {
	return archiveStepRunResult(ctx, s.queries, s.pool, tenantId, stepRunId, userErr)
}

func archiveStepRunResult(ctx context.Context, queries *dbsqlc.Queries, db dbsqlc.DBTX, tenantId, stepRunId string, userErr *string) error {
	params := dbsqlc.ArchiveStepRunResultFromStepRunParams{
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
		Steprunid: sqlchelpers.UUIDFromStr(stepRunId),
	}

	if userErr != nil {
		params.Error = sqlchelpers.TextFromStr(*userErr)
	}

	_, err := queries.ArchiveStepRunResultFromStepRun(ctx, db, params)

	return err
}

func (s *stepRunEngineRepository) RefreshTimeoutBy(ctx context.Context, tenantId, stepRunId string, opts repository.RefreshTimeoutBy) (pgtype.Timestamp, error) {
	stepRunUUID := sqlchelpers.UUIDFromStr(stepRunId)
	tenantUUID := sqlchelpers.UUIDFromStr(tenantId)

	incrementTimeoutBy := opts.IncrementTimeoutBy

	err := s.v.Validate(opts)

	if err != nil {
		return pgtype.Timestamp{}, err
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, s.pool, s.l, 5000)

	if err != nil {
		return pgtype.Timestamp{}, err
	}

	defer rollback()

	res, err := s.queries.RefreshTimeoutBy(ctx, tx, dbsqlc.RefreshTimeoutByParams{
		Steprunid:          stepRunUUID,
		Tenantid:           tenantUUID,
		IncrementTimeoutBy: sqlchelpers.TextFromStr(incrementTimeoutBy),
	})

	if err != nil {
		return pgtype.Timestamp{}, err
	}

	if err := commit(ctx); err != nil {
		return pgtype.Timestamp{}, err
	}

	sev := dbsqlc.StepRunEventSeverityINFO
	reason := dbsqlc.StepRunEventReasonTIMEOUTREFRESHED

	defer s.deferredStepRunEvent(
		tenantId,
		repository.CreateStepRunEventOpts{
			StepRunId:     stepRunId,
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

func (s *stepRunEngineRepository) releaseWorkerSemaphoreSlot(ctx context.Context, tenantId, stepRunId string) error {
	_, err := s.bulkSemaphoreReleaser.FireAndWait(ctx, tenantId, semaphoreReleaseOpts{
		StepRunId: sqlchelpers.UUIDFromStr(stepRunId),
		TenantId:  sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return fmt.Errorf("could not buffer semaphore release: %w", err)
	}

	return nil
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
	StepRunId  string `json:"step_run_id"`
	Hash       int    `json:"hash"`
	TenantId   string `json:"tenant_id"`
	RetryCount int    `json:"retry_count,omitempty"`

	// Event *repository.CreateStepRunEventOpts `json:"event,omitempty"`

	StartedAt       *time.Time `json:"started_at,omitempty"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
	CancelledAt     *time.Time `json:"cancelled_at,omitempty"`
	Output          []byte     `json:"output"`
	CancelledReason *string    `json:"cancelled_reason,omitempty"`
	Error           *string    `json:"error,omitempty"`
	Status          *string    `json:"status,omitempty"`
}

func bulkInsertInternalQueueItem(
	ctx context.Context,
	dbtx dbsqlc.DBTX,
	queries *dbsqlc.Queries,
	tenantIds []pgtype.UUID,
	queues []dbsqlc.InternalQueue,
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

	insertQueues := make([]string, len(queues))

	for i, q := range queues {
		insertQueues[i] = string(q)
	}

	err := queries.CreateInternalQueueItemsBulk(ctx, dbtx, dbsqlc.CreateInternalQueueItemsBulkParams{
		Tenantids: tenantIds,
		Queues:    insertQueues,
		Datas:     insertData,
	})

	if err != nil {
		return err
	}

	return nil
}

func hashToBucket(id pgtype.UUID, buckets int) int {
	hasher := fnv.New32a()
	hasher.Write(id.Bytes[:])
	return int(hasher.Sum32()) % buckets
}
