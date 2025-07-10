package webhooksv1

import (
	"encoding/json"
	"fmt"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/cel"
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

	payloadMap := make(map[string]interface{})

	if request.Body != nil {
		payloadMap = map[string]interface{}(*request.Body)
		delete(payloadMap, "tenant")
		delete(payloadMap, "v1-webhook")
	}

	eventKey, err := w.celParser.EvaluateIncomingWebhookExpression(webhook.EventKeyExpression, cel.NewInput(
		cel.WithInput(payloadMap),
	),
	)

	if err != nil {
		// TODO: Store this error
		return gen.V1WebhookReceive400JSONResponse{
			Errors: []gen.APIError{
				{
					Description: fmt.Sprintf("failed to evaluate event key expression: %v", err),
				},
			},
		}, nil
	}

	payload, err := json.Marshal(payloadMap)
	if err != nil {
		return gen.V1WebhookReceive400JSONResponse{
			Errors: []gen.APIError{
				{
					Description: fmt.Sprintf("failed to marshal request body: %v", err),
				},
			},
		}, nil
	}

	_, err = w.config.Ingestor.IngestEvent(
		ctx.Request().Context(),
		tenant,
		eventKey,
		payload,
		nil,
		nil,
		nil,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to ingest event")
	}

	msg := "ok"

	response := gen.V1WebhookReceive200JSONResponse{
		Message: &msg,
	}

	return gen.V1WebhookReceive200JSONResponse(response), nil
}
