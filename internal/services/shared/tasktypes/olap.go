package tasktypes

import (
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/repository/olap"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v2/sqlcv2"
)

type CreatedTaskPayload struct {
	// (required) the external id
	ExternalId string `validate:"required,uuid"`

	SourceId uint64 `json:"source_id"`

	InsertedAt time.Time `json:"inserted_at"`

	// (required) the queue
	Queue string

	// (required) the action id
	ActionId string `validate:"required,actionId"`

	// (required) the step id
	StepId string `validate:"required,uuid"`

	// (required) the workflow id
	WorkflowId string `validate:"required,uuid"`

	// (required) the schedule timeout
	ScheduleTimeout string `validate:"required,duration"`

	// (required) the step timeout
	StepTimeout string `validate:"required,duration"`

	// (required) the task display name
	DisplayName string

	// (required) the input bytes to the task
	Input string

	// (optional) the additional metadata for the task
	AdditionalMetadata map[string]interface{}

	// (optional) the priority of the task
	Priority *int

	// (optional) the sticky strategy
	// TODO: validation
	StickyStrategy *string

	// (optional) the desired worker id
	DesiredWorkerId *string
}

func CreatedTaskMessage(tenantId string, task *sqlcv2.V2Task) (*msgqueue.Message, error) {
	return msgqueue.NewSingletonTenantMessage(
		tenantId,
		"created-task",
		CreatedTaskPayload{
			ExternalId:      sqlchelpers.UUIDToStr(task.ExternalID),
			SourceId:        uint64(task.ID),
			InsertedAt:      task.InsertedAt.Time,
			Queue:           task.Queue,
			ActionId:        task.ActionID,
			StepId:          sqlchelpers.UUIDToStr(task.StepID),
			WorkflowId:      sqlchelpers.UUIDToStr(task.WorkflowID),
			ScheduleTimeout: task.ScheduleTimeout,
			StepTimeout:     task.StepTimeout.String,
			DisplayName:     task.DisplayName,
			Input:           string(task.Input),
			// AdditionalMetadata: task.AdditionalMetadata,
			// Priority:           task.Priority,
			// StickyStrategy:     task.StickyStrategy,
			// DesiredWorkerId:    task.DesiredWorkerId,
		},
		false,
		true,
	)
}

type CreateMonitoringEventPayload struct {
	// Either one of TaskId or TaskExternalId must be set
	TaskId         *int64  `json:"task_id"`
	TaskExternalId *string `json:"task_external_id"`

	RetryCount int32 `json:"retry_count"`

	WorkerId *string `json:"worker_id,omitempty"`

	EventType olap.EventType `json:"event_type"`

	EventTimestamp time.Time `json:"event_timestamp" validate:"required"`
	EventPayload   string    `json:"event_payload" validate:"required"`
	EventMessage   string    `json:"event_message,omitempty"`
}

func MonitoringEventMessageFromActionEvent(tenantId string, taskId int64, retryCount int32, request *contracts.StepActionEvent) (*msgqueue.Message, error) {
	payload := CreateMonitoringEventPayload{
		TaskId:         &taskId,
		RetryCount:     retryCount,
		WorkerId:       &request.WorkerId,
		EventTimestamp: request.EventTimestamp.AsTime(),
		EventPayload:   request.EventPayload,
	}

	switch request.EventType {
	case contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED:
		payload.EventType = olap.EVENT_TYPE_FINISHED
	case contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED:
		payload.EventType = olap.EVENT_TYPE_FAILED
	case contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED:
		payload.EventType = olap.EVENT_TYPE_STARTED
	default:
		return nil, fmt.Errorf("unknown event type: %s", request.EventType.String())
	}

	return msgqueue.NewSingletonTenantMessage(
		tenantId,
		"create-monitoring-event",
		payload,
		false,
		false,
	)
}

func MonitoringEventMessageFromInternal(tenantId string, payload CreateMonitoringEventPayload) (*msgqueue.Message, error) {
	return msgqueue.NewSingletonTenantMessage(
		tenantId,
		"create-monitoring-event",
		payload,
		false,
		false,
	)
}
