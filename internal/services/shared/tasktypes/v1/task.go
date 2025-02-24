package v1

import (
	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

type TriggerTaskPayload struct {
	// (required) the task id
	TaskExternalId string `json:"task_external_id" validate:"required,uuid"`

	// (required) the workflow name
	WorkflowName string `json:"workflow_name" validate:"required"`

	// (optional) the workflow run data
	Data []byte `json:"data"`

	// (optional) the workflow run metadata
	AdditionalMetadata []byte `json:"additional_metadata"`

	// (optional) the parent task id
	ParentTaskId *int64 `json:"parent_task_id"`

	// (optional) the child index
	ChildIndex *int64 `json:"child_index"`

	// (optional) the child key
	ChildKey *string `json:"child_key"`
}

func TriggerTaskMessage(tenantId string, taskExternalId, name string, data []byte, additionalMetadata []byte, parentTaskId *int64, childIndex *int64, childKey *string) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		"task-trigger",
		false,
		true,
		TriggerTaskPayload{
			TaskExternalId:     taskExternalId,
			WorkflowName:       name,
			Data:               data,
			AdditionalMetadata: additionalMetadata,
			ParentTaskId:       parentTaskId,
			ChildIndex:         childIndex,
			ChildKey:           childKey,
		},
	)
}

type CompletedTaskPayload struct {
	// (required) the task id
	TaskId int64 `validate:"required"`

	// (required) the retry count
	RetryCount int32

	// (optional) the output data
	Output []byte
}

func CompletedTaskMessage(tenantId string, taskId int64, retryCount int32, output []byte) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		"task-completed",
		false,
		true,
		CompletedTaskPayload{
			TaskId:     taskId,
			RetryCount: retryCount,
			Output:     output,
		},
	)
}

type FailedTaskPayload struct {
	// (required) the task id
	TaskId int64 `validate:"required"`

	// (required) the retry count
	RetryCount int32

	// (required) whether this is an application-level error or an internal error on the Hatchet side
	IsAppError bool

	// (optional) the error message
	ErrorMsg string
}

func FailedTaskMessage(tenantId string, taskId int64, retryCount int32, isAppError bool, errorMsg string) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		"task-failed",
		false,
		true,
		FailedTaskPayload{
			TaskId:     taskId,
			RetryCount: retryCount,
			IsAppError: isAppError,
			ErrorMsg:   errorMsg,
		},
	)
}

type CancelledTaskPayload struct {
	// (required) the task id
	TaskId int64 `validate:"required"`

	// (required) the retry count
	RetryCount int32

	// (required) the reason for cancellation
	EventType sqlcv1.V1EventTypeOlap

	// (optional) whether the task should notify the worker
	ShouldNotify bool
}

func CancelledTaskMessage(tenantId string, taskId int64, retryCount int32, eventType sqlcv1.V1EventTypeOlap, shouldNotify bool) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		"task-cancelled",
		false,
		true,
		CancelledTaskPayload{
			TaskId:       taskId,
			RetryCount:   retryCount,
			EventType:    eventType,
			ShouldNotify: shouldNotify,
		},
	)
}

type SignalTaskCancelledPayload struct {
	// (required) the worker id
	WorkerId string `validate:"required,uuid"`

	// (required) the task id
	TaskId int64 `validate:"required"`

	// (required) the retry count
	RetryCount int32
}
