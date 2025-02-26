package workflowruns

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *V1WorkflowRunsService) V1WorkflowRunList(ctx echo.Context, request gen.V1WorkflowRunListRequestObject) (gen.V1WorkflowRunListResponseObject, error) {
	code := uint64(501)
	return gen.V1WorkflowRunList501JSONResponse(gen.APIErrors{
		Errors: []gen.APIError{{
			Code:        &code,
			Description: "Not implemented",
		}},
	}), nil

}
