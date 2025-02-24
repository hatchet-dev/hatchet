package workflowruns

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type V1WorkflowRunsService struct {
	config *server.ServerConfig
}

func NewV1WorkflowRunsService(config *server.ServerConfig) *V1WorkflowRunsService {
	return &V1WorkflowRunsService{
		config: config,
	}
}
