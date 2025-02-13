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
	v2 "github.com/hatchet-dev/hatchet/pkg/repository/v2"
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

	StartedAfter time.Time

	FinishedBefore *time.Time

	AdditionalMetadata map[string]interface{}

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
	ListTaskRuns(ctx context.Context, tenantId string, opts ListTaskRunOpts) ([]*olap.TaskRunDataRow, int, error)
	ListTaskRunEvents(ctx context.Context, tenantId string, taskId int64, taskInsertedAt pgtype.Timestamptz, limit, offset int64) ([]*timescalev2.ListTaskEventsRow, error)
	ReadTaskRunMetrics(ctx context.Context, tenantId string, opts ReadTaskRunMetricsOpts) ([]olap.TaskRunMetric, error)
	CreateTasks(ctx context.Context, tenantId string, tasks []*sqlcv2.V2Task) error
	CreateTaskEvents(ctx context.Context, tenantId string, events []timescalev2.CreateTaskEventsOLAPParams) error
	CreateDAGs(ctx context.Context, tenantId string, dags []*v2.DAGWithData) error
	GetTaskPointMetrics(ctx context.Context, tenantId string, startTimestamp *time.Time, endTimestamp *time.Time, bucketInterval time.Duration) ([]*timescalev2.GetTaskPointMetricsRow, error)
	UpdateTaskStatuses(ctx context.Context, tenantId string) (bool, error)
	UpdateDAGStatuses(ctx context.Context, tenantId string) (bool, error)
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
	partitionCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err = queries.CreateOLAPTaskEventTmpPartitions(partitionCtx, timescalePool, NUM_PARTITIONS)

	if err != nil {
		log.Fatal(err)
	}

	err = queries.CreateOLAPTaskStatusUpdateTmpPartitions(partitionCtx, timescalePool, NUM_PARTITIONS)

	if err != nil {
		log.Fatal(err)
	}

	setupRangePartition(
		partitionCtx,
		timescalePool,
		queries.CreateOLAPTaskPartition,
		queries.ListOLAPTaskPartitionsBeforeDate,
		"v2_tasks_olap",
	)

	setupRangePartition(
		partitionCtx,
		timescalePool,
		queries.CreateOLAPDAGPartition,
		queries.ListOLAPDAGPartitionsBeforeDate,
		"v2_dags_olap",
	)

	setupRangePartition(
		partitionCtx,
		timescalePool,
		queries.CreateOLAPRunsPartition,
		queries.ListOLAPRunsPartitionsBeforeDate,
		"v2_runs_olap",
	)

	return &olapEventRepository{
		pool:       timescalePool,
		l:          l,
		queries:    queries,
		eventCache: eventCache,
	}
}

