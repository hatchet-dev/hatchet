package repository

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/hatchet-dev/hatchet/pkg/repository/olap"
)

type OLAPEventRepository interface {
	ReadTaskRun(tenantId, taskRunId uuid.UUID) (olap.WorkflowRun, error)
	ReadTaskRuns(tenantId uuid.UUID, limit, offset int64) ([]olap.WorkflowRun, error)
	ReadTaskRunEvents(tenantId, taskId uuid.UUID, limit, offset int64) ([]olap.TaskRunEvent, error)
	ReadTaskRunMetrics(tenantId uuid.UUID, since time.Time) ([]olap.TaskRunMetric, error)
	CreateTasks(tasks []olap.Task) error
	CreateTaskEvents(events []olap.TaskEvent) error
}

type olapEventRepository struct {
	conn clickhouse.Conn
}

func NewOLAPEventRepository() OLAPEventRepository {
	conn, err := olap.CreateClickhouseConnection()

	if err != nil {
		log.Fatal(err)
	}

	return &olapEventRepository{
		conn: conn,
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

func (r *olapEventRepository) ReadTaskRuns(tenantId uuid.UUID, limit, offset int64) ([]olap.WorkflowRun, error) {
	ctx := context.Background()
	rows, err := r.conn.Query(ctx, `
		WITH candidate_tasks AS (
			SELECT *
			FROM tasks
			WHERE tenant_id = ?
			ORDER BY created_at DESC
			LIMIT ?
			OFFSET ?
		), max_retry_counts AS (
			SELECT task_id, MAX(retry_count) AS max_retry_count
			FROM task_events
			WHERE
				tenant_id = ?
				AND task_id IN (SELECT id FROM candidate_tasks)
			GROUP BY task_id
		), relevant_task_events AS (
			SELECT te.*
			FROM task_events te
			JOIN max_retry_counts mrc ON te.task_id = mrc.task_id AND te.retry_count = mrc.max_retry_count
			WHERE
				te.tenant_id = ?
				AND task_id IN (SELECT id FROM candidate_tasks)
				-- Filter for the max retry count within the task here
				-- AND status IN (
				--  ...
				-- )
		), task_creation_times AS (
			SELECT
				task_id,
				MIN(timestamp) AS created_at
			FROM relevant_task_events
			GROUP BY task_id
		), task_start_times AS (
			SELECT
				task_id,
				MIN(timestamp) AS started_at
			FROM relevant_task_events
			WHERE event_type = 'STARTED'
			GROUP BY task_id
		), task_finish_times AS (
			SELECT
				task_id,
				MAX(timestamp) AS finished_at,
				MAX(readable_status) AS status
			FROM relevant_task_events

			-- 3 indicates a terminal event. See enum definition
			WHERE event_type >= 3
			GROUP BY task_id
		), task_event_metadata AS (
			SELECT
				tct.task_id AS task_id,
				tct.created_at AS created_at,
				tst.started_at AS started_at,
				tft.finished_at AS finished_at,
				timeDiff(tst.started_at, tft.finished_at) * 1000 AS duration,
				tft.status AS status
			FROM task_creation_times tct
			JOIN task_start_times tst ON tct.task_id = tst.task_id
			JOIN task_finish_times tft ON tct.task_id = tft.task_id
		), error_messages AS (
			SELECT
				task_id,
				error_message
			FROM relevant_task_events
			WHERE readable_status = 'FAILED'
		), outputs AS (
			SELECT
				task_id,
				LAST_VALUE(output) OVER(PARTITION BY task_id ORDER BY retry_count) AS output
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
			ct.created_at AS timestamp
		FROM candidate_tasks ct
		JOIN task_event_metadata tem ON ct.id = tem.task_id
		LEFT JOIN error_messages em ON ct.id = em.task_id
		LEFT JOIN outputs o ON ct.id = o.task_id
		ORDER BY ct.created_at DESC
		`,
		tenantId,
		limit,
		offset,
		tenantId,
		tenantId,
	)

	if err != nil {
		return []olap.WorkflowRun{}, err
	}

	records := make([]olap.WorkflowRun, 0)

	for rows.Next() {
		var (
			taskRun olap.WorkflowRun
		)

		err = rows.Scan(
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
		)

		if err != nil {
			log.Fatal(err)
		}

		taskRun.Status = string(StringToReadableStatus(taskRun.Status))

		records = append(records, taskRun)
	}

	return records, nil
}

func (r *olapEventRepository) ReadTaskRunMetrics(tenantId uuid.UUID, since time.Time) ([]olap.TaskRunMetric, error) {
	ctx := context.Background()
	rows, err := r.conn.Query(ctx, `
		WITH candidate_tasks AS (
			SELECT *
			FROM tasks
			WHERE tenant_id = ? AND created_at > ?
		), max_retry_counts AS (
			SELECT task_id, MAX(retry_count) AS max_retry_count
			FROM task_events
			WHERE
				tenant_id = ?
				AND task_id IN (SELECT id FROM candidate_tasks)
			GROUP BY task_id
		)

		SELECT te.readable_status, COUNT(te.readable_status) AS count
		FROM task_events te
		JOIN max_retry_counts mrc ON te.task_id = mrc.task_id AND te.retry_count = mrc.max_retry_count
		WHERE
			te.tenant_id = ?
			AND te.task_id IN (SELECT id FROM candidate_tasks)
		GROUP BY te.readable_status
		`,
		tenantId,
		since,
		tenantId,
		tenantId,
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
	ctx := context.Background()
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
				MIN(timestamp) AS created_at
			FROM relevant_task_events
			GROUP BY task_id
		), task_start_times AS (
			SELECT
				task_id,
				MIN(timestamp) AS started_at
			FROM relevant_task_events
			WHERE event_type = 'STARTED'
			GROUP BY task_id
		), task_finish_times AS (
			SELECT
				task_id,
				MAX(timestamp) AS finished_at,
				MAX(readable_status) AS status
			FROM relevant_task_events

			-- 3 indicates a terminal event. See enum definition
			WHERE event_type >= 3
			GROUP BY task_id
		), task_event_metadata AS (
			SELECT
				tct.task_id AS task_id,
				tct.created_at AS created_at,
				tst.started_at AS started_at,
				tft.finished_at AS finished_at,
				timeDiff(tst.started_at, tft.finished_at) * 1000 AS duration,
				tft.status AS status
			FROM task_creation_times tct
			JOIN task_start_times tst ON tct.task_id = tst.task_id
			JOIN task_finish_times tft ON tct.task_id = tft.task_id
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
			ct.created_at AS timestamp
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
	)

	if err != nil {
		log.Fatal(err)
		return olap.WorkflowRun{}, err
	}

	taskRun.Status = string(StringToReadableStatus(taskRun.Status))

	return taskRun, nil
}

func (r *olapEventRepository) ReadTaskRunEvents(tenantId, taskRunId uuid.UUID, limit, offset int64) ([]olap.TaskRunEvent, error) {
	ctx := context.Background()
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
		ORDER BY te.retry_count, te.readable_status DESC
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

func (r *olapEventRepository) writeTaskEventBatch(c context.Context, events []olap.TaskEvent) error {
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

	for _, event := range events {
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

	return batch.Send()
}

func (r *olapEventRepository) writeTaskBatch(c context.Context, tasks []olap.Task) error {
	// Clickhouse recommends using bulk (batch) inserts for performance
	// https://clickhouse.com/docs/en/integrations/go#batch-insert
	// If https://clickhouse.com/docs/en/cloud/bestpractices/bulk-inserts#ingest-data-in-bulk
	batch, err := r.conn.PrepareBatch(c, `
		INSERT INTO tasks (
			id,
			tenant_id,
			queue,
			action_id,
			schedule_timeout,
			step_timeout,
			priority,
			sticky,
			desired_worker_id,
			display_name,
			input,
			additional_metadata
		)
	`)

	if err != nil {
		return err
	}

	for _, task := range tasks {
		err := batch.Append(
			task.Id,
			task.TenantId,
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
