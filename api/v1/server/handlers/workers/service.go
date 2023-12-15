package workers

import (
	"github.com/hatchet-dev/hatchet/internal/config/server"
)

type WorkerService struct {
	config *server.ServerConfig
}

func NewWorkerService(config *server.ServerConfig) *WorkerService {
	return &WorkerService{
		config: config,
	}
}
