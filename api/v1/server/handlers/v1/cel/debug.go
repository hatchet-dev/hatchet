package celv1

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/internal/cel"
	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/labstack/echo/v4"
)

func (c *V1CELService) V1CelDebug(ctx echo.Context, request gen.V1CelDebugRequestObject) (gen.V1CelDebugResponseObject, error) {
	tenant := ctx.Get("tenant").(*dbsqlc.Tenant)

	additionalMetadata := make(map[string]interface{})
	if request.Body.AdditionalMetadata != nil {
		additionalMetadata = *request.Body.AdditionalMetadata
	}

	filterPayload := make(map[string]interface{})
	if request.Body.FilterPayload != nil {
		filterPayload = *request.Body.FilterPayload
	}

	result, err := c.celParser.EvaluateEventExpression(request.Body.Expression, cel.NewInput(
		cel.WithInput(request.Body.Input),
		cel.WithAdditionalMetadata(additionalMetadata),
		cel.WithPayload(filterPayload),
	),
	)

	if err != nil {
		msg, innerErr := tasktypes.CELEvaluationFailureMessage(
			tenant.ID.String(),
			[]v1.CELEvaluationFailure{
				{
					Source:       sqlcv1.V1CelEvaluationFailureSourceDEBUG,
					ErrorMessage: err.Error(),
				},
			},
		)

		if innerErr != nil {
			return nil, fmt.Errorf("failed to create CEL evaluation failure message: %w", innerErr)
		}

		innerErr = c.config.MessageQueueV1.SendMessage(ctx.Request().Context(), msgqueue.OLAP_QUEUE, msg)

		if innerErr != nil {
			return nil, fmt.Errorf("failed to send CEL evaluation failure message: %w", innerErr)
		}
	}

	var output *bool
	var errorMessage *string

	success := err == nil

	if success {
		output = &result
	} else {
		msg := err.Error()
		errorMessage = &msg
	}

	return gen.V1CelDebug200JSONResponse(transformers.ToV1CELDebugResponse(
		err == nil,
		output,
		errorMessage,
	)), nil
}
