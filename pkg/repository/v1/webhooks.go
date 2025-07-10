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
	Username string `json:"username"`
	Password []byte `json:"password"`
}

type APIKeyAuthCredentials struct {
	HeaderName string `json:"header_name"`
	Key        []byte `json:"key"`
}

type HMACAuthCredentials struct {
	Algorithm            sqlcv1.V1IncomingWebhookHmacAlgorithm `json:"algorithm"`
	Encoding             sqlcv1.V1IncomingWebhookHmacEncoding  `json:"encoding"`
	SignatureHeaderName  string                                `json:"signature_header_name"`
	WebhookSigningSecret []byte                                `json:"webhook_signing_secret"`
}

type AuthConfig struct {
	Type       sqlcv1.V1IncomingWebhookAuthType `json:"type"`
	BasicAuth  *BasicAuthCredentials            `json:"basic_auth,omitempty"`
	APIKeyAuth *APIKeyAuthCredentials           `json:"api_key_auth,omitempty"`
	HMACAuth   *HMACAuthCredentials             `json:"hmac_auth,omitempty"`
}

type CreateWebhookOpts struct {
	ID                 pgtype.UUID `json:"id"`
	Tenantid           pgtype.UUID `json:"tenantid"`
	Sourcename         string      `json:"sourcename"`
	Name               string      `json:"name"`
	Eventkeyexpression string      `json:"eventkeyexpression"`
	AuthConfig         AuthConfig  `json:"auth_config,omitempty"`
}

func (r *webhookRepository) CreateWebhook(ctx context.Context, tenantId string, opts CreateWebhookOpts) (*sqlcv1.V1IncomingWebhook, error) {
	params := sqlcv1.CreateWebhookParams{
		ID:                 opts.ID,
		Tenantid:           sqlchelpers.UUIDFromStr(tenantId),
		Sourcename:         opts.Sourcename,
		Name:               opts.Name,
		Eventkeyexpression: opts.Eventkeyexpression,
		Authmethod:         sqlcv1.V1IncomingWebhookAuthType(opts.AuthConfig.Type),
	}

	switch opts.AuthConfig.Type {
	case sqlcv1.V1IncomingWebhookAuthTypeBASICAUTH:
		params.Authbasicusername = opts.AuthConfig.BasicAuth.Username
		params.Authbasicpassword = opts.AuthConfig.BasicAuth.Password
	case sqlcv1.V1IncomingWebhookAuthTypeAPIKEY:
		params.Authapikeyheadername = opts.AuthConfig.APIKeyAuth.HeaderName
		params.Authapikeykey = opts.AuthConfig.APIKeyAuth.Key
	case sqlcv1.V1IncomingWebhookAuthTypeHMAC:
		params.AuthHmacAlgorithm = sqlcv1.NullV1IncomingWebhookHmacAlgorithm{
			V1IncomingWebhookHmacAlgorithm: opts.AuthConfig.HMACAuth.Algorithm,
			Valid:                          true,
		}
		params.AuthHmacEncoding = sqlcv1.NullV1IncomingWebhookHmacEncoding{
			V1IncomingWebhookHmacEncoding: opts.AuthConfig.HMACAuth.Encoding,
			Valid:                         true,
		}
		params.Authhmacsignatureheadername = opts.AuthConfig.HMACAuth.SignatureHeaderName
		params.Authhmacwebhooksigningsecret = opts.AuthConfig.HMACAuth.WebhookSigningSecret
	default:
		return nil, fmt.Errorf("unsupported auth type: %s", opts.AuthConfig.Type)
	}

	return r.queries.CreateWebhook(ctx, r.pool, params)
}

type ListWebhooksOpts struct {
	WebhookNames       []string `json:"webhook_names"`
	WebhookSourceNames []string `json:"webhook_source_names"`
	Limit              *int64   `json:"limit" validate:"omitnil,min=1"`
	Offset             *int64   `json:"offset" validate:"omitnil,min=0"`
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

func (r *webhookRepository) DeleteWebhook(ctx context.Context, tenantId, webhookId string) (*sqlcv1.V1IncomingWebhook, error) {
	return r.queries.DeleteWebhook(ctx, r.pool, sqlcv1.DeleteWebhookParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		ID:       sqlchelpers.UUIDFromStr(webhookId),
	})
}

func (r *webhookRepository) GetWebhook(ctx context.Context, tenantId, webhookId string) (*sqlcv1.V1IncomingWebhook, error) {
	return r.queries.GetWebhook(ctx, r.pool, sqlcv1.GetWebhookParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		ID:       sqlchelpers.UUIDFromStr(webhookId),
	})
}
