package workflows

import (
	"github.com/hatchet-dev/hatchet/internal/config/server"
)

type WorkflowService struct {
	config *server.ServerConfig
}

func NewWorkflowService(config *server.ServerConfig) *WorkflowService {
	return &WorkflowService{
		config: config,
	}
}
