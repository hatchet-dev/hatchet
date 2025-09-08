package tasks

import (
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"

	"github.com/labstack/echo/v4"

	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
)

func (t *TasksService) V1TaskListStatusMetrics(ctx echo.Context, request gen.V1TaskListStatusMetricsRequestObject) (gen.V1TaskListStatusMetricsResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)
	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	var workflowIds []uuid.UUID

	if request.Params.WorkflowIds != nil {
		workflowIds = *request.Params.WorkflowIds
	}

	var parentTaskExternalId *pgtype.UUID

	if request.Params.ParentTaskExternalId != nil {
		uuidPtr := *request.Params.ParentTaskExternalId
		uuidVal := sqlchelpers.UUIDFromStr(uuidPtr.String())
		parentTaskExternalId = &uuidVal
	}

	var triggeringEventExternalId *pgtype.UUID
	if request.Params.TriggeringEventExternalId != nil {
		uuidVal := sqlchelpers.UUIDFromStr(request.Params.TriggeringEventExternalId.String())
		triggeringEventExternalId = &uuidVal
	}

	additionalMetadataFilters := make(map[string]interface{})

	if request.Params.AdditionalMetadata != nil {
		for _, v := range *request.Params.AdditionalMetadata {
			kv_pairs := strings.Split(v, ":")
			if len(kv_pairs) == 2 {
				additionalMetadataFilters[kv_pairs[0]] = kv_pairs[1]
			}
		}
	}

	metrics, err := t.config.V1.OLAP().ReadTaskRunMetrics(ctx.Request().Context(), tenantId, v1.ReadTaskRunMetricsOpts{
		CreatedAfter:              request.Params.Since,
		CreatedBefore:             request.Params.Until,
		WorkflowIds:               workflowIds,
		ParentTaskExternalID:      parentTaskExternalId,
		TriggeringEventExternalId: triggeringEventExternalId,
		AdditionalMetadata:        additionalMetadataFilters,
	})

	if err != nil {
		return nil, err
	}

	result := transformers.ToTaskRunMetrics(&metrics)

	// Search for api errors to see how we handle errors in other cases
	return gen.V1TaskListStatusMetrics200JSONResponse(
		result,
	), nil
}
