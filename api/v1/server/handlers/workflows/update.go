package workflows

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *WorkflowService) WorkflowUpdate(ctx echo.Context, request gen.WorkflowUpdateRequestObject) (gen.WorkflowUpdateResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	workflow := ctx.Get("workflow").(*sqlcv1.GetWorkflowByIdRow)

	if request.Body.IsPaused == nil &&
		request.Body.QueueCronOnPause == nil &&
		request.Body.QueueScheduledOnPause == nil {
		return gen.WorkflowUpdate400JSONResponse(gen.APIErrors{
			Errors: []gen.APIError{{Description: "at least one of isPaused, queueCronOnPause, or queueScheduledOnPause is required"}},
		}), nil
	}

	if request.Body.QueueCronOnPause != nil || request.Body.QueueScheduledOnPause != nil {
		updated, err := t.config.V1.Workflows().UpdateWorkflowPauseSettings(
			ctx.Request().Context(),
			workflow.Workflow.ID,
			request.Body.QueueCronOnPause,
			request.Body.QueueScheduledOnPause,
		)
		if err != nil {
			return nil, err
		}
		workflow.Workflow = *updated
	}

	if request.Body.IsPaused != nil {
		resp, err := t.proxyPause.Do(ctx.Request().Context(), tenant, &contracts.UpdateWorkflowPauseRequest{
			WorkflowId: workflow.Workflow.ID.String(),
			IsPaused:   *request.Body.IsPaused,
		})
		if err != nil {
			return nil, err
		}
		workflow.Workflow.IsPaused.Bool = resp.IsPaused
		workflow.Workflow.IsPaused.Valid = true
		workflow.Workflow.UpdatedAt.Time = resp.UpdatedAt.AsTime()
		workflow.Workflow.UpdatedAt.Valid = true
	}

	apiResp := transformers.ToWorkflowFromSQLC(&workflow.Workflow)
	return gen.WorkflowUpdate200JSONResponse(*apiResp), nil
}
