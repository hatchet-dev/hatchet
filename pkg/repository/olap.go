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
	ReadTaskRuns(tenantId uuid.UUID) ([]olap.WorkflowRun, error)
	ReadTaskRun(stepRunId int) (olap.WorkflowRun, error)
	CreateEvent(event olap.Event) error
	CreateEvents(events []olap.Event) error
}

type olapEventRepository struct {
	conn   clickhouse.Conn
	buffer *buffer.TenantBufferManager[olap.Event, olap.Event]
}

func NewOLAPEventRepository() OLAPEventRepository {
	opts := buffer.TenantBufManagerOpts[olap.Event, olap.Event]{
		Name:       "create-event-buffer",
		OutputFunc: WriteEventBatch,
		L:          &zerolog.Logger{},
		Config: buffer.ConfigFileBuffer{
			FlushPeriodMilliseconds: 500,
		},
		SizeFunc: func(e olap.Event) int {
			return 16 + 16 + len(e.Status)
		},
		V: validator.NewDefaultValidator(),
	}

	buff, err := buffer.NewTenantBufManager(opts)

	defer buff.Cleanup()

	if err != nil {
		log.Fatal(err)
	}

	conn, err := olap.CreateClickhouseConnection()

	if err != nil {
		log.Fatal(err)
	}

	return &olapEventRepository{
		conn:   conn,
		buffer: buff,
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

func (r *olapEventRepository) ReadTaskRuns(tenantId uuid.UUID) ([]olap.WorkflowRun, error) {
	ctx := context.Background()
	rows, err := r.conn.Query(ctx, `
		WITH rows_assigned AS (
			SELECT id, ROW_NUMBER() OVER (PARTITION BY task_id ORDER BY timestamp DESC) AS row_num
			FROM events
			WHERE tenant_id = ?
		)
		SELECT
			e.id,
			e.task_id,
			e.worker_id,
			e.tenant_id,
			e.status,
			e.timestamp,
			e.created_at,
			e.retry_count,
			e.error_message
		FROM events e
		JOIN rows_assigned ra ON e.id = ra.id
		WHERE ra.row_num = 1`,
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
			&taskRun.Id,
			&taskRun.TaskId,
			&taskRun.WorkerId,
			&taskRun.TenantId,
			&taskRun.Status,
			&taskRun.Timestamp,
			&taskRun.CreatedAt,
			&taskRun.RetryCount,
			&taskRun.ErrorMessage,
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
			worker_id,
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
		&taskRun.Id,
		&taskRun.TaskId,
		&taskRun.WorkerId,
		&taskRun.TenantId,
		&taskRun.Status,
		&taskRun.Timestamp,
		&taskRun.CreatedAt,
		&taskRun.RetryCount,
		&taskRun.ErrorMessage,
	)

	if err != nil {
		return olap.WorkflowRun{}, err
	}

	return taskRun, nil
}

func WriteEventBatch(c context.Context, events []olap.Event) ([]*olap.Event, error) {
	conn, err := olap.CreateClickhouseConnection()

	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	// Clickhouse recommends using bulk (batch) inserts for performance
	// https://clickhouse.com/docs/en/integrations/go#batch-insert
	// If https://clickhouse.com/docs/en/cloud/bestpractices/bulk-inserts#ingest-data-in-bulk
	batch, err := conn.PrepareBatch(ctx, `
		INSERT INTO events (
			task_id,
			worker_id,
			tenant_id,
			status,
			timestamp,
			retry_count,
			error_message
		)
	`)

	if err != nil {
		return nil, err
	}

	for _, event := range events {
		err := batch.Append(
			event.TaskId,
			event.WorkerId,
			event.TenantId,
			event.Status,
			event.Timestamp,
			event.RetryCount,
			event.ErrorMsg,
		)

		if err != nil {
			return nil, err
		}
	}

	err = batch.Send()

	if err != nil {
		return nil, err
	}

	eventPtrs := make([]*olap.Event, len(events))

	for i := range events {
		eventPtrs[i] = &events[i]
	}

	return eventPtrs, nil
}

func (r *olapEventRepository) CreateEvent(event olap.Event) error {
	_, err := r.buffer.FireAndWait(context.Background(), event.TenantId.String(), event)

	return err
}

func (r *olapEventRepository) CreateEvents(events []olap.Event) error {
	_, err := WriteEventBatch(context.Background(), events)

	return err
}
