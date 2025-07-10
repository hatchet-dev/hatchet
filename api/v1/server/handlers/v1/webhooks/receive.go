package webhooksv1

import (
	"encoding/json"
	"fmt"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/labstack/echo/v4"
)

func (w *V1WebhooksService) V1WebhookReceive(ctx echo.Context, request gen.V1WebhookReceiveRequestObject) (gen.V1WebhookReceiveResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	webhook := ctx.Get("v1-webhook").(*sqlcv1.V1IncomingWebhook)

	switch webhook.AuthMethod {
	case sqlcv1.V1IncomingWebhookAuthTypeBASICAUTH:
		username, password, ok := ctx.Request().BasicAuth()

		if !ok {
			return gen.V1WebhookReceive400JSONResponse{
				Errors: []gen.APIError{
					{
						Description: "missing or invalid authorization header",
					},
				},
			}, nil
		}

		decryptedPassword, err := w.config.Encryption.Decrypt(webhook.AuthBasicPassword, "v1_webhook_basic_auth_password")

		if err != nil {
			return gen.V1WebhookReceive400JSONResponse{
				Errors: []gen.APIError{
					{
						Description: fmt.Sprintf("failed to decrypt basic auth password: %v", err),
					},
				},
			}, nil
		}

		if username != webhook.AuthBasicUsername.String || password != string(decryptedPassword) {
			return gen.V1WebhookReceive403JSONResponse{
				Errors: []gen.APIError{
					{
						Description: "invalid basic auth credentials",
					},
				},
			}, nil
		}

		fmt.Println("Received webhook with Basic Auth", password)
	case sqlcv1.V1IncomingWebhookAuthTypeAPIKEY:
		fmt.Println("Received webhook with API Key Auth")
	case sqlcv1.V1IncomingWebhookAuthTypeHMAC:
		fmt.Println("Received webhook with HMAC Auth")
	default:
		return gen.V1WebhookReceive400JSONResponse{
			Errors: []gen.APIError{
				{
					Description: fmt.Sprintf("unsupported auth type: %s", webhook.AuthMethod),
				},
			},
		}, nil
	}

	var payload []byte
	var err error

	if request.Body != nil {
		payloadMap := map[string]interface{}(*request.Body)

		payload, err = json.Marshal(payloadMap)

		if err != nil {
			return gen.V1WebhookReceive400JSONResponse{
				Errors: []gen.APIError{
					{
						Description: fmt.Sprintf("failed to marshal request body: %v", err),
					},
				},
			}, nil
		}
	}

	w.config.Ingestor.IngestEvent(
		ctx.Request().Context(),
		tenant,
		"foo",
		payload,
		nil,
		nil,
		nil,
	)

	msg := "ok"

	response := gen.V1WebhookReceive200JSONResponse{
		Message: &msg,
	}

	return gen.V1WebhookReceive200JSONResponse(response), nil
}
