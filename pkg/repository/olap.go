package repository

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/olap"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v2/sqlcv2"
	"github.com/hatchet-dev/hatchet/pkg/repository/v2/timescalev2"
)

// TODO: make this dynamic for the instance
const NUM_PARTITIONS = 4

type ListTaskRunOpts struct {
	CreatedAfter time.Time

	Statuses []gen.V2TaskStatus

	WorkflowIds []uuid.UUID

	WorkerId *uuid.UUID

	Limit int64

	Offset int64
}

type ReadTaskRunMetricsOpts struct {
	CreatedAfter time.Time

	WorkflowIds []uuid.UUID
}

type OLAPEventRepository interface {
	ReadTaskRun(ctx context.Context, taskExternalId string) (*timescalev2.V2TasksOlap, error)
	ReadTaskRunData(ctx context.Context, tenantId pgtype.UUID, taskId int64, taskInsertedAt pgtype.Timestamptz) (*timescalev2.PopulateSingleTaskRunDataRow, error)
	ListTaskRuns(ctx context.Context, tenantId string, opts ListTaskRunOpts) ([]*timescalev2.PopulateTaskRunDataRow, int, error)
	ListTaskRunEvents(ctx context.Context, tenantId string, taskId int64, taskInsertedAt pgtype.Timestamptz, limit, offset int64) ([]*timescalev2.ListTaskEventsRow, error)
	ReadTaskRunMetrics(ctx context.Context, tenantId string, opts ReadTaskRunMetricsOpts) ([]olap.TaskRunMetric, error)
	CreateTasks(ctx context.Context, tenantId string, tasks []*sqlcv2.V2Task) error
	CreateTaskEvents(ctx context.Context, tenantId string, events []timescalev2.CreateTaskEventsOLAPParams) error
	GetTaskPointMetrics(ctx context.Context, tenantId string, startTimestamp *time.Time, endTimestamp *time.Time, bucketInterval time.Duration) ([]*timescalev2.GetTaskPointMetricsRow, error)
	UpdateTaskStatuses(ctx context.Context, tenantId string) (bool, error)
}

type olapEventRepository struct {
	pool *pgxpool.Pool
	l    *zerolog.Logger

	eventCache *lru.Cache[string, bool]
	queries    *timescalev2.Queries
}

func NewOLAPEventRepository(l *zerolog.Logger) OLAPEventRepository {
	timescaleUrl := os.Getenv("TIMESCALE_URL")

	if timescaleUrl == "" {
		log.Fatal("TIMESCALE_URL is not set")
	}

	timescaleConfig, err := pgxpool.ParseConfig(timescaleUrl)

	if err != nil {
		log.Fatal(err)
	}

	timescaleConfig.MaxConns = 150
	timescaleConfig.MinConns = 10
	timescaleConfig.MaxConnLifetime = 15 * 60 * time.Second

	timescalePool, err := pgxpool.NewWithConfig(context.Background(), timescaleConfig)

	if err != nil {
		log.Fatalf("Unable to create connection pool: %v\n", err)
	}

	eventCache, err := lru.New[string, bool](100000)

	if err != nil {
		log.Fatal(err)
	}

	queries := timescalev2.New()

	// create partitions of the events OLAP table
	partitionCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = queries.CreateOLAPPartitions(partitionCtx, timescalePool, NUM_PARTITIONS)

	if err != nil {
		log.Fatal(err)
	}

	return &olapEventRepository{
		pool:       timescalePool,
		l:          l,
		queries:    queries,
		eventCache: eventCache,
	}
}

func StringToReadableStatus(status string) olap.ReadableTaskStatus {
	switch status {
	case "QUEUED":
		return olap.READABLE_TASK_STATUS_QUEUED
	case "RUNNING":
		return olap.READABLE_TASK_STATUS_RUNNING
	case "COMPLETED":
		return olap.READABLE_TASK_STATUS_COMPLETED
	case "CANCELLED":
		return olap.READABLE_TASK_STATUS_CANCELLED
	case "FAILED":
		return olap.READABLE_TASK_STATUS_FAILED
	default:
		return olap.READABLE_TASK_STATUS_QUEUED
	}
}

type getRelevantTaskEventsRow struct {
	TaskId         uuid.UUID
	Timestamp      time.Time
	EventType      string
	ReadableStatus string
}

