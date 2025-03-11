package repository

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

type CreateAPITokenOpts struct {
	// The id of the token
	ID string `validate:"required,uuid"`

	// When the token expires
	ExpiresAt time.Time

	// (optional) A tenant ID for this API token
	TenantId *string `validate:"omitempty,uuid"`

	// (optional) A name for this API token
	Name *string `validate:"omitempty,max=255"`

	Internal bool
}

type APITokenGenerator func(ctx context.Context, tenantId, name string, internal bool, expires *time.Time) (string, error)

type APITokenRepository interface {
	CreateAPIToken(ctx context.Context, opts *CreateAPITokenOpts) (*dbsqlc.APIToken, error)
	GetAPITokenById(ctx context.Context, id string) (*dbsqlc.APIToken, error)
	ListAPITokensByTenant(ctx context.Context, tenantId string) ([]*dbsqlc.APIToken, error)
	RevokeAPIToken(ctx context.Context, id string) error
	DeleteAPIToken(ctx context.Context, tenantId, id string) error
}
