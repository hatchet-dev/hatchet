package workflowruns

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *V1WorkflowRunsService) V1WorkflowRunTaskEventsList(ctx echo.Context, request gen.V1WorkflowRunTaskEventsListRequestObject) (gen.V1WorkflowRunTaskEventsListResponseObject, error) {
	code := uint64(501)
	return gen.V1WorkflowRunTaskEventsList501JSONResponse(gen.APIErrors{
		Errors: []gen.APIError{{
			Code:        &code,
			Description: "Not implemented",
		}},
	}), nil

}
