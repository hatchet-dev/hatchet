package tenants

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type TenantService struct {
	config *server.ServerConfig
}

func NewTenantService(config *server.ServerConfig) *TenantService {
	return &TenantService{
		config: config,
	}
}
