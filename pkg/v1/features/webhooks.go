// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package features

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// Deprecated: WebhookAuth is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type WebhookAuth interface {
	toCreateRequest(opts CreateWebhookOpts) (rest.V1CreateWebhookRequest, error)
}

// Deprecated: BasicAuth is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type BasicAuth struct {
	Username string
	Password string
}

func (a BasicAuth) toCreateRequest(opts CreateWebhookOpts) (rest.V1CreateWebhookRequest, error) {
	var req rest.V1CreateWebhookRequest
	err := req.FromV1CreateWebhookRequestBasicAuth(rest.V1CreateWebhookRequestBasicAuth{
		Name:               opts.Name,
		SourceName:         opts.SourceName,
		EventKeyExpression: opts.EventKeyExpression,
		AuthType:           rest.V1CreateWebhookRequestBasicAuthAuthType("BASIC"),
		Auth: rest.V1WebhookBasicAuth{
			Username: a.Username,
			Password: a.Password,
		},
	})
	return req, err
}

// Deprecated: APIKeyAuth is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type APIKeyAuth struct {
	HeaderName string
	APIKey     string
}

func (a APIKeyAuth) toCreateRequest(opts CreateWebhookOpts) (rest.V1CreateWebhookRequest, error) {
	var req rest.V1CreateWebhookRequest
	err := req.FromV1CreateWebhookRequestAPIKey(rest.V1CreateWebhookRequestAPIKey{
		Name:               opts.Name,
		SourceName:         opts.SourceName,
		EventKeyExpression: opts.EventKeyExpression,
		AuthType:           rest.V1CreateWebhookRequestAPIKeyAuthType("API_KEY"),
		Auth: rest.V1WebhookAPIKeyAuth{
			HeaderName: a.HeaderName,
			ApiKey:     a.APIKey,
		},
	})
	return req, err
}

// Deprecated: HMACAuth is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type HMACAuth struct {
	SigningSecret       string
	SignatureHeaderName string
	Algorithm           rest.V1WebhookHMACAlgorithm
	Encoding            rest.V1WebhookHMACEncoding
}

func (a HMACAuth) toCreateRequest(opts CreateWebhookOpts) (rest.V1CreateWebhookRequest, error) {
	var req rest.V1CreateWebhookRequest
	err := req.FromV1CreateWebhookRequestHMAC(rest.V1CreateWebhookRequestHMAC{
		Name:               opts.Name,
		SourceName:         opts.SourceName,
		EventKeyExpression: opts.EventKeyExpression,
		AuthType:           rest.V1CreateWebhookRequestHMACAuthType("HMAC"),
		Auth: rest.V1WebhookHMACAuth{
			SigningSecret:       a.SigningSecret,
			SignatureHeaderName: a.SignatureHeaderName,
			Algorithm:           a.Algorithm,
			Encoding:            a.Encoding,
		},
	})
	return req, err
}

// Deprecated: CreateWebhookOpts is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type CreateWebhookOpts struct {
	Name               string
	SourceName         rest.V1WebhookSourceName
	EventKeyExpression string
	Auth               WebhookAuth
}

// Deprecated: UpdateWebhookOpts is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type UpdateWebhookOpts struct {
	EventKeyExpression string
}

// Deprecated: WebhooksClient is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
//
// WebhooksClient provides methods for managing webhook configurations.
type WebhooksClient interface {
	// List retrieves a collection of webhooks based on the provided parameters.
	List(ctx context.Context, opts rest.V1WebhookListParams) (*rest.V1WebhookList, error)

	// Get retrieves a specific webhook by its name.
	Get(ctx context.Context, webhookName string) (*rest.V1Webhook, error)

	// Create creates a new webhook configuration.
	Create(ctx context.Context, opts CreateWebhookOpts) (*rest.V1Webhook, error)

	// Update updates an existing webhook configuration.
	Update(ctx context.Context, webhookName string, opts UpdateWebhookOpts) (*rest.V1Webhook, error)

	// Delete removes a webhook configuration.
	Delete(ctx context.Context, webhookName string) error
}

type webhooksClientImpl struct {
	api      *rest.ClientWithResponses
	tenantId uuid.UUID
}

// Deprecated: NewWebhooksClient is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
//
// NewWebhooksClient creates a new client for managing webhook configurations.
func NewWebhooksClient(
	api *rest.ClientWithResponses,
	tenantId *string,
) WebhooksClient {
	tenantIdUUID := uuid.MustParse(*tenantId)

	return &webhooksClientImpl{
		api:      api,
		tenantId: tenantIdUUID,
	}
}

func (c *webhooksClientImpl) List(ctx context.Context, opts rest.V1WebhookListParams) (*rest.V1WebhookList, error) {
	resp, err := c.api.V1WebhookListWithResponse(
		ctx,
		c.tenantId,
		&opts,
	)
	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

func (c *webhooksClientImpl) Get(ctx context.Context, webhookName string) (*rest.V1Webhook, error) {
	resp, err := c.api.V1WebhookGetWithResponse(
		ctx,
		c.tenantId,
		webhookName,
	)
	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

func (c *webhooksClientImpl) Create(ctx context.Context, opts CreateWebhookOpts) (*rest.V1Webhook, error) {
	if opts.Auth == nil {
		return nil, fmt.Errorf("auth is required")
	}

	req, err := opts.Auth.toCreateRequest(opts)
	if err != nil {
		return nil, err
	}

	resp, err := c.api.V1WebhookCreateWithResponse(
		ctx,
		c.tenantId,
		req,
	)
	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

func (c *webhooksClientImpl) Update(ctx context.Context, webhookName string, opts UpdateWebhookOpts) (*rest.V1Webhook, error) {
	resp, err := c.api.V1WebhookUpdateWithResponse(
		ctx,
		c.tenantId,
		webhookName,
		rest.V1UpdateWebhookRequest{
			EventKeyExpression: opts.EventKeyExpression,
		},
	)
	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

func (c *webhooksClientImpl) Delete(ctx context.Context, webhookName string) error {
	_, err := c.api.V1WebhookDeleteWithResponse(
		ctx,
		c.tenantId,
		webhookName,
	)
	return err
}
