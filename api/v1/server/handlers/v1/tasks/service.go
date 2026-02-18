package tasks

import (
	"context"

	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/v1/proxy"
	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	admincontracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/config/server"

	client "github.com/hatchet-dev/hatchet/pkg/client/v1"
)

type TasksService struct {
	config       *server.ServerConfig
	proxyCancel  *proxy.Proxy[admincontracts.CancelTasksRequest, admincontracts.CancelTasksResponse]
	proxyReplay  *proxy.Proxy[admincontracts.ReplayTasksRequest, admincontracts.ReplayTasksResponse]
	proxyRestore *proxy.Proxy[dispatchercontracts.RestoreEvictedTaskRequest, dispatchercontracts.RestoreEvictedTaskResponse]
}

func NewTasksService(config *server.ServerConfig) *TasksService {
	proxyCancel := proxy.NewProxy(config, func(ctx context.Context, cli *client.GRPCClient, in *admincontracts.CancelTasksRequest) (*admincontracts.CancelTasksResponse, error) {
		return cli.Admin().CancelTasks(ctx, in)
	})

	proxyReplay := proxy.NewProxy(config, func(ctx context.Context, cli *client.GRPCClient, in *admincontracts.ReplayTasksRequest) (*admincontracts.ReplayTasksResponse, error) {
		return cli.Admin().ReplayTasks(ctx, in)
	})

	proxyRestore := proxy.NewProxy(config, func(ctx context.Context, cli *client.GRPCClient, in *dispatchercontracts.RestoreEvictedTaskRequest) (*dispatchercontracts.RestoreEvictedTaskResponse, error) {
		return cli.Dispatcher().RestoreEvictedTask(ctx, in)
	})

	return &TasksService{
		config:       config,
		proxyCancel:  proxyCancel,
		proxyReplay:  proxyReplay,
		proxyRestore: proxyRestore,
	}
}
