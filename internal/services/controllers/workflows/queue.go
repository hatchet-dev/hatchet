package workflows

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/internal/telemetry/servertel"
)

func (wc *WorkflowsControllerImpl) handleWorkflowRunQueued(ctx context.Context, task *taskqueue.Task) error {
	ctx, span := telemetry.NewSpan(ctx, "handle-workflow-run-queued")
	defer span.End()

	payload := tasktypes.WorkflowRunQueuedTaskPayload{}
	metadata := tasktypes.WorkflowRunQueuedTaskMetadata{}

	err := wc.dv.DecodeAndValidate(task.Payload, &payload)

	if err != nil {
		return fmt.Errorf("could not decode job task payload: %w", err)
	}

	err = wc.dv.DecodeAndValidate(task.Metadata, &metadata)

	if err != nil {
		return fmt.Errorf("could not decode job task metadata: %w", err)
	}

	// get the workflow run in the database
	workflowRun, err := wc.repo.WorkflowRun().GetWorkflowRunById(metadata.TenantId, payload.WorkflowRunId)

	if err != nil {
		return fmt.Errorf("could not get job run: %w", err)
	}

	servertel.WithWorkflowRunModel(span, workflowRun)

	// determine if we should start this workflow run or we need to limit its concurrency
	// TODO: determine if we should limit concurrency, right now this just starts the workflow run
	err = wc.queueWorkflowRunJobs(ctx, workflowRun)

	if err != nil {
		return fmt.Errorf("could not start workflow run: %w", err)
	}

	return nil
}

func (wc *WorkflowsControllerImpl) queueWorkflowRunJobs(ctx context.Context, workflowRun *db.WorkflowRunModel) error {
	ctx, span := telemetry.NewSpan(ctx, "process-event")
	defer span.End()

	jobRuns := workflowRun.JobRuns()

	var err error

	for i := range jobRuns {
		err := wc.tq.AddTask(
			context.Background(),
			taskqueue.JOB_PROCESSING_QUEUE,
			tasktypes.JobRunQueuedToTask(jobRuns[i].Job(), &jobRuns[i]),
		)

		if err != nil {
			err = multierror.Append(err, fmt.Errorf("could not add job run to task queue: %w", err))
		}
	}

	return err
}
