package repository

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/olap"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v2/sqlcv2"
	"github.com/hatchet-dev/hatchet/pkg/repository/v2/timescalev2"
)

type OLAPEventRepository interface {
	ReadTaskRun(ctx context.Context, taskExternalId string) (*timescalev2.V2TasksOlap, error)
	ListTaskRuns(ctx context.Context, tenantId string, since time.Time, statuses []gen.V2TaskStatus, workflowIds []uuid.UUID, limit, offset int64) ([]*timescalev2.PopulateTaskRunDataRow, int, error)
	ListTaskRunEvents(ctx context.Context, tenantId string, taskId int64, taskInsertedAt pgtype.Timestamptz, limit, offset int64) ([]*timescalev2.ListTaskEventsRow, error)
	ReadTaskRunMetrics(ctx context.Context, tenantId string, since time.Time) ([]olap.TaskRunMetric, error)
	CreateTasks(ctx context.Context, tenantId string, tasks []*sqlcv2.V2Task) error
	CreateTaskEvents(ctx context.Context, tenantId string, events []timescalev2.CreateTaskEventsOLAPParams) error
}

type olapEventRepository struct {
	pool *pgxpool.Pool

	eventCache *lru.Cache[string, bool]
	queries    *timescalev2.Queries
}

func NewOLAPEventRepository() OLAPEventRepository {
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

	return &olapEventRepository{
		pool:       timescalePool,
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

func (r *olapEventRepository) ListTaskRuns(ctx context.Context, tenantId string, since time.Time, statuses []gen.V2TaskStatus, workflowIds []uuid.UUID, limit, offset int64) ([]*timescalev2.PopulateTaskRunDataRow, int, error) {
	tx, err := r.pool.Begin(ctx)

	if err != nil {
		return nil, 0, err
	}

	defer tx.Rollback(ctx)

	lastSucceededAggTs, err := r.queries.LastSucceededStatusAggregate(ctx, tx)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, err
	}

	if !lastSucceededAggTs.Valid {
		lastSucceededAggTs = sqlchelpers.TimestamptzFromTime(time.Time{}) // zero value
	}

	taskIds := make([]int64, 0)
	tenantIds := make([]pgtype.UUID, 0)
	taskInsertedAts := make([]pgtype.Timestamptz, 0)
	retryCounts := make([]int32, 0)
	queryStatuses := make([]string, 0)

	realTimeParams := timescalev2.ListTasksRealTimeParams{
		Tenantid:      sqlchelpers.UUIDFromStr(tenantId),
		Insertedafter: lastSucceededAggTs,
		Tasklimit:     int32(limit),
	}

	// if we're filtering by all statuses or no statuses, we can provide an earlier CTE limit to speed up the query
	if len(statuses) == 0 || len(statuses) == 5 {
		realTimeParams.EventLimit = pgtype.Int4{
			Int32: int32(limit),
			Valid: true,
		}

		realTimeParams.Eventstatuses = []string{
			string(timescalev2.V2ReadableStatusOlapQUEUED),
		}

		realTimeParams.EventType = timescalev2.NullV2EventTypeOlap{
			V2EventTypeOlap: timescalev2.V2EventTypeOlapCREATED,
			Valid:           true,
		}
	} else {
		realTimeParams.Eventstatuses = make([]string, 0)
		realTimeParams.Statuses = make([]string, 0)

		for _, status := range statuses {
			realTimeParams.Eventstatuses = append(realTimeParams.Eventstatuses, string(status))
			realTimeParams.Statuses = append(realTimeParams.Statuses, string(status))
		}
	}

	var workflowIdParams []pgtype.UUID

	if len(workflowIds) > 0 {
		workflowIdParams = make([]pgtype.UUID, 0)

		for _, id := range workflowIds {
			workflowIdParams = append(workflowIdParams, sqlchelpers.UUIDFromStr(id.String()))
		}
	}

	realTimeParams.WorkflowIds = workflowIdParams

	uniqueTasks := make(map[int64]struct{}, 0)

	realTimeTasks, err := r.queries.ListTasksRealTime(ctx, tx, realTimeParams)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, 0, err
	}

	for _, task := range realTimeTasks {
		uniqueTasks[task.TaskID] = struct{}{}
		taskIds = append(taskIds, task.TaskID)
		tenantIds = append(tenantIds, task.TenantID)
		taskInsertedAts = append(taskInsertedAts, task.TaskInsertedAt)
		retryCounts = append(retryCounts, task.MaxRetryCount)
		queryStatuses = append(queryStatuses, string(task.Status))
	}

	if len(taskIds) != int(limit) {
		historicalParams := timescalev2.ListTasksFromAggregateParams{
			Tenantid:     sqlchelpers.UUIDFromStr(tenantId),
			Createdafter: sqlchelpers.TimestamptzFromTime(since),
			Tasklimit:    int32(limit),
			WorkflowIds:  workflowIdParams,
		}

		if len(statuses) != 0 && len(statuses) != 5 {
			historicalParams.Statuses = make([]string, 0)

			for _, status := range statuses {
				historicalParams.Statuses = append(historicalParams.Statuses, string(status))
			}
		} else {

		}

		historicalTasks, err := r.queries.ListTasksFromAggregate(ctx, tx, historicalParams)

		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, 0, err
		}

		for _, task := range historicalTasks {
			if _, ok := uniqueTasks[task.TaskID]; ok {
				continue
			}

			taskIds = append(taskIds, task.TaskID)
			tenantIds = append(tenantIds, task.TenantID)
			taskInsertedAts = append(taskInsertedAts, task.InsertedAt)
			retryCounts = append(retryCounts, task.MaxRetryCount)
			queryStatuses = append(queryStatuses, string(task.Status))
		}
	}

	// get the task rows
	rows, err := r.queries.PopulateTaskRunData(ctx, tx, timescalev2.PopulateTaskRunDataParams{
		Taskids:         taskIds,
		Tenantids:       tenantIds,
		Taskinsertedats: taskInsertedAts,
		Retrycounts:     retryCounts,
		Statuses:        queryStatuses,
		Tasklimit:       int32(limit),
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

func (r *olapEventRepository) ReadTaskRunMetrics(ctx context.Context, tenantId string, since time.Time) ([]olap.TaskRunMetric, error) {
	res, err := r.queries.GetTenantStatusMetrics(context.Background(), r.pool, timescalev2.GetTenantStatusMetricsParams{
		Tenantid:     sqlchelpers.UUIDFromStr(tenantId),
		Createdafter: sqlchelpers.TimestamptzFromTime(since),
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

	for _, event := range events {
		key := getCacheKey(event)

		if _, ok := r.eventCache.Get(key); !ok {
			eventsToWrite = append(eventsToWrite, event)
		}
	}

	if len(eventsToWrite) == 0 {
		return nil
	}

	_, err := r.queries.CreateTaskEventsOLAP(ctx, r.pool, eventsToWrite)

	if err != nil {
		return err
	}

	r.saveEventsToCache(eventsToWrite)

	return nil
}

func (r *olapEventRepository) writeTaskBatch(ctx context.Context, tenantId string, tasks []*sqlcv2.V2Task) error {
	params := make([]timescalev2.CreateTasksOLAPParams, 0)

	for _, task := range tasks {
		params = append(params, timescalev2.CreateTasksOLAPParams{
			TenantID:        task.TenantID,
			ID:              task.ID,
			InsertedAt:      task.InsertedAt,
			Queue:           task.Queue,
			ActionID:        task.ActionID,
			StepID:          task.StepID,
			WorkflowID:      task.WorkflowID,
			ScheduleTimeout: task.ScheduleTimeout,
			StepTimeout:     task.StepTimeout,
			Priority:        task.Priority,
			Sticky:          timescalev2.V2StickyStrategyOlap(task.Sticky),
			DesiredWorkerID: task.DesiredWorkerID,
			ExternalID:      task.ExternalID,
			DisplayName:     task.DisplayName,
			Input:           task.Input,
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
