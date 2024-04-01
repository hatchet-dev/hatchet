package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
)

type ChildWorkflow struct {
	workflowRunId string
	client        client.Client
	l             *zerolog.Logger
}

type ChildWorkflowResult struct {
	workflowRun *rest.WorkflowRun
}

func (r *ChildWorkflowResult) StepOutput(key string, v interface{}) error {
	var outputBytes []byte
	for _, jobRun := range *r.workflowRun.JobRuns {
		for _, stepRun := range *jobRun.StepRuns {
			if stepRun.Step.ReadableId == key && stepRun.Output != nil {
				outputBytes = []byte(*stepRun.Output)
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resChan := make(chan *ChildWorkflowResult)
	workflowErrChan := make(chan error)
	errChan := make(chan error)

	f := func() {
		resp, err := c.client.API().WorkflowRunGetWithResponse(
			ctx,
			uuid.MustParse(c.client.TenantId()),
			uuid.MustParse(c.workflowRunId),
		)

		if err != nil {
			errChan <- fmt.Errorf("failed to get workflow run: %w", err)
			return
		}

		if workflowRun := resp.JSON200; workflowRun != nil {
			if workflowRun.Status == rest.SUCCEEDED {
				// write the workflow run to the channel
				resChan <- &ChildWorkflowResult{
					workflowRun: workflowRun,
				}
			}

			if workflowRun.Status == rest.FAILED || workflowRun.Status == rest.CANCELLED {
				// write the error to the channel
				workflowErrChan <- fmt.Errorf("workflow run failed with status %s", workflowRun.Status)
			}
		} else {
			errChan <- fmt.Errorf("request failed with status %d", resp.StatusCode())
			return
		}
	}

	// start two goroutines: one which polls the API for the workflow run result, and one which listens for
	// workflow finished events
	go func() {
		f()

		// poll the API for the workflow run result
		ticker := time.NewTicker(5 * time.Second)

		for {
			select {
			case <-ticker.C:
				f()
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		// listen for workflow finished events
		err := c.client.Subscribe().On(
			ctx,
			c.workflowRunId,
			func(event client.RunEvent) error {
				if event.ResourceType == dispatchercontracts.ResourceType_RESOURCE_TYPE_WORKFLOW_RUN {
					if event.EventType == dispatchercontracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED {
						f()
					}
				}

				return nil
			},
		)

		if err != nil {
			errChan <- fmt.Errorf("failed to listen for workflow events: %w", err)
		}
	}()

	select {
	case res := <-resChan:
		return res, nil
	case err := <-workflowErrChan:
		return nil, err
	case err := <-errChan:
		c.l.Err(err).Msg("error occurred")
		return nil, err
	}
}
