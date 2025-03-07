package workflowruns

import (
	"context"

	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/v1/proxy"
	admincontracts "github.com/hatchet-dev/hatchet/internal/services/admin/contracts/v1"
	"github.com/hatchet-dev/hatchet/pkg/config/server"

	client "github.com/hatchet-dev/hatchet/pkg/client/v1"
)

type V1WorkflowRunsService struct {
	config       *server.ServerConfig
	proxyTrigger *proxy.Proxy[admincontracts.TriggerWorkflowRunRequest, admincontracts.TriggerWorkflowRunResponse]
}

func NewV1WorkflowRunsService(config *server.ServerConfig) *V1WorkflowRunsService {
	proxyTrigger := proxy.NewProxy(config, func(ctx context.Context, cli *client.GRPCClient, in *admincontracts.TriggerWorkflowRunRequest) (*admincontracts.TriggerWorkflowRunResponse, error) {
		return cli.Admin().TriggerWorkflowRun(ctx, in)
	})

	return &V1WorkflowRunsService{
		config:       config,
		proxyTrigger: proxyTrigger,
	}
}
