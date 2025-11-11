package features

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// WorkflowRunRef is a type that represents a reference to a workflow run.
type WorkflowRunRef struct {
	RunId      string
	v0Workflow *client.Workflow
}

// NewWorkflowRunRef creates a new WorkflowRunRef from a runId and v0Workflow.
func NewWorkflowRunRef(v0Workflow *client.Workflow) *WorkflowRunRef {
	return &WorkflowRunRef{RunId: v0Workflow.RunId(), v0Workflow: v0Workflow}
}

// Result returns the result of the workflow run.
func (wr *WorkflowRunRef) Result() (*WorkflowResult, error) {
	result, err := wr.v0Workflow.Result()
	if err != nil {
		return nil, err
	}

	workflowResult, err := result.Results()
	if err != nil {
		return nil, err
	}

	return NewWorkflowResult(wr.RunId, workflowResult), nil
}

// WorkflowResult wraps workflow execution results and provides type-safe conversion methods.
type WorkflowResult struct {
	RunId  string
	Result any
}

// NewWorkflowResult creates a new WorkflowResult.
func NewWorkflowResult(runId string, result any) *WorkflowResult {
	return &WorkflowResult{RunId: runId, Result: result}
}

// TaskResult wraps a single task's output and provides type-safe conversion methods.
type TaskResult struct {
	RunId  string
	Result any
}

// TaskOutput extracts the output of a specific task from the workflow result.
// Returns a TaskResult that can be used to convert the task output into the desired type.
//
// Example usage:
//
//	taskResult := workflowResult.TaskOutput("myTask")
//	var output MyOutputType
//	err := taskResult.Into(&output)
func (wr *WorkflowResult) TaskOutput(taskName string) *TaskResult {
	// Handle different result structures that might come from workflow execution
	resultData := wr.Result

	taskResult := &TaskResult{RunId: wr.RunId}

	// Check if this is a raw client.WorkflowResult that we need to extract from
	if workflowResult, ok := resultData.(*client.WorkflowResult); ok {
		// Try to get the workflow results as a map
		results, err := workflowResult.Results()
		if err != nil {
			// Return empty TaskResult if we can't extract results
			return taskResult
		}
		resultData = results
	}

	// If the result is a map, look for the specific task
	if resultMap, ok := resultData.(map[string]any); ok {
		if taskOutput, exists := resultMap[taskName]; exists {
			taskResult.Result = taskOutput
			return taskResult
		}
	}

	// If we can't find the specific task, return the entire result
	// This handles cases where there's only one task
	taskResult.Result = resultData
	return taskResult
}

// Into converts the task result into the provided destination using JSON marshal/unmarshal.
// The destination should be a pointer to the desired type.
//
// Example usage:
//
//	var output MyOutputType
//	err := taskResult.Into(&output)
func (tr *TaskResult) Into(dest any) error {
	// Handle different result structures that might come from task execution
	resultData := tr.Result

	// If the result is a pointer to interface{}, dereference it
	if ptr, ok := resultData.(*any); ok && ptr != nil {
		resultData = *ptr
	}

	// If the result is a pointer to string (JSON), unmarshal it directly
	if strPtr, ok := resultData.(*string); ok && strPtr != nil {
		return json.Unmarshal([]byte(*strPtr), dest)
	}

	// Convert the result to JSON and then unmarshal to destination
	jsonData, err := json.Marshal(resultData)
	if err != nil {
		return fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	if err := json.Unmarshal(jsonData, dest); err != nil {
		return fmt.Errorf("failed to unmarshal JSON to destination: %w", err)
	}

	return nil
}

// Raw returns the raw workflow result as interface{}.
func (wr *WorkflowResult) Raw() any {
	return wr.Result
}

// RunsClient provides methods for interacting with workflow runs
type RunsClient struct {
	api      *rest.ClientWithResponses
	tenantId uuid.UUID
	v0Client client.Client
	l        *zerolog.Logger
}

// NewRunsClient creates a new client for interacting with workflow runs.
func NewRunsClient(
	api *rest.ClientWithResponses,
	tenantId string,
	v0Client client.Client,
) *RunsClient {
	tenantIdUUID := uuid.MustParse(tenantId)
	logger := v0Client.Logger()

	return &RunsClient{
		api:      api,
		tenantId: tenantIdUUID,
		v0Client: v0Client,
		l:        logger,
	}
}

// Get retrieves a workflow run by its ID.
func (r *RunsClient) Get(ctx context.Context, runId string) (*rest.V1WorkflowRunDetails, error) {
	resp, err := r.api.V1WorkflowRunGetWithResponse(
		ctx,
		uuid.MustParse(runId),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workflow run")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// GetStatus retrieves the status of a workflow run by its ID.
func (r *RunsClient) GetStatus(ctx context.Context, runId string) (*rest.V1TaskStatus, error) {
	resp, err := r.api.V1WorkflowRunGetStatusWithResponse(
		ctx,
		uuid.MustParse(runId),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workflow run status")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// List retrieves a collection of workflow runs based on the provided parameters.
func (r *RunsClient) List(ctx context.Context, opts rest.V1WorkflowRunListParams) (*rest.V1TaskSummaryList, error) {
	resp, err := r.api.V1WorkflowRunListWithResponse(
		ctx,
		r.tenantId,
		&opts,
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list workflow runs")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Replay requests a task to be replayed within a workflow run.
func (r *RunsClient) Replay(ctx context.Context, opts rest.V1ReplayTaskRequest) (*rest.V1ReplayedTasks, error) {
	json, err := json.Marshal(opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal rest.V1ReplayTaskRequest")
	}

	resp, err := r.api.V1TaskReplayWithBodyWithResponse(
		ctx,
		r.tenantId,
		"application/json; charset=utf-8",
		bytes.NewReader(json),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to replay task")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

// Cancel requests cancellation of a specific task within a workflow run.
func (r *RunsClient) Cancel(ctx context.Context, opts rest.V1CancelTaskRequest) (*rest.V1CancelledTasks, error) {
	json, err := json.Marshal(opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed marshal rest.V1CancelTaskRequest")
	}

	resp, err := r.api.V1TaskCancelWithBodyWithResponse(
		ctx,
		r.tenantId,
		"application/json; charset=utf-8",
		bytes.NewReader(json),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to cancel task")
	}

	if err := validateJSON200Response(resp.StatusCode(), resp.Body, resp.JSON200); err != nil {
		return nil, err
	}

	return resp.JSON200, nil
}

func (r *RunsClient) GetRunRef(ctx context.Context, runId string) (*WorkflowRunRef, error) {
	listener, err := r.v0Client.Subscribe().SubscribeToWorkflowRunEvents(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to subscribe to workflow run events")
	}

	v0Workflow := client.NewWorkflow(runId, listener)

	return &WorkflowRunRef{RunId: runId, v0Workflow: v0Workflow}, nil
}

// SubscribeToStream subscribes to streaming events for a specific workflow run.
func (r *RunsClient) SubscribeToStream(ctx context.Context, workflowRunId string) <-chan string {
	ch := make(chan string)

	go func() {
		defer func() {
			close(ch)
			r.l.Info().Str("workflowRunId", workflowRunId).Msg("stream subscription ended")
		}()

		r.l.Info().Str("workflowRunId", workflowRunId).Msg("starting stream subscription")

		err := r.v0Client.Subscribe().Stream(ctx, workflowRunId, func(event client.StreamEvent) error {
			select {
			case ch <- string(event.Message):
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		})
		if err != nil {
			r.l.Error().Err(err).Str("workflowRunId", workflowRunId).Msg("failed to subscribe to stream")
		}
	}()

	return ch
}
