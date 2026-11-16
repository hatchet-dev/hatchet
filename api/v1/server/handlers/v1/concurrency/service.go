package concurrencyv1

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

// V1ConcurrencyService serves the tenant concurrency overview endpoints.
type V1ConcurrencyService struct {
	config *server.ServerConfig
}

// NewV1ConcurrencyService returns a V1ConcurrencyService using the given server config.
func NewV1ConcurrencyService(config *server.ServerConfig) *V1ConcurrencyService {
	return &V1ConcurrencyService{
		config: config,
	}
}
