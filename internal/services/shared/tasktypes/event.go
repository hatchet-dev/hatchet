package tasktypes

import (
	"encoding/json"
	"time"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	v2 "github.com/hatchet-dev/hatchet/pkg/repository/v2"
)

type UserEventTaskPayload struct {
	EventId                 string `json:"event_id" validate:"required,uuid"`
	EventKey                string `json:"event_key" validate:"required"`
	EventData               []byte `json:"event_data" validate:"required"`
	EventAdditionalMetadata []byte `json:"event_additional_metadata"`
}

type InternalEventTaskPayload struct {
	EventTimestamp time.Time `json:"event_timestamp" validate:"required"`
	EventKey       string    `json:"event_key" validate:"required"`
	EventData      []byte    `json:"event_data" validate:"required"`
}

func NewInternalEventMessage(tenantId string, timestamp time.Time, key string, data []byte) (*msgqueue.Message, error) {
	return msgqueue.NewSingletonTenantMessage(
		tenantId,
		"internal-event",
		InternalEventTaskPayload{
			EventTimestamp: timestamp,
			EventKey:       key,
			EventData:      data,
		},
		false,
		true,
	)
}

func NewInternalCompletedEventMessage(tenantId string, timestamp time.Time, key, stepReadableId string, output []byte) (*msgqueue.Message, error) {
	failureData := v2.CompletedData{
		StepReadableId: stepReadableId,
		Output:         output,
	}

	data, err := json.Marshal(failureData)

	if err != nil {
		return nil, err
	}

	return NewInternalEventMessage(tenantId, timestamp, key, data)
}

func NewInternalFailureEventMessage(tenantId string, timestamp time.Time, key, stepReadableId string, errorMsg string) (*msgqueue.Message, error) {
	failureData := v2.FailedData{
		StepReadableId: stepReadableId,
		Error:          errorMsg,
	}

	data, err := json.Marshal(failureData)

	if err != nil {
		return nil, err
	}

	return NewInternalEventMessage(tenantId, timestamp, key, data)
}
