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
	tenantId := tenant.ID
	workflow := ctx.Get("workflow").(*sqlcv1.GetWorkflowByIdRow)

	if request.Body.IsPaused == nil {
		return gen.WorkflowUpdate400JSONResponse(gen.APIErrors{
			Errors: []gen.APIError{{Description: "isPaused is required"}},
		}), nil
	}

	var updated *sqlcv1.Workflow
	var err error

	if *request.Body.IsPaused {
		updated, err = t.config.V1.Workflows().PauseWorkflow(ctx.Request().Context(), tenantId, workflow.Workflow.ID)
		if err != nil {
			return nil, err
		}

		// cancel any running tasks for the workflow
		_, err = t.proxyCancel.Do(ctx.Request().Context(), tenant, &contracts.CancelTasksRequest{
    		Filter: &contracts.TasksFilter{
        		WorkflowIds: []string{workflow.Workflow.ID.String()},
        		Statuses:    []string{"RUNNING"},
    		},
		})

		if err != nil {
			return nil, err
		}
	} else {
		updated, err = t.config.V1.Workflows().UnpauseWorkflow(ctx.Request().Context(), tenantId, workflow.Workflow.ID)
		if err != nil {
			return nil, err
		}
	}

	resp := transformers.ToWorkflowFromSQLC(updated)

	return gen.WorkflowUpdate200JSONResponse(*resp), nil
}