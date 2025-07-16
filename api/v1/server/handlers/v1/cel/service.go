package celv1

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type V1CELService struct {
	config *server.ServerConfig
}

func NewV1CELService(config *server.ServerConfig) *V1CELService {

	return &V1CELService{
		config: config,
	}
}
