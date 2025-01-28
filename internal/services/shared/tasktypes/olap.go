package tasktypes

import (
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/repository/olap"
	v2 "github.com/hatchet-dev/hatchet/pkg/repository/v2"
)

type CreatedTaskPayload struct {
	// (required) the external id
	ExternalId string `validate:"required,uuid"`

	// (required) the queue
	Queue string

	// (required) the action id
	ActionId string `validate:"required,actionId"`

	// (required) the step id
	StepId string `validate:"required,uuid"`

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

type CreatedTaskMetadata struct {
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}

func TaskOptToMessage(tenantId string, opt v2.CreateTaskOpts) (*msgqueue.Message, error) {
	return msgqueue.NewSingletonTenantMessage(
		tenantId,
		"created-task",
		CreatedTaskPayload{
			ExternalId:         opt.ExternalId,
			Queue:              opt.Queue,
			ActionId:           opt.ActionId,
			StepId:             opt.StepId,
			ScheduleTimeout:    opt.ScheduleTimeout,
			StepTimeout:        opt.StepTimeout,
			DisplayName:        opt.DisplayName,
			Input:              string(opt.Input),
			AdditionalMetadata: opt.AdditionalMetadata,
			Priority:           opt.Priority,
			StickyStrategy:     opt.StickyStrategy,
			DesiredWorkerId:    opt.DesiredWorkerId,
		},
		false,
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
	)
}

func MonitoringEventMessageFromInternal(tenantId string, payload CreateMonitoringEventPayload) (*msgqueue.Message, error) {
	return msgqueue.NewSingletonTenantMessage(
		tenantId,
		"create-monitoring-event",
		payload,
		false,
	)
}
