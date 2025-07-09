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

func (t *V1WebhooksService) V1WebhookCreate(ctx echo.Context, request gen.V1WebhookCreateRequestObject) (gen.V1WebhookCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)

	params, err := constructCreateOpts(tenant.ID.String(), *request.Body)
	if err != nil {
		return gen.V1WebhookCreate400JSONResponse{
			Errors: []gen.APIError{
				{
					Description: fmt.Sprintf("failed to construct webhook create params: %v", err),
				},
			},
		}, nil
	}

	webhook, err := t.config.V1.Webhooks().CreateWebhook(
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

// parseAuthConfig extracts the auth configuration from the discriminated union
func constructCreateOpts(tenantId string, request gen.V1CreateWebhookRequest) (v1.CreateWebhookOpts, error) {
	discriminator, err := request.Discriminator()

	params := v1.CreateWebhookOpts{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
	}

	if err != nil {
		return params, fmt.Errorf("failed to get discriminator: %w", err)
	}

	authConfig := v1.AuthConfig{}

	switch discriminator {
	case "BASIC_AUTH":
		basicAuth, err := request.AsV1CreateWebhookRequestBasicAuth()
		if err != nil {
			return params, fmt.Errorf("failed to parse basic auth: %w", err)
		}

		authConfig.Type = sqlcv1.V1IncomingWebhookAuthTypeBASICAUTH

		if basicAuth.Auth.Username != nil && basicAuth.Auth.Password != nil {
			authConfig.BasicAuth = &v1.BasicAuthCredentials{
				Username: *basicAuth.Auth.Username,
				Password: *basicAuth.Auth.Password,
			}
		}

		params.Sourcename = basicAuth.SourceName
		params.Name = basicAuth.Name
		params.Eventkeyexpression = basicAuth.EventKeyExpression
		params.AuthConfig = authConfig
	case "API_KEY":
		apiKeyAuth, err := request.AsV1CreateWebhookRequestAPIKey()
		if err != nil {
			return params, fmt.Errorf("failed to parse api key auth: %w", err)
		}

		authConfig.Type = sqlcv1.V1IncomingWebhookAuthTypeAPIKEY

		authConfig := v1.AuthConfig{
			Type: sqlcv1.V1IncomingWebhookAuthTypeAPIKEY,
		}

		if apiKeyAuth.Auth.HeaderName != nil && apiKeyAuth.Auth.ApiKey != nil {
			authConfig.APIKeyAuth = &v1.APIKeyAuthCredentials{
				HeaderName: *apiKeyAuth.Auth.HeaderName,
				Key:        *apiKeyAuth.Auth.ApiKey,
			}
		}

		params.Sourcename = apiKeyAuth.SourceName
		params.Name = apiKeyAuth.Name
		params.Eventkeyexpression = apiKeyAuth.EventKeyExpression
		params.AuthConfig = authConfig
	case "HMAC":
		hmacAuth, err := request.AsV1CreateWebhookRequestHMAC()
		if err != nil {
			return params, fmt.Errorf("failed to parse hmac auth: %w", err)
		}

		authConfig := v1.AuthConfig{
			Type: sqlcv1.V1IncomingWebhookAuthTypeHMAC,
		}

		if hmacAuth.Auth.Algorithm != nil && hmacAuth.Auth.Encoding != nil &&
			hmacAuth.Auth.SignatureHeaderName != nil && hmacAuth.Auth.SigningSecret != nil {
			authConfig.HMACAuth = &v1.HMACAuthCredentials{
				Algorithm:            sqlcv1.V1IncomingWebhookHmacAlgorithm(*hmacAuth.Auth.Algorithm),
				Encoding:             sqlcv1.V1IncomingWebhookHmacEncoding(*hmacAuth.Auth.Encoding),
				SignatureHeaderName:  *hmacAuth.Auth.SignatureHeaderName,
				WebhookSigningSecret: *hmacAuth.Auth.SigningSecret,
			}
		}

		params.Sourcename = hmacAuth.SourceName
		params.Name = hmacAuth.Name
		params.Eventkeyexpression = hmacAuth.EventKeyExpression
		params.AuthConfig = authConfig
	default:
		return params, fmt.Errorf("unsupported auth type: %s", discriminator)
	}

	return params, nil
}
