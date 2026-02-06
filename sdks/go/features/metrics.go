package features

import (
	"context"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// MetricsClient provides methods for retrieving metrics for workflows, tasks, and tenant queue
type MetricsClient struct {
	api       *rest.ClientWithResponses
	workflows *WorkflowsClient
	tenantId  uuid.UUID
}

// NewMetricsClient creates a new MetricsClient
func NewMetricsClient(
	api *rest.ClientWithResponses,
	tenantId string,
) *MetricsClient {
	tenantIdUUID := uuid.MustParse(tenantId)
	workflows := NewWorkflowsClient(api, tenantId)

	return &MetricsClient{
		api:       api,
		workflows: workflows,
		tenantId:  tenantIdUUID,
	}
}

// GetWorkflowMetrics retrieves metrics for a specific workflow.
func (m *MetricsClient) GetWorkflowMetrics(ctx context.Context, workflowName string, opts *rest.WorkflowGetMetricsParams) (*rest.WorkflowMetrics, error) {
	workflowId, err := (*m.workflows).GetId(ctx, workflowName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workflow metrics")
	}

	resp, err := m.api.WorkflowGetMetricsWithResponse(
		ctx,
		workflowId,
		opts,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workflow metrics")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// GetQueueMetrics retrieves tenant-wide queue metrics.
func (m *MetricsClient) GetQueueMetrics(ctx context.Context, opts *rest.TenantGetQueueMetricsParams) (*rest.TenantQueueMetrics, error) {
	resp, err := m.api.TenantGetQueueMetricsWithResponse(
		ctx,
		m.tenantId,
		opts,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get queue metrics")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// GetTaskQueueMetrics retrieves tenant-wide step run queue metrics.
func (m *MetricsClient) GetTaskQueueMetrics(ctx context.Context) (*rest.TenantStepRunQueueMetrics, error) {
	resp, err := m.api.TenantGetStepRunQueueMetricsWithResponse(
		ctx,
		m.tenantId,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get task queue metrics")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}
