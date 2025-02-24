package workflowruns

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type V2WorkflowRunsService struct {
	config *server.ServerConfig
}

func NewV2WorkflowRunsService(config *server.ServerConfig) *V2WorkflowRunsService {
	return &V2WorkflowRunsService{
		config: config,
	}
}
