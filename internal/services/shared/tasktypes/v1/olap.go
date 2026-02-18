package v1

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type CELEvaluationFailures struct {
	Failures []v1.CELEvaluationFailure
}

func CELEvaluationFailureMessage(tenantId uuid.UUID, failures []v1.CELEvaluationFailure) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDCELEvaluationFailure,
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

func CreatedTaskMessage(tenantId uuid.UUID, task *v1.V1TaskWithPayload) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDCreatedTask,
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

func CreatedDAGMessage(tenantId uuid.UUID, dag *v1.DAGWithData) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDCreatedDAG,
		false,
		true,
		CreatedDAGPayload{
			DAGWithData: dag,
		},
	)
}

type CreatedEventTriggerPayloadSingleton struct {
	MaybeRunId              *int64     `json:"run_id"`
	MaybeRunInsertedAt      *time.Time `json:"run_inserted_at"`
	EventSeenAt             time.Time  `json:"event_seen_at"`
	EventKey                string     `json:"event_key"`
	EventExternalId         uuid.UUID  `json:"event_id"`
	EventPayload            []byte     `json:"event_payload"`
	EventAdditionalMetadata []byte     `json:"event_additional_metadata,omitempty"`
	EventScope              *string    `json:"event_scope,omitempty"`
	FilterId                *uuid.UUID `json:"filter_id,omitempty"`
	TriggeringWebhookName   *string    `json:"triggering_webhook_name,omitempty"`
}

type CreatedEventTriggerPayload struct {
	Payloads []CreatedEventTriggerPayloadSingleton `json:"payloads"`
}

func CreatedEventTriggerMessage(tenantId uuid.UUID, eventTriggers CreatedEventTriggerPayload) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDCreatedEventTrigger,
		false,
		true,
		eventTriggers,
	)
}

type CreateMonitoringEventPayload struct {
	TaskId int64 `json:"task_id"`

	RetryCount int32 `json:"retry_count"`

	WorkerId *uuid.UUID `json:"worker_id,omitempty"`

	EventType sqlcv1.V1EventTypeOlap `json:"event_type"`

	EventTimestamp time.Time `json:"event_timestamp" validate:"required"`
	EventPayload   string    `json:"event_payload" validate:"required"`
	EventMessage   string    `json:"event_message,omitempty"`
}

func MonitoringEventMessageFromActionEvent(tenantId uuid.UUID, taskId int64, retryCount int32, request *contracts.StepActionEvent) (*msgqueue.Message, error) {
	var workerId *uuid.UUID
	parsedId, err := uuid.Parse(request.WorkerId)

	if err == nil {
		workerId = &parsedId
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
	case contracts.StepActionEventType_STEP_EVENT_TYPE_CANCELLED_CONFIRMED:
		// If a worker reports "cancelled" with reason=evicted, treat it as a durable eviction
		// (purely informational; should not move the run into a CANCELLED status on the frontend).
		// TODO-DURABLE: This is a hack to get the durable eviction event type to work. We should remove this once we have a proper durable eviction event types.
		if isEvictedReason(request.EventPayload) {
			payload.EventType = sqlcv1.V1EventTypeOlapDURABLEEVICTED
		} else {
			payload.EventType = sqlcv1.V1EventTypeOlapCANCELLEDCONFIRMED
		}
	case contracts.StepActionEventType_STEP_EVENT_TYPE_CANCELLATION_FAILED:
		if isEvictedReason(request.EventPayload) {
			payload.EventType = sqlcv1.V1EventTypeOlapDURABLEEVICTED
		} else {
			payload.EventType = sqlcv1.V1EventTypeOlapCANCELLATIONFAILED
		}
	case contracts.StepActionEventType_STEP_EVENT_TYPE_CANCELLING:
		if isEvictedReason(request.EventPayload) {
			payload.EventType = sqlcv1.V1EventTypeOlapDURABLEEVICTED
		} else {
			payload.EventType = sqlcv1.V1EventTypeOlapCANCELLING
		}
	default:
		return nil, fmt.Errorf("unknown event type: %s", request.EventType.String())
	}

	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDCreateMonitoringEvent,
		false,
		true,
		payload,
	)
}

func isEvictedReason(eventPayload string) bool {
	if eventPayload == "" {
		return false
	}

	var p struct {
		Reason string `json:"reason"`
	}

	if err := json.Unmarshal([]byte(eventPayload), &p); err != nil {
		return false
	}

	reason := strings.ToLower(strings.TrimSpace(p.Reason))

	// TODO-DURABLE: This is a hack to get the durable eviction event type to work. We should remove this once we have a proper durable eviction event types.
	// Python SDK uses CancellationReason.EVICTED.value ("evicted").
	// Be defensive in case other SDKs/versions use slightly different strings.
	switch reason {
	case "evicted", "durable_evicted", "durable-evicted":
		return true
	default:
		return false
	}
}

func MonitoringEventMessageFromInternal(tenantId uuid.UUID, payload CreateMonitoringEventPayload) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDCreateMonitoringEvent,
		false,
		true,
		payload,
	)
}
