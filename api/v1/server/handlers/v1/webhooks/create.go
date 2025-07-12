package webhooksv1

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/labstack/echo/v4"
)

func (w *V1WebhooksService) V1WebhookCreate(ctx echo.Context, request gen.V1WebhookCreateRequestObject) (gen.V1WebhookCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)

	webhookLimit := int32(w.config.Runtime.Limits.DefaultIncomingWebhookLimit)

	canCreate, err := w.config.V1.Webhooks().CanCreate(
		ctx.Request().Context(),
		tenant.ID.String(),
		webhookLimit,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to check if webhook can be created: %w", err)
	}

	if !canCreate {
		return gen.V1WebhookCreate400JSONResponse{
			Errors: []gen.APIError{
				{
					Description: fmt.Sprintf("cannot create more than %d webhooks", webhookLimit),
				},
			},
		}, nil
	}

	params, err := w.constructCreateOpts(tenant.ID.String(), *request.Body)
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
		tenant.ID.String(),
		params,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create webhook")
	}

	transformed := transformers.ToV1Webhook(webhook)

	return gen.V1WebhookCreate200JSONResponse(transformed), nil
}

func (w *V1WebhooksService) constructCreateOpts(tenantId string, request gen.V1CreateWebhookRequest) (v1.CreateWebhookOpts, error) {
	unparsedDiscriminator, err := request.Discriminator()

	params := v1.CreateWebhookOpts{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	}

	if err != nil {
		return params, fmt.Errorf("failed to get discriminator: %w", err)
	}

	authConfig := v1.AuthConfig{}
	discriminator := sqlcv1.V1IncomingWebhookAuthType(unparsedDiscriminator)

	switch discriminator {
	case sqlcv1.V1IncomingWebhookAuthTypeBASIC:
		basicAuth, err := request.AsV1CreateWebhookRequestBasicAuth()
		if err != nil {
			return params, fmt.Errorf("failed to parse basic auth: %w", err)
		}

		authConfig.Type = sqlcv1.V1IncomingWebhookAuthTypeBASIC

		if basicAuth.Auth.Username != nil && basicAuth.Auth.Password != nil {
			passwordEncrypted, err := w.config.Encryption.Encrypt([]byte(*basicAuth.Auth.Password), "v1_webhook_basic_auth_password")

			if err != nil {
				return params, fmt.Errorf("failed to encrypt basic auth password: %s", err.Error())
			}

			authConfig.BasicAuth = &v1.BasicAuthCredentials{
				Username: *basicAuth.Auth.Username,
				Password: passwordEncrypted,
			}
		} else {
			return params, fmt.Errorf("basic auth requires both username and password")
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

		if apiKeyAuth.Auth.HeaderName != nil && apiKeyAuth.Auth.ApiKey != nil {
			apiKeyEncrypted, err := w.config.Encryption.Encrypt([]byte(*apiKeyAuth.Auth.ApiKey), "v1_webhook_api_key")

			if err != nil {
				return params, fmt.Errorf("failed to encrypt api key: %s", err.Error())
			}

			authConfig.APIKeyAuth = &v1.APIKeyAuthCredentials{
				HeaderName: *apiKeyAuth.Auth.HeaderName,
				Key:        apiKeyEncrypted,
			}
		} else {
			return params, fmt.Errorf("api key auth requires both header name and api key")
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

		if hmacAuth.Auth.Algorithm != nil && hmacAuth.Auth.Encoding != nil &&
			hmacAuth.Auth.SignatureHeaderName != nil && hmacAuth.Auth.SigningSecret != nil {
			signingSecretEncrypted, err := w.config.Encryption.Encrypt([]byte(*hmacAuth.Auth.SigningSecret), "v1_webhook_hmac_signing_secret")

			if err != nil {
				return params, fmt.Errorf("failed to encrypt api key: %s", err.Error())
			}

			authConfig.HMACAuth = &v1.HMACAuthCredentials{
				Algorithm:            sqlcv1.V1IncomingWebhookHmacAlgorithm(*hmacAuth.Auth.Algorithm),
				Encoding:             sqlcv1.V1IncomingWebhookHmacEncoding(*hmacAuth.Auth.Encoding),
				SignatureHeaderName:  *hmacAuth.Auth.SignatureHeaderName,
				WebhookSigningSecret: signingSecretEncrypted,
			}
		} else {
			return params, fmt.Errorf("hmac auth requires algorithm, encoding, signature header name, and signing secret")
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
