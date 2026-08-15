package workflows

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *WorkflowService) WorkflowUpdate(echoCtx echo.Context, request gen.WorkflowUpdateRequestObject) (gen.WorkflowUpdateResponseObject, error) {
	tenant := echoCtx.Get("tenant").(*sqlcv1.Tenant)
	workflow := echoCtx.Get("workflow").(*sqlcv1.Workflow)
	ctx := echoCtx.Request().Context()

	var updateOpts repository.UpdateWorkflowOpts

	if request.Body.IsPaused != nil && request.Body.PausedWorkflowCronRunQueueBehavior != nil && request.Body.PausedWorkflowScheduledRunQueueBehavior != nil {
		cronBehavior := string(*request.Body.PausedWorkflowCronRunQueueBehavior)
		scheduledBehavior := string(*request.Body.PausedWorkflowScheduledRunQueueBehavior)

		updateOpts.PauseOpts = &repository.WorkflowPauseOpts{
			IsPaused:                                *request.Body.IsPaused,
			PausedWorkflowCronRunQueueBehavior:      repository.WorkflowPauseScheduledCronRunQueueBehavior(cronBehavior),
			PausedWorkflowScheduledRunQueueBehavior: repository.WorkflowPauseScheduledCronRunQueueBehavior(scheduledBehavior),
		}
	}

	result, err := t.config.V1.Workflows().UpdateWorkflow(ctx, tenant.ID, workflow.ID, updateOpts)

	if err != nil {
		return nil, fmt.Errorf("failed to update workflow")
	}

	return gen.WorkflowUpdate200JSONResponse(
		*transformers.ToWorkflowFromSQLC(result),
	), nil
}
