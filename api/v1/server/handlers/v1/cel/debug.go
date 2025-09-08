package celv1

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/internal/cel"
	"github.com/labstack/echo/v4"
)

func (c *V1CELService) V1CelDebug(ctx echo.Context, request gen.V1CelDebugRequestObject) (gen.V1CelDebugResponseObject, error) {
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
	))

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
