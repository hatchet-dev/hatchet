package repository

import (
	"context"
	"log"

	"github.com/google/uuid"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/hatchet-dev/hatchet/pkg/repository/buffer"
	"github.com/hatchet-dev/hatchet/pkg/repository/olap"
	"github.com/hatchet-dev/hatchet/pkg/validator"
	"github.com/rs/zerolog"
)

type OLAPEventRepository interface {
	Connect(ctx context.Context) error
	ReadTaskRuns(tenantId uuid.UUID, limit, offset int64) ([]olap.WorkflowRun, error)
	ReadTaskRun(stepRunId int) (olap.WorkflowRun, error)
	CreateTask(task olap.Task) error
	CreateTasks(tasks []olap.Task) error
	CreateTaskEvent(event olap.TaskEvent) error
	CreateTaskEvents(events []olap.TaskEvent) error
}

type olapEventRepository struct {
	conn            clickhouse.Conn
	taskBuffer      *buffer.TenantBufferManager[olap.Task, olap.Task]
	taskEventBuffer *buffer.TenantBufferManager[olap.TaskEvent, olap.TaskEvent]
}

func NewOLAPEventRepository() OLAPEventRepository {
	task_event_buffer_opts := buffer.TenantBufManagerOpts[olap.TaskEvent, olap.TaskEvent]{
		Name:       "task-event-buffer",
		OutputFunc: WriteTaskEventBatch,
		L:          &zerolog.Logger{},
		Config: buffer.ConfigFileBuffer{
			FlushPeriodMilliseconds: 500,
		},
		SizeFunc: func(e olap.TaskEvent) int {
			return 16 + 16 + len(e.Status)
		},
		V: validator.NewDefaultValidator(),
	}

	task_event_buffer, err := buffer.NewTenantBufManager(task_event_buffer_opts)

	defer task_event_buffer.Cleanup()

	if err != nil {
		log.Fatal(err)
	}

	task_buffer_opts := buffer.TenantBufManagerOpts[olap.Task, olap.Task]{
		Name:       "task-event-buffer",
		OutputFunc: WriteTaskBatch,
		L:          &zerolog.Logger{},
		Config: buffer.ConfigFileBuffer{
			FlushPeriodMilliseconds: 500,
		},
		SizeFunc: func(e olap.Task) int {
			return 16 + 16 + len(e.DisplayName)
		},
		V: validator.NewDefaultValidator(),
	}

	task_buffer, err := buffer.NewTenantBufManager(task_buffer_opts)

	defer task_buffer.Cleanup()

	if err != nil {
		log.Fatal(err)
	}

	conn, err := olap.CreateClickhouseConnection()

	if err != nil {
		log.Fatal(err)
	}

	return &olapEventRepository{
		conn:            conn,
		taskBuffer:      task_buffer,
		taskEventBuffer: task_event_buffer,
	}
}

