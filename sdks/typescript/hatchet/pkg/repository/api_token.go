package repository

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
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

type APITokenRepository interface {
	GetAPITokenById(id string) (*db.APITokenModel, error)
	RevokeAPIToken(id string) error
	ListAPITokensByTenant(tenantId string) ([]db.APITokenModel, error)
}

type EngineTokenRepository interface {
	CreateAPIToken(ctx context.Context, opts *CreateAPITokenOpts) (*dbsqlc.APIToken, error)
	GetAPITokenById(ctx context.Context, id string) (*dbsqlc.APIToken, error)
}
