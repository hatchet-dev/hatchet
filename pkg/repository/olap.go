package repository

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/olap"
)

type OLAPEventRepository interface {
	ReadTaskRun(tenantId, taskRunId uuid.UUID) (olap.WorkflowRun, error)
	ReadTaskRuns(tenantId uuid.UUID, since time.Time, statuses []gen.V2TaskStatus, workflowIds []uuid.UUID, limit, offset int64) ([]olap.WorkflowRun, uint64, error)
	ReadTaskRunEvents(tenantId, taskId uuid.UUID, limit, offset int64) ([]olap.TaskRunEvent, error)
	ReadTaskRunMetrics(tenantId uuid.UUID, since time.Time) ([]olap.TaskRunMetric, error)
	CreateTasks(tasks []olap.Task) error
	CreateTaskEvents(events []olap.TaskEvent) error
}

type olapEventRepository struct {
	conn clickhouse.Conn

	eventCache *lru.Cache[string, bool]
}

func NewOLAPEventRepository() OLAPEventRepository {
	conn, err := olap.CreateClickhouseConnection()

	if err != nil {
		log.Fatal(err)
	}

	eventCache, err := lru.New[string, bool](100000)

	if err != nil {
		log.Fatal(err)
	}

	return &olapEventRepository{
		conn:       conn,
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

func CreateContext() context.Context {
	return clickhouse.Context(context.Background(), clickhouse.WithSettings(clickhouse.Settings{
		"join_use_nulls": "1",
	}))
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

func (r *olapEventRepository) ReadTaskRuns(tenantId uuid.UUID, since time.Time, statuses []gen.V2TaskStatus, workflowIds []uuid.UUID, limit, offset int64) ([]olap.WorkflowRun, uint64, error) {
	var stringifiedStatuses = make([]string, len(statuses))
	for i, status := range statuses {
		stringifiedStatuses[i] = string(status)
	}

	if len(stringifiedStatuses) == 0 {
		stringifiedStatuses = []string{
			string(olap.READABLE_TASK_STATUS_QUEUED),
		}
	}

	ctx := CreateContext()

	// when we need to get all "queued" tasks, we need an unpaginated query of all tasks which have a queued event,
	// and then filter out the tasks which have a later event which is not "queued"

	var eventRows []getRelevantTaskEventsRow
	var err error

	// FIXME: this is ugly, but the frontend will pass in all statuses in which case we don't need a status filter
	if len(stringifiedStatuses) > 0 && len(stringifiedStatuses) != 5 {
		eventRows, err = r.getRelevantRowsWithStatusFilter(ctx, tenantId, since, limit, offset, stringifiedStatuses)
	} else {
		eventRows, err = r.getRelevantRowsNoStatusFilter(ctx, tenantId, since, limit, offset)
	}

	if err != nil {
		return nil, 0, err
	}

	greatestTaskStatus := make(map[string]int)
	taskStatuses := make(map[string]olap.ReadableTaskStatus)
	taskStartedAt := make(map[string]time.Time)
	taskFinishedAt := make(map[string]time.Time)

	for _, row := range eventRows {
		strVal := olap.ReadableTaskStatus(row.ReadableStatus)
		enumVal := strVal.EnumValue()

		currVal, ok := greatestTaskStatus[row.TaskId.String()]

		if !ok {
			greatestTaskStatus[row.TaskId.String()] = enumVal
			taskStatuses[row.TaskId.String()] = strVal
		} else if enumVal > currVal {
			greatestTaskStatus[row.TaskId.String()] = enumVal
			taskStatuses[row.TaskId.String()] = strVal
		}

		if row.EventType == string(olap.EVENT_TYPE_STARTED) {
			taskStartedAt[row.TaskId.String()] = row.Timestamp
		}

		if enumVal >= 3 {
			taskFinishedAt[row.TaskId.String()] = row.Timestamp
		}
	}

	taskDuration := make(map[string]int64)

	for taskId, finishedAt := range taskFinishedAt {
		startedAt, ok := taskStartedAt[taskId]

		if !ok {
			continue
		}

		taskDuration[taskId] = finishedAt.Sub(startedAt).Milliseconds()
	}

	taskIds := make([]string, 0)

	for taskId := range taskStatuses {
		taskIds = append(taskIds, taskId)
	}

	taskRows, err := r.conn.Query(ctx, `
		SELECT
			t.id AS id,
			t.additional_metadata,
			t.display_name,
			t.tenant_id,
			t.inserted_at AS timestamp,
			t.workflow_id
		FROM tasks t
		WHERE
			t.tenant_id = ?
			AND t.id IN (?)
			AND (
				? = []
				OR workflow_id IN (?)
			)
		ORDER BY t.inserted_at DESC, t.id ASC
		`,
		tenantId,
		taskIds,
		workflowIds,
		workflowIds,
	)

	if err != nil {
		return nil, 0, err
	}

	tasks := make([]getTaskRow, 0)

	for taskRows.Next() {
		var row getTaskRow

		err := taskRows.Scan(
			&row.Id,
			&row.AdditionalMetadata,
			&row.DisplayName,
			&row.TenantId,
			&row.CreatedAt,
			&row.WorkflowId,
		)

		if err != nil {
			return nil, 0, err
		}

		tasks = append(tasks, row)
	}

	res := make([]olap.WorkflowRun, 0)

	for _, task := range tasks {
		status, ok := taskStatuses[task.Id.String()]

		if !ok {
			status = olap.READABLE_TASK_STATUS_QUEUED
		}

		startedAt, ok := taskStartedAt[task.Id.String()]

		if !ok {
			startedAt = time.Time{}
		}

		finishedAt, ok := taskFinishedAt[task.Id.String()]

		if !ok {
			finishedAt = time.Time{}
		}

		duration, ok := taskDuration[task.Id.String()]

		if !ok {
			duration = 0
		}

		res = append(res, olap.WorkflowRun{
			AdditionalMetadata: &task.AdditionalMetadata,
			DisplayName:        &task.DisplayName,
			Duration:           &duration,
			ErrorMessage:       nil,
			CreatedAt:          task.CreatedAt,
			FinishedAt:         &finishedAt,
			Id:                 task.Id,
			Input:              "",
			Output:             "",
			StartedAt:          &startedAt,
			Status:             string(status),
			TaskId:             task.Id,
			TenantId:           &task.TenantId,
			Timestamp:          task.CreatedAt,
			WorkflowId:         task.WorkflowId,
		})
	}

	count := r.conn.QueryRow(ctx, `
		WITH task_ids AS (
			SELECT DISTINCT(task_id)
			FROM task_events
			WHERE
				tenant_id = ?
				AND timestamp > ?
				AND readable_status IN (?)
		)
		SELECT COUNT(*)
		FROM task_ids
		`,
		tenantId,
		since,
		stringifiedStatuses,
	)

	var total uint64
	if err := count.Scan(&total); err != nil {
		return []olap.WorkflowRun{}, 0, fmt.Errorf("failed to scan count: %w", err)
	}

	return res, 0, nil
}

func (r *olapEventRepository) getRelevantRowsNoStatusFilter(ctx context.Context, tenantId uuid.UUID, since time.Time, limit, offset int64) ([]getRelevantTaskEventsRow, error) {
	taskEventRows, err := r.conn.Query(ctx, `
		-- name: GetRelevantTaskEvents
		WITH task_ids AS (
			SELECT
				id AS task_id,
				tenant_id
			FROM tasks
			WHERE
				tenant_id = ?
				AND inserted_at > ?
			ORDER BY inserted_at DESC, id ASC
			LIMIT ?
			OFFSET ?
		), max_retry_counts AS (
			SELECT task_id, MAX(retry_count) AS max_retry_count
			FROM task_events
			WHERE
				tenant_id = ?
				AND timestamp > ?
				AND task_id = ANY((
					SELECT task_id FROM task_ids
				))
			GROUP BY task_id
		)
		SELECT
			te.task_id,
			te.timestamp,
			te.event_type,
			te.readable_status
		FROM task_events te
		JOIN max_retry_counts mrc ON te.task_id = mrc.task_id AND te.retry_count = mrc.max_retry_count
		WHERE
			te.tenant_id = ?
			AND te.timestamp > ?
			AND te.task_id = ANY((
				SELECT task_id FROM task_ids
			))
		`,
		tenantId,
		since,
		limit,
		offset,
		tenantId,
		since,
		tenantId,
		since,
	)

	if err != nil {
		return nil, err
	}

	resRows := make([]getRelevantTaskEventsRow, 0)

	for taskEventRows.Next() {
		var row getRelevantTaskEventsRow

		err := taskEventRows.Scan(
			&row.TaskId,
			&row.Timestamp,
			&row.EventType,
			&row.ReadableStatus,
		)

		if err != nil {
			return nil, err
		}

		resRows = append(resRows, row)
	}

	return resRows, nil
}

func (r *olapEventRepository) getRelevantRowsWithStatusFilter(ctx context.Context, tenantId uuid.UUID, since time.Time, limit, offset int64, statuses []string) ([]getRelevantTaskEventsRow, error) {
	taskEventRows, err := r.conn.Query(ctx, `
		-- name: GetRelevantTaskEvents
		WITH selected_tasks AS (
			SELECT *
			FROM task_events
			WHERE
				tenant_id = ?
				AND timestamp > ?
				AND readable_status IN (?)
		), max_retry_counts AS (
			SELECT task_id, MAX(retry_count) AS max_retry_count
			FROM selected_tasks
			GROUP BY task_id
		), max_readable_statuses AS (
			SELECT te.task_id, MAX(te.readable_status) AS max_readable_status
			FROM task_events te
			JOIN max_retry_counts mrc ON te.task_id = mrc.task_id AND te.retry_count = mrc.max_retry_count
			WHERE te.task_id IN (
				SELECT task_id
				FROM selected_tasks
			)
			GROUP BY te.task_id
		)

		SELECT
			te.task_id,
			te.timestamp,
			te.event_type,
			te.readable_status
		FROM selected_tasks te
		INNER JOIN max_readable_statuses mrs ON te.task_id = mrs.task_id AND te.readable_status = mrs.max_readable_status
		JOIN tasks t ON te.task_id = t.id
		WHERE t.tenant_id = ?
		ORDER BY t.inserted_at DESC, t.id ASC
		LIMIT ?
		OFFSET ?
		`,
		tenantId,
		since,
		statuses,
		tenantId,
		limit,
		offset,
	)

	if err != nil {
		return nil, err
	}

	resRows := make([]getRelevantTaskEventsRow, 0)

	for taskEventRows.Next() {
		var row getRelevantTaskEventsRow

		err := taskEventRows.Scan(
			&row.TaskId,
			&row.Timestamp,
			&row.EventType,
			&row.ReadableStatus,
		)

		if err != nil {
			return nil, err
		}

		resRows = append(resRows, row)
	}

	return resRows, nil
}

func (r *olapEventRepository) ReadTaskRunMetrics(tenantId uuid.UUID, since time.Time) ([]olap.TaskRunMetric, error) {
	ctx := CreateContext()
	rows, err := r.conn.Query(ctx, `
		WITH candidate_tasks AS (
			SELECT *
			FROM task_events
			WHERE
				tenant_id = ?
				AND timestamp > ?
		), max_retry_counts AS (
			SELECT
				task_id,
				MAX(retry_count) AS max_retry_count
			FROM candidate_tasks
			GROUP BY task_id
		), final_status_events AS (
			SELECT ct.task_id, MAX(readable_status) AS readable_status
			FROM candidate_tasks ct
			JOIN max_retry_counts mrc ON ct.task_id = mrc.task_id AND ct.retry_count = mrc.max_retry_count
			GROUP BY ct.task_id
		)
		SELECT readable_status, COUNT(readable_status) AS count
		FROM final_status_events
		GROUP BY readable_status
		`,
		tenantId,
		since,
	)

	if err != nil {
		return []olap.TaskRunMetric{}, err
	}

	records := make([]olap.TaskRunMetric, 0)

	for rows.Next() {
		var (
			metric olap.TaskRunMetric
		)

		err = rows.Scan(
			&metric.Status,
			&metric.Count,
		)

		if err != nil {
			log.Fatal(err)
		}

		metric.Status = string(StringToReadableStatus(metric.Status))

		records = append(records, metric)
	}

	return records, nil
}

func (r *olapEventRepository) ReadTaskRun(tenantId, taskRunId uuid.UUID) (olap.WorkflowRun, error) {
	ctx := CreateContext()
	row := r.conn.QueryRow(ctx, `
		WITH max_retry_counts AS (
			SELECT task_id, MAX(retry_count) AS max_retry_count
			FROM task_events
			WHERE
				tenant_id = ?
				AND task_id = ?
			GROUP BY task_id
		), relevant_task_events AS (
			SELECT te.*
			FROM task_events te
			JOIN max_retry_counts mrc ON te.task_id = mrc.task_id AND te.retry_count = mrc.max_retry_count
			WHERE
				te.tenant_id = ?
				AND task_id = ?
		), task_creation_times AS (
			SELECT
				task_id,
				MIN(timestamp) AS value,
				MAX(readable_status) AS status
			FROM relevant_task_events
			GROUP BY task_id
		), task_start_times AS (
			SELECT
				task_id,
				MIN(timestamp) AS value
			FROM relevant_task_events
			WHERE event_type = 'STARTED'
			GROUP BY task_id
		), task_finish_times AS (
			SELECT
				task_id,
				MAX(timestamp) AS value,
				MAX(readable_status) AS status
			FROM relevant_task_events

			-- 3 indicates a terminal event. See enum definition
			WHERE readable_status >= 3
			GROUP BY task_id
		), task_event_metadata AS (
			SELECT
				tct.task_id AS task_id,
				tct.value AS created_at,
				tst.value AS started_at,
				tft.value AS finished_at,
				timeDiff(tst.value, tft.value) AS duration,
				tct.status AS status
			FROM task_creation_times tct
			LEFT JOIN task_start_times tst ON tct.task_id = tst.task_id
			LEFT JOIN task_finish_times tft ON tct.task_id = tft.task_id
		), error_messages AS (
			SELECT
				task_id,
				error_message
			FROM relevant_task_events
			WHERE readable_status = 'FAILED'
		), outputs AS (
			SELECT
				LAST_VALUE(task_id) AS task_id,
				LAST_VALUE(output) AS output
			FROM relevant_task_events
			WHERE readable_status = 'COMPLETED'
		)

		SELECT
			ct.additional_metadata,
			ct.display_name,
			tem.duration,
			em.error_message,
			tem.finished_at,
			ct.id AS id,
			ct.input,
			o.output,
			tem.started_at,
			tem.created_at,
			toString(tem.status) AS status,
			ct.id AS task_id,
			ct.tenant_id,
			-- NOTE: This is probably a bug, figure out which timestamp to use.
			ct.inserted_at AS timestamp,
			ct.workflow_id
		FROM tasks ct
		JOIN task_event_metadata tem ON ct.id = tem.task_id
		LEFT JOIN error_messages em ON ct.id = em.task_id
		LEFT JOIN outputs o ON ct.id = o.task_id
		WHERE ct.id = ?
		`,
		tenantId,
		taskRunId,
		tenantId,
		taskRunId,
		taskRunId,
	)

	var (
		taskRun olap.WorkflowRun
	)

	err := row.Scan(
		&taskRun.AdditionalMetadata,
		&taskRun.DisplayName,
		&taskRun.Duration,
		&taskRun.ErrorMessage,
		&taskRun.FinishedAt,
		&taskRun.Id,
		&taskRun.Input,
		&taskRun.Output,
		&taskRun.StartedAt,
		&taskRun.CreatedAt,
		&taskRun.Status,
		&taskRun.TaskId,
		&taskRun.TenantId,
		&taskRun.Timestamp,
		&taskRun.WorkflowId,
	)

	if err != nil {
		log.Fatal(err)
		return olap.WorkflowRun{}, err
	}

	taskRun.Status = string(StringToReadableStatus(taskRun.Status))

	return taskRun, nil
}

func (r *olapEventRepository) ReadTaskRunEvents(tenantId, taskRunId uuid.UUID, limit, offset int64) ([]olap.TaskRunEvent, error) {
	ctx := CreateContext()
	rows, err := r.conn.Query(ctx, `
		SELECT
			te.id,
			te.task_id,
			te.additional__event_message AS message,
			te.timestamp,
			te.additional__event_data,
			te.event_type,
			te.error_message,
			te.worker_id,
			t.display_name AS task_display_name,
			t.input AS task_input,
			t.additional_metadata
		FROM task_events te
		JOIN tasks t ON te.task_id = t.id
   		WHERE task_id = ? AND tenant_id = ?
		ORDER BY te.retry_count DESC, te.readable_status DESC
		`,
		taskRunId,
		tenantId,
	)

	if err != nil {
		return []olap.TaskRunEvent{}, err
	}

	records := make([]olap.TaskRunEvent, 0)

	for rows.Next() {
		var (
			taskRunEvent olap.TaskRunEvent
		)

		err := rows.Scan(
			&taskRunEvent.Id,
			&taskRunEvent.TaskId,
			&taskRunEvent.Message,
			&taskRunEvent.Timestamp,
			&taskRunEvent.Data,
			&taskRunEvent.EventType,
			&taskRunEvent.ErrorMsg,
			&taskRunEvent.WorkerId,
			&taskRunEvent.TaskDisplayName,
			&taskRunEvent.TaskInput,
			&taskRunEvent.AdditionalMetadata,
		)

		if err != nil {
			log.Fatal(err)
		}

		records = append(records, taskRunEvent)
	}

	return records, nil
}

func (r *olapEventRepository) saveEventsToCache(events []olap.TaskEvent) {
	for _, event := range events {
		key := getCacheKey(event)
		r.eventCache.Add(key, true)
	}
}

func getCacheKey(event olap.TaskEvent) string {
	// key on the task_id, retry_count, and event_type
	return fmt.Sprintf("%s-%s-%d", event.TaskId.String(), event.EventType, event.RetryCount)
}

func (r *olapEventRepository) writeTaskEventBatch(c context.Context, events []olap.TaskEvent) error {
	// skip any events which have a corresponding event already
	eventsToWrite := make([]olap.TaskEvent, 0)

	for _, event := range events {
		key := getCacheKey(event)

		if _, ok := r.eventCache.Get(key); !ok {
			eventsToWrite = append(eventsToWrite, event)
		}
	}

	// Clickhouse recommends using bulk (batch) inserts for performance
	// https://clickhouse.com/docs/en/integrations/go#batch-insert
	// If https://clickhouse.com/docs/en/cloud/bestpractices/bulk-inserts#ingest-data-in-bulk

	batch, err := r.conn.PrepareBatch(c, `
		INSERT INTO task_events (
			task_id,
			tenant_id,
			event_type,
			readable_status,
			timestamp,
			retry_count,
			error_message,
			output,
			worker_id,
			additional__event_data,
			additional__event_message
		)
	`)

	if err != nil {
		return err
	}

	for _, event := range eventsToWrite {
		readableStatus := olap.READABLE_TASK_STATUS_QUEUED

		switch event.EventType {
		case olap.EVENT_TYPE_REQUEUED_NO_WORKER:
			readableStatus = olap.READABLE_TASK_STATUS_QUEUED
		case olap.EVENT_TYPE_REQUEUED_RATE_LIMIT:
			readableStatus = olap.READABLE_TASK_STATUS_QUEUED
		case olap.EVENT_TYPE_SCHEDULING_TIMED_OUT:
			readableStatus = olap.READABLE_TASK_STATUS_FAILED
		case olap.EVENT_TYPE_ASSIGNED:
			readableStatus = olap.READABLE_TASK_STATUS_RUNNING
		case olap.EVENT_TYPE_STARTED:
			readableStatus = olap.READABLE_TASK_STATUS_RUNNING
		case olap.EVENT_TYPE_FINISHED:
			readableStatus = olap.READABLE_TASK_STATUS_COMPLETED
		case olap.EVENT_TYPE_FAILED:
			readableStatus = olap.READABLE_TASK_STATUS_FAILED
		case olap.EVENT_TYPE_RETRYING:
			readableStatus = olap.READABLE_TASK_STATUS_RUNNING
		case olap.EVENT_TYPE_CANCELLED:
			readableStatus = olap.READABLE_TASK_STATUS_CANCELLED
		case olap.EVENT_TYPE_TIMED_OUT:
			readableStatus = olap.READABLE_TASK_STATUS_FAILED
		case olap.EVENT_TYPE_REASSIGNED:
			readableStatus = olap.READABLE_TASK_STATUS_RUNNING
		case olap.EVENT_TYPE_SLOT_RELEASED:
			readableStatus = olap.READABLE_TASK_STATUS_QUEUED
		case olap.EVENT_TYPE_TIMEOUT_REFRESHED:
			readableStatus = olap.READABLE_TASK_STATUS_RUNNING
		case olap.EVENT_TYPE_RETRIED_BY_USER:
			readableStatus = olap.READABLE_TASK_STATUS_RUNNING
		case olap.EVENT_TYPE_SENT_TO_WORKER:
			readableStatus = olap.READABLE_TASK_STATUS_RUNNING
		case olap.EVENT_TYPE_RATE_LIMIT_ERROR:
			readableStatus = olap.READABLE_TASK_STATUS_FAILED
		case olap.EVENT_TYPE_ACKNOWLEDGED:
			readableStatus = olap.READABLE_TASK_STATUS_RUNNING
		case olap.EVENT_TYPE_CREATED:
			readableStatus = olap.READABLE_TASK_STATUS_QUEUED
		case olap.EVENT_TYPE_QUEUED:
			readableStatus = olap.READABLE_TASK_STATUS_QUEUED
		}

		err := batch.Append(
			event.TaskId,
			event.TenantId,
			string(event.EventType),
			string(readableStatus),
			event.Timestamp,
			event.RetryCount,
			event.ErrorMsg,
			event.Output,
			event.WorkerId,
			event.AdditionalEventData,
			event.AdditionalEventMessage,
		)

		if err != nil {
			return err
		}
	}

	err = batch.Send()

	if err != nil {
		return err
	}

	r.saveEventsToCache(eventsToWrite)

	return nil
}

func (r *olapEventRepository) writeTaskBatch(c context.Context, tasks []olap.Task) error {
	// Clickhouse recommends using bulk (batch) inserts for performance
	// https://clickhouse.com/docs/en/integrations/go#batch-insert
	// If https://clickhouse.com/docs/en/cloud/bestpractices/bulk-inserts#ingest-data-in-bulk
	batch, err := r.conn.PrepareBatch(c, `
		INSERT INTO tasks (
			id,
			source_id,
			tenant_id,
			workflow_id,
			queue,
			action_id,
			schedule_timeout,
			step_timeout,
			priority,
			sticky,
			desired_worker_id,
			display_name,
			input,
			additional_metadata,
			inserted_at
		)
	`)

	if err != nil {
		return err
	}

	for _, task := range tasks {
		err := batch.Append(
			task.Id,
			task.SourceId,
			task.TenantId,
			task.WorkflowId,
			task.Queue,
			task.ActionId,
			task.ScheduleTimeout,
			task.StepTimeout,
			task.Priority,
			string(task.Sticky),
			task.DesiredWorkerId,
			task.DisplayName,
			task.Input,
			task.AdditionalMetadata,
			task.InsertedAt,
		)

		if err != nil {
			return err
		}
	}

	return batch.Send()
}

func (r *olapEventRepository) CreateTaskEvents(events []olap.TaskEvent) error {
	return r.writeTaskEventBatch(context.Background(), events)
}

func (r *olapEventRepository) CreateTasks(tasks []olap.Task) error {
	return r.writeTaskBatch(context.Background(), tasks)
}
