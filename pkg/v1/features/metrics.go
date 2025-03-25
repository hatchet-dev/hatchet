package features

import (
	"context"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// MetricsClient provides methods for retrieving metrics data
// in the Hatchet platform.
type MetricsClient interface {
	// GetWorkflowMetrics retrieves metrics for a specific workflow.
	GetWorkflowMetrics(workflowId string, opts *rest.WorkflowGetMetricsParams, ctx ...context.Context) (*rest.WorkflowMetrics, error)

	// GetQueueMetrics retrieves tenant-wide queue metrics.
	GetQueueMetrics(opts *rest.TenantGetQueueMetricsParams, ctx ...context.Context) (*rest.TenantGetQueueMetricsResponse, error)

	// GetTaskQueueMetrics retrieves tenant-wide step run queue metrics.
	GetTaskQueueMetrics(ctx ...context.Context) (*rest.TenantGetStepRunQueueMetricsResponse, error)
}

// metricsClientImpl implements the MetricsClient interface.
type metricsClientImpl struct {
	api       *rest.ClientWithResponses
	tenantId  uuid.UUID
	workflows *WorkflowsClient
}

// NewMetricsClient creates a new client for interacting with metrics.
func NewMetricsClient(
	api *rest.ClientWithResponses,
	tenantId *string,
) MetricsClient {
	tenantIdUUID := uuid.MustParse(*tenantId)
	workflows := NewWorkflowsClient(api, tenantId)

	return &metricsClientImpl{
		api:       api,
		workflows: &workflows,
		tenantId:  tenantIdUUID,
	}
}

// GetWorkflowMetrics retrieves metrics for a specific workflow.
func (m *metricsClientImpl) GetWorkflowMetrics(workflowName string, opts *rest.WorkflowGetMetricsParams, ctx ...context.Context) (*rest.WorkflowMetrics, error) {

	workflowId, err := (*m.workflows).GetId(workflowName, ctx...)

	if err != nil {
		return nil, err
	}

	resp, err := m.api.WorkflowGetMetricsWithResponse(
		getContext(ctx...),
		workflowId,
		opts,
	)

	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// GetQueueMetrics retrieves tenant-wide queue metrics.
func (m *metricsClientImpl) GetQueueMetrics(opts *rest.TenantGetQueueMetricsParams, ctx ...context.Context) (*rest.TenantGetQueueMetricsResponse, error) {
	return m.api.TenantGetQueueMetricsWithResponse(
		getContext(ctx...),
		m.tenantId,
		opts,
	)

}

// GetTaskQueueMetrics retrieves tenant-wide step run queue metrics.
func (m *metricsClientImpl) GetTaskQueueMetrics(ctx ...context.Context) (*rest.TenantGetStepRunQueueMetricsResponse, error) {
	return m.api.TenantGetStepRunQueueMetricsWithResponse(
		getContext(ctx...),
		m.tenantId,
	)
}
