package prisma

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
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
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/internal/services/shared/defaults"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/buffer"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
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
	pool                     *pgxpool.Pool
	v                        validator.Validator
	l                        *zerolog.Logger
	queries                  *dbsqlc.Queries
	cf                       *server.ConfigFileRuntime
	cachedMinQueuedIds       sync.Map
	cachedStepIdHasRateLimit *cache.Cache
	callbacks                []repository.TenantScopedCallback[*dbsqlc.ResolveWorkflowRunStatusRow]

	bulkStatusBuffer       *buffer.TenantBufferManager[*updateStepRunQueueData, pgtype.UUID]
	bulkEventBuffer        *buffer.BulkEventWriter
	bulkSemaphoreReleaser  *buffer.BulkSemaphoreReleaser
	bulkQueuer             *buffer.BulkStepRunQueuer
	queueActionTenantCache *cache.Cache

	updateConcurrentFactor int
	maxHashFactor          int
}

func (s *stepRunEngineRepository) cleanup() error {
	if err := s.bulkStatusBuffer.Cleanup(); err != nil {
		return err
	}

	if err := s.bulkSemaphoreReleaser.Cleanup(); err != nil {
		return err
	}

	if err := s.bulkQueuer.Cleanup(); err != nil {
		return err
	}

	return s.bulkEventBuffer.Cleanup()
}

func NewStepRunEngineRepository(pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger, cf *server.ConfigFileRuntime, rlCache *cache.Cache, queueCache *cache.Cache) (*stepRunEngineRepository, func() error, error) {
	queries := dbsqlc.New()

	eventBuffer, err := buffer.NewBulkEventWriter(pool, v, l, cf.EventBuffer)

	if err != nil {
		return nil, nil, err
	}

	semReleaser, err := buffer.NewBulkSemaphoreReleaser(pool, v, l, cf.ReleaseSemaphoreBuffer)

	if err != nil {
		return nil, nil, err
	}

	bulkQueuer, err := buffer.NewBulkStepRunQueuer(pool, v, l, cf.QueueStepRunBuffer)

	if err != nil {
		return nil, nil, err
	}

	s := &stepRunEngineRepository{
		pool:                     pool,
		v:                        v,
		l:                        l,
		queries:                  queries,
		cf:                       cf,
		cachedStepIdHasRateLimit: rlCache,
		updateConcurrentFactor:   cf.UpdateConcurrentFactor,
		maxHashFactor:            cf.UpdateHashFactor,
		bulkEventBuffer:          eventBuffer,
		bulkSemaphoreReleaser:    semReleaser,
		bulkQueuer:               bulkQueuer,
		queueActionTenantCache:   queueCache,
	}

	err = s.startBuffers()

	if err != nil {
		l.Err(err).Msg("could not start buffers")
		return nil, nil, err
	}

	return s, s.cleanup, nil
}

func sizeOfUpdateData(item *updateStepRunQueueData) int {
	size := len(item.Output) + len(item.StepRunId)

	if item.Error != nil {
		errorLength := len(*item.Error)
		size += errorLength
	}

	return size
}

func (s *stepRunEngineRepository) startBuffers() error {
	statusBufOpts := buffer.TenantBufManagerOpts[*updateStepRunQueueData, pgtype.UUID]{
		Name:       "update_step_run_status",
		OutputFunc: s.bulkUpdateStepRunStatuses,
		SizeFunc:   sizeOfUpdateData,
		L:          s.l,
		V:          s.v,
	}

	var err error
	s.bulkStatusBuffer, err = buffer.NewTenantBufManager(statusBufOpts)

	if err != nil {
		return err
	}

	return err
}

