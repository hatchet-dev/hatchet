package workflowruns

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (t *V2WorkflowRunsService) V2WorkflowRunTaskEventsList(ctx echo.Context, request gen.V2WorkflowRunTaskEventsListRequestObject) (gen.V2WorkflowRunTaskEventsListResponseObject, error) {
	code := uint64(501)
	return gen.V2WorkflowRunTaskEventsList501JSONResponse(gen.APIErrors{
		Errors: []gen.APIError{{
			Code:        &code,
			Description: "Not implemented",
		}},
	}), nil

}
