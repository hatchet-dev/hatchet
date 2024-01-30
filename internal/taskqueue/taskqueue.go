package taskqueue

import (
	"context"
)

type QueueType string

const (
	EVENT_PROCESSING_QUEUE    QueueType = "event_processing_queue"
	JOB_PROCESSING_QUEUE      QueueType = "job_processing_queue"
	WORKFLOW_PROCESSING_QUEUE QueueType = "workflow_processing_queue"
	DISPATCHER_POOL_QUEUE     QueueType = "dispatcher_pool_queue"
	SCHEDULING_QUEUE          QueueType = "scheduling_queue"
)

func QueueTypeFromDispatcherID(d string) QueueType {
	return QueueType(d)
}

func QueueTypeFromTickerID(t string) QueueType {
	return QueueType(t)
}

type Task struct {
	// ID is the ID of the task.
	ID string `json:"id"`

	// Queue is the queue of the task.
	Queue QueueType `json:"queue"`

	// Payload is the payload of the task.
	Payload map[string]interface{} `json:"payload"`

	// Metadata is the metadata of the task.
	Metadata map[string]interface{} `json:"metadata"`

	// Retries is the number of retries for the task.
	Retries int `json:"retries"`

	// RetryDelay is the delay between retries.
	RetryDelay int `json:"retry_delay"`
}

type TaskQueue interface {
	// AddTask adds a task to the queue. Implementations should ensure that Start().
	AddTask(ctx context.Context, queue QueueType, task *Task) error

	// Subscribe subscribes to the task queue.
	Subscribe(ctx context.Context, queueType QueueType) (<-chan *Task, error)
}
