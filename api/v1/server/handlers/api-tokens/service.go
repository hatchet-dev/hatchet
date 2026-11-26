package apitokens

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/authz"
	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type APITokenService struct {
	config      *server.ServerConfig
	permissions func() (*pkgauthz.RESTPermissions, *pkgauthz.TokenPermissions, error)
}

func NewAPITokenService(config *server.ServerConfig) *APITokenService {
	return &APITokenService{
		config:      config,
		permissions: authz.CachedPermissions(config),
	}
}
