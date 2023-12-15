package workflows

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/labstack/echo/v4"
)

func (t *WorkflowService) WorkflowGet(ctx echo.Context, request gen.WorkflowGetRequestObject) (gen.WorkflowGetResponseObject, error) {
	workflow := ctx.Get("workflow").(*db.WorkflowModel)

	resp, err := transformers.ToWorkflow(workflow, nil)

	if err != nil {
		return nil, err
	}

	return gen.WorkflowGet200JSONResponse(*resp), nil
}
