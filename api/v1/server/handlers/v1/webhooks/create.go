package webhooksv1

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (w *V1WebhooksService) V1WebhookCreate(ctx echo.Context, request gen.V1WebhookCreateRequestObject) (gen.V1WebhookCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	canCreate, _, err := w.config.V1.TenantLimit().CanCreate(ctx.Request().Context(), sqlcv1.LimitResourceINCOMINGWEBHOOK, tenant.ID, 1)

	if err != nil {
		return nil, fmt.Errorf("failed to check if webhook can be created: %w", err)
	}

	if !canCreate {
		return gen.V1WebhookCreate400JSONResponse{
			Errors: []gen.APIError{
				{
					Description: "incoming webhook limit reached",
				},
			},
		}, nil
	}

	params, err := w.constructCreateOpts(tenant.ID, *request.Body)
	if err != nil {
		return gen.V1WebhookCreate400JSONResponse{
			Errors: []gen.APIError{
				{
					Description: fmt.Sprintf("failed to construct webhook create params: %v", err),
				},
			},
		}, nil
	}

	webhook, err := w.config.V1.Webhooks().CreateWebhook(
		ctx.Request().Context(),
		tenant.ID,
		params,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create webhook: %w", err)
	}

	transformed := transformers.ToV1Webhook(webhook)

	return gen.V1WebhookCreate200JSONResponse(transformed), nil
}

func extractAuthType(request gen.V1CreateWebhookRequest) (sqlcv1.V1IncomingWebhookAuthType, error) {
	j, err := request.MarshalJSON()

	if err != nil {
		return "", fmt.Errorf("failed to get marshal request: %w", err)
	}

	parsedBody := make(map[string]interface{})
	if err := json.Unmarshal(j, &parsedBody); err != nil {
		return "", fmt.Errorf("failed to unmarshal request body: %w", err)
	}

	unparsedDiscriminator, ok := parsedBody["authType"]
	if !ok {
		return "", fmt.Errorf("authType field is missing in the request body")
	}

	discriminator := unparsedDiscriminator.(string)

	authType := sqlcv1.V1IncomingWebhookAuthType(discriminator)
	if authType == "" {
		return "", fmt.Errorf("invalid auth type: %s", unparsedDiscriminator)
	}

	return authType, nil
}

func (w *V1WebhooksService) constructCreateOpts(tenantId uuid.UUID, request gen.V1CreateWebhookRequest) (v1.CreateWebhookOpts, error) {
	params := v1.CreateWebhookOpts{
		Tenantid: tenantId,
	}

	discriminator, err := extractAuthType(request)

	if err != nil {
		return params, fmt.Errorf("failed to get discriminator: %w", err)
	}

	authConfig := v1.AuthConfig{}

	switch discriminator {
	case sqlcv1.V1IncomingWebhookAuthTypeBASIC:
		basicAuth, err := request.AsV1CreateWebhookRequestBasicAuth()
		if err != nil {
			return params, fmt.Errorf("failed to parse basic auth: %w", err)
		}

		authConfig.Type = sqlcv1.V1IncomingWebhookAuthTypeBASIC

		passwordEncrypted, err := w.config.Encryption.Encrypt([]byte(basicAuth.Auth.Password), "v1_webhook_basic_auth_password")

		if err != nil {
			return params, fmt.Errorf("failed to encrypt basic auth password: %s", err.Error())
		}

		authConfig.BasicAuth = &v1.BasicAuthCredentials{
			Username:          basicAuth.Auth.Username,
			EncryptedPassword: passwordEncrypted,
		}

		params.Sourcename = sqlcv1.V1IncomingWebhookSourceName(basicAuth.SourceName)
		params.Name = basicAuth.Name
		params.Eventkeyexpression = basicAuth.EventKeyExpression
		params.AuthConfig = authConfig
	case sqlcv1.V1IncomingWebhookAuthTypeAPIKEY:
		apiKeyAuth, err := request.AsV1CreateWebhookRequestAPIKey()
		if err != nil {
			return params, fmt.Errorf("failed to parse api key auth: %w", err)
		}

		authConfig.Type = sqlcv1.V1IncomingWebhookAuthTypeAPIKEY

		authConfig := v1.AuthConfig{
			Type: sqlcv1.V1IncomingWebhookAuthTypeAPIKEY,
		}

		apiKeyEncrypted, err := w.config.Encryption.Encrypt([]byte(apiKeyAuth.Auth.ApiKey), "v1_webhook_api_key")

		if err != nil {
			return params, fmt.Errorf("failed to encrypt api key: %s", err.Error())
		}

		authConfig.APIKeyAuth = &v1.APIKeyAuthCredentials{
			HeaderName:   apiKeyAuth.Auth.HeaderName,
			EncryptedKey: apiKeyEncrypted,
		}

		params.Sourcename = sqlcv1.V1IncomingWebhookSourceName(apiKeyAuth.SourceName)
		params.Name = apiKeyAuth.Name
		params.Eventkeyexpression = apiKeyAuth.EventKeyExpression
		params.AuthConfig = authConfig
	case sqlcv1.V1IncomingWebhookAuthTypeHMAC:
		hmacAuth, err := request.AsV1CreateWebhookRequestHMAC()
		if err != nil {
			return params, fmt.Errorf("failed to parse hmac auth: %w", err)
		}

		authConfig := v1.AuthConfig{
			Type: sqlcv1.V1IncomingWebhookAuthTypeHMAC,
		}

		signingSecretEncrypted, err := w.config.Encryption.Encrypt([]byte(hmacAuth.Auth.SigningSecret), "v1_webhook_hmac_signing_secret")

		if err != nil {
			return params, fmt.Errorf("failed to encrypt api key: %s", err.Error())
		}

		authConfig.HMACAuth = &v1.HMACAuthCredentials{
			Algorithm:                     sqlcv1.V1IncomingWebhookHmacAlgorithm(hmacAuth.Auth.Algorithm),
			Encoding:                      sqlcv1.V1IncomingWebhookHmacEncoding(hmacAuth.Auth.Encoding),
			SignatureHeaderName:           hmacAuth.Auth.SignatureHeaderName,
			EncryptedWebhookSigningSecret: signingSecretEncrypted,
		}

		params.Sourcename = sqlcv1.V1IncomingWebhookSourceName(hmacAuth.SourceName)
		params.Name = hmacAuth.Name
		params.Eventkeyexpression = hmacAuth.EventKeyExpression
		params.AuthConfig = authConfig
	default:
		return params, fmt.Errorf("unsupported auth type: %s", discriminator)
	}

	return params, nil
}