func (s *stepRunEngineRepository) bulkUpdateStepRunStatuses(ctx context.Context, opts []*updateStepRunQueueData) ([]pgtype.UUID, error) {
	stepRunIds := make([]pgtype.UUID, 0, len(opts))

	eventTimeSeen := make([]time.Time, 0, len(opts))
	eventReasons := make([]dbsqlc.StepRunEventReason, 0, len(opts))
	eventStepRunIds := make([]pgtype.UUID, 0, len(opts))
	eventTenantIds := make([]string, 0, len(opts))
	eventSeverities := make([]dbsqlc.StepRunEventSeverity, 0, len(opts))
	eventMessages := make([]string, 0, len(opts))
	eventData := make([]map[string]interface{}, 0, len(opts))

	for _, item := range opts {
		stepRunId := sqlchelpers.UUIDFromStr(item.StepRunId)
		stepRunIds = append(stepRunIds, stepRunId)

		if item.Status == nil {
			continue
		}

		switch dbsqlc.StepRunStatus(*item.Status) {
		case dbsqlc.StepRunStatusRUNNING:
			eventStepRunIds = append(eventStepRunIds, stepRunId)
			eventTenantIds = append(eventTenantIds, item.TenantId)
			eventTimeSeen = append(eventTimeSeen, *item.StartedAt)
			eventReasons = append(eventReasons, dbsqlc.StepRunEventReasonSTARTED)
			eventSeverities = append(eventSeverities, dbsqlc.StepRunEventSeverityINFO)
			eventMessages = append(eventMessages, fmt.Sprintf("Step run started at %s", item.StartedAt.Format(time.RFC1123)))
			eventData = append(eventData, map[string]interface{}{})
		case dbsqlc.StepRunStatusFAILED:
			eventTimeSeen = append(eventTimeSeen, *item.FinishedAt)

			eventStepRunIds = append(eventStepRunIds, stepRunId)
			eventTenantIds = append(eventTenantIds, item.TenantId)
			eventMessage := fmt.Sprintf("Step run failed on %s", item.FinishedAt.Format(time.RFC1123))
			eventReason := dbsqlc.StepRunEventReasonFAILED

			if item.Error != nil && *item.Error == "TIMED_OUT" {
				eventReason = dbsqlc.StepRunEventReasonTIMEDOUT
				eventMessage = "Step exceeded timeout duration"
			}

			eventReasons = append(eventReasons, eventReason)
			eventSeverities = append(eventSeverities, dbsqlc.StepRunEventSeverityCRITICAL)
			eventMessages = append(eventMessages, eventMessage)
			eventData = append(eventData, map[string]interface{}{
				"retry_count": item.RetryCount,
			})
		case dbsqlc.StepRunStatusCANCELLED:
			eventTimeSeen = append(eventTimeSeen, *item.CancelledAt)
			eventStepRunIds = append(eventStepRunIds, stepRunId)
			eventTenantIds = append(eventTenantIds, item.TenantId)
			eventReasons = append(eventReasons, dbsqlc.StepRunEventReasonCANCELLED)
			eventSeverities = append(eventSeverities, dbsqlc.StepRunEventSeverityWARNING)
			eventMessages = append(eventMessages, fmt.Sprintf("Step run was cancelled on %s for the following reason: %s", item.CancelledAt.Format(time.RFC1123), *item.CancelledReason))
			eventData = append(eventData, map[string]interface{}{})
		case dbsqlc.StepRunStatusSUCCEEDED:
			eventTimeSeen = append(eventTimeSeen, *item.FinishedAt)
			eventStepRunIds = append(eventStepRunIds, stepRunId)
			eventTenantIds = append(eventTenantIds, item.TenantId)
			eventReasons = append(eventReasons, dbsqlc.StepRunEventReasonFINISHED)
			eventSeverities = append(eventSeverities, dbsqlc.StepRunEventSeverityINFO)
			eventMessages = append(eventMessages, fmt.Sprintf("Step run finished at %s", item.FinishedAt.Format(time.RFC1123)))
			eventData = append(eventData, map[string]interface{}{})
		}
	}

	eg := errgroup.Group{}

	if len(opts) > 0 {
		eg.Go(func() error {
			insertInternalQITenantIds := make([]pgtype.UUID, 0, len(opts))
			insertInternalQIQueues := make([]dbsqlc.InternalQueue, 0, len(opts))
			insertInternalQIData := make([]any, 0, len(opts))

			for _, item := range opts {
				if item.Status == nil {
					continue
				}

				itemCp := item

				insertInternalQITenantIds = append(insertInternalQITenantIds, sqlchelpers.UUIDFromStr(itemCp.TenantId))
				insertInternalQIQueues = append(insertInternalQIQueues, dbsqlc.InternalQueueSTEPRUNUPDATEV2)
				insertInternalQIData = append(insertInternalQIData, itemCp)
			}

			err := bulkInsertInternalQueueItem(
				ctx,
				s.pool,
				s.queries,
				insertInternalQITenantIds,
				insertInternalQIQueues,
				insertInternalQIData,
			)

			if err != nil {
				return err
			}

			return nil
		})
	}

	if len(eventStepRunIds) > 0 {
		for i, stepRunId := range eventStepRunIds {
			_, err := s.bulkEventBuffer.BuffItem(eventTenantIds[i], &repository.CreateStepRunEventOpts{
				StepRunId:     sqlchelpers.UUIDToStr(stepRunId),
				EventMessage:  &eventMessages[i],
				EventReason:   &eventReasons[i],
				EventSeverity: &eventSeverities[i],
				Timestamp:     &eventTimeSeen[i],
				EventData:     eventData[i],
			})

			if err != nil {
				s.l.Err(err).Msg("could not buffer step run event")
			}
		}
	}

	err := eg.Wait()

	if err != nil {
		return nil, err
	}

	return stepRunIds, nil
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
	results, err := s.queries.ListStepRunsToReassign(ctx, tx, pgTenantId)

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

		_, err := s.bulkEventBuffer.BuffItem(tenantId, &repository.CreateStepRunEventOpts{
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
			Int32: int32(limit),
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

func (s *stepRunEngineRepository) deferredStepRunEvent(
	tenantId string,
	opts repository.CreateStepRunEventOpts,
) {
	// fire-and-forget for events
	_, err := s.bulkEventBuffer.BuffItem(tenantId, &opts)

	if err != nil {
		s.l.Error().Err(err).Msg("could not buffer event")
	}
}

func (s *stepRunEngineRepository) bulkStepRunsAssigned(
	tenantId string,
	assignedAt time.Time,
	stepRunIds []pgtype.UUID,
	workerIds []pgtype.UUID,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	workerIdToStepRunIds := make(map[string][]string)

	for i := range stepRunIds {
		workerId := sqlchelpers.UUIDToStr(workerIds[i])

		if _, ok := workerIdToStepRunIds[workerId]; !ok {
			workerIdToStepRunIds[workerId] = make([]string, 0)
		}

		workerIdToStepRunIds[workerId] = append(workerIdToStepRunIds[workerId], sqlchelpers.UUIDToStr(stepRunIds[i]))
		message := fmt.Sprintf("Assigned to worker %s", workerId)
		timeSeen := assignedAt
		reasons := dbsqlc.StepRunEventReasonASSIGNED
		severity := dbsqlc.StepRunEventSeverityINFO
		data := map[string]interface{}{"worker_id": workerId}

		_, err := s.bulkEventBuffer.BuffItem(tenantId, &repository.CreateStepRunEventOpts{
			StepRunId:     sqlchelpers.UUIDToStr(stepRunIds[i]),
			EventMessage:  &message,
			EventReason:   &reasons,
			EventSeverity: &severity,
			Timestamp:     &timeSeen,
			EventData:     data,
		})

		if err != nil {
			s.l.Err(err).Msg("could not buffer step run event")
		}
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
}

func (s *stepRunEngineRepository) bulkStepRunsUnassigned(
	tenantId string,
	stepRunIds []pgtype.UUID,
) {
	for _, stepRunId := range stepRunIds {
		message := "No worker available"
		timeSeen := time.Now().UTC()
		severity := dbsqlc.StepRunEventSeverityWARNING
		reason := dbsqlc.StepRunEventReasonREQUEUEDNOWORKER
		data := map[string]interface{}{}

		_, err := s.bulkEventBuffer.BuffItem(tenantId, &repository.CreateStepRunEventOpts{
			StepRunId:     sqlchelpers.UUIDToStr(stepRunId),
			EventMessage:  &message,
			EventReason:   &reason,
			EventSeverity: &severity,
			Timestamp:     &timeSeen,
			EventData:     data,
		})

		if err != nil {
			s.l.Err(err).Msg("could not buffer step run event")
		}
	}
}

func (s *stepRunEngineRepository) bulkStepRunsRateLimited(
	tenantId string,
	rateLimits scheduling.RateLimitedResult,
) {
	stepRunIds := rateLimits.StepRuns

	for i, stepRunId := range stepRunIds {
		message := fmt.Sprintf("Rate limit exceeded for key %s, attempting to consume %d units", rateLimits.Keys[i], rateLimits.Units[i])
		reason := dbsqlc.StepRunEventReasonREQUEUEDRATELIMIT
		severity := dbsqlc.StepRunEventSeverityWARNING
		timeSeen := time.Now().UTC()
		data := map[string]interface{}{
			"rate_limit_key": rateLimits.Keys[i],
		}

		_, err := s.bulkEventBuffer.BuffItem(tenantId, &repository.CreateStepRunEventOpts{
			StepRunId:     sqlchelpers.UUIDToStr(stepRunId),
			EventMessage:  &message,
			EventReason:   &reason,
			EventSeverity: &severity,
			Timestamp:     &timeSeen,
			EventData:     data,
		})

		if err != nil {
			s.l.Err(err).Msg("could not buffer step run event")
		}
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

	defer sqlchelpers.DeferRollback(ctx, s.l, tx.Rollback)

	durationPrepareTx := time.Since(startedAt)

	startListQueues := time.Now().UTC()

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

	durationListQueues := time.Since(startListQueues)
	startedListQueueItems := time.Now().UTC()

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

	durationListQueueItems := time.Since(startedListQueueItems)
	startRateLimits := time.Now().UTC()

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
	rateLimits, currRateLimitValues, err := s.getStepRunRateLimits(ctx, tx, tenantId, queueItems)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return emptyRes, fmt.Errorf("could not list rate limits for tenant: %w", err)
	}

	durationListRateLimits := time.Since(startRateLimits)
	startGetWorkerCounts := time.Now()

	// list workers to assign
	workers, err := s.queries.GetWorkerDispatcherActions(ctx, tx, dbsqlc.GetWorkerDispatcherActionsParams{
		Tenantid:  pgTenantId,
		Actionids: uniqueActionsArr,
	})

	if err != nil {
		return emptyRes, fmt.Errorf("could not get worker dispatcher actions: %w", err)
	}

	workerIds := make([]pgtype.UUID, 0, len(workers))

	for _, worker := range workers {
		workerIds = append(workerIds, worker.ID)
	}

	availableSlots, err := s.queries.ListAvailableSlotsForWorkers(ctx, tx, dbsqlc.ListAvailableSlotsForWorkersParams{
		Tenantid:  pgTenantId,
		Workerids: workerIds,
	})

	if err != nil {
		return emptyRes, fmt.Errorf("could not list available slots for workers: %w", err)
	}

	workersToCounts := make(map[string]int)

	for _, worker := range availableSlots {
		workersToCounts[sqlchelpers.UUIDToStr(worker.ID)] = int(worker.AvailableSlots)
	}

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
	startGetLabels := time.Now().UTC()

	// GET UNIQUE STEP IDS
	stepIdSet := UniqueSet(queueItems, func(x *scheduling.QueueItemWithOrder) string {
		return sqlchelpers.UUIDToStr(x.StepId)
	})

	desiredLabels := make(map[string][]*dbsqlc.GetDesiredLabelsRow)
	hasDesired := false

	// GET DESIRED LABELS
	// OPTIMIZATION: CACHEABLE
	stepIds := make([]pgtype.UUID, 0, len(stepIdSet))
	for stepId := range stepIdSet {
		stepIds = append(stepIds, sqlchelpers.UUIDFromStr(stepId))
	}

	labels, err := s.queries.GetDesiredLabels(ctx, tx, stepIds)

	if err != nil {
		return emptyRes, fmt.Errorf("could not get desired labels: %w", err)
	}

	for _, label := range labels {
		stepId := sqlchelpers.UUIDToStr(label.StepId)
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

	durationGetLabels := time.Since(startGetLabels)
	startScheduling := time.Now().UTC()

	plan, err := scheduling.GeneratePlan(
		ctx,
		slots,
		uniqueActionsArr,
		queueItems,
		rateLimits,
		currRateLimitValues,
		workerLabels,
		desiredLabels,
	)

	if err != nil {
		return emptyRes, fmt.Errorf("could not generate scheduling: %w", err)
	}

	durationScheduling := time.Since(startScheduling)
	startUpdateRateLimits := time.Now()

	// save rate limits as a subtransaction, but don't throw an error if it fails
	func() {
		updateKeys := []string{}
		updateUnits := []int32{}
		didConsume := false

		for key, value := range plan.RateLimitUnitsConsumed {
			if value == 0 {
				continue
			}

			didConsume = true
			updateKeys = append(updateKeys, key)
			updateUnits = append(updateUnits, value)
		}

		if !didConsume {
			return
		}

		subtx, err := tx.Begin(ctx)

		if err != nil {
			s.l.Err(err).Msg("could not start subtransaction")
			return
		}

		defer sqlchelpers.DeferRollback(ctx, s.l, subtx.Rollback)

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

	durationUpdateRateLimits := time.Since(startUpdateRateLimits)
	startAssignTime := time.Now()

	numAssigns := make(map[string]int)

	for _, workerId := range plan.WorkerIds {
		numAssigns[sqlchelpers.UUIDToStr(workerId)]++
	}

	_, err = s.queries.UpdateStepRunsToAssigned(ctx, tx, dbsqlc.UpdateStepRunsToAssignedParams{
		Steprunids:      plan.StepRunIds,
		Workerids:       plan.WorkerIds,
		Stepruntimeouts: plan.StepRunTimeouts,
		Tenantid:        pgTenantId,
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

	defer s.bulkStepRunsAssigned(tenantId, time.Now().UTC(), plan.StepRunIds, plan.WorkerIds)
	defer s.bulkStepRunsUnassigned(tenantId, plan.UnassignedStepRunIds)
	defer s.bulkStepRunsRateLimited(tenantId, plan.RateLimitedStepRuns)

	// update the cache with the min queued id
	for name, qiId := range plan.MinQueuedIds {
		s.cachedMinQueuedIds.Store(getCacheName(tenantId, name), qiId)
	}

	timedOutStepRunsStr := make([]string, len(plan.TimedOutStepRuns))

	for i, id := range plan.TimedOutStepRuns {
		timedOutStepRunsStr[i] = sqlchelpers.UUIDToStr(id)
	}

	defer printQueueDebugInfo(
		ql,
		tenantId,
		queues,
		queueItems,
		duplicates,
		finalized,
		plan,
		slots,
		startedAt,
		durationPrepareTx,
		durationListQueues,
		durationListQueueItems,
		durationListRateLimits,
		finishedGetWorkerCounts,
		durationGetLabels,
		durationScheduling,
		durationUpdateRateLimits,
		finishedAssignTime,
		finishQueueTime,
	)

	return repository.QueueStepRunsResult{
		Queued:             plan.QueuedStepRuns,
		SchedulingTimedOut: timedOutStepRunsStr,
		Continue:           plan.ShouldContinue,
	}, nil
}

func (s *stepRunEngineRepository) getStepRunRateLimits(ctx context.Context, dbtx dbsqlc.DBTX, tenantId string, queueItems []*scheduling.QueueItemWithOrder) (map[string]map[string]int32, map[string]*dbsqlc.ListRateLimitsForTenantWithMutateRow, error) {
	stepRunIds := make([]pgtype.UUID, 0, len(queueItems))
	stepIds := make([]pgtype.UUID, 0, len(queueItems))
	stepsWithRateLimits := make(map[string]bool)

	for _, item := range queueItems {
		stepRunIds = append(stepRunIds, item.StepRunId)
		stepIds = append(stepIds, item.StepId)
	}

	stepIdToStepRuns := make(map[string][]string)
	stepRunIdToStepId := make(map[string]string)

	for i, stepRunId := range stepRunIds {
		stepId := sqlchelpers.UUIDToStr(stepIds[i])
		stepRunIdStr := sqlchelpers.UUIDToStr(stepRunId)

		if _, ok := stepIdToStepRuns[stepId]; !ok {
			stepIdToStepRuns[stepId] = make([]string, 0)
		}

		stepIdToStepRuns[stepId] = append(stepIdToStepRuns[stepId], stepRunIdStr)
		stepRunIdToStepId[stepRunIdStr] = stepId
	}

	// check if we have any rate limits for these step ids
	skipRateLimiting := true

	for stepIdStr := range stepIdToStepRuns {
		if hasRateLimit, ok := s.cachedStepIdHasRateLimit.Get(stepIdStr); !ok || hasRateLimit.(bool) {
			skipRateLimiting = false
			break
		}
	}

	if skipRateLimiting {
		return nil, nil, nil
	}

	// get all step run expression evals which correspond to rate limits, grouped by step run id
	expressionEvals, err := s.queries.ListStepRunExpressionEvals(ctx, dbtx, stepRunIds)

	if err != nil {
		return nil, nil, err
	}

	stepRunAndGlobalKeyToKey := make(map[string]string)
	stepRunToKeys := make(map[string][]string)

	for _, eval := range expressionEvals {
		stepRunId := sqlchelpers.UUIDToStr(eval.StepRunId)
		globalKey := eval.Key

		// Only append if this is a key expression. Note that we have a uniqueness constraint on
		// the stepRunId, kind, and key, so we will not insert duplicate values into the array.
		if eval.Kind == dbsqlc.StepExpressionKindDYNAMICRATELIMITKEY {
			stepsWithRateLimits[stepRunIdToStepId[stepRunId]] = true

			k := eval.ValueStr.String

			if _, ok := stepRunToKeys[stepRunId]; !ok {
				stepRunToKeys[stepRunId] = make([]string, 0)
			}

			stepRunToKeys[stepRunId] = append(stepRunToKeys[stepRunId], k)

			stepRunAndGlobalKey := fmt.Sprintf("%s-%s", stepRunId, globalKey)

			stepRunAndGlobalKeyToKey[stepRunAndGlobalKey] = k
		}
	}

	rateLimitKeyToEvals := make(map[string][]*dbsqlc.StepRunExpressionEval)

	for _, eval := range expressionEvals {
		k := stepRunAndGlobalKeyToKey[fmt.Sprintf("%s-%s", sqlchelpers.UUIDToStr(eval.StepRunId), eval.Key)]

		if _, ok := rateLimitKeyToEvals[k]; !ok {
			rateLimitKeyToEvals[k] = make([]*dbsqlc.StepRunExpressionEval, 0)
		}

		rateLimitKeyToEvals[k] = append(rateLimitKeyToEvals[k], eval)
	}

	upsertRateLimitBulkParams := dbsqlc.UpsertRateLimitsBulkParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	}

	stepRunToKeyToUnits := make(map[string]map[string]int32)

	for key, evals := range rateLimitKeyToEvals {
		var duration string
		var limitValue int
		var skip bool

		for _, eval := range evals {
			// add to stepRunToKeyToUnits
			stepRunId := sqlchelpers.UUIDToStr(eval.StepRunId)

			// throw an error if there are multiple rate limits with the same keys, but different limit values or durations
			if eval.Kind == dbsqlc.StepExpressionKindDYNAMICRATELIMITWINDOW {
				if duration == "" {
					duration = eval.ValueStr.String
				} else if duration != eval.ValueStr.String {
					largerDuration, err := getLargerDuration(duration, eval.ValueStr.String)

					if err != nil {
						skip = true
						break
					}

					message := fmt.Sprintf("Multiple rate limits with key %s have different durations: %s vs %s. Using longer window %s.", key, duration, eval.ValueStr.String, largerDuration)
					timeSeen := time.Now().UTC()
					reason := dbsqlc.StepRunEventReasonRATELIMITERROR
					severity := dbsqlc.StepRunEventSeverityWARNING
					data := map[string]interface{}{}

					_, buffErr := s.bulkEventBuffer.BuffItem(tenantId, &repository.CreateStepRunEventOpts{
						StepRunId:     sqlchelpers.UUIDToStr(eval.StepRunId),
						EventMessage:  &message,
						EventReason:   &reason,
						EventSeverity: &severity,
						Timestamp:     &timeSeen,
						EventData:     data,
					})

					if buffErr != nil {
						s.l.Err(buffErr).Msg("could not buffer step run event")
					}

					duration = largerDuration
				}
			}

			if eval.Kind == dbsqlc.StepExpressionKindDYNAMICRATELIMITVALUE {
				if limitValue == 0 {
					limitValue = int(eval.ValueInt.Int32)
				} else if limitValue != int(eval.ValueInt.Int32) {
					message := fmt.Sprintf("Multiple rate limits with key %s have different limit values: %d vs %d. Using lower value %d.", key, limitValue, eval.ValueInt.Int32, min(limitValue, int(eval.ValueInt.Int32)))
					timeSeen := time.Now().UTC()
					reason := dbsqlc.StepRunEventReasonRATELIMITERROR
					severity := dbsqlc.StepRunEventSeverityWARNING
					data := map[string]interface{}{}

					_, buffErr := s.bulkEventBuffer.BuffItem(tenantId, &repository.CreateStepRunEventOpts{
						StepRunId:     sqlchelpers.UUIDToStr(eval.StepRunId),
						EventMessage:  &message,
						EventReason:   &reason,
						EventSeverity: &severity,
						Timestamp:     &timeSeen,
						EventData:     data,
					})

					if buffErr != nil {
						s.l.Err(buffErr).Msg("could not buffer step run event")
					}

					limitValue = min(limitValue, int(eval.ValueInt.Int32))
				}
			}

			if eval.Kind == dbsqlc.StepExpressionKindDYNAMICRATELIMITUNITS {
				if _, ok := stepRunToKeyToUnits[stepRunId]; !ok {
					stepRunToKeyToUnits[stepRunId] = make(map[string]int32)
				}

				stepRunToKeyToUnits[stepRunId][key] = eval.ValueInt.Int32
			}
		}

		if skip {
			continue
		}

		upsertRateLimitBulkParams.Keys = append(upsertRateLimitBulkParams.Keys, key)
		upsertRateLimitBulkParams.Windows = append(upsertRateLimitBulkParams.Windows, getWindowParamFromDurString(duration))
		upsertRateLimitBulkParams.Limitvalues = append(upsertRateLimitBulkParams.Limitvalues, int32(limitValue)) // nolint: gosec
	}

	var stepRateLimits []*dbsqlc.StepRateLimit

	if len(upsertRateLimitBulkParams.Keys) > 0 {
		// upsert all rate limits based on the keys, limit values, and durations
		err = s.queries.UpsertRateLimitsBulk(ctx, dbtx, upsertRateLimitBulkParams)

		if err != nil {
			return nil, nil, fmt.Errorf("could not bulk upsert dynamic rate limits: %w", err)
		}
	}

	// get all existing static rate limits for steps to the mapping, mapping back from step ids to step run ids
	uniqueStepIds := make([]pgtype.UUID, 0, len(stepIdToStepRuns))

	for stepId := range stepIdToStepRuns {
		uniqueStepIds = append(uniqueStepIds, sqlchelpers.UUIDFromStr(stepId))
	}

	stepRateLimits, err = s.queries.ListRateLimitsForSteps(ctx, dbtx, dbsqlc.ListRateLimitsForStepsParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Stepids:  uniqueStepIds,
	})

	if err != nil {
		return nil, nil, fmt.Errorf("could not list rate limits for steps: %w", err)
	}

	for _, row := range stepRateLimits {
		stepsWithRateLimits[sqlchelpers.UUIDToStr(row.StepId)] = true
		stepId := sqlchelpers.UUIDToStr(row.StepId)
		stepRuns := stepIdToStepRuns[stepId]

		for _, stepRunId := range stepRuns {
			if _, ok := stepRunToKeyToUnits[stepRunId]; !ok {
				stepRunToKeyToUnits[stepRunId] = make(map[string]int32)
			}

			stepRunToKeyToUnits[stepRunId][row.RateLimitKey] = row.Units
		}
	}

	rateLimitsForTenant, err := s.queries.ListRateLimitsForTenantWithMutate(ctx, dbtx, sqlchelpers.UUIDFromStr(tenantId))

	if err != nil {
		return nil, nil, fmt.Errorf("could not list rate limits for tenant: %w", err)
	}

	mapRateLimitsForTenant := make(map[string]*dbsqlc.ListRateLimitsForTenantWithMutateRow)

	for _, row := range rateLimitsForTenant {
		mapRateLimitsForTenant[row.Key] = row
	}

	// store all step ids in the cache, so we can skip rate limiting for steps without rate limits
	for stepId := range stepIdToStepRuns {
		hasRateLimit := stepsWithRateLimits[stepId]
		s.cachedStepIdHasRateLimit.Set(stepId, hasRateLimit)
	}

	return stepRunToKeyToUnits, mapRateLimitsForTenant, nil
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

func (s *stepRunEngineRepository) ProcessStepRunUpdates(ctx context.Context, qlp *zerolog.Logger, tenantId string) (repository.ProcessStepRunUpdatesResult, error) {
	ql := qlp.With().Str("tenant_id", tenantId).Logger()
	// startedAt := time.Now().UTC()

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

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, s.pool, s.l, 25000)

	if err != nil {
		return emptyRes, err
	}

	defer rollback()

	// list queues
	queueItems, err := s.queries.ListInternalQueueItems(ctx, tx, dbsqlc.ListInternalQueueItemsParams{
		Tenantid: pgTenantId,
		Queue:    dbsqlc.InternalQueueSTEPRUNUPDATE,
		Limit: pgtype.Int4{
			Int32: int32(limit), // nolint: gosec
			Valid: true,
		},
	})

	if err != nil {
		return emptyRes, fmt.Errorf("could not list queues: %w", err)
	}

	data, err := toQueueItemData[updateStepRunQueueDataV0](queueItems)

	if err != nil {
		return emptyRes, fmt.Errorf("could not convert internal queue item data to worker semaphore queue data: %w", err)
	}

	succeededStepRuns, completedWorkflowRuns, err := s.processStepRunUpdates(ctx, &ql, tenantId, tx, data)

	if err != nil {
		return emptyRes, fmt.Errorf("could not process step run updates v0: %w", err)
	}

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

	return repository.ProcessStepRunUpdatesResult{
		SucceededStepRuns:     succeededStepRuns,
		CompletedWorkflowRuns: completedWorkflowRuns,
		Continue:              len(queueItems) == limit,
	}, nil
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

func (s *stepRunEngineRepository) processStepRunUpdates(
	ctx context.Context,
	qlp *zerolog.Logger,
	tenantId string,
	tx dbsqlc.DBTX,
	data []updateStepRunQueueDataV0,
) (succeededStepRuns []*dbsqlc.GetStepRunForEngineRow, completedWorkflowRuns []*dbsqlc.ResolveWorkflowRunStatusRow, err error) {
	// startedAt := time.Now().UTC()
	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	startParams := dbsqlc.BulkStartStepRunParams{}
	failParams := dbsqlc.BulkFailStepRunParams{}
	cancelParams := dbsqlc.BulkCancelStepRunParams{}
	finishParams := dbsqlc.BulkFinishStepRunParams{}

	stepRunIds := make([]pgtype.UUID, 0, len(data))
	eventTimeSeen := make([]time.Time, 0, len(data))
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

			if item.Event.Timestamp != nil {
				eventTimeSeen = append(eventTimeSeen, *item.Event.Timestamp)
			} else {
				eventTimeSeen = append(eventTimeSeen, time.Now().UTC())
			}

			continue
		}

		if item.Status == nil {
			continue
		}

		stepRunIds = append(stepRunIds, stepRunId)

		switch dbsqlc.StepRunStatus(*item.Status) {
		case dbsqlc.StepRunStatusRUNNING:
			startParams.Steprunids = append(startParams.Steprunids, stepRunId)
			startParams.Startedats = append(startParams.Startedats, sqlchelpers.TimestampFromTime(*item.StartedAt))
			eventStepRunIds = append(eventStepRunIds, stepRunId)
			eventTimeSeen = append(eventTimeSeen, *item.StartedAt)
			eventReasons = append(eventReasons, dbsqlc.StepRunEventReasonSTARTED)
			eventSeverities = append(eventSeverities, dbsqlc.StepRunEventSeverityINFO)
			eventMessages = append(eventMessages, fmt.Sprintf("Step run started at %s", item.StartedAt.Format(time.RFC1123)))
			eventData = append(eventData, map[string]interface{}{})
		case dbsqlc.StepRunStatusFAILED:
			failParams.Steprunids = append(failParams.Steprunids, stepRunId)
			failParams.Finishedats = append(failParams.Finishedats, sqlchelpers.TimestampFromTime(*item.FinishedAt))
			eventTimeSeen = append(eventTimeSeen, *item.FinishedAt)
			failParams.Errors = append(failParams.Errors, *item.Error)

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
			eventData = append(eventData, map[string]interface{}{
				"retry_count": item.RetryCount,
			})
		case dbsqlc.StepRunStatusCANCELLED:
			cancelParams.Steprunids = append(cancelParams.Steprunids, stepRunId)
			cancelParams.Cancelledats = append(cancelParams.Cancelledats, sqlchelpers.TimestampFromTime(*item.CancelledAt))
			cancelParams.Finishedats = append(cancelParams.Finishedats, sqlchelpers.TimestampFromTime(*item.CancelledAt))
			eventTimeSeen = append(eventTimeSeen, *item.CancelledAt)
			cancelParams.Cancelledreasons = append(cancelParams.Cancelledreasons, *item.CancelledReason)
			eventStepRunIds = append(eventStepRunIds, stepRunId)
			eventReasons = append(eventReasons, dbsqlc.StepRunEventReasonCANCELLED)
			eventSeverities = append(eventSeverities, dbsqlc.StepRunEventSeverityWARNING)
			eventMessages = append(eventMessages, fmt.Sprintf("Step run was cancelled on %s for the following reason: %s", item.CancelledAt.Format(time.RFC1123), *item.CancelledReason))
			eventData = append(eventData, map[string]interface{}{})
		case dbsqlc.StepRunStatusSUCCEEDED:
			finishParams.Steprunids = append(finishParams.Steprunids, stepRunId)
			finishParams.Finishedats = append(finishParams.Finishedats, sqlchelpers.TimestampFromTime(*item.FinishedAt))
			eventTimeSeen = append(eventTimeSeen, *item.FinishedAt)
			finishParams.Outputs = append(finishParams.Outputs, item.Output)
			eventStepRunIds = append(eventStepRunIds, stepRunId)
			eventReasons = append(eventReasons, dbsqlc.StepRunEventReasonFINISHED)
			eventSeverities = append(eventSeverities, dbsqlc.StepRunEventSeverityINFO)
			eventMessages = append(eventMessages, fmt.Sprintf("Step run finished at %s", item.FinishedAt.Format(time.RFC1123)))
			eventData = append(eventData, map[string]interface{}{})
		}
	}

	if len(startParams.Steprunids) > 0 {
		err = s.queries.BulkStartStepRun(ctx, tx, startParams)

		if err != nil {
			return nil, nil, fmt.Errorf("could not start step runs: %w", err)
		}
	}

	if len(failParams.Steprunids) > 0 {
		err = s.queries.BulkFailStepRun(ctx, tx, failParams)

		if err != nil {
			return nil, nil, fmt.Errorf("could not fail step runs: %w", err)
		}
	}

	if len(cancelParams.Steprunids) > 0 {
		err = s.queries.BulkCancelStepRun(ctx, tx, cancelParams)

		if err != nil {
			return nil, nil, fmt.Errorf("could not cancel step runs: %w", err)
		}
	}

	if len(finishParams.Steprunids) > 0 {
		err = s.queries.BulkFinishStepRun(ctx, tx, finishParams)

		if err != nil {
			return nil, nil, fmt.Errorf("could not finish step runs: %w", err)
		}
	}

	// durationUpdateStepRuns := time.Since(startedAt)

	// startResolveJobRunStatus := time.Now()

	// update the job runs and workflow runs as well
	jobRunIds, err := s.queries.ResolveJobRunStatus(ctx, tx,
		stepRunIds,
	)

	if err != nil {
		return nil, nil, fmt.Errorf("could not resolve job run status: %w", err)
	}

	// durationResolveJobRunStatus := time.Since(startResolveJobRunStatus)

	// startResolveWorkflowRuns := time.Now()

	succeededStepRuns, err = s.queries.GetStepRunForEngine(ctx, tx, dbsqlc.GetStepRunForEngineParams{
		Ids:      finishParams.Steprunids,
		TenantId: pgTenantId,
	})

	if err != nil {
		return nil, nil, fmt.Errorf("could not get succeeded step runs: %w", err)
	}

	completedWorkflowRuns, err = s.queries.ResolveWorkflowRunStatus(ctx, tx, dbsqlc.ResolveWorkflowRunStatusParams{
		Jobrunids: jobRunIds,
		Tenantid:  pgTenantId,
	})

	if err != nil {
		return nil, nil, fmt.Errorf("could not resolve workflow run status: %w", err)
	}

	// durationResolveWorkflowRuns := time.Since(startResolveWorkflowRuns)

	for i, stepRunId := range eventStepRunIds {
		_, err = s.bulkEventBuffer.BuffItem(tenantId, &repository.CreateStepRunEventOpts{
			StepRunId:     sqlchelpers.UUIDToStr(stepRunId),
			EventMessage:  &eventMessages[i],
			EventReason:   &eventReasons[i],
			EventSeverity: &eventSeverities[i],
			Timestamp:     &eventTimeSeen[i],
			EventData:     eventData[i],
		})

		if err != nil {
			s.l.Err(err).Msg("could not buffer step run event")
		}
	}

	// defer printProcessStepRunUpdateInfo(ql, tenantId, startedAt, len(stepRunIds), durationUpdateStepRuns, durationResolveJobRunStatus, durationResolveWorkflowRuns, durationMarkQueueItemsProcessed, durationRunEvents)

	return succeededStepRuns, completedWorkflowRuns, nil
}

func (s *stepRunEngineRepository) processStepRunUpdatesV2(
	ctx context.Context,
	qlp *zerolog.Logger,
	tenantId string,
	outerTx dbsqlc.DBTX,
	data []updateStepRunQueueData,
) (succeededStepRuns []*dbsqlc.GetStepRunForEngineRow, completedWorkflowRuns []*dbsqlc.ResolveWorkflowRunStatusRow, err error) {
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
				}
			}

			innerCompletedWorkflowRuns, err := s.bulkProcessStepRunUpdates(ctx, startParams, failParams, cancelParams, finishParams, batchStepRunIds, pgTenantId)

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

				innerCompletedWorkflowRuns, err = s.bulkProcessStepRunUpdates(ctx, startParams, failParams, cancelParams, finishParams, batchStepRunIds, pgTenantId)

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

	_, err := s.bulkStatusBuffer.BuffItem(tenantId, data)

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

	_, err := s.bulkEventBuffer.BuffItem(tenantId, data)

	if err != nil {
		return fmt.Errorf("could not buffer event: %w", err)
	}

	return nil
}

func (s *stepRunEngineRepository) StepRunSucceeded(ctx context.Context, tenantId, workflowRunId, stepRunId string, finishedAt time.Time, output []byte) error {
	ctx, span := telemetry.NewSpan(ctx, "step-run-started-db")
	defer span.End()

	finished := string(dbsqlc.StepRunStatusSUCCEEDED)

	data := &updateStepRunQueueData{
		Hash:       hashToBucket(sqlchelpers.UUIDFromStr(workflowRunId), s.maxHashFactor),
		StepRunId:  stepRunId,
		TenantId:   tenantId,
		FinishedAt: &finishedAt,
		Status:     &finished,
		Output:     output,
	}

	// we write to the buffer first so we don't get race conditions when we resolve workflow run statuses
	done, err := s.bulkStatusBuffer.BuffItem(tenantId, data)

	if err != nil {
		return fmt.Errorf("could not buffer step run succeeded: %w", err)
	}

	var response *buffer.FlushResponse[pgtype.UUID]

	select {
	case response = <-done:
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(20 * time.Second):
		return fmt.Errorf("timeout waiting for step run succeeded to be flushed to db")
	}

	if response.Err != nil {
		return fmt.Errorf("could not flush step run succeeded: %w", response.Err)
	}

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

	_, err = s.bulkStatusBuffer.BuffItem(tenantId, data)

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

			_, err := s.bulkStatusBuffer.BuffItem(tenantId, &updateStepRunQueueData{
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

	_, err = s.bulkStatusBuffer.BuffItem(tenantId, data)

	if err != nil {
		return fmt.Errorf("could not buffer step run succeeded: %w", err)
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

		_, err := s.bulkStatusBuffer.BuffItem(tenantId, &updateStepRunQueueData{
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

func (s *stepRunEngineRepository) doCachedUpsertOfQueue(ctx context.Context, tx dbsqlc.DBTX, tenantId string, innerStepRun *dbsqlc.GetStepRunForEngineRow) error {
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
		err := s.releaseWorkerSemaphoreSlot(ctx, tenantId, stepRunId)

		if err != nil {
			return nil, fmt.Errorf("could not release worker semaphore queue items: %w", err)
		}

		// retries get highest priority to ensure that they're run immediately
		priority = 4
	}

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

	err = s.doCachedUpsertOfQueue(ctx, s.pool, tenantId, innerStepRun)

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

	_, err = s.bulkQueuer.BuffItem(tenantId, buffer.BulkQueueStepRunOpts{
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

func (s *stepRunEngineRepository) ListInitialStepRunsForJobRun(ctx context.Context, tenantId, jobRunId string) ([]*dbsqlc.GetStepRunForEngineRow, error) {
	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return nil, err
	}

	defer sqlchelpers.DeferRollback(ctx, s.l, tx.Rollback)

	srs, err := s.queries.ListInitialStepRuns(ctx, tx, sqlchelpers.UUIDFromStr(jobRunId))

	if err != nil {
		return nil, fmt.Errorf("could not list initial step runs: %w", err)
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

func (s *stepRunEngineRepository) releaseWorkerSemaphoreSlot(ctx context.Context, tenantId, stepRunId string) error {
	done, err := s.bulkSemaphoreReleaser.BuffItem(tenantId, buffer.SemaphoreReleaseOpts{
		StepRunId: sqlchelpers.UUIDFromStr(stepRunId),
		TenantId:  sqlchelpers.UUIDFromStr(tenantId),
	})

	if err != nil {
		return fmt.Errorf("could not buffer semaphore release: %w", err)
	}

	var response *buffer.FlushResponse[pgtype.UUID]

	select {
	case response = <-done:
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(15 * time.Second):
		return fmt.Errorf("timeout waiting for semaphore slot to be flushed to db")
	}

	if response.Err != nil {
		return fmt.Errorf("could not release worker semaphore slot: %w", response.Err)
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

type updateStepRunQueueDataV0 struct {
	StepRunId  string `json:"step_run_id"`
	TenantId   string `json:"tenant_id"`
	RetryCount int    `json:"retry_count,omitempty"`

	Event *repository.CreateStepRunEventOpts `json:"event,omitempty"`

	StartedAt       *time.Time `json:"started_at,omitempty"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
	CancelledAt     *time.Time `json:"cancelled_at,omitempty"`
	Output          []byte     `json:"output"`
	CancelledReason *string    `json:"cancelled_reason,omitempty"`
	Error           *string    `json:"error,omitempty"`
	Status          *string    `json:"status,omitempty"`
}

// func insertStepRunQueueItem(
// 	ctx context.Context,
// 	dbtx dbsqlc.DBTX,
// 	queries *dbsqlc.Queries,
// 	tenantId string,
// 	data updateStepRunQueueData,
// ) error {
// 	insertData := make([]any, 1)
// 	insertData[0] = data

// 	return bulkInsertInternalQueueItem(
// 		ctx,
// 		dbtx,
// 		queries,
// 		tenantId,
// 		dbsqlc.InternalQueueSTEPRUNUPDATEV2,
// 		insertData,
// 	)
// }

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
	durationPrepareTx,
	durationListQueues,
	durationListQueueItems,
	durationListRateLimits,
	durationGetWorkerCounts,
	durationGetLabels,
	durationScheduling,
	durationUpdateRateLimits,
	durationAssignQueueItems,
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
		"duration_prepare_tx", durationPrepareTx,
	).Dur(
		"duration_list_queues", durationListQueues,
	).Dur(
		"duration_list_queue_items", durationListQueueItems,
	).Dur(
		"duration_list_rate_limits", durationListRateLimits,
	).Dur(
		"duration_get_worker_counts", durationGetWorkerCounts,
	).Dur(
		"duration_get_labels", durationGetLabels,
	).Dur(
		"duration_scheduling", durationScheduling,
	).Dur(
		"duration_update_rate_limits", durationUpdateRateLimits,
	).Dur(
		"duration_assign_queue_items", durationAssignQueueItems,
	).Dur(
		"duration_pop_queue_items", durationPopQueueItems,
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

func hashToBucket(id pgtype.UUID, buckets int) int {
	hasher := fnv.New32a()
	hasher.Write(id.Bytes[:])
	return int(hasher.Sum32()) % buckets
}
