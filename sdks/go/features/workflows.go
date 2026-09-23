package features

import (
	"context"
	"fmt"
	"strconv"
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

// PauseWorkflowOpts contains the configuration for pausing a workflow.
type PauseWorkflowOpts struct {
	// QueueTTL is how long runs stay queued while the workflow is paused before they are dropped.
	QueueTTL time.Duration

	// (optional) CronRunQueueBehavior is the behavior of cron runs triggered while the workflow is paused. Defaults to QUEUE.
	CronRunQueueBehavior rest.WorkflowPauseScheduledCronRunQueueBehavior

	// (optional) ScheduledRunQueueBehavior is the behavior of scheduled runs triggered while the workflow is paused. Defaults to QUEUE.
	ScheduledRunQueueBehavior rest.WorkflowPauseScheduledCronRunQueueBehavior
}

func queueBehaviorOrDefault(behavior rest.WorkflowPauseScheduledCronRunQueueBehavior) rest.WorkflowPauseScheduledCronRunQueueBehavior {
	if behavior == "" {
		return rest.QUEUE
	}

	return behavior
}

// Pause pauses a workflow by its name. While paused, new runs of the workflow are queued but not started.
func (w *WorkflowsClient) Pause(ctx context.Context, workflowName string, opts PauseWorkflowOpts) (*rest.Workflow, error) {
	var pauseRequest rest.PauseWorkflowRequest

	err := pauseRequest.FromPauseWorkflowRequestPause(rest.PauseWorkflowRequestPause{
		Action:                                  rest.Pause,
		PausedWorkflowQueueTTL:                  strconv.FormatFloat(opts.QueueTTL.Seconds(), 'f', -1, 64) + "s",
		PausedWorkflowCronRunQueueBehavior:      queueBehaviorOrDefault(opts.CronRunQueueBehavior),
		PausedWorkflowScheduledRunQueueBehavior: queueBehaviorOrDefault(opts.ScheduledRunQueueBehavior),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to build pause request")
	}

	return w.update(ctx, workflowName, rest.WorkflowUpdateRequest{Pause: &pauseRequest})
}

// Unpause unpauses a workflow by its name.
func (w *WorkflowsClient) Unpause(ctx context.Context, workflowName string) (*rest.Workflow, error) {
	var unpauseRequest rest.PauseWorkflowRequest

	err := unpauseRequest.FromPauseWorkflowRequestUnpause(rest.PauseWorkflowRequestUnpause{
		Action: rest.Unpause,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to build unpause request")
	}

	return w.update(ctx, workflowName, rest.WorkflowUpdateRequest{Pause: &unpauseRequest})
}

func (w *WorkflowsClient) update(ctx context.Context, workflowName string, request rest.WorkflowUpdateRequest) (*rest.Workflow, error) {
	workflowId, err := w.GetId(ctx, workflowName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workflow ID")
	}

	resp, err := w.api.WorkflowUpdateWithResponse(ctx, workflowId, request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update workflow")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	w.cache.Set(workflowName, resp.JSON200)

	return resp.JSON200, nil
}
