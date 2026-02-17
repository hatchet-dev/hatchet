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
	// (required) the task id
	TaskId int64 `validate:"required"`

	// (required) the task inserted at
	InsertedAt pgtype.Timestamptz

	// (required) the task external id
	ExternalId uuid.UUID

	// (required) the workflow run id
	WorkflowRunId uuid.UUID

	// (required) the retry count
	RetryCount int32

	// (optional) the output data
	Output []byte
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
	// (required) the task id
	TaskId int64 `validate:"required"`

	// (required) the task inserted at
	InsertedAt pgtype.Timestamptz

	// (required) the task external id
	ExternalId uuid.UUID

	// (required) the workflow run id
	WorkflowRunId uuid.UUID

	// (required) the retry count
	RetryCount int32

	// (required) whether this is an application-level error or an internal error on the Hatchet side
	IsAppError bool

	// (optional) the error message
	ErrorMsg string

	// (optional) A boolean flag to indicate whether the error is non-retryable, meaning it should _not_ be retried. Defaults to false.
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
	// (required) the task id
	TaskId int64 `validate:"required"`

	// (required) the task inserted at
	InsertedAt pgtype.Timestamptz

	// (required) the task external id
	ExternalId uuid.UUID

	// (required) the workflow run id
	WorkflowRunId uuid.UUID

	// (required) the retry count
	RetryCount int32

	// (optional) the event message
	EventMessage string

	// (required) the reason for cancellation
	EventType sqlcv1.V1EventTypeOlap

	// (optional) whether the task should notify the worker
	ShouldNotify bool
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
	// (required) the worker id
	WorkerId uuid.UUID `validate:"required"`

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

type TaskIdInsertedAtRetryCountWithExternalId struct {
	v1.TaskIdInsertedAtRetryCount `json:"task"`
	WorkflowRunExternalId         uuid.UUID `json:"workflow_run_external_id,omitempty"`
	TaskExternalId                uuid.UUID `json:"task_external_id,omitempty"`
}

type ReplayTasksPayload struct {
	Tasks []TaskIdInsertedAtRetryCountWithExternalId `json:"tasks"`
}

type NotifyFinalizedPayload struct {
	// (required) the external id (can either be a workflow run id or single task)
	ExternalId uuid.UUID `validate:"required"`

	// (required) the status of the task
	Status sqlcv1.V1ReadableStatusOlap
}

type CandidateFinalizedPayload struct {
	// (required) the workflow run id (can either be a workflow run id or single task)
	WorkflowRunId uuid.UUID `validate:"required"`
}

type DurableCallbackCompletedPayload struct {
	Payload         []byte
	NodeId          int64
	InvocationCount int64
	TaskExternalId  uuid.UUID
}

func DurableCallbackCompletedMessage(
	tenantId, taskExternalId uuid.UUID, nodeId int64, payload []byte,
) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDDurableCallbackCompleted,
		false,
		true,
		DurableCallbackCompletedPayload{
			TaskExternalId: taskExternalId,
			NodeId:         nodeId,
			Payload:        payload,
		},
	)
}
