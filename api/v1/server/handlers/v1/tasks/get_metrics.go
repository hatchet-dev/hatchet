package tasks

import (
	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"

	"github.com/labstack/echo/v4"

	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
)

func (t *TasksService) V1TaskListStatusMetrics(ctx echo.Context, request gen.V1TaskListStatusMetricsRequestObject) (gen.V1TaskListStatusMetricsResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	tenantId := tenant.ID

	var workflowIds []uuid.UUID

	if request.Params.WorkflowIds != nil {
		workflowIds = *request.Params.WorkflowIds
	}

	var parentTaskExternalId *uuid.UUID

	if request.Params.ParentTaskExternalId != nil {
		parentTaskExternalId = request.Params.ParentTaskExternalId
	}

	var triggeringEventExternalId *uuid.UUID
	if request.Params.TriggeringEventExternalId != nil {
		triggeringEventExternalId = request.Params.TriggeringEventExternalId
	}

	additionalMetadataFilters := make(map[string]interface{})

	if filters, _, _ := v1.ParseAdditionalMetadataFilters(request.Params.AdditionalMetadata); filters != nil {
		additionalMetadataFilters = filters
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

	result := transformers.StatusToTaskRunMetrics(&metrics)

	// Search for api errors to see how we handle errors in other cases
	return gen.V1TaskListStatusMetrics200JSONResponse(
		result,
	), nil
}
