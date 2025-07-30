package v1

type FailedWebhookValidationPayload struct {
	WebhookName string `json:"webhook_name" validate:"required"`
	ErrorText   string `json:"error_text" validate:"required"`
}
