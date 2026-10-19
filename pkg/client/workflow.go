// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// Workflow represents a running workflow instance and provides methods to retrieve its results.
//
// The workflow listener uses a multi-layer best-effort retry strategy to handle transient failures
// and provides robust recovery from temporary connection issues like brief DB downtime
// or network interruptions without requiring manual intervention.
type Workflow struct {
	workflowRunId string
	listener      *WorkflowRunsListener
}

func NewWorkflow(
	workflowRunId string,
	listener *WorkflowRunsListener,
) *Workflow {
	return &Workflow{
		workflowRunId: workflowRunId,
		listener:      listener,
	}
}

func (r *Workflow) RunId() string {
	return r.workflowRunId
}

// Deprecated: Use RunId instead
func (r *Workflow) WorkflowRunId() string {
	return r.workflowRunId
}

type WorkflowResult struct {
	workflowRun *workflowRunEvent
}

func (r *WorkflowResult) StepOutput(key string, v interface{}) error {
	var outputBytes []byte
	for _, stepRunResult := range r.workflowRun.Results {
		if stepRunResult.StepReadableId == key {
			if stepRunResult.Error != nil {
				return fmt.Errorf("%s", *stepRunResult.Error)
			}

			if stepRunResult.Output != nil {
				outputBytes = []byte(*stepRunResult.Output)
			}
		}
	}

	if outputBytes == nil {
		return fmt.Errorf("step output for %s not found", key)
	}

	if err := json.Unmarshal(outputBytes, v); err != nil {
		return fmt.Errorf("failed to unmarshal output: %w", err)
	}

	return nil
}

// Results returns a map of all step outputs from the workflow run.
//
// Note: This method operates on an already-fetched WorkflowResult. The retry logic
// is handled by Workflow.Result() which obtains the WorkflowResult.
func (r *WorkflowResult) Results() (interface{}, error) {
	results := make(map[string]interface{})

	for _, stepRunResult := range r.workflowRun.Results {
		if stepRunResult.Error != nil {
			return nil, fmt.Errorf("run failed: %s", *stepRunResult.Error)
		}

		if stepRunResult.Output != nil {
			results[stepRunResult.StepReadableId] = stepRunResult.Output
		}
	}

	return results, nil
}

// Result waits for the workflow run to complete and returns the results.
// It waits indefinitely; use ResultWithContext to bound the wait.
//
// AddWorkflowRun is attempted once; it uses bounded synchronous reconnect
// (StreamSyncMaxAttempts) for send and subscribe failures. The background
// listen loop reconnects unboundedly while the listener remains open.
func (r *Workflow) Result() (*WorkflowResult, error) {
	return r.result(nil)
}

// ResultWithContext waits for the workflow run to complete and returns the
// results. The whole wait is bounded by ctx, including the subscription
// setup (connect and send retries with their backoff sleeps): if ctx ends
// first, the wait stops and the returned error wraps ctx.Err(), so callers
// can check errors.Is(err, context.DeadlineExceeded) or context.Canceled.
// A connect or send attempt already in flight when ctx ends never blocks the
// caller; it is left to finish in the background, bounded by the stream's
// own lifecycle, and any registration it makes is removed. Ending the wait
// only drops this subscription; the shared listener stays up for other
// waiters and the run itself keeps executing on the engine.
func (r *Workflow) ResultWithContext(ctx context.Context) (*WorkflowResult, error) {
	return r.result(ctx)
}

// result subscribes to the run's terminal event and waits for it. A nil ctx
// waits indefinitely; otherwise both the subscription setup and the wait end
// with ctx.Err() when ctx ends. The subscription is removed on every return
// path, and the run keeps executing on the engine regardless of how the wait
// ended.
func (r *Workflow) result(ctx context.Context) (*WorkflowResult, error) {
	resChan := make(chan *WorkflowResult, 1)
	failChan := make(chan error, 1)
	sessionId := uuid.NewString()

	onEvent := func(event WorkflowRunEvent) error {
		// Non-blocking: the waiter may already have returned on ctx and
		// stopped reading, and this runs on the listener's dispatch
		// goroutine.
		select {
		case resChan <- &WorkflowResult{workflowRun: event}:
		default:
		}
		return nil
	}
	onError := func(err error) {
		select {
		case failChan <- err:
		default:
		}
	}

	var err error
	if ctx == nil {
		err = r.listener.addWorkflowRun(r.workflowRunId, sessionId, onEvent, onError)
	} else {
		err = r.listener.addWorkflowRunContext(ctx, r.workflowRunId, sessionId, onEvent, onError)
	}
	if err != nil {
		r.listener.RemoveWorkflowRun(r.workflowRunId, sessionId)
		return nil, fmt.Errorf("failed to listen for workflow events: %w", err)
	}
	defer r.listener.RemoveWorkflowRun(r.workflowRunId, sessionId)

	var contextDone <-chan struct{}
	if ctx != nil {
		contextDone = ctx.Done()
	}

	return awaitWorkflowResult(r.workflowRunId, resChan, failChan, contextDone, func() error { return ctx.Err() })
}

// awaitWorkflowResult is the wait loop behind result, split out so it can be
// exercised without a listener. contextDone may be nil, in which case the
// wait never ends on its own; ctxErr is only called once contextDone fires.
func awaitWorkflowResult(
	workflowRunId string,
	resChan <-chan *WorkflowResult,
	failChan <-chan error,
	contextDone <-chan struct{},
	ctxErr func() error,
) (*WorkflowResult, error) {
	select {
	case res := <-resChan:
		for _, stepRunResult := range res.workflowRun.Results {
			if stepRunResult.Error != nil {
				return nil, fmt.Errorf("%s", *stepRunResult.Error)
			}
		}
		return res, nil
	case err := <-failChan:
		return nil, fmt.Errorf("workflow run listener terminated while waiting for %s: %w", workflowRunId, err)
	case <-contextDone:
		return nil, fmt.Errorf("waiting for workflow run %s result: %w", workflowRunId, ctxErr())
	}
}
