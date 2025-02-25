package tasks

import (
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"

	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
)

func (t *TasksService) V1DagListTasks(ctx echo.Context, request gen.V1DagListTasksRequestObject) (gen.V1DagListTasksResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)
	dagIds := request.Params.DagIds

	pguuids := make([]pgtype.UUID, 0)
	for _, dagId := range dagIds {
		pguuids = append(pguuids, sqlchelpers.UUIDFromStr(dagId.String()))
	}

	tasks, taskIdToDagExternalId, err := t.config.V1.OLAP().ListTasksByDAGId(
		ctx.Request().Context(),
		tenantId,
		pguuids,
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
