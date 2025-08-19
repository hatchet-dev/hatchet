package features

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
)

// WorkflowsClient provides methods for interacting with workflows
// in the Hatchet platform.
type WorkflowsClient interface {
	// Get retrieves a workflow by its name.
	Get(ctx context.Context, workflowName string) (*rest.Workflow, error)

	// GetId retrieves a workflow by its name.
	GetId(ctx context.Context, workflowName string) (uuid.UUID, error)

	// List retrieves all workflows for the tenant with optional filtering parameters.
	List(ctx context.Context, opts *rest.WorkflowListParams) (*rest.WorkflowList, error)

	// Delete removes a workflow by its name.
	Delete(ctx context.Context, workflowName string) (*rest.WorkflowDeleteResponse, error)

	// // IsPaused checks if a workflow is paused.
	// IsPaused(ctx context.Context, workflowName string) (bool, error)

	// // Pause pauses a workflow and prevents runs from being scheduled.
	// Pause(ctx context.Context, workflowName string) (*rest.Workflow, error)

	// // Unpause unpauses a workflow and allows runs to be scheduled.
	// Unpause(ctx context.Context, workflowName string) (*rest.Workflow, error)
}

// workflowsClientImpl implements the WorkflowsClient interface.
type workflowsClientImpl struct {
	api      *rest.ClientWithResponses
	tenantId uuid.UUID
	cache    *cache.Cache
}

// NewWorkflowsClient creates a new client for interacting with workflows.
func NewWorkflowsClient(
	api *rest.ClientWithResponses,
	tenantId *string,
) WorkflowsClient {
	tenantIdUUID := uuid.MustParse(*tenantId)

	// Create a cache with the specified TTL
	workflowCache := cache.New(time.Minute * 5)

	return &workflowsClientImpl{
		api:      api,
		tenantId: tenantIdUUID,
		cache:    workflowCache,
	}
}

// Get retrieves a workflow by its ID or name.
func (w *workflowsClientImpl) Get(ctx context.Context, workflowName string) (*rest.Workflow, error) {
	// Try to get the workflow from cache first
	cacheKey := workflowName
	cachedWorkflow, found := w.cache.Get(cacheKey)
	if found {
		return cachedWorkflow.(*rest.Workflow), nil
	}

	// FIXME: this is a hack to get the workflow by name
	resp, err := w.api.WorkflowListWithResponse(
		ctx,
		w.tenantId,
		&rest.WorkflowListParams{
			Name: &workflowName,
		},
	)

	if err != nil {
		return nil, err
	}

	if resp.JSON200 == nil || len(*resp.JSON200.Rows) == 0 {
		return nil, fmt.Errorf("workflow with name %s not found", workflowName)
	}

	workflow := (*resp.JSON200.Rows)[0]

	// Update cache
	w.cache.Set(cacheKey, &workflow)

	return &workflow, nil
}

// GetId retrieves a workflow by its name.
func (w *workflowsClientImpl) GetId(ctx context.Context, workflowName string) (uuid.UUID, error) {
	workflow, err := w.Get(ctx, workflowName)
	if err != nil {
		return uuid.Nil, err
	}

	return uuid.MustParse(workflow.Metadata.Id), nil
}

// List retrieves all workflows for the tenant with optional filtering parameters.
func (w *workflowsClientImpl) List(ctx context.Context, opts *rest.WorkflowListParams) (*rest.WorkflowList, error) {
	resp, err := w.api.WorkflowListWithResponse(
		ctx,
		w.tenantId,
		opts,
	)
	if err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Delete removes a workflow by its ID or name.
func (w *workflowsClientImpl) Delete(ctx context.Context, workflowName string) (*rest.WorkflowDeleteResponse, error) {
	// FIXME: this is a hack to get the workflow by name
	workflowId, err := w.GetId(ctx, workflowName)
	if err != nil {
		return nil, err
	}

	resp, err := w.api.WorkflowDeleteWithResponse(
		ctx,
		workflowId,
	)

	if err != nil {
		return nil, err
	}

	// Remove from cache after deletion
	w.cache.Set(workflowName, nil)

	return resp, nil
}

// // IsPaused checks if a workflow is paused.
// func (w *workflowsClientImpl) IsPaused(ctx context.Context, workflowName string) (bool, error) {
// 	workflow, err := w.Get(ctx, workflowName)
// 	if err != nil {
// 		return false, err
// 	}

// 	if workflow.IsPaused == nil {
// 		return false, nil
// 	}

// 	return *workflow.IsPaused, nil
// }

// // Pause pauses a workflow.
// func (w *workflowsClientImpl) Pause(ctx context.Context, workflowName string) (*rest.Workflow, error) {
// 	// FIXME: this is a hack to get the workflow by name
// 	workflow, err := w.Get(ctx, workflowName)
// 	if err != nil {
// 		return nil, err
// 	}

// 	id := uuid.MustParse(workflow.Metadata.Id)

// 	paused := true

// 	request := rest.WorkflowUpdateJSONRequestBody{
// 		IsPaused: &paused,
// 	}

// 	resp, err := w.api.WorkflowUpdateWithResponse(
// 		ctx,
// 		id,
// 		request,
// 	)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Update cache with new paused state
// 	if resp.JSON200 != nil {
// 		w.cache.Set(workflowName, resp.JSON200)
// 	}

// 	return resp.JSON200, nil
// }

// // Unpause unpauses a workflow.
// func (w *workflowsClientImpl) Unpause(ctx context.Context, workflowName string) (*rest.Workflow, error) {
// 	// FIXME: this is a hack to get the workflow by name
// 	workflow, err := w.Get(ctx, workflowName)
// 	if err != nil {
// 		return nil, err
// 	}

// 	id := uuid.MustParse(workflow.Metadata.Id)

// 	paused := false

// 	request := rest.WorkflowUpdateJSONRequestBody{
// 		IsPaused: &paused,
// 	}

// 	resp, err := w.api.WorkflowUpdateWithResponse(
// 		ctx,
// 		id,
// 		request,
// 	)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Update cache with new unpaused state
// 	if resp.JSON200 != nil {
// 		w.cache.Set(workflowName, resp.JSON200)
// 	}

// 	return resp.JSON200, nil
// }
