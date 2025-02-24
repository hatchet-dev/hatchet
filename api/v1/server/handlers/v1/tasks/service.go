package tasks

import (
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type TasksService struct {
	config *server.ServerConfig
}

func NewTasksService(config *server.ServerConfig) *TasksService {
	return &TasksService{
		config: config,
	}
}
