package repository

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository/buffer"
)

type WorkflowRun struct {
	Id           uuid.UUID `json:"id"`
	TaskId       int32     `json:"task_id"`
	WorkerId     int32     `json:"worker_id"`
	TenantId     uuid.UUID `json:"tenant_id"`
	Status       string    `json:"status"`
	Timestamp    time.Time `json:"timestamp"`
	CreatedAt    time.Time `json:"created_at"`
	RetryCount   int32     `json:"retry_count"`
	ErrorMessage string    `json:"error_message"`
}

type Event struct {
	TaskId     int32     `json:"task_id"`
	WorkerId   int32     `json:"worker_id"`
	TenantId   uuid.UUID `json:"tenant_id"`
	Status     string    `json:"status"`
	Timestamp  time.Time `json:"timestamp"`
	RetryCount int32     `json:"retry_count"`
	ErrorMsg   string    `json:"error_message"`
}

type ClickhouseRepository interface {
	Connect() (*clickhouse.Conn, error)
	ReadTaskRun(taskId int) (WorkflowRun, error)
	ReadTaskRuns(tenantId string) ([]WorkflowRun, error)
	CreateEvent(events []Event) error
}

type OLAPEventRepository interface {
	ListStepExpressions(ctx context.Context, stepId string) ([]*WorkflowRun, error)
}

func Connect() (clickhouse.Conn, error) {
	return clickhouse.Open(&clickhouse.Options{
		Addr: []string{os.Getenv("CLICKHOUSE_SECURE_NATIVE_HOSTNAME") + ":9440"},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: os.Getenv("CLICKHOUSE_PASSWORD"),
		},

		Debugf: func(format string, v ...interface{}) {
			fmt.Printf(format, v)
		},
	})
}

func ReadTaskRuns(tenantId uuid.UUID) ([]WorkflowRun, error) {
	conn, err := Connect()

	if err != nil {
		return []WorkflowRun{}, err
	}

	ctx := context.Background()
	rows, err := conn.Query(ctx, `
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
		return []WorkflowRun{}, err
	}

	records := make([]WorkflowRun, 0)

	for rows.Next() {
		var (
			taskRun WorkflowRun
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

func ReadTaskRun(taskRunId int) (WorkflowRun, error) {
	conn, err := Connect()

	if err != nil {
		return WorkflowRun{}, err
	}

	ctx := context.Background()
	row := conn.QueryRow(ctx, `
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

	var taskRun WorkflowRun

	err = row.Scan(
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
		return WorkflowRun{}, err
	}

	return taskRun, nil
}

func WriteEvents(c context.Context, events []Event) ([]*Event, error) {
	conn, _ := Connect()
	ctx := context.Background()

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

	eventPtrs := make([]*Event, len(events))

	for i := range events {
		eventPtrs[i] = &events[i]
	}

	return eventPtrs, nil
}

func CreateEvent(e Event, b *buffer.IngestBuf[Event, Event]) error {
	return b.FireForget(e)
}

func Example() error {
	opts := buffer.IngestBufOpts[Event, Event]{
		Name:               "create-event-buffer",
		MaxCapacity:        5,
		FlushPeriod:        5 * time.Second,
		MaxDataSizeInQueue: 10_000,
		OutputFunc:         WriteEvents,
		L:                  &zerolog.Logger{},
		FlushStrategy:      buffer.Dynamic,
	}

	buff := buffer.NewIngestBuffer(opts)

	event := Event{
		TaskId:     1,
		WorkerId:   1,
		TenantId:   uuid.New(),
		Status:     "success",
		Timestamp:  time.Now(),
		RetryCount: 0,
		ErrorMsg:   "",
	}

	return CreateEvent(event, buff)
}