type getTaskRow struct {
	Id                 uuid.UUID
	AdditionalMetadata string
	DisplayName        string
	TenantId           uuid.UUID
	CreatedAt          time.Time
	WorkflowId         uuid.UUID
}

func (r *olapEventRepository) ReadTaskRun(ctx context.Context, taskExternalId string) (*timescalev2.V2TasksOlap, error) {
	return r.queries.ReadTaskByExternalID(ctx, r.pool, sqlchelpers.UUIDFromStr(taskExternalId))
}

func (r *olapEventRepository) ReadTaskRunData(ctx context.Context, tenantId pgtype.UUID, taskId int64, taskInsertedAt pgtype.Timestamptz) (*timescalev2.PopulateSingleTaskRunDataRow, error) {
	return r.queries.PopulateSingleTaskRunData(ctx, r.pool, timescalev2.PopulateSingleTaskRunDataParams{
		Taskid:         taskId,
		Tenantid:       tenantId,
		Taskinsertedat: taskInsertedAt,
	})
}

func (r *olapEventRepository) ListTaskRuns(ctx context.Context, tenantId string, opts ListTaskRunOpts) ([]*timescalev2.PopulateTaskRunDataRow, int, error) {
	tx, err := r.pool.Begin(ctx)

	if err != nil {
		return nil, 0, err
	}

	defer tx.Rollback(ctx)

	// lastSucceededAggTs, err := r.queries.LastSucceededStatusAggregate(ctx, tx)

	// if err != nil && !errors.Is(err, pgx.ErrNoRows) {
	// 	return nil, 0, err
	// }

	// if !lastSucceededAggTs.Valid {
	// 	lastSucceededAggTs = sqlchelpers.TimestamptzFromTime(time.Time{}) // zero value
	// } else if lastSucceededAggTs.Time.After(time.Now().Add(-5 * time.Minute)) {
	// 	// always search the last 5 minutes of data
	// 	lastSucceededAggTs = sqlchelpers.TimestamptzFromTime(time.Now().Add(-5 * time.Minute))
	// }

	taskIds := make([]int64, 0)
	tenantIds := make([]pgtype.UUID, 0)
	taskInsertedAts := make([]pgtype.Timestamptz, 0)
	retryCounts := make([]int32, 0)
	queryStatuses := make([]string, 0)

	realTimeParams := timescalev2.ListTasksParams{
		Tenantid:      sqlchelpers.UUIDFromStr(tenantId),
		Insertedafter: sqlchelpers.TimestamptzFromTime(opts.CreatedAfter),
		Tasklimit:     int32(opts.Limit),
	}

	// if we're filtering by all statuses or no statuses, we don't pass status in
	if len(opts.Statuses) != 0 && len(opts.Statuses) != 5 {
		realTimeParams.Statuses = make([]string, 0)

		for _, status := range opts.Statuses {
			realTimeParams.Statuses = append(realTimeParams.Statuses, string(status))
		}
	}

	var workflowIdParams []pgtype.UUID

	if len(opts.WorkflowIds) > 0 {
		workflowIdParams = make([]pgtype.UUID, 0)

		for _, id := range opts.WorkflowIds {
			workflowIdParams = append(workflowIdParams, sqlchelpers.UUIDFromStr(id.String()))
		}
	}

	// TODO: FIX
	// if opts.WorkerId != nil {
	// 	realTimeParams.WorkerId = sqlchelpers.UUIDFromStr(opts.WorkerId.String())
	// }

	realTimeParams.WorkflowIds = workflowIdParams

	uniqueTasks := make(map[int64]struct{}, 0)

	realTimeTasks, err := r.queries.ListTasks(ctx, tx, realTimeParams)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, err
	}

	for _, task := range realTimeTasks {
		uniqueTasks[task.ID] = struct{}{}
		taskIds = append(taskIds, task.ID)
		tenantIds = append(tenantIds, task.TenantID)
		taskInsertedAts = append(taskInsertedAts, task.InsertedAt)
		retryCounts = append(retryCounts, task.LatestRetryCount)
		queryStatuses = append(queryStatuses, string(task.ReadableStatus))
	}

	// get the task rows
	rows, err := r.queries.PopulateTaskRunData(ctx, tx, timescalev2.PopulateTaskRunDataParams{
		Taskids:         taskIds,
		Tenantids:       tenantIds,
		Taskinsertedats: taskInsertedAts,
		Retrycounts:     retryCounts,
		Statuses:        queryStatuses,
		Tasklimit:       int32(opts.Limit),
	})

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, err
	}

	if err := tx.Commit(context.Background()); err != nil {
		return nil, 0, err
	}

	return rows, len(rows), nil
}

