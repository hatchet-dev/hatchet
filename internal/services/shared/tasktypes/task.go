package tasktypes

import (
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/pkg/repository/v2/timescalev2"
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
	return msgqueue.NewSingletonTenantMessage(
		tenantId,
		"task-trigger",
		TriggerTaskPayload{
			TaskExternalId:     taskExternalId,
			WorkflowName:       name,
			Data:               data,
			AdditionalMetadata: additionalMetadata,
			ParentTaskId:       parentTaskId,
			ChildIndex:         childIndex,
			ChildKey:           childKey,
		},
		false,
		true,
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
	return msgqueue.NewSingletonTenantMessage(
		tenantId,
		"task-completed",
		CompletedTaskPayload{
			TaskId:     taskId,
			RetryCount: retryCount,
			Output:     output,
		},
		false,
		true,
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
	return msgqueue.NewSingletonTenantMessage(
		tenantId,
		"task-failed",
		FailedTaskPayload{
			TaskId:     taskId,
			RetryCount: retryCount,
			IsAppError: isAppError,
			ErrorMsg:   errorMsg,
		},
		false,
		true,
	)
}

type CancelledTaskPayload struct {
	// (required) the task id
	TaskId int64 `validate:"required"`

	// (required) the retry count
	RetryCount int32

	// (required) the reason for cancellation
	EventType timescalev2.V2EventTypeOlap
}

func CancelledTaskMessage(tenantId string, taskId int64, retryCount int32, eventType timescalev2.V2EventTypeOlap) (*msgqueue.Message, error) {
	return msgqueue.NewSingletonTenantMessage(
		tenantId,
		"task-cancelled",
		CancelledTaskPayload{
			TaskId:     taskId,
			RetryCount: retryCount,
			EventType:  eventType,
		},
		false,
		true,
	)
}
