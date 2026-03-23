package workflows

import (
	"context"

	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/v1/proxy"
	admincontracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	client "github.com/hatchet-dev/hatchet/pkg/client/v1"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type WorkflowService struct {
	config *server.ServerConfig
	proxyCancel *proxy.Proxy[admincontracts.CancelTasksRequest, admincontracts.CancelTasksResponse]
}

func NewWorkflowService(config *server.ServerConfig) *WorkflowService {
	proxyCancel := proxy.NewProxy(config, func(ctx context.Context, cli *client.GRPCClient, in *admincontracts.CancelTasksRequest) (*admincontracts.CancelTasksResponse, error) {
		return cli.Admin().CancelTasks(ctx, in)
	})

	return &WorkflowService{
		config: config,
		proxyCancel: proxyCancel,
	}
}
