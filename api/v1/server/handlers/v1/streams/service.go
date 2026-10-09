package streams

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type V1StreamsService struct {
	config *server.ServerConfig
}

func NewV1StreamsService(config *server.ServerConfig) *V1StreamsService {
	return &V1StreamsService{
		config: config,
	}
}
