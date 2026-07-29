package durabletasks

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type DurableTasksService struct {
	config *server.ServerConfig
}

func NewDurableTasksService(config *server.ServerConfig) *DurableTasksService {
	return &DurableTasksService{
		config: config,
	}
}
