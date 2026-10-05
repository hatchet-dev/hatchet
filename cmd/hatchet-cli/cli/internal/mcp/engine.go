package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/client" //nolint:staticcheck
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	cliconfig "github.com/hatchet-dev/hatchet/pkg/config/cli"
)

// Engine is the narrow slice of the Hatchet clients that the MCP tools need.
// Tool handlers depend on this interface so they can be tested with fakes.
type Engine interface {
	// TenantID returns the tenant the engine's credentials are scoped to.
	TenantID() string

	// TriggerWorkflow triggers a workflow run and returns the new run ID.
	TriggerWorkflow(ctx context.Context, workflow string, input map[string]any, meta map[string]any) (string, error)

	// GetRun fetches details for a run (task or DAG) by external ID.
	GetRun(ctx context.Context, runID string) (*rest.V1WorkflowRunDetails, error)

	// ListTaskEvents lists lifecycle events for a single task run.
	ListTaskEvents(ctx context.Context, taskID string) ([]rest.V1TaskEvent, error)

	// ListWorkers lists the workers registered with the tenant.
	ListWorkers(ctx context.Context) ([]rest.Worker, error)

	// ReplayRun replays an existing run in place (same run ID).
	ReplayRun(ctx context.Context, runID string) error

	// WorkflowName resolves a workflow ID to its name.
	WorkflowName(ctx context.Context, workflowID string) (string, error)

	// Meta fetches the unauthenticated server metadata.
	Meta(ctx context.Context) (*rest.APIMeta, error)

	// Version fetches the engine version, if the endpoint is available.
	Version(ctx context.Context) (string, error)
}

// EngineFactory builds an Engine for a profile. The CLI wires this to
// NewClientFromProfile; tests substitute fakes.
type EngineFactory func(profile *cliconfig.Profile) (Engine, error)

// clientEngine implements Engine on top of the SDK client (gRPC admin + REST).
type clientEngine struct {
	client client.Client //nolint:staticcheck
}

// NewClientEngine wraps an SDK client in the Engine interface.
func NewClientEngine(c client.Client) Engine { //nolint:staticcheck
	return &clientEngine{client: c}
}

func (e *clientEngine) TenantID() string {
	return e.client.TenantId()
}

func (e *clientEngine) TriggerWorkflow(ctx context.Context, workflow string, input map[string]any, meta map[string]any) (string, error) {
	// RunWorkflow does not take a context, so run it in a goroutine and bound
	// it with the caller's context (mirrors the CLI trigger command).
	type result struct {
		runID string
		err   error
	}

	resultCh := make(chan result, 1)

	go func() {
		opts := []client.RunOptFunc{} //nolint:staticcheck
		if len(meta) > 0 {
			opts = append(opts, client.WithRunMetadata(meta))
		}

		workflowRun, err := e.client.Admin().RunWorkflow(workflow, input, opts...)
		if err != nil {
			resultCh <- result{err: err}
			return
		}

		resultCh <- result{runID: workflowRun.RunId()}
	}()

	triggerCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	select {
	case res := <-resultCh:
		return res.runID, res.err
	case <-triggerCtx.Done():
		return "", fmt.Errorf("workflow trigger timed out; this may indicate a connection issue with the Hatchet server")
	}
}

func (e *clientEngine) GetRun(ctx context.Context, runID string) (*rest.V1WorkflowRunDetails, error) {
	runUUID, err := uuid.Parse(runID)
	if err != nil {
		return nil, fmt.Errorf("invalid run ID %q: %w", runID, err)
	}

	resp, err := e.client.API().V1WorkflowRunGetWithResponse(ctx, runUUID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get run: %w", err)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("run %s not found (status %d)", runID, resp.StatusCode())
	}

	return resp.JSON200, nil
}

func (e *clientEngine) ListTaskEvents(ctx context.Context, taskID string) ([]rest.V1TaskEvent, error) {
	taskUUID, err := uuid.Parse(taskID)
	if err != nil {
		return nil, fmt.Errorf("invalid run ID %q: %w", taskID, err)
	}

	limit := int64(1000)
	offset := int64(0)

	resp, err := e.client.API().V1TaskEventListWithResponse(ctx, taskUUID, &rest.V1TaskEventListParams{
		Limit:  &limit,
		Offset: &offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events: %w", err)
	}
	if resp.JSON200 == nil || resp.JSON200.Rows == nil {
		return nil, nil
	}

	return *resp.JSON200.Rows, nil
}

func (e *clientEngine) ListWorkers(ctx context.Context) ([]rest.Worker, error) {
	tenantUUID, err := uuid.Parse(e.client.TenantId())
	if err != nil {
		return nil, fmt.Errorf("invalid tenant ID: %w", err)
	}

	resp, err := e.client.API().WorkerListWithResponse(ctx, tenantUUID, &rest.WorkerListParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to list workers: %w", err)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("unexpected response from API (status %d)", resp.StatusCode())
	}
	if resp.JSON200.Rows == nil {
		return nil, nil
	}

	return *resp.JSON200.Rows, nil
}

func (e *clientEngine) ReplayRun(ctx context.Context, runID string) error {
	runUUID, err := uuid.Parse(runID)
	if err != nil {
		return fmt.Errorf("invalid run ID %q: %w", runID, err)
	}

	tenantUUID, err := uuid.Parse(e.client.TenantId())
	if err != nil {
		return fmt.Errorf("invalid tenant ID: %w", err)
	}

	resp, err := e.client.API().V1TaskReplayWithResponse(ctx, tenantUUID, rest.V1ReplayTaskRequest{
		ExternalIds: &[]uuid.UUID{runUUID},
	})
	if err != nil {
		return fmt.Errorf("failed to replay run: %w", err)
	}
	if resp.JSON200 == nil {
		return fmt.Errorf("unexpected response from API (status %d)", resp.StatusCode())
	}

	return nil
}

func (e *clientEngine) WorkflowName(ctx context.Context, workflowID string) (string, error) {
	workflowUUID, err := uuid.Parse(workflowID)
	if err != nil {
		return "", fmt.Errorf("invalid workflow ID %q: %w", workflowID, err)
	}

	resp, err := e.client.API().WorkflowGetWithResponse(ctx, workflowUUID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch workflow: %w", err)
	}
	if resp.JSON200 == nil {
		return "", fmt.Errorf("workflow %s not found (status %d)", workflowID, resp.StatusCode())
	}

	return resp.JSON200.Name, nil
}

func (e *clientEngine) Meta(ctx context.Context) (*rest.APIMeta, error) {
	resp, err := e.client.API().MetadataGetWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not reach the API server: %w", err)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("unexpected response from API (status %d)", resp.StatusCode())
	}

	return resp.JSON200, nil
}

func (e *clientEngine) Version(ctx context.Context) (string, error) {
	resp, err := e.client.API().InfoGetVersionWithResponse(ctx)
	if err != nil {
		return "", err
	}
	if resp.JSON200 == nil {
		return "", fmt.Errorf("version endpoint returned status %d", resp.StatusCode())
	}

	return resp.JSON200.Version, nil
}
