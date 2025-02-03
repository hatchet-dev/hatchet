package tasktypes

type EventTaskPayload struct {
	EventId                 string `json:"event_id" validate:"required,uuid"`
	EventKey                string `json:"event_key" validate:"required"`
	EventData               []byte `json:"event_data" validate:"required"`
	EventAdditionalMetadata []byte `json:"event_additional_metadata"`
}
