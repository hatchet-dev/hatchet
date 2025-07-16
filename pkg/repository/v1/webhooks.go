package v1

import (
	"context"
	"fmt"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/jackc/pgx/v5/pgtype"
)

type WebhookRepository interface {
	CreateWebhook(ctx context.Context, tenantId string, params CreateWebhookOpts) (*sqlcv1.V1IncomingWebhook, error)
	ListWebhooks(ctx context.Context, tenantId string, params ListWebhooksOpts) ([]*sqlcv1.V1IncomingWebhook, error)
	DeleteWebhook(ctx context.Context, tenantId, webhookId string) (*sqlcv1.V1IncomingWebhook, error)
	GetWebhook(ctx context.Context, tenantId, webhookId string) (*sqlcv1.V1IncomingWebhook, error)
	CanCreate(ctx context.Context, tenantId string, webhookLimit int32) (bool, error)
}

type webhookRepository struct {
	*sharedRepository
}

func newWebhookRepository(shared *sharedRepository) WebhookRepository {
	return &webhookRepository{
		sharedRepository: shared,
	}
}

type BasicAuthCredentials struct {
	Username          string `json:"username" validate:"required"`
	EncryptedPassword []byte `json:"password" validate:"required"`
}

type APIKeyAuthCredentials struct {
	HeaderName   string `json:"header_name" validate:"required"`
	EncryptedKey []byte `json:"key" validate:"required"`
}

type HMACAuthCredentials struct {
	Algorithm                     sqlcv1.V1IncomingWebhookHmacAlgorithm `json:"algorithm" validate:"required"`
	Encoding                      sqlcv1.V1IncomingWebhookHmacEncoding  `json:"encoding" validate:"required"`
	SignatureHeaderName           string                                `json:"signature_header_name" validate:"required"`
	EncryptedWebhookSigningSecret []byte                                `json:"webhook_signing_secret" validate:"required"`
}

type AuthConfig struct {
	Type       sqlcv1.V1IncomingWebhookAuthType `json:"type" validate:"required"`
	BasicAuth  *BasicAuthCredentials            `json:"basic_auth,omitempty"`
	APIKeyAuth *APIKeyAuthCredentials           `json:"api_key_auth,omitempty"`
	HMACAuth   *HMACAuthCredentials             `json:"hmac_auth,omitempty"`
}

func (ac *AuthConfig) Validate() error {
	authMethodsSet := 0

	if ac.BasicAuth != nil {
		authMethodsSet++
	}
	if ac.APIKeyAuth != nil {
		authMethodsSet++
	}
	if ac.HMACAuth != nil {
		authMethodsSet++
	}

	if authMethodsSet != 1 {
		return fmt.Errorf("exactly one auth method must be set, but %d were provided", authMethodsSet)
	}

	switch ac.Type {
	case sqlcv1.V1IncomingWebhookAuthTypeBASIC:
		if ac.BasicAuth == nil {
			return fmt.Errorf("basic auth credentials must be provided when type is BASIC")
		}
	case sqlcv1.V1IncomingWebhookAuthTypeAPIKEY:
		if ac.APIKeyAuth == nil {
			return fmt.Errorf("api key auth credentials must be provided when type is API_KEY")
		}
	case sqlcv1.V1IncomingWebhookAuthTypeHMAC:
		if ac.HMACAuth == nil {
			return fmt.Errorf("hmac auth credentials must be provided when type is HMAC")
		}
	default:
		return fmt.Errorf("unsupported auth type: %s", ac.Type)
	}

	return nil
}

type CreateWebhookOpts struct {
	Tenantid           pgtype.UUID                        `json:"tenantid"`
	Sourcename         sqlcv1.V1IncomingWebhookSourceName `json:"sourcename"`
	Name               string                             `json:"name" validate:"required"`
	Eventkeyexpression string                             `json:"eventkeyexpression"`
	AuthConfig         AuthConfig                         `json:"auth_config,omitempty"`
}

