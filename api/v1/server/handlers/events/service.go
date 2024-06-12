package events

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type EventService struct {
	config *server.ServerConfig
}

func NewEventService(config *server.ServerConfig) *EventService {
	return &EventService{
		config: config,
	}
}
