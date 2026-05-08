package workflows

import (
	"context"

	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/v1/proxy"
	admincontracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	client "github.com/hatchet-dev/hatchet/pkg/client/v1"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type WorkflowService struct {
	config     *server.ServerConfig
	proxyPause *proxy.Proxy[admincontracts.UpdateWorkflowPauseRequest, admincontracts.UpdateWorkflowPauseResponse]
}

func NewWorkflowService(config *server.ServerConfig) *WorkflowService {
	proxyPause := proxy.NewProxy(config, func(ctx context.Context, cli *client.GRPCClient, in *admincontracts.UpdateWorkflowPauseRequest) (*admincontracts.UpdateWorkflowPauseResponse, error) {
		return cli.Admin().UpdateWorkflowPause(ctx, in)
	})

	return &WorkflowService{
		config:     config,
		proxyPause: proxyPause,
	}
}