func (r *olapEventRepository) Connect(ctx context.Context) error {
	conn, err := olap.CreateClickhouseConnection()
	if err != nil {
		return err
	}
	r.conn = conn
	return nil
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
		), task_statuses AS (
			SELECT task_id, LAST_VALUE(status) AS status
			FROM task_events
			WHERE
				tenant_id = ?
				AND task_id IN (SELECT id FROM candidate_tasks)
			GROUP BY task_id
		), task_durations AS (
			SELECT
				task_id,
				-- NOTE: These are bugs, should filter by status to determine these
				MIN(timestamp) AS started_at,
				MAX(timestamp) AS finished_at,
				MAX(timestamp) - MIN(timestamp) AS duration
			FROM task_events
			WHERE
				tenant_id = ?
				AND task_id IN (SELECT id FROM candidate_tasks)
			GROUP BY task_id
		), error_messages AS (
			SELECT
				task_id,
				error_message
			FROM task_events
			WHERE
				tenant_id = ?
				AND task_id IN (SELECT id FROM candidate_tasks)
				AND status = 'FAILED'
		)

		SELECT
			ct.additional_metadata,
			ct.display_name,
			td.duration,
			em.error_message,
			td.finished_at,
			ct.id AS id,
			td.started_at,
			ts.status,
			ct.id AS task_id,
			ct.tenant_id,
			-- NOTE: This is probably a bug, figure out which timestamp to use.
			ct.created_at AS timestamp
		FROM candidate_tasks ct
		JOIN task_statuses ts ON ct.id = ts.task_id
		JOIN task_durations td ON ct.id = td.task_id
		LEFT JOIN error_messages em ON ct.id = em.task_id
		ORDER BY ct.created_at DESC
		`,
		tenantId,
		limit,
		offset,
		tenantId,
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
			&taskRun.StartedAt,
			&taskRun.Status,
			&taskRun.TaskId,
			&taskRun.TenantId,
			&taskRun.Timestamp,
		)

		if err != nil {
			log.Fatal(err)
		}

		records = append(records, taskRun)
	}

	return records, nil
}

func (r *olapEventRepository) ReadTaskRun(taskRunId int) (olap.WorkflowRun, error) {
	ctx := context.Background()
	row := r.conn.QueryRow(ctx, `
		SELECT
			id,
			task_id,
			tenant_id,
			status,
			timestamp,
			created_at,
			retry_count,
			error_message
		FROM events
   		WHERE task_id = ?
		`,
		taskRunId,
	)

	var taskRun olap.WorkflowRun

	err := row.Scan(
		&taskRun.TaskId,
		&taskRun.TenantId,
		&taskRun.Status,
		&taskRun.Timestamp,
	)

	if err != nil {
		return olap.WorkflowRun{}, err
	}

	return taskRun, nil
}

func WriteTaskEventBatch(c context.Context, events []olap.TaskEvent) ([]*olap.TaskEvent, error) {
	conn, err := olap.CreateClickhouseConnection()

	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	// Clickhouse recommends using bulk (batch) inserts for performance
	// https://clickhouse.com/docs/en/integrations/go#batch-insert
	// If https://clickhouse.com/docs/en/cloud/bestpractices/bulk-inserts#ingest-data-in-bulk
	batch, err := conn.PrepareBatch(ctx, `
		INSERT INTO task_events (
			task_id,
			tenant_id,
			status,
			timestamp,
			retry_count,
			error_message,
			output,
			additional__event_data,
			additional__event_message,
			additional__event_severity,
			additional__event_reason
		)
	`)

	if err != nil {
		return nil, err
	}

	for _, event := range events {
		err := batch.Append(
			event.TaskId,
			event.TenantId,
			event.Status,
			event.Timestamp,
			event.RetryCount,
			event.ErrorMsg,
			event.Output,
			event.AdditionalEventData,
			event.AdditionalEventMessage,
			event.AdditionalEventSeverity,
			event.AdditionalEventReason,
		)

		if err != nil {
			return nil, err
		}
	}

	err = batch.Send()

	if err != nil {
		return nil, err
	}

	eventPtrs := make([]*olap.TaskEvent, len(events))

	for i := range events {
		eventPtrs[i] = &events[i]
	}

	return eventPtrs, nil
}

func WriteTaskBatch(c context.Context, tasks []olap.Task) ([]*olap.Task, error) {
	conn, err := olap.CreateClickhouseConnection()

	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	// Clickhouse recommends using bulk (batch) inserts for performance
	// https://clickhouse.com/docs/en/integrations/go#batch-insert
	// If https://clickhouse.com/docs/en/cloud/bestpractices/bulk-inserts#ingest-data-in-bulk
	batch, err := conn.PrepareBatch(ctx, `
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
		return nil, err
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
			task.Sticky,
			task.DesiredWorkerId,
			task.DisplayName,
			task.Input,
			task.AdditionalMetadata,
		)

		if err != nil {
			return nil, err
		}
	}

	err = batch.Send()

	if err != nil {
		return nil, err
	}

	taskPtrs := make([]*olap.Task, len(tasks))

	for i := range tasks {
		taskPtrs[i] = &tasks[i]
	}

	return taskPtrs, nil
}

func (r *olapEventRepository) CreateTaskEvent(event olap.TaskEvent) error {
	_, err := r.taskEventBuffer.FireAndWait(context.Background(), event.TenantId.String(), event)

	return err
}

func (r *olapEventRepository) CreateTaskEvents(events []olap.TaskEvent) error {
	_, err := WriteTaskEventBatch(context.Background(), events)

	return err
}

func (r *olapEventRepository) CreateTask(task olap.Task) error {
	_, err := r.taskBuffer.FireAndWait(context.Background(), task.TenantId.String(), task)

	return err
}

func (r *olapEventRepository) CreateTasks(tasks []olap.Task) error {
	_, err := WriteTaskBatch(context.Background(), tasks)

	return err
}
