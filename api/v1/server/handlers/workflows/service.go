package workflows

import (
	"context"

	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/v1/proxy"
	admincontracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	clientv1 "github.com/hatchet-dev/hatchet/pkg/client/v1"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type WorkflowService struct {
	config                       *server.ServerConfig
	proxyTriggerScheduledWorkflow *proxy.Proxy[admincontracts.TriggerScheduledWorkflowRunRequest, admincontracts.TriggerScheduledWorkflowRunResponse]
}

func NewWorkflowService(config *server.ServerConfig) *WorkflowService {
	proxyTriggerScheduledWorkflow := proxy.NewProxy(config, func(ctx context.Context, cli *clientv1.GRPCClient, in *admincontracts.TriggerScheduledWorkflowRunRequest) (*admincontracts.TriggerScheduledWorkflowRunResponse, error) {
		return cli.Admin().TriggerScheduledWorkflowRun(ctx, in)
	})

	return &WorkflowService{
		config:                       config,
		proxyTriggerScheduledWorkflow: proxyTriggerScheduledWorkflow,
	}
}
