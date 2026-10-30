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

type PendingDurableRunTriggerPayload struct {
	NodeId      int64                       `json:"node_id"`
	BranchId    int64                       `json:"branch_id"`
	TriggerOpts *v1.WorkflowNameTriggerOpts `json:"trigger_opts"`
}

type DurableRunTriggerMessage struct {
	DurableTaskId         int64                             `json:"durable_task_id"`
	DurableTaskInsertedAt pgtype.Timestamptz                `json:"durable_task_inserted_at"`
	DurableTaskExternalId uuid.UUID                         `json:"durable_task_external_id"`
	Entries               []PendingDurableRunTriggerPayload `json:"entries"`
}

func DurableRunTriggerTaskMessage(tenantId uuid.UUID, msg DurableRunTriggerMessage) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDDurableRunTrigger,
		false,
		true,
		msg,
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

	// (optional) the id the core and OLAP copies of this completion share: the task controller
	// stores the completion event payload under it and the OLAP event row points at it
	PayloadReferenceId *uuid.UUID
}

func CompletedTaskMessage(
	tenantId uuid.UUID,
	taskId int64,
	taskInsertedAt pgtype.Timestamptz,
	taskExternalId uuid.UUID,
	workflowRunId uuid.UUID,
	retryCount int32,
	output []byte,
	payloadReferenceId *uuid.UUID,
) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDTaskCompleted,
		false,
		true,
		CompletedTaskPayload{
			TaskId:             taskId,
			InsertedAt:         taskInsertedAt,
			ExternalId:         taskExternalId,
			WorkflowRunId:      workflowRunId,
			RetryCount:         retryCount,
			Output:             output,
			PayloadReferenceId: payloadReferenceId,
		},
	)
}

func AckedTaskMessage(
	tenantId uuid.UUID,
	taskId int64,
	taskInsertedAt pgtype.Timestamptz,
	retryCount int32,
) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDTaskAcked,
		false,
		true,
		v1.TaskIdInsertedAtRetryCount{
			Id:         taskId,
			InsertedAt: taskInsertedAt,
			RetryCount: retryCount,
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

	// (optional) the delay in milliseconds before the next attempt. Ignored when IsNonRetryable is set.
	RetryAfterMs *int64 `json:"retry_after_ms,omitempty"`
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
	retryAfterMs *int64,
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
			RetryAfterMs:   retryAfterMs,
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

	CancellationReason v1.TaskCancellationReason

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
	cancellationReason v1.TaskCancellationReason,
	shouldNotify bool,
) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDTaskCancelled,
		false,
		true,
		CancelledTaskPayload{
			TaskId:             taskId,
			InsertedAt:         taskInsertedAt,
			ExternalId:         taskExternalId,
			WorkflowRunId:      workflowRunId,
			RetryCount:         retryCount,
			EventType:          eventType,
			EventMessage:       eventMessage,
			CancellationReason: cancellationReason,
			ShouldNotify:       shouldNotify,
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

type TaskToCancel struct {
	v1.TaskIdInsertedAtRetryCount
	CancellationReason v1.TaskCancellationReason
}

type CancelTasksPayload struct {
	Tasks []TaskToCancel `json:"tasks"`
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

type DurableRestoreTaskPayload struct {
	Reason         string
	TaskExternalId uuid.UUID
}

func DurableRestoreTaskMessage(tenantId uuid.UUID, taskExternalId uuid.UUID, reason string) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDDurableRestoreTask,
		false,
		true,
		DurableRestoreTaskPayload{
			TaskExternalId: taskExternalId,
			Reason:         reason,
		},
	)
}

type DurableCallbackCompletedPayload struct {
	Payload               []byte
	SatisfiedOrder        *int64
	ChildTaskIsFailure    bool
	ChildTaskErrorMessage *string
	BranchId              int64
	NodeId                int64
	InvocationCount       int32
	TaskExternalId        uuid.UUID
	// RedeliveryCount is bumped each time the dead-letter path re-dispatches this callback
	// because no dispatcher had an active durable session for the task. It bounds that retry
	// loop (see handleDeadLetteredDurableCallbackCompleted).
	RedeliveryCount int32 `json:"redelivery_count,omitempty"`
}

func DurableCallbackCompletedMessage(
	tenantId, taskExternalId uuid.UUID, invocationCount int32, branchId, nodeId int64, payload []byte, satisfiedOrder *int64, childTaskIsFailure bool, childTaskErrorMessage *string, redeliveryCount int32,
) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDDurableCallbackCompleted,
		false,
		true,
		DurableCallbackCompletedPayload{
			TaskExternalId:        taskExternalId,
			InvocationCount:       invocationCount,
			BranchId:              branchId,
			NodeId:                nodeId,
			Payload:               payload,
			SatisfiedOrder:        satisfiedOrder,
			ChildTaskIsFailure:    childTaskIsFailure,
			ChildTaskErrorMessage: childTaskErrorMessage,
			RedeliveryCount:       redeliveryCount,
		},
	)
}

type TogglePauseWorkflowPayload struct {
	WorkflowID uuid.UUID
	IsPaused   bool
}

func NewTogglePauseWorkflowMessage(tenantId, workflowId uuid.UUID, isPaused bool) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDTogglePauseWorkflow,
		false,
		true,
		TogglePauseWorkflowPayload{
			WorkflowID: workflowId,
			IsPaused:   isPaused,
		},
	)
}

// RateLimitPausesChangedPayload tells the task controller to hold or release the tenant's queued runs behind rate limit pauses.
type RateLimitPausesChangedPayload struct {
	// Paused is true when a key was paused, and false when a pause ended
	Paused bool
}

// NewRateLimitPausesChangedMessage is sent to the task controller when a rate limit key is paused, so it holds the runs
// already waiting on the key, and when a pause ends, so it releases them.
func NewRateLimitPausesChangedMessage(tenantId uuid.UUID, paused bool) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDRateLimitPausesChanged,
		false,
		true,
		RateLimitPausesChangedPayload{
			Paused: paused,
		},
	)
}
