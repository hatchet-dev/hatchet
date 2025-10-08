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

type CELEvaluationFailures struct {
	Failures []v1.CELEvaluationFailure
}

func CELEvaluationFailureMessage(tenantId string, failures []v1.CELEvaluationFailure) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		"cel-evaluation-failure",
		false,
		true,
		CELEvaluationFailures{
			Failures: failures,
		},
	)
}

type CreatedTaskPayload struct {
	*v1.V1TaskWithPayload
}

func CreatedTaskMessage(tenantId string, task *v1.V1TaskWithPayload) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		"created-task",
		false,
		true,
		CreatedTaskPayload{
			V1TaskWithPayload: task,
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

type PutOLAPPayloadOpts struct {
	*v1.StorePayloadOpts
	Location sqlcv1.V1PayloadLocationOlap
}

type Payloads struct {
	Payloads []PutOLAPPayloadOpts
}

func PutPayloadMessage(tenantId string, payloads []PutOLAPPayloadOpts) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		"put-payloads",
		false,
		true,
		Payloads{
			Payloads: payloads,
		},
	)
}

type CreatedEventTriggerPayloadSingleton struct {
	MaybeRunId              *int64     `json:"run_id"`
	MaybeRunInsertedAt      *time.Time `json:"run_inserted_at"`
	EventSeenAt             time.Time  `json:"event_seen_at"`
	EventKey                string     `json:"event_key"`
	EventExternalId         string     `json:"event_id"`
	EventPayload            []byte     `json:"event_payload"`
	EventAdditionalMetadata []byte     `json:"event_additional_metadata,omitempty"`
	EventScope              *string    `json:"event_scope,omitempty"`
	FilterId                *string    `json:"filter_id,omitempty"`
	TriggeringWebhookName   *string    `json:"triggering_webhook_name,omitempty"`
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
		true,
		payload,
	)
}

func MonitoringEventMessageFromInternal(tenantId string, payload CreateMonitoringEventPayload) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		"create-monitoring-event",
		false,
		true,
		payload,
	)
}
