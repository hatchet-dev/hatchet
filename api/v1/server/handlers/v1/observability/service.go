package observability

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type V1ObservabilityService struct {
	config *server.ServerConfig
}

func NewV1ObservabilityService(config *server.ServerConfig) *V1ObservabilityService {

	return &V1ObservabilityService{
		config: config,
	}
}