func (r *olapEventRepository) ListTaskRunEvents(ctx context.Context, tenantId string, taskId int64, taskInsertedAt pgtype.Timestamptz, limit, offset int64) ([]*timescalev2.ListTaskEventsRow, error) {
	rows, err := r.queries.ListTaskEvents(ctx, r.pool, timescalev2.ListTaskEventsParams{
		Tenantid:       sqlchelpers.UUIDFromStr(tenantId),
		Taskid:         taskId,
		Taskinsertedat: taskInsertedAt,
	})

	if err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *olapEventRepository) ReadTaskRunMetrics(ctx context.Context, tenantId string, opts ReadTaskRunMetricsOpts) ([]olap.TaskRunMetric, error) {
	var workflowIds []pgtype.UUID

	if len(opts.WorkflowIds) > 0 {
		workflowIds = make([]pgtype.UUID, 0)

		for _, id := range opts.WorkflowIds {
			workflowIds = append(workflowIds, sqlchelpers.UUIDFromStr(id.String()))
		}
	}

	res, err := r.queries.GetTenantStatusMetrics(context.Background(), r.pool, timescalev2.GetTenantStatusMetricsParams{
		Tenantid:     sqlchelpers.UUIDFromStr(tenantId),
		Createdafter: sqlchelpers.TimestamptzFromTime(opts.CreatedAfter),
		WorkflowIds:  workflowIds,
	})

	if err != nil {
		return nil, err
	}

	metrics := make([]olap.TaskRunMetric, 0)

	metrics = append(metrics, olap.TaskRunMetric{
		Status: "QUEUED",
		Count:  uint64(res.TotalQueued),
	})

	metrics = append(metrics, olap.TaskRunMetric{
		Status: "RUNNING",
		Count:  uint64(res.TotalRunning),
	})

	metrics = append(metrics, olap.TaskRunMetric{
		Status: "COMPLETED",
		Count:  uint64(res.TotalCompleted),
	})

	metrics = append(metrics, olap.TaskRunMetric{
		Status: "CANCELLED",
		Count:  uint64(res.TotalCancelled),
	})

	metrics = append(metrics, olap.TaskRunMetric{
		Status: "FAILED",
		Count:  uint64(res.TotalFailed),
	})

	return metrics, nil
}

func (r *olapEventRepository) saveEventsToCache(events []timescalev2.CreateTaskEventsOLAPParams) {
	for _, event := range events {
		key := getCacheKey(event)
		r.eventCache.Add(key, true)
	}
}

func getCacheKey(event timescalev2.CreateTaskEventsOLAPParams) string {
	// key on the task_id, retry_count, and event_type
	return fmt.Sprintf("%d-%s-%d", event.TaskID, event.EventType, event.RetryCount)
}

func (r *olapEventRepository) writeTaskEventBatch(ctx context.Context, tenantId string, events []timescalev2.CreateTaskEventsOLAPParams) error {
	// skip any events which have a corresponding event already
	eventsToWrite := make([]timescalev2.CreateTaskEventsOLAPParams, 0)
	tmpEventsToWrite := make([]timescalev2.CreateTaskEventsOLAPTmpParams, 0)

	for _, event := range events {
		key := getCacheKey(event)

		if _, ok := r.eventCache.Get(key); !ok {
			eventsToWrite = append(eventsToWrite, event)

			tmpEventsToWrite = append(tmpEventsToWrite, timescalev2.CreateTaskEventsOLAPTmpParams{
				TenantID:       event.TenantID,
				TaskID:         event.TaskID,
				TaskInsertedAt: event.TaskInsertedAt,
				EventType:      event.EventType,
				RetryCount:     event.RetryCount,
				ReadableStatus: event.ReadableStatus,
			})
		}
	}

	if len(eventsToWrite) == 0 {
		return nil
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)

	if err != nil {
		return err
	}

	defer rollback()

	_, err = r.queries.CreateTaskEventsOLAP(ctx, tx, eventsToWrite)

	if err != nil {
		return err
	}

	_, err = r.queries.CreateTaskEventsOLAPTmp(ctx, tx, tmpEventsToWrite)

	if err != nil {
		return err
	}

	if err := commit(ctx); err != nil {
		return err
	}

	r.saveEventsToCache(eventsToWrite)

	return nil
}

