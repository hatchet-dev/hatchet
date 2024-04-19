package worker

import (
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/client"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
)

type ChildWorkflow struct {
	workflowRunId string
	// client        client.Client
	l        *zerolog.Logger
	listener *client.WorkflowRunsListener
}

type ChildWorkflowResult struct {
	workflowRun *dispatchercontracts.WorkflowRunEvent
}

func (r *ChildWorkflowResult) StepOutput(key string, v interface{}) error {
	var outputBytes []byte
	for _, stepRunResult := range r.workflowRun.Results {
		if stepRunResult.StepReadableId == key {
			if stepRunResult.Error != nil {
				return fmt.Errorf("step run failed: %s", *stepRunResult.Error)
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

func (c *ChildWorkflow) Result() (*ChildWorkflowResult, error) {
	resChan := make(chan *ChildWorkflowResult)

	err := c.listener.AddWorkflowRun(
		c.workflowRunId,
		func(event client.WorkflowRunEvent) error {
			// non-blocking send
			select {
			case resChan <- &ChildWorkflowResult{
				workflowRun: event,
			}: // continue
			default:
			}

			return nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to listen for workflow events: %w", err)
	}

	res := <-resChan

	return res, nil
}
