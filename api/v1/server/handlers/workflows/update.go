package workflows

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *WorkflowService) WorkflowUpdate(echoCtx echo.Context, request gen.WorkflowUpdateRequestObject) (gen.WorkflowUpdateResponseObject, error) {
	tenant := echoCtx.Get("tenant").(*sqlcv1.Tenant)
	workflow := echoCtx.Get("workflow").(*sqlcv1.GetWorkflowByIdRow)
	ctx := echoCtx.Request().Context()

	hasPauseChanges := request.Body.Pause != nil

	var updateOpts repository.UpdateWorkflowOpts

	if hasPauseChanges {
		cronBehavior := string(request.Body.Pause.PausedWorkflowCronRunQueueBehavior)
		scheduledBehavior := string(request.Body.Pause.PausedWorkflowScheduledRunQueueBehavior)

		updateOpts.PauseOpts = &repository.WorkflowPauseOpts{
			IsPaused:                                request.Body.Pause.IsPaused,
			PausedWorkflowCronRunQueueBehavior:      repository.WorkflowPauseScheduledCronRunQueueBehavior(cronBehavior),
			PausedWorkflowScheduledRunQueueBehavior: repository.WorkflowPauseScheduledCronRunQueueBehavior(scheduledBehavior),
			PausedWorkflowQueueTTL:                  request.Body.Pause.PausedWorkflowQueueTTL,
		}
	}

	result, err := t.config.V1.Workflows().UpdateWorkflow(ctx, tenant.ID, workflow.Workflow.ID, updateOpts)

	if err != nil {
		t.config.Logger.Err(err).Msg("failed to update workflow")
		return nil, fmt.Errorf("failed to update workflow")
	}

	if hasPauseChanges && result.IsPaused.Valid {
		msg, err := tasktypes.NewPauseWorkflowMessage(tenant.ID, workflow.Workflow.ID, result.IsPaused.Bool)

		if err != nil {
			t.config.Logger.Err(err).Msg("failed to create pause workflow message")
			return nil, fmt.Errorf("failed to update workflow")
		}

		if err = t.config.MessageQueueV1.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, msg); err != nil {
			t.config.Logger.Err(err).Msg("could not send workflow update message")
			return nil, fmt.Errorf("failed to update workflow")
		}
	}

	return gen.WorkflowUpdate200JSONResponse(
		*transformers.ToWorkflowFromSQLC(result),
	), nil
}
