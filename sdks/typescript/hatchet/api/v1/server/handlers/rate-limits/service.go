package rate_limits

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type RateLimitService struct {
	config *server.ServerConfig
}

func NewRateLimitService(config *server.ServerConfig) *RateLimitService {
	return &RateLimitService{
		config: config,
	}
}
