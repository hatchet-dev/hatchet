package concurrencyv1

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type V1ConcurrencyService struct {
	config *server.ServerConfig
}

func NewV1ConcurrencyService(config *server.ServerConfig) *V1ConcurrencyService {
	return &V1ConcurrencyService{
		config: config,
	}
}
