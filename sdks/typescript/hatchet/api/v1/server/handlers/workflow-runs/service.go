package workflowruns

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type WorkflowRunsService struct {
	config *server.ServerConfig
}

func NewWorkflowRunsService(config *server.ServerConfig) *WorkflowRunsService {
	return &WorkflowRunsService{
		config: config,
	}
}