func setupRangePartition(
	ctx context.Context,
	pool *pgxpool.Pool,
	create func(ctx context.Context, db timescalev2.DBTX, date pgtype.Date) error,
	listBeforeDate func(ctx context.Context, db timescalev2.DBTX, date pgtype.Date) ([]string, error),
	tableName string,
) {
	today := time.Now().UTC()
	tomorrow := today.AddDate(0, 0, 1)
	sevenDaysAgo := today.AddDate(0, 0, -7)

	err := create(ctx, pool, pgtype.Date{
		Time:  today,
		Valid: true,
	})

	if err != nil {
		log.Fatal(err)
	}

	err = create(ctx, pool, pgtype.Date{
		Time:  tomorrow,
		Valid: true,
	})

	if err != nil {
		log.Fatal(err)
	}

	partitions, err := listBeforeDate(ctx, pool, pgtype.Date{
		Time:  sevenDaysAgo,
		Valid: true,
	})

	if err != nil {
		log.Fatal(err)
	}

	for _, partition := range partitions {
		_, err := pool.Exec(
			ctx,
			fmt.Sprintf("ALTER TABLE %s DETACH PARTITION %s CONCURRENTLY", tableName, partition),
		)

		if err != nil {
			log.Fatal(err)
		}

		_, err = pool.Exec(
			ctx,
			fmt.Sprintf("DROP TABLE %s", partition),
		)

		if err != nil {
			log.Fatal(err)
		}
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
	row, err := r.queries.ReadTaskByExternalID(ctx, r.pool, sqlchelpers.UUIDFromStr(taskExternalId))

	if err != nil {
		return nil, err
	}

	return &timescalev2.V2TasksOlap{
		TenantID:           row.TenantID,
		ID:                 row.ID,
		InsertedAt:         row.InsertedAt,
		Queue:              row.Queue,
		ActionID:           row.ActionID,
		StepID:             row.StepID,
		WorkflowID:         row.WorkflowID,
		ScheduleTimeout:    row.ScheduleTimeout,
		StepTimeout:        row.StepTimeout,
		Priority:           row.Priority,
		Sticky:             row.Sticky,
		DesiredWorkerID:    row.DesiredWorkerID,
		DisplayName:        row.DisplayName,
		Input:              row.Input,
		AdditionalMetadata: row.AdditionalMetadata,
		DagID:              row.DagID,
		DagInsertedAt:      row.DagInsertedAt,
		ReadableStatus:     row.ReadableStatus,
		ExternalID:         row.ExternalID,
		LatestRetryCount:   row.LatestRetryCount,
		LatestWorkerID:     row.LatestWorkerID,
	}, nil
}

func (r *olapEventRepository) ReadTaskRunData(ctx context.Context, tenantId pgtype.UUID, taskId int64, taskInsertedAt pgtype.Timestamptz) (*timescalev2.PopulateSingleTaskRunDataRow, error) {
	return r.queries.PopulateSingleTaskRunData(ctx, r.pool, timescalev2.PopulateSingleTaskRunDataParams{
		Taskid:         taskId,
		Tenantid:       tenantId,
		Taskinsertedat: taskInsertedAt,
	})
}

func uniq[T comparable](arr []T) []T {
	const t = true

	seen := make(map[T]bool)
	uniq := make([]T, 0)

	for _, item := range arr {
		if _, ok := seen[item]; !ok {
			seen[item] = t
			uniq = append(uniq, item)
		}
	}

	return uniq
}

func (r *olapEventRepository) ListTaskRuns(ctx context.Context, tenantId string, opts ListTaskRunOpts) ([]*olap.TaskRunDataRow, int, error) {
	tx, err := r.pool.Begin(ctx)

	if err != nil {
		return nil, 0, err
	}

	defer tx.Rollback(ctx)

	statuses := make([]string, 0)
	for _, status := range opts.Statuses {
		statuses = append(statuses, string(status))
	}

	var workflowIdParams []pgtype.UUID

	if len(opts.WorkflowIds) > 0 {
		workflowIdParams = make([]pgtype.UUID, 0)

		for _, id := range opts.WorkflowIds {
			workflowIdParams = append(workflowIdParams, sqlchelpers.UUIDFromStr(id.String()))
		}
	}

	params := timescalev2.ListWorkflowRunsParams{
		WorkflowIds:            workflowIdParams,
		Statuses:               statuses,
		Since:                  sqlchelpers.TimestamptzFromTime(opts.CreatedAfter),
		Listworkflowrunsoffset: int32(opts.Offset),
		Listworkflowrunslimit:  int32(opts.Limit),
	}

	until := opts.FinishedBefore
	if until != nil {
		params.Until = sqlchelpers.TimestamptzFromTime(*until)
	}

	workerId := opts.WorkerId
	if workerId != nil {
		params.WorkerId = sqlchelpers.UUIDFromStr(workerId.String())
	}

	for key, value := range opts.AdditionalMetadata {
		params.Keys = append(params.Keys, key)
		params.Values = append(params.Values, value.(string))
	}

	rows, err := r.queries.ListWorkflowRuns(ctx, r.pool, params)

	if err != nil {
		return nil, 0, err
	}

	dagIds := make([]int64, 0)

	for _, row := range rows {
		dagIds = append(dagIds, row.DagID.Int64)
	}

	dagIds = uniq(dagIds)

	children, err := r.queries.ListDAGChildren(ctx, r.pool, dagIds)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, err
	}

	if err := tx.Commit(context.Background()); err != nil {
		return nil, 0, err
	}

	records := make([]*olap.TaskRunDataRow, 0)
	for _, row := range rows {
		parent := &timescalev2.ListWorkflowRunsRow{
			TenantID:           row.TenantID,
			RunID:              row.RunID,
			InsertedAt:         row.InsertedAt,
			ExternalID:         row.ExternalID,
			WorkflowID:         row.WorkflowID,
			DisplayName:        row.DisplayName,
			AdditionalMetadata: row.AdditionalMetadata,
			ReadableStatus:     row.ReadableStatus,
			FinishedAt:         row.FinishedAt,
			StartedAt:          row.StartedAt,
			Output:             row.Output,
			ErrorMessage:       row.ErrorMessage,
		}

		rowChildren := make([]*timescalev2.ListDAGChildrenRow, 0)

		for _, child := range children {
			if child.DagID == row.DagID.Int64 {
				rowChildren = append(rowChildren, child)
			}
		}

		record := &olap.TaskRunDataRow{
			Parent:   parent,
			Children: rowChildren,
		}

		records = append(records, record)
	}

	countParams := timescalev2.CountRunsParams{
		WorkflowIds: workflowIdParams,
		Statuses:    statuses,
		Since:       sqlchelpers.TimestamptzFromTime(opts.CreatedAfter),
	}

	count, err := r.queries.CountRuns(ctx, r.pool, countParams)

	if err != nil {
		count = int64(len(records))
	}

	return records, int(count), nil
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
				WorkerID:       event.WorkerID,
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

func (r *olapEventRepository) UpdateDAGStatuses(ctx context.Context, tenantId string) (bool, error) {
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

			count, err := r.queries.UpdateDAGStatuses(ctx, tx, timescalev2.UpdateDAGStatusesParams{
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
			DagID:              task.DagID,
			DagInsertedAt:      task.DagInsertedAt,
		})
	}

	_, err := r.queries.CreateTasksOLAP(ctx, r.pool, params)

	return err
}

