package v1

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// The OLAP payload types and message constructors live in pkg/repository (see
// olap_outbox.go) so the repository can stage OLAP messages inside its transactions.
type CELEvaluationFailures = v1.CELEvaluationFailures

type CreatedTaskPayload = v1.CreatedTaskPayload

type CreatedDAGPayload = v1.CreatedDAGPayload

type CreatedEventTriggerPayloadSingleton = v1.CreatedEventTriggerPayloadSingleton

type CreatedEventTriggerPayload = v1.CreatedEventTriggerPayload

type CreateMonitoringEventPayload = v1.CreateMonitoringEventPayload

// MonitoringEventPayloadFromActionEvent maps a dispatcher action event to a monitoring
// event payload. It stays in this package because the contracts types must not leak
// into pkg/repository.
func MonitoringEventPayloadFromActionEvent(taskId int64, retryCount int32, durableInvocationCount int32, request *contracts.StepActionEvent) (v1.CreateMonitoringEventPayload, error) {
	var workerId *uuid.UUID
	parsedId, err := uuid.Parse(request.WorkerId)

	if err == nil {
		workerId = &parsedId
	}

	payload := v1.CreateMonitoringEventPayload{
		TaskId:                 taskId,
		RetryCount:             retryCount,
		DurableInvocationCount: durableInvocationCount,
		WorkerId:               workerId,
		EventTimestamp:         request.EventTimestamp.AsTime(),
		EventPayload:           request.EventPayload,
	}

	switch request.EventType {
	case contracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED:
		payload.EventType = sqlcv1.V1EventTypeOlapFINISHED
	case contracts.StepActionEventType_STEP_EVENT_TYPE_FAILED:
		payload.EventType = sqlcv1.V1EventTypeOlapFAILED
	case contracts.StepActionEventType_STEP_EVENT_TYPE_STARTED:
		payload.EventType = sqlcv1.V1EventTypeOlapSTARTED
	default:
		return payload, fmt.Errorf("unknown event type: %s", request.EventType.String())
	}

	return payload, nil
}
