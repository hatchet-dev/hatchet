package billing

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type BillingService struct {
	config *server.ServerConfig
}

func NewBillingService(config *server.ServerConfig) *BillingService {
	return &BillingService{
		config: config,
	}
}
