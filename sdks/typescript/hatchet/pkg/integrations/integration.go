package integrations

import (
	"net/http"

	"github.com/hatchet-dev/hatchet/pkg/client/types"
)

type AuthError interface {
	error

	StatusCode() int
}

type defaultAuthError struct {
	err        error
	statusCode int
}

func (e *defaultAuthError) Error() string {
	return e.err.Error()
}

func (e *defaultAuthError) StatusCode() int {
	return e.statusCode
}

func NewAuthError(err error, statusCode int) AuthError {
	return &defaultAuthError{err, statusCode}
}

type IntegrationWebhook interface {
	// Returns the method for this webhook (GET, POST, etc)
	GetMethod() string

	// Returns the default API paths that this webhook listens on.
	GetDefaultPaths() string

	// ValidatePayload validates the payload of the webhook request. Different webhooks have different
	// signature validation strategies, so this method is implemented by each integration differently.
	ValidatePayload(r *http.Request) AuthError

	// Returns the action that this webhook triggers.
	GetAction() types.Action

	// Returns the data that was sent as part of this webhook.
	GetData(r *http.Request) (map[string]interface{}, error)
}

type Integration interface {
	GetId() string
	Actions() []string
	ActionHandler(action string) any

	// GetWebhooks returns a list of webhooks that the integration supports.
	GetWebhooks() []IntegrationWebhook
}
