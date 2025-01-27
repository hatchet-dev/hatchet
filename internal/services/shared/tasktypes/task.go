package tasktypes

import (
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
)

type CompletedTaskPayload struct {
	// (required) the task id
	TaskId int64 `validate:"required"`

	// (required) the retry count
	RetryCount int32
}

type CompletedTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
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

type FailedTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

func FailedTaskMessage(tenantId string, taskId int64, retryCount int32) (*msgqueue.Message, error) {
	return msgqueue.NewSingletonTenantMessage(
		tenantId,
		"task-failed",
		FailedTaskPayload{
			TaskId:     taskId,
			RetryCount: retryCount,
		},
		false,
	)
}
