package tasktypes

type EventTaskPayload struct {
	EventId   string `json:"event_id" validate:"required,uuid"`
	EventKey  string `json:"event_key" validate:"required"`
	EventData string `json:"event_data" validate:"required"`
}

type EventTaskMetadata struct {
	EventKey string `json:"event_key" validate:"required"`
	TenantId string `json:"tenant_id" validate:"required,uuid"`
}
