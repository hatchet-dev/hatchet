package operatorclient

import (
	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// These aliases re-export internal symbols to be used by external operators.
type (
	AssignedAction      = dispatchercontracts.AssignedAction
	StepActionEvent     = dispatchercontracts.StepActionEvent
	ActionEventResponse = dispatchercontracts.ActionEventResponse
	ActionType          = dispatchercontracts.ActionType
	StepActionEventType = dispatchercontracts.StepActionEventType
)

type (
	CreateWorkflowVersionRequest  = v1.CreateWorkflowVersionRequest
	CreateWorkflowVersionResponse = v1.CreateWorkflowVersionResponse
	CreateTaskOpts                = v1.CreateTaskOpts
	CreateTaskRateLimit           = v1.CreateTaskRateLimit
	DesiredWorkerLabels           = v1.DesiredWorkerLabels
	Concurrency                   = v1.Concurrency
	TaskConditions                = v1.TaskConditions
	TaskBatchConfig               = v1.TaskBatchConfig
	StickyStrategy                = v1.StickyStrategy
	DefaultFilter                 = v1.DefaultFilter
	IdempotencyConfig             = v1.IdempotencyConfig
)

type DurableTaskClient = v1.V1Dispatcher_DurableTaskClient

const (
	ActionTypeStartStepRun     = dispatchercontracts.ActionType_START_STEP_RUN
	ActionTypeCancelStepRun    = dispatchercontracts.ActionType_CANCEL_STEP_RUN
	ActionTypeStartGetGroupKey = dispatchercontracts.ActionType_START_GET_GROUP_KEY
	ActionTypeStartBatch       = dispatchercontracts.ActionType_START_BATCH

	StepEventTypeUnknown      = dispatchercontracts.StepActionEventType_STEP_EVENT_TYPE_UNKNOWN
	StepEventTypeStarted      = dispatchercontracts.StepActionEventType_STEP_EVENT_TYPE_STARTED
	StepEventTypeCompleted    = dispatchercontracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED
	StepEventTypeFailed       = dispatchercontracts.StepActionEventType_STEP_EVENT_TYPE_FAILED
	StepEventTypeAcknowledged = dispatchercontracts.StepActionEventType_STEP_EVENT_TYPE_ACKNOWLEDGED
	StepEventTypeCancelled    = dispatchercontracts.StepActionEventType_STEP_EVENT_TYPE_CANCELLED
)