func (r *webhookRepository) CreateWebhook(ctx context.Context, tenantId string, opts CreateWebhookOpts) (*sqlcv1.V1IncomingWebhook, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	if err := opts.AuthConfig.Validate(); err != nil {
		return nil, err
	}

	params := sqlcv1.CreateWebhookParams{
		Tenantid:           sqlchelpers.UUIDFromStr(tenantId),
		Sourcename:         sqlcv1.V1IncomingWebhookSourceName(opts.Sourcename),
		Name:               opts.Name,
		Eventkeyexpression: opts.Eventkeyexpression,
		Authmethod:         sqlcv1.V1IncomingWebhookAuthType(opts.AuthConfig.Type),
	}

	switch opts.AuthConfig.Type {
	case sqlcv1.V1IncomingWebhookAuthTypeBASIC:
		params.AuthBasicUsername = pgtype.Text{
			String: opts.AuthConfig.BasicAuth.Username,
			Valid:  true,
		}
		params.Authbasicpassword = opts.AuthConfig.BasicAuth.EncryptedPassword
	case sqlcv1.V1IncomingWebhookAuthTypeAPIKEY:
		params.AuthApiKeyHeaderName = pgtype.Text{
			String: opts.AuthConfig.APIKeyAuth.HeaderName,
			Valid:  true,
		}

		params.Authapikeykey = opts.AuthConfig.APIKeyAuth.EncryptedKey
	case sqlcv1.V1IncomingWebhookAuthTypeHMAC:
		params.AuthHmacAlgorithm = sqlcv1.NullV1IncomingWebhookHmacAlgorithm{
			V1IncomingWebhookHmacAlgorithm: opts.AuthConfig.HMACAuth.Algorithm,
			Valid:                          true,
		}
		params.AuthHmacEncoding = sqlcv1.NullV1IncomingWebhookHmacEncoding{
			V1IncomingWebhookHmacEncoding: opts.AuthConfig.HMACAuth.Encoding,
			Valid:                         true,
		}
		params.AuthHmacSignatureHeaderName = pgtype.Text{
			String: opts.AuthConfig.HMACAuth.SignatureHeaderName,
			Valid:  true,
		}
		params.Authhmacwebhooksigningsecret = opts.AuthConfig.HMACAuth.EncryptedWebhookSigningSecret
	default:
		return nil, fmt.Errorf("unsupported auth type: %s", opts.AuthConfig.Type)
	}

	return r.queries.CreateWebhook(ctx, r.pool, params)
}

type ListWebhooksOpts struct {
	WebhookNames       []string                             `json:"webhook_names"`
	WebhookSourceNames []sqlcv1.V1IncomingWebhookSourceName `json:"webhook_source_names"`
	Limit              *int64                               `json:"limit" validate:"omitnil,min=1"`
	Offset             *int64                               `json:"offset" validate:"omitnil,min=0"`
}

func (r *webhookRepository) ListWebhooks(ctx context.Context, tenantId string, opts ListWebhooksOpts) ([]*sqlcv1.V1IncomingWebhook, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	var limit pgtype.Int8
	var offset pgtype.Int8

	if opts.Limit != nil {
		limit = pgtype.Int8{
			Int64: *opts.Limit,
			Valid: true,
		}
	}

	if opts.Offset != nil {
		offset = pgtype.Int8{
			Int64: *opts.Offset,
			Valid: true,
		}
	}

	return r.queries.ListWebhooks(ctx, r.pool, sqlcv1.ListWebhooksParams{
		Tenantid:      sqlchelpers.UUIDFromStr(tenantId),
		Webhooknames:  opts.WebhookNames,
		Sourcenames:   opts.WebhookSourceNames,
		WebhookLimit:  limit,
		WebhookOffset: offset,
	})

}

func (r *webhookRepository) DeleteWebhook(ctx context.Context, tenantId, name string) (*sqlcv1.V1IncomingWebhook, error) {
	return r.queries.DeleteWebhook(ctx, r.pool, sqlcv1.DeleteWebhookParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Name:     name,
	})
}

func (r *webhookRepository) GetWebhook(ctx context.Context, tenantId, name string) (*sqlcv1.V1IncomingWebhook, error) {
	return r.queries.GetWebhook(ctx, r.pool, sqlcv1.GetWebhookParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Name:     name,
	})
}

func (r *webhookRepository) CanCreate(ctx context.Context, tenantId string, webhookLimit int32) (bool, error) {
	return r.queries.CanCreateWebhook(ctx, r.pool, sqlcv1.CanCreateWebhookParams{
		Tenantid:     sqlchelpers.UUIDFromStr(tenantId),
		Webhooklimit: webhookLimit,
	})
}
