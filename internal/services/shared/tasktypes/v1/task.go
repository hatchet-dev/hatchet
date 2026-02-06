package v1

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TriggerTaskMessage(tenantId uuid.UUID, payloads ...*v1.WorkflowNameTriggerOpts) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDTaskTrigger,
		false,
		true,
		payloads...,
	)
}

type CompletedTaskPayload struct {
	InsertedAt    pgtype.Timestamptz
	Output        []byte
	TaskId        int64 `validate:"required"`
	RetryCount    int32
	ExternalId    uuid.UUID
	WorkflowRunId uuid.UUID
}

func CompletedTaskMessage(
	tenantId uuid.UUID,
	taskId int64,
	taskInsertedAt pgtype.Timestamptz,
	taskExternalId uuid.UUID,
	workflowRunId uuid.UUID,
	retryCount int32,
	output []byte,
) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDTaskCompleted,
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
	InsertedAt     pgtype.Timestamptz
	ErrorMsg       string
	TaskId         int64 `validate:"required"`
	RetryCount     int32
	ExternalId     uuid.UUID
	WorkflowRunId  uuid.UUID
	IsAppError     bool
	IsNonRetryable bool `json:"is_non_retryable"`
}

func FailedTaskMessage(
	tenantId uuid.UUID,
	taskId int64,
	taskInsertedAt pgtype.Timestamptz,
	taskExternalId uuid.UUID,
	workflowRunId uuid.UUID,
	retryCount int32,
	isAppError bool,
	errorMsg string,
	isNonRetryable bool,
) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDTaskFailed,
		false,
		true,
		FailedTaskPayload{
			TaskId:         taskId,
			InsertedAt:     taskInsertedAt,
			ExternalId:     taskExternalId,
			WorkflowRunId:  workflowRunId,
			RetryCount:     retryCount,
			IsAppError:     isAppError,
			ErrorMsg:       errorMsg,
			IsNonRetryable: isNonRetryable,
		},
	)
}

type CancelledTaskPayload struct {
	InsertedAt    pgtype.Timestamptz
	EventMessage  string
	EventType     sqlcv1.V1EventTypeOlap
	TaskId        int64 `validate:"required"`
	RetryCount    int32
	ExternalId    uuid.UUID
	WorkflowRunId uuid.UUID
	ShouldNotify  bool
}

func CancelledTaskMessage(
	tenantId uuid.UUID,
	taskId int64,
	taskInsertedAt pgtype.Timestamptz,
	taskExternalId uuid.UUID,
	workflowRunId uuid.UUID,
	retryCount int32,
	eventType sqlcv1.V1EventTypeOlap,
	eventMessage string,
	shouldNotify bool,
) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDTaskCancelled,
		false,
		true,
		CancelledTaskPayload{
			TaskId:        taskId,
			InsertedAt:    taskInsertedAt,
			ExternalId:    taskExternalId,
			WorkflowRunId: workflowRunId,
			RetryCount:    retryCount,
			EventType:     eventType,
			EventMessage:  eventMessage,
			ShouldNotify:  shouldNotify,
		},
	)
}

type SignalTaskCancelledPayload struct {
	InsertedAt pgtype.Timestamptz
	TaskId     int64 `validate:"required"`
	RetryCount int32
	WorkerId   uuid.UUID `validate:"required"`
}

type CancelTasksPayload struct {
	Tasks []v1.TaskIdInsertedAtRetryCount `json:"tasks"`
}

type TaskIdInsertedAtRetryCountWithExternalId struct {
	v1.TaskIdInsertedAtRetryCount `json:"task"`
	WorkflowRunExternalId         uuid.UUID `json:"workflow_run_external_id,omitempty"`
	TaskExternalId                uuid.UUID `json:"task_external_id,omitempty"`
}

type ReplayTasksPayload struct {
	Tasks []TaskIdInsertedAtRetryCountWithExternalId `json:"tasks"`
}

type NotifyFinalizedPayload struct {
	Status     sqlcv1.V1ReadableStatusOlap
	ExternalId uuid.UUID `validate:"required"`
}

type CandidateFinalizedPayload struct {
	// (required) the workflow run id (can either be a workflow run id or single task)
	WorkflowRunId uuid.UUID `validate:"required"`
}
