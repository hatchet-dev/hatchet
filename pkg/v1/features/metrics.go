// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package features

import (
	"context"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// Deprecated: MetricsClient is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type MetricsClient interface {
	GetWorkflowMetrics(ctx context.Context, workflowId string, opts *rest.WorkflowGetMetricsParams) (*rest.WorkflowMetrics, error)

	GetQueueMetrics(ctx context.Context, opts *rest.TenantGetQueueMetricsParams) (*rest.TenantGetQueueMetricsResponse, error)

	GetTaskQueueMetrics(ctx context.Context) (*rest.TenantGetStepRunQueueMetricsResponse, error)
}

// metricsClientImpl implements the MetricsClient interface.
type metricsClientImpl struct {
	api       *rest.ClientWithResponses
	tenantId  uuid.UUID
	workflows *WorkflowsClient
}

// Deprecated: NewMetricsClient is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
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

// Deprecated: GetWorkflowMetrics is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
//
// GetWorkflowMetrics retrieves metrics for a specific workflow.
func (m *metricsClientImpl) GetWorkflowMetrics(ctx context.Context, workflowName string, opts *rest.WorkflowGetMetricsParams) (*rest.WorkflowMetrics, error) {

	workflowId, err := (*m.workflows).GetId(ctx, workflowName)

	if err != nil {
		return nil, err
	}

	resp, err := m.api.WorkflowGetMetricsWithResponse(
		ctx,
		workflowId,
		opts,
	)

	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Deprecated: GetQueueMetrics is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
//
// GetQueueMetrics retrieves tenant-wide queue metrics.
func (m *metricsClientImpl) GetQueueMetrics(ctx context.Context, opts *rest.TenantGetQueueMetricsParams) (*rest.TenantGetQueueMetricsResponse, error) {
	return m.api.TenantGetQueueMetricsWithResponse(
		ctx,
		m.tenantId,
		opts,
	)
}

// Deprecated: GetTaskQueueMetrics is part of the old generics-based v1 Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
//
// GetTaskQueueMetrics retrieves tenant-wide step run queue metrics.
func (m *metricsClientImpl) GetTaskQueueMetrics(ctx context.Context) (*rest.TenantGetStepRunQueueMetricsResponse, error) {
	return m.api.TenantGetStepRunQueueMetricsWithResponse(
		ctx,
		m.tenantId,
	)
}
