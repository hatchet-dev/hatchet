package celv1

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/internal/cel"
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

	res, err := c.celParser.EvaluateDebugExpression(request.Body.Expression, cel.NewInput(
		cel.WithInput(request.Body.Input),
		cel.WithAdditionalMetadata(additionalMetadata),
		cel.WithPayload(filterPayload),
	))

	var output *bool
	var outputStr *string
	var outputInt *int
	var outputType *gen.V1CELDebugResponseOutputType
	var errorMessage *string

	if err != nil {
		msg := err.Error()
		errorMessage = &msg
	} else {
		switch {
		case res.Bool != nil:
			t := gen.Bool
			output, outputType = res.Bool, &t
		case res.String != nil:
			t := gen.String
			outputStr, outputType = res.String, &t
		case res.Int != nil:
			t := gen.Int
			outputInt, outputType = res.Int, &t
		}
	}

	return gen.V1CelDebug200JSONResponse(transformers.ToV1CELDebugResponse(
		err == nil,
		output,
		outputStr,
		outputInt,
		outputType,
		errorMessage,
	)), nil
}