func (r *olapEventRepository) writeDAGBatch(ctx context.Context, tenantId string, dags []*v2.DAGWithData) error {
	params := make([]timescalev2.CreateDAGsOLAPParams, 0)

	for _, dag := range dags {
		params = append(params, timescalev2.CreateDAGsOLAPParams{
			TenantID:           dag.TenantID,
			ID:                 dag.ID,
			InsertedAt:         dag.InsertedAt,
			WorkflowID:         dag.WorkflowID,
			WorkflowVersionID:  dag.WorkflowVersionID,
			ExternalID:         dag.ExternalID,
			DisplayName:        dag.DisplayName,
			Input:              dag.Input,
			AdditionalMetadata: dag.AdditionalMetadata,
		})
	}

	_, err := r.queries.CreateDAGsOLAP(ctx, r.pool, params)

	return err
}

func (r *olapEventRepository) CreateTaskEvents(ctx context.Context, tenantId string, events []timescalev2.CreateTaskEventsOLAPParams) error {
	return r.writeTaskEventBatch(ctx, tenantId, events)
}

func (r *olapEventRepository) CreateTasks(ctx context.Context, tenantId string, tasks []*sqlcv2.V2Task) error {
	return r.writeTaskBatch(ctx, tenantId, tasks)
}

func (r *olapEventRepository) CreateDAGs(ctx context.Context, tenantId string, dags []*v2.DAGWithData) error {
	return r.writeDAGBatch(ctx, tenantId, dags)
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
