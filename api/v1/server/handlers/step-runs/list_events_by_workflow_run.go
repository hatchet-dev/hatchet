package stepruns

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (t *StepRunService) WorkflowRunListStepRunEvents(ctx echo.Context, request gen.WorkflowRunListStepRunEventsRequestObject) (gen.WorkflowRunListStepRunEventsResponseObject, error) {
	tenant, err := populator.FromContext(ctx).GetTenant()
	if err != nil {
		return nil, err
	}
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	reqCtx, cancel := context.WithTimeout(ctx.Request().Context(), 5*time.Second)
	defer cancel()

	lastId := request.Params.LastId

	listRes, err := t.config.APIRepository.StepRun().ListStepRunEventsByWorkflowRunId(
		reqCtx,
		tenantId,
		request.WorkflowRun.String(),
		lastId,
	)

	if err != nil {
		return nil, err
	}

	rows := make([]gen.StepRunEvent, len(listRes.Rows))

	for i := range listRes.Rows {
		e := listRes.Rows[i]

		eventData := transformers.ToStepRunEvent(e)

		rows[i] = *eventData
	}

	return gen.WorkflowRunListStepRunEvents200JSONResponse(
		gen.StepRunEventList{
			Rows: &rows,
		},
	), nil
}
