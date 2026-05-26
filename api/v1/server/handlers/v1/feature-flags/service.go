package featureflags

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type V1FeatureFlagsService struct {
	config *server.ServerConfig
}

func NewV1FeatureFlagsService(config *server.ServerConfig) *V1FeatureFlagsService {
	return &V1FeatureFlagsService{
		config: config,
	}
}
