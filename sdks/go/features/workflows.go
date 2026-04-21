package features

import (
	"context"
	"fmt"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/repository/cache"
)

// WorkflowsClient provides methods for interacting with workflows
type WorkflowsClient struct {
	api      *rest.ClientWithResponses
	tenantId uuid.UUID
	cache    *cache.Cache
}

// NewWorkflowsClient creates a new WorkflowsClient
func NewWorkflowsClient(
	api *rest.ClientWithResponses,
	tenantId string,
) *WorkflowsClient {
	tenantIdUUID := uuid.MustParse(tenantId)

	// Create a cache with the specified TTL
	workflowCache := cache.New(time.Minute * 5)

	return &WorkflowsClient{
		api:      api,
		tenantId: tenantIdUUID,
		cache:    workflowCache,
	}
}

// Get retrieves a workflow by its ID or name.
func (w *WorkflowsClient) Get(ctx context.Context, workflowName string) (*rest.Workflow, error) {
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
		return nil, errors.Wrap(err, "failed to get workflow")
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
func (w *WorkflowsClient) GetId(ctx context.Context, workflowName string) (uuid.UUID, error) {
	workflow, err := w.Get(ctx, workflowName)
	if err != nil {
		return uuid.Nil, errors.Wrap(err, "failed to get workflow ID")
	}

	return uuid.MustParse(workflow.Metadata.Id), nil
}

// List retrieves all workflows for the tenant with optional filtering parameters.
func (w *WorkflowsClient) List(ctx context.Context, opts *rest.WorkflowListParams) (*rest.WorkflowList, error) {
	resp, err := w.api.WorkflowListWithResponse(
		ctx,
		w.tenantId,
		opts,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list workflows")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Delete removes a workflow by its ID or name.
func (w *WorkflowsClient) Delete(ctx context.Context, workflowName string) (*rest.WorkflowDeleteResponse, error) {
	// FIXME: this is a hack to get the workflow by name
	workflowId, err := w.GetId(ctx, workflowName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workflow ID")
	}

	resp, err := w.api.WorkflowDeleteWithResponse(
		ctx,
		workflowId,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to delete workflow")
	}

	if err := validateStatusCodeResponse(resp.StatusCode(), resp.Body); err != nil {
		return nil, err
	}

	// Remove from cache after deletion
	w.cache.Set(workflowName, nil)

	return resp, nil
}

// Pause pauses a workflow by its name.
func (w *WorkflowsClient) Pause(ctx context.Context, workflowName string) (*rest.Workflow, error) {
	workflowId, err := w.GetId(ctx, workflowName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workflow ID")
	}

	paused := true

	resp, err := w.api.WorkflowUpdateWithResponse(
		ctx,
		workflowId,
		rest.WorkflowUpdateJSONRequestBody{
			IsPaused: &paused,
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to pause workflow")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	w.cache.Set(workflowName, resp.JSON200)

	return resp.JSON200, nil
}

// Unpause unpauses a workflow by its name.
func (w *WorkflowsClient) Unpause(ctx context.Context, workflowName string) (*rest.Workflow, error) {
	workflowId, err := w.GetId(ctx, workflowName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workflow ID")
	}

	paused := false

	resp, err := w.api.WorkflowUpdateWithResponse(
		ctx,
		workflowId,
		rest.WorkflowUpdateJSONRequestBody{
			IsPaused: &paused,
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unpause workflow")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	w.cache.Set(workflowName, resp.JSON200)

	return resp.JSON200, nil
}

// IsPaused reports whether a workflow is currently paused.
func (w *WorkflowsClient) IsPaused(ctx context.Context, workflowName string) (bool, error) {
	// Bypass the in-memory cache so we always read the live server state.
	w.cache.Remove(workflowName)

	workflow, err := w.Get(ctx, workflowName)
	if err != nil {
		return false, errors.Wrap(err, "failed to get workflow")
	}

	if workflow.IsPaused == nil {
		return false, nil
	}

	return *workflow.IsPaused, nil
}
