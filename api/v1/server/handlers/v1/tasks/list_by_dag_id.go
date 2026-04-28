package tasks

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"

	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
)

func (t *TasksService) V1DagListTasks(ctx echo.Context, request gen.V1DagListTasksRequestObject) (gen.V1DagListTasksResponseObject, error) {
	user := ctx.Get("user").(*sqlcv1.User)
	tenantId := request.Params.Tenant

	// validate tenant membership
	unauthorized := echo.NewHTTPError(http.StatusUnauthorized, "Not authorized to view this resource")

	tenantMember, err := t.config.V1.Tenant().GetTenantMemberByUserID(ctx.Request().Context(), tenantId, user.ID)

	if err != nil {
		return nil, unauthorized
	}

	if tenantMember == nil {
		return nil, unauthorized
	}

	dagIds := request.Params.DagIds

	tasks, taskIdToDagExternalId, err := t.config.V1.OLAP().ListTasksByDAGId(
		ctx.Request().Context(),
		tenantId,
		dagIds,
		false,
	)

	if err != nil {
		return nil, err
	}

	result := transformers.ToDagChildren(tasks, taskIdToDagExternalId)

	// Search for api errors to see how we handle errors in other cases
	return gen.V1DagListTasks200JSONResponse(
		result,
	), nil
}
