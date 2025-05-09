package eventsv1

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type V1EventsService struct {
	config *server.ServerConfig
}

func NewV1EventsService(config *server.ServerConfig) *V1EventsService {

	return &V1EventsService{
		config: config,
	}
}
