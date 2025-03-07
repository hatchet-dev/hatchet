package v1

import (
	"github.com/jackc/pgx/v5/pgtype"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func TriggerTaskMessage(tenantId string, payloads ...*v1.WorkflowNameTriggerOpts) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		"task-trigger",
		false,
		true,
		payloads...,
	)
}

type CompletedTaskPayload struct {
	// (required) the task id
	TaskId int64 `validate:"required"`

	// (required) the task inserted at
	InsertedAt pgtype.Timestamptz

	// (required) the task external id
	ExternalId string

	// (required) the workflow run id
	WorkflowRunId string

	// (required) the retry count
	RetryCount int32

	// (optional) the output data
	Output []byte
}

func CompletedTaskMessage(
	tenantId string,
	taskId int64,
	taskInsertedAt pgtype.Timestamptz,
	taskExternalId string,
	workflowRunId string,
	retryCount int32,
	output []byte,
) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		"task-completed",
		false,
		true,
		CompletedTaskPayload{
			TaskId:        taskId,
			InsertedAt:    taskInsertedAt,
			ExternalId:    taskExternalId,
			WorkflowRunId: workflowRunId,
			RetryCount:    retryCount,
			Output:        output,
		},
	)
}

type FailedTaskPayload struct {
	// (required) the task id
	TaskId int64 `validate:"required"`

	// (required) the task inserted at
	InsertedAt pgtype.Timestamptz

	// (required) the task external id
	ExternalId string

	// (required) the workflow run id
	WorkflowRunId string

	// (required) the retry count
	RetryCount int32

	// (required) whether this is an application-level error or an internal error on the Hatchet side
	IsAppError bool

	// (optional) the error message
	ErrorMsg string
}

func FailedTaskMessage(
	tenantId string,
	taskId int64,
	taskInsertedAt pgtype.Timestamptz,
	taskExternalId string,
	workflowRunId string,
	retryCount int32,
	isAppError bool,
	errorMsg string,
) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		"task-failed",
		false,
		true,
		FailedTaskPayload{
			TaskId:        taskId,
			InsertedAt:    taskInsertedAt,
			ExternalId:    taskExternalId,
			WorkflowRunId: workflowRunId,
			RetryCount:    retryCount,
			IsAppError:    isAppError,
			ErrorMsg:      errorMsg,
		},
	)
}

type CancelledTaskPayload struct {
	// (required) the task id
	TaskId int64 `validate:"required"`

	// (required) the task inserted at
	InsertedAt pgtype.Timestamptz

	// (required) the task external id
	ExternalId string

	// (required) the workflow run id
	WorkflowRunId string

	// (required) the retry count
	RetryCount int32

	// (required) the reason for cancellation
	EventType sqlcv1.V1EventTypeOlap

	// (optional) whether the task should notify the worker
	ShouldNotify bool
}

func CancelledTaskMessage(
	tenantId string,
	taskId int64,
	taskInsertedAt pgtype.Timestamptz,
	taskExternalId string,
	workflowRunId string,
	retryCount int32,
	eventType sqlcv1.V1EventTypeOlap,
	shouldNotify bool,
) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		"task-cancelled",
		false,
		true,
		CancelledTaskPayload{
			TaskId:        taskId,
			InsertedAt:    taskInsertedAt,
			ExternalId:    taskExternalId,
			WorkflowRunId: workflowRunId,
			RetryCount:    retryCount,
			EventType:     eventType,
			ShouldNotify:  shouldNotify,
		},
	)
}

type SignalTaskCancelledPayload struct {
	// (required) the worker id
	WorkerId string `validate:"required,uuid"`

	// (required) the task id
	TaskId int64 `validate:"required"`

	// (required) the task inserted at
	InsertedAt pgtype.Timestamptz

	// (required) the retry count
	RetryCount int32
}

type CancelTasksPayload struct {
	Tasks []v1.TaskIdInsertedAtRetryCount `json:"tasks"`
}

type ReplayTasksPayload struct {
	Tasks []v1.TaskIdInsertedAtRetryCount `json:"tasks"`
}

type NotifyFinalizedPayload struct {
	// (required) the external id (can either be a workflow run id or single task)
	ExternalId string `validate:"required"`

	// (required) the status of the task
	Status sqlcv1.V1ReadableStatusOlap
}
