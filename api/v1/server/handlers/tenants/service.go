package tenants

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/authz"
	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type TenantService struct {
	config      *server.ServerConfig
	permissions func() (*pkgauthz.RESTPermissions, *pkgauthz.TokenPermissions, error)
}

func NewTenantService(config *server.ServerConfig) *TenantService {
	return &TenantService{
		config:      config,
		permissions: authz.CachedPermissions(config),
	}
}
