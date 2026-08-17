// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

const (
	defaultResultPollGrace    = 5 * time.Second
	defaultResultPollInterval = time.Second
)

// Workflow represents a running workflow instance and provides methods to retrieve its results.
//
// The workflow listener uses a multi-layer best-effort retry strategy to handle transient failures
// and provides robust recovery from temporary connection issues like brief DB downtime
// or network interruptions without requiring manual intervention.
type Workflow struct {
	workflowRunId      string
	listener           *WorkflowRunsListener
	fetchRunDetails    func(context.Context, uuid.UUID) (*RunDetails, error)
	resultPollGrace    time.Duration
	resultPollInterval time.Duration
}

func NewWorkflow(
	workflowRunId string,
	listener *WorkflowRunsListener,
	fetchers ...func(context.Context, uuid.UUID) (*RunDetails, error),
) *Workflow {
	var fetchRunDetails func(context.Context, uuid.UUID) (*RunDetails, error)
	if len(fetchers) > 0 {
		fetchRunDetails = fetchers[0]
	}

	return &Workflow{
		workflowRunId:   workflowRunId,
		listener:        listener,
		fetchRunDetails: fetchRunDetails,
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
// AddWorkflowRun is attempted once; it uses bounded synchronous reconnect
// (StreamSyncMaxAttempts) for send and subscribe failures. The background
// listen loop reconnects unboundedly while the listener remains open.
func (r *Workflow) Result() (*WorkflowResult, error) {
	return r.result(nil)
}

// ResultWithContext waits for the workflow run to complete and logs if the
// caller's context ends after the result subscription was sent successfully.
// Context cancellation is diagnostic only and does not change Result behavior.
func (r *Workflow) ResultWithContext(ctx context.Context) (*WorkflowResult, error) {
	return r.result(ctx)
}

func (r *Workflow) result(ctx context.Context) (*WorkflowResult, error) {
	resChan := make(chan *WorkflowResult, 1)
	failChan := make(chan error, 1)
	sessionId := uuid.NewString()

	err := r.listener.addWorkflowRun(r.workflowRunId, sessionId,
		func(event WorkflowRunEvent) error {
			resChan <- &WorkflowResult{workflowRun: event}
			return nil
		},
		func(err error) {
			select {
			case failChan <- err:
			default:
			}
		},
	)
	if err != nil {
		r.listener.RemoveWorkflowRun(r.workflowRunId, sessionId)
		return nil, fmt.Errorf("failed to listen for workflow events: %w", err)
	}
	defer r.listener.RemoveWorkflowRun(r.workflowRunId, sessionId)

	waitStarted := time.Now()
	var contextDone <-chan struct{}
	if ctx != nil {
		contextDone = ctx.Done()
	}

	var (
		poll          <-chan time.Time
		ticker        *time.Ticker
		workflowRunId uuid.UUID
	)
	if r.fetchRunDetails != nil {
		workflowRunId, err = uuid.Parse(r.workflowRunId)
		if err != nil {
			r.listener.l.Debug().
				Err(err).
				Str("workflow_run_id", r.workflowRunId).
				Msg("workflow result polling disabled for invalid workflow run id")
		} else {
			grace := r.resultPollGrace
			if grace == 0 {
				grace = defaultResultPollGrace
			}
			graceTimer := time.NewTimer(grace)
			defer graceTimer.Stop()
			poll = graceTimer.C
		}
	}
	defer func() {
		if ticker != nil {
			ticker.Stop()
		}
	}()

	for {
		select {
		case res := <-resChan:
			return resultFromEvent(res)
		case err := <-failChan:
			return nil, fmt.Errorf("workflow run listener terminated while waiting for %s: %w", r.workflowRunId, err)
		case <-contextDone:
			r.listener.l.Warn().
				Err(ctx.Err()).
				Str("workflow_run_id", r.workflowRunId).
				Dur("wait_duration", time.Since(waitStarted)).
				Msg("workflow result still pending after subscription send succeeded")
			contextDone = nil
		case <-poll:
			if ticker == nil {
				interval := r.resultPollInterval
				if interval == 0 {
					interval = defaultResultPollInterval
				}
				ticker = time.NewTicker(interval)
				poll = ticker.C
			}

			select {
			case res := <-resChan:
				return resultFromEvent(res)
			default:
			}

			fetchCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			details, err := r.fetchRunDetails(fetchCtx, workflowRunId)
			cancel()
			if err != nil {
				r.listener.l.Debug().
					Err(err).
					Str("workflow_run_id", r.workflowRunId).
					Msg("workflow result polling failed")
				continue
			}
			if details == nil || !details.Done {
				continue
			}

			return runDetailsToResult(details)
		}
	}
}

func resultFromEvent(res *WorkflowResult) (*WorkflowResult, error) {
	for _, stepRunResult := range res.workflowRun.Results {
		if stepRunResult.Error != nil {
			return nil, fmt.Errorf("%s", *stepRunResult.Error)
		}
	}
	return res, nil
}

func runDetailsToResult(details *RunDetails) (*WorkflowResult, error) {
	if details.Status == rest.V1TaskStatusCOMPLETED {
		return runDetailsToWorkflowResult(details), nil
	}

	if msg := firstTaskError(details); msg != "" {
		return nil, fmt.Errorf("%s", msg)
	}
	if details.Status == rest.V1TaskStatusCANCELLED {
		return nil, fmt.Errorf("workflow run %s was cancelled", details.ExternalId)
	}
	return nil, fmt.Errorf("workflow run %s failed", details.ExternalId)
}

func firstTaskError(details *RunDetails) string {
	ids := make([]string, 0, len(details.TaskRuns))
	for id := range details.TaskRuns {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		taskRun := details.TaskRuns[id]
		if taskRun != nil && taskRun.Error != nil && *taskRun.Error != "" {
			return *taskRun.Error
		}
	}
	return ""
}

func runDetailsToWorkflowResult(details *RunDetails) *WorkflowResult {
	results := make([]*StepRunResult, 0, len(details.TaskRuns))
	for _, taskRun := range details.TaskRuns {
		if taskRun == nil {
			continue
		}

		result := &StepRunResult{
			StepRunId:      taskRun.ExternalId.String(),
			StepReadableId: taskRun.ReadableId,
		}
		if len(taskRun.Output) > 0 {
			output := string(taskRun.Output)
			result.Output = &output
		}
		results = append(results, result)
	}

	return &WorkflowResult{
		workflowRun: &workflowRunEvent{
			WorkflowRunId: details.ExternalId.String(),
			Results:       results,
			EventType:     WorkflowRunEventType_WORKFLOW_RUN_EVENT_TYPE_FINISHED,
		},
	}
}
