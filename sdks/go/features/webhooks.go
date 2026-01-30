package features

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

type WebhookAuth interface {
	toCreateRequest(opts CreateWebhookOpts) (rest.V1CreateWebhookRequest, error)
}

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

type CreateWebhookOpts struct {
	Name               string
	SourceName         rest.V1WebhookSourceName
	EventKeyExpression string
	Auth               WebhookAuth
}

type UpdateWebhookOpts struct {
	EventKeyExpression string
}

// WebhooksClient provides methods for managing webhook configurations
type WebhooksClient struct {
	api      *rest.ClientWithResponses
	tenantId uuid.UUID
}

// NewWebhooksClient creates a new WebhooksClient
func NewWebhooksClient(
	api *rest.ClientWithResponses,
	tenantId uuid.UUID,
) *WebhooksClient {
	tenantIdUUID := tenantId

	return &WebhooksClient{
		api:      api,
		tenantId: tenantIdUUID,
	}
}

// List retrieves a collection of webhooks based on the provided parameters.
func (c *WebhooksClient) List(ctx context.Context, opts rest.V1WebhookListParams) (*rest.V1WebhookList, error) {
	resp, err := c.api.V1WebhookListWithResponse(
		ctx,
		c.tenantId,
		&opts,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list webhooks")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Get retrieves a specific webhook by its name.
func (c *WebhooksClient) Get(ctx context.Context, webhookName string) (*rest.V1Webhook, error) {
	resp, err := c.api.V1WebhookGetWithResponse(
		ctx,
		c.tenantId,
		webhookName,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get webhook")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Create creates a new webhook configuration.
func (c *WebhooksClient) Create(ctx context.Context, opts CreateWebhookOpts) (*rest.V1Webhook, error) {
	if opts.Auth == nil {
		return nil, errors.New("auth is required")
	}

	req, err := opts.Auth.toCreateRequest(opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create webhook request")
	}

	resp, err := c.api.V1WebhookCreateWithResponse(
		ctx,
		c.tenantId,
		req,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create webhook")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Update updates an existing webhook configuration.
func (c *WebhooksClient) Update(ctx context.Context, webhookName string, opts UpdateWebhookOpts) (*rest.V1Webhook, error) {
	resp, err := c.api.V1WebhookUpdateWithResponse(
		ctx,
		c.tenantId,
		webhookName,
		rest.V1UpdateWebhookRequest{
			EventKeyExpression: opts.EventKeyExpression,
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update webhook")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Delete removes a webhook configuration.
func (c *WebhooksClient) Delete(ctx context.Context, webhookName string) error {
	resp, err := c.api.V1WebhookDeleteWithResponse(
		ctx,
		c.tenantId,
		webhookName,
	)
	if err != nil {
		return errors.Wrap(err, "failed to delete webhook")
	}

	if err := validateStatusCodeResponse(resp.StatusCode(), resp.Body); err != nil {
		return err
	}

	return nil
}
