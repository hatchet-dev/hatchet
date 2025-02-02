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
}

func TriggerTaskMessage(tenantId string, taskExternalId, name string, data []byte, additionalMetadata []byte) (*msgqueue.Message, error) {
	return msgqueue.NewSingletonTenantMessage(
		tenantId,
		"task-trigger",
		TriggerTaskPayload{
			TaskExternalId:     taskExternalId,
			WorkflowName:       name,
			Data:               data,
			AdditionalMetadata: additionalMetadata,
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
}

func CompletedTaskMessage(tenantId string, taskId int64, retryCount int32) (*msgqueue.Message, error) {
	return msgqueue.NewSingletonTenantMessage(
		tenantId,
		"task-completed",
		CompletedTaskPayload{
			TaskId:     taskId,
			RetryCount: retryCount,
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
}

func FailedTaskMessage(tenantId string, taskId int64, retryCount int32, isAppError bool) (*msgqueue.Message, error) {
	return msgqueue.NewSingletonTenantMessage(
		tenantId,
		"task-failed",
		FailedTaskPayload{
			TaskId:     taskId,
			RetryCount: retryCount,
			IsAppError: isAppError,
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
