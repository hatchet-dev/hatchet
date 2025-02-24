package workflowruns

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *V1WorkflowRunsService) V1WorkflowRunGet(ctx echo.Context, request gen.V1WorkflowRunGetRequestObject) (gen.V1WorkflowRunGetResponseObject, error) {
	code := uint64(501)
	return gen.V1WorkflowRunGet501JSONResponse(gen.APIErrors{
		Errors: []gen.APIError{{
			Code:        &code,
			Description: "Not implemented",
		}},
	}), nil

}
