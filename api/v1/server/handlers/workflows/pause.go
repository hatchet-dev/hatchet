package workflows

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func extractIsPaused(request gen.PauseWorkflowRequest) (bool, error) {
	j, err := request.MarshalJSON()

	if err != nil {
		return false, fmt.Errorf("failed to marshal request: %w", err)
	}

	parsedBody := make(map[string]interface{})

	if err := json.Unmarshal(j, &parsedBody); err != nil {
		return false, fmt.Errorf("failed to unmarshal request body: %w", err)
	}

	unparsedDiscriminator, ok := parsedBody["isPaused"]

	if !ok {
		return false, fmt.Errorf("isPaused field is missing in the request body")
	}

	isPaused, ok := unparsedDiscriminator.(bool)

	if !ok {
		return false, fmt.Errorf("invalid isPaused value: %v", unparsedDiscriminator)
	}

	return isPaused, nil
}

func (t *WorkflowService) applyPauseWorkflowRequest(ctx context.Context, tenantId, workflowId uuid.UUID, request gen.PauseWorkflowRequest) (*sqlcv1.Workflow, error) {
	isPaused, err := extractIsPaused(request)

	if err != nil {
		return nil, fmt.Errorf("failed to parse request body: %w", err)
	}

	var result *sqlcv1.Workflow

	if isPaused {
		pauseRequest, err := request.AsPauseWorkflowRequestPause()

		if err != nil {
			return nil, fmt.Errorf("failed to parse pause request: %w", err)
		}

		result, err = t.config.V1.Workflows().PauseWorkflow(ctx, workflowId, repository.PauseWorkflowOpts{
			CronRunQueueBehavior:      repository.WorkflowPauseScheduledCronRunQueueBehavior(pauseRequest.PausedWorkflowCronRunQueueBehavior),
			ScheduledRunQueueBehavior: repository.WorkflowPauseScheduledCronRunQueueBehavior(pauseRequest.PausedWorkflowScheduledRunQueueBehavior),
			QueueTTL:                  pauseRequest.PausedWorkflowQueueTTL,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to pause workflow: %w", err)
		}
	} else {
		result, err = t.config.V1.Workflows().UnpauseWorkflow(ctx, workflowId)

		if err != nil {
			return nil, fmt.Errorf("failed to unpause workflow: %w", err)
		}
	}

	msg, err := tasktypes.NewTogglePauseWorkflowMessage(tenantId, workflowId, isPaused)

	if err != nil {
		return nil, fmt.Errorf("failed to create pause workflow message: %w", err)
	}

	if err = t.config.MessageQueueV1.SendMessage(ctx, msgqueue.TASK_PROCESSING_QUEUE, msg); err != nil {
		return nil, fmt.Errorf("could not send workflow update message: %w", err)
	}

	return result, nil
}

func (t *WorkflowService) WorkflowUpdatePause(echoCtx echo.Context, request gen.WorkflowUpdatePauseRequestObject) (gen.WorkflowUpdatePauseResponseObject, error) {
	tenant := echoCtx.Get("tenant").(*sqlcv1.Tenant)
	workflow := echoCtx.Get("workflow").(*sqlcv1.GetWorkflowByIdRow)
	ctx := echoCtx.Request().Context()

	result, err := t.applyPauseWorkflowRequest(ctx, tenant.ID, workflow.Workflow.ID, *request.Body)

	if err != nil {
		t.config.Logger.Err(err).Msg("failed to update workflow pause state")
		return nil, fmt.Errorf("failed to update workflow")
	}

	return gen.WorkflowUpdatePause200JSONResponse(
		*transformers.ToWorkflowFromSQLC(result),
	), nil
}
