package webhooksv1

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type V1WebhooksService struct {
	config *server.ServerConfig
}

func NewV1WebhooksService(config *server.ServerConfig) *V1WebhooksService {

	return &V1WebhooksService{
		config: config,
	}
}
