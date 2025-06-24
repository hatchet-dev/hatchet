package tenants

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/serverutils"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type TenantService struct {
	config      *server.ServerConfig
	rateLimiter *serverutils.AuthAPIRateLimiter
}

func NewTenantService(config *server.ServerConfig) *TenantService {
	return &TenantService{
		config:      config,
		rateLimiter: serverutils.NewAuthAPIRateLimiter(config),
	}
}

func (t *TenantService) GetRateLimiter() *serverutils.AuthAPIRateLimiter {
	return t.rateLimiter
}
