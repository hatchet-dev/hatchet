package workflowruns

import (
	"context"

	"github.com/hatchet-dev/hatchet/api/v1/server/handlers/v1/proxy"
	admincontracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/config/server"

	client "github.com/hatchet-dev/hatchet/pkg/client/v1"
)

type V1WorkflowRunsService struct {
	config                 *server.ServerConfig
	proxyTrigger           *proxy.Proxy[admincontracts.TriggerWorkflowRunRequest, admincontracts.TriggerWorkflowRunResponse]
	proxyBranchDurableTask *proxy.Proxy[admincontracts.BranchDurableTaskRequest, admincontracts.BranchDurableTaskResponse]
}

func NewV1WorkflowRunsService(config *server.ServerConfig) *V1WorkflowRunsService {
	proxyTrigger := proxy.NewProxy(config, func(ctx context.Context, cli *client.GRPCClient, in *admincontracts.TriggerWorkflowRunRequest) (*admincontracts.TriggerWorkflowRunResponse, error) {
		return cli.Admin().TriggerWorkflowRun(ctx, in)
	})

	proxyBranchDurableTask := proxy.NewProxy(config, func(ctx context.Context, cli *client.GRPCClient, in *admincontracts.BranchDurableTaskRequest) (*admincontracts.BranchDurableTaskResponse, error) {
		return cli.Admin().BranchDurableTask(ctx, in)
	})

	return &V1WorkflowRunsService{
		config:                 config,
		proxyTrigger:           proxyTrigger,
		proxyBranchDurableTask: proxyBranchDurableTask,
	}
}
