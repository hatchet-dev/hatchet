// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"encoding/json"
	"fmt"
	"time"

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
//
// Retry strategy (best-effort):
// 1. This function retries AddWorkflowRun up to DefaultActionListenerRetryCount times with DefaultActionListenerRetryInterval intervals
// 2. AddWorkflowRun calls retrySend which retries up to DefaultActionListenerRetryCount times with DefaultActionListenerRetryInterval intervals
// 3. Each retrySend attempt calls retrySubscribe which itself retries up to DefaultActionListenerRetryCount times with DefaultActionListenerRetryInterval intervals
func (c *Workflow) Result() (*WorkflowResult, error) {
	resChan := make(chan *WorkflowResult, 1)
	sessionId := uuid.NewString()

	var err error
	retries := 0

	for retries < DefaultActionListenerRetryCount {
		if retries > 0 {
			time.Sleep(DefaultActionListenerRetryInterval)
		}

		err = c.listener.AddWorkflowRun(
			c.workflowRunId,
			sessionId,
			func(event WorkflowRunEvent) error {
				resChan <- &WorkflowResult{
					workflowRun: event,
				}

				return nil
			},
		)

		if err == nil {
			defer c.listener.RemoveWorkflowRun(c.workflowRunId, sessionId)

			break
		}
	}

	if retries == DefaultActionListenerRetryCount && err != nil {
		return nil, fmt.Errorf("failed to listen for workflow events: %w", err)
	}

	res := <-resChan

	for _, stepRunResult := range res.workflowRun.Results {
		if stepRunResult.Error != nil {
			return nil, fmt.Errorf("%s", *stepRunResult.Error)
		}
	}

	return res, nil
}
