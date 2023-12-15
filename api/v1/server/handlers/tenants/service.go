package tenants

import (
	"github.com/hatchet-dev/hatchet/internal/config/server"
)

type TenantService struct {
	config *server.ServerConfig
}

func NewTenantService(config *server.ServerConfig) *TenantService {
	return &TenantService{
		config: config,
	}
}