func (r *olapEventRepository) UpdateTaskStatuses(ctx context.Context, tenantId string) (bool, error) {
	var limit int32 = 10000

	// each partition gets its own goroutine
	eg := &errgroup.Group{}
	mu := sync.Mutex{}

	// if any of the partitions are saturated, we return true
	isSaturated := false

	for i := 0; i < NUM_PARTITIONS; i++ {
		partitionNumber := i

		eg.Go(func() error {
			tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)

			if err != nil {
				return err
			}

			defer rollback()

			count, err := r.queries.UpdateTaskStatuses(ctx, tx, timescalev2.UpdateTaskStatusesParams{
				Partitionnumber: int32(partitionNumber), // nolint: gosec
				Tenantid:        sqlchelpers.UUIDFromStr(tenantId),
				Eventlimit:      limit,
			})

			if err != nil {
				return err
			}

			if err := commit(ctx); err != nil {
				return err
			}

			mu.Lock()
			isSaturated = isSaturated || count == int64(limit)
			mu.Unlock()

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return false, err
	}

	return isSaturated, nil
}

func (r *olapEventRepository) writeTaskBatch(ctx context.Context, tenantId string, tasks []*sqlcv2.V2Task) error {
	params := make([]timescalev2.CreateTasksOLAPParams, 0)

	for _, task := range tasks {
		params = append(params, timescalev2.CreateTasksOLAPParams{
			TenantID:           task.TenantID,
			ID:                 task.ID,
			InsertedAt:         task.InsertedAt,
			Queue:              task.Queue,
			ActionID:           task.ActionID,
			StepID:             task.StepID,
			WorkflowID:         task.WorkflowID,
			ScheduleTimeout:    task.ScheduleTimeout,
			StepTimeout:        task.StepTimeout,
			Priority:           task.Priority,
			Sticky:             timescalev2.V2StickyStrategyOlap(task.Sticky),
			DesiredWorkerID:    task.DesiredWorkerID,
			ExternalID:         task.ExternalID,
			DisplayName:        task.DisplayName,
			Input:              task.Input,
			AdditionalMetadata: task.AdditionalMetadata,
		})
	}

	_, err := r.queries.CreateTasksOLAP(ctx, r.pool, params)

	return err
}

func (r *olapEventRepository) CreateTaskEvents(ctx context.Context, tenantId string, events []timescalev2.CreateTaskEventsOLAPParams) error {
	return r.writeTaskEventBatch(ctx, tenantId, events)
}

func (r *olapEventRepository) CreateTasks(ctx context.Context, tenantId string, tasks []*sqlcv2.V2Task) error {
	return r.writeTaskBatch(ctx, tenantId, tasks)
}

func (r *olapEventRepository) GetTaskPointMetrics(ctx context.Context, tenantId string, startTimestamp *time.Time, endTimestamp *time.Time, bucketInterval time.Duration) ([]*timescalev2.GetTaskPointMetricsRow, error) {
	rows, err := r.queries.GetTaskPointMetrics(ctx, r.pool, timescalev2.GetTaskPointMetricsParams{
		Interval:      durationToPgInterval(bucketInterval),
		Tenantid:      sqlchelpers.UUIDFromStr(tenantId),
		Createdafter:  sqlchelpers.TimestamptzFromTime(*startTimestamp),
		Createdbefore: sqlchelpers.TimestamptzFromTime(*endTimestamp),
	})

	if err != nil {
		return nil, err
	}

	return rows, nil
}

func durationToPgInterval(d time.Duration) pgtype.Interval {
	// Convert the time.Duration to microseconds
	microseconds := d.Microseconds()

	return pgtype.Interval{
		Microseconds: microseconds,
		Valid:        true,
	}
}
