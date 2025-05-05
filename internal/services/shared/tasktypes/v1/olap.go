package v1

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

type CreatedTaskPayload struct {
	*sqlcv1.V1Task
}

func CreatedTaskMessage(tenantId string, task *sqlcv1.V1Task) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		"created-task",
		false,
		true,
		CreatedTaskPayload{
			V1Task: task,
		},
	)
}

type CreatedDAGPayload struct {
	*v1.DAGWithData
}

func CreatedDAGMessage(tenantId string, dag *v1.DAGWithData) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		"created-dag",
		false,
		true,
		CreatedDAGPayload{
			DAGWithData: dag,
		},
	)
}

type CreatedEventTriggerPayloadSingleton struct {
	TaskExternalId          string    `json:"task_external_id"`
	TaskInsertedAt          time.Time `json:"task_inserted_at"`
	EventGeneratedAt        time.Time `json:"event_generated_at"`
	EventKey                string    `json:"event_key"`
	EventExternalId         string    `json:"event_external_id"`
	EventPayload            []byte    `json:"event_payload"`
	EventAdditionalMetadata []byte    `json:"event_additional_metadata,omitempty"`
}

type CreatedEventTriggerPayload struct {
	Payloads []CreatedEventTriggerPayloadSingleton `json:"payloads"`
}

func CreatedEventTriggerMessage(tenantId string, eventTriggers CreatedEventTriggerPayload) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		"created-event-trigger",
		false,
		true,
		eventTriggers,
	)
}

type CreateMonitoringEventPayload struct {
	TaskId int64 `json:"task_id"`

	RetryCount int32 `json:"retry_count"`

	WorkerId *string `json:"worker_id,omitempty"`

	EventType sqlcv1.V1EventTypeOlap `json:"event_type"`

	EventTimestamp time.Time `json:"event_timestamp" validate:"required"`
	EventPayload   string    `json:"event_payload" validate:"required"`
	EventMessage   string    `json:"event_message,omitempty"`
}

func MonitoringEventMessageFromActionEvent(tenantId string, taskId int64, retryCount int32, request *contracts.StepActionEvent) (*msgqueue.Message, error) {
	var workerId *string

	if _, err := uuid.Parse(request.WorkerId); err == nil {
		workerId = &request.WorkerId
	}

	payload := CreateMonitoringEventPayload{
		TaskId:         taskId,
		RetryCount:     retryCount,
		WorkerId:       workerId,
		EventTimestamp: request.EventTimestamp.AsTime(),
		EventPayload:   request.EventPayload,
	}

	switch request.EventType {
	case contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED:
		payload.EventType = sqlcv1.V1EventTypeOlapFINISHED
	case contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED:
		payload.EventType = sqlcv1.V1EventTypeOlapFAILED
	case contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED:
		payload.EventType = sqlcv1.V1EventTypeOlapSTARTED
	default:
		return nil, fmt.Errorf("unknown event type: %s", request.EventType.String())
	}

	return msgqueue.NewTenantMessage(
		tenantId,
		"create-monitoring-event",
		false,
		false,
		payload,
	)
}

func MonitoringEventMessageFromInternal(tenantId string, payload CreateMonitoringEventPayload) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		"create-monitoring-event",
		false,
		false,
		payload,
	)
}
