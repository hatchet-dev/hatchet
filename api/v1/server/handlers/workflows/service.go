package workflows

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

type WorkflowService struct {
	config            *server.ServerConfig
	workflowSchedules v1.WorkflowScheduleRepository
}

func NewWorkflowService(config *server.ServerConfig) *WorkflowService {
	var workflowSchedules v1.WorkflowScheduleRepository
	if config != nil && config.V1 != nil {
		workflowSchedules = config.V1.WorkflowSchedules()
	}

	return &WorkflowService{
		config:            config,
		workflowSchedules: workflowSchedules,
	}
}
