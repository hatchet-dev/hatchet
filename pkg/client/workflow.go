package client

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
)

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
	workflowRun *dispatchercontracts.WorkflowRunEvent
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

func (c *Workflow) Result() (*WorkflowResult, error) {
	resChan := make(chan *WorkflowResult, 1)
	sessionId := uuid.NewString()

	err := c.listener.AddWorkflowRun(
		c.workflowRunId,
		sessionId,
		func(event WorkflowRunEvent) error {
			resChan <- &WorkflowResult{
				workflowRun: event,
			}

			return nil
		},
	)

	defer func() {
		c.listener.RemoveWorkflowRun(c.workflowRunId, sessionId)
	}()

	if err != nil {
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
