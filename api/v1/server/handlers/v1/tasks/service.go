package tasks

import (
	"context"

	admincontracts "github.com/hatchet-dev/hatchet/internal/services/admin/contracts/v1"
	"github.com/hatchet-dev/hatchet/pkg/config/server"

	client "github.com/hatchet-dev/hatchet/pkg/client/v1"
)

type TasksService struct {
	config      *server.ServerConfig
	proxyCancel *Proxy[admincontracts.CancelTasksRequest, admincontracts.CancelTasksResponse]
}

func NewTasksService(config *server.ServerConfig) *TasksService {
	proxyCancel := &Proxy[admincontracts.CancelTasksRequest, admincontracts.CancelTasksResponse]{
		config: config,
		method: func(ctx context.Context, cli *client.GRPCClient, in *admincontracts.CancelTasksRequest) (*admincontracts.CancelTasksResponse, error) {
			return cli.Admin().CancelTasks(ctx, in)
		},
	}

	return &TasksService{
		config:      config,
		proxyCancel: proxyCancel,
	}
}
