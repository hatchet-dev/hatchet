package celv1

import (
	"strconv"

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

	var output *string
	var outputType *gen.V1CELDebugResponseOutputType
	var errorMessage *string

	if err != nil {
		msg := err.Error()
		errorMessage = &msg
	} else {
		switch {
		case res.Bool != nil:
			s := strconv.FormatBool(*res.Bool)
			t := gen.Bool
			output, outputType = &s, &t
		case res.String != nil:
			t := gen.String
			output, outputType = res.String, &t
		case res.Int != nil:
			s := strconv.Itoa(*res.Int)
			t := gen.Int
			output, outputType = &s, &t
		}
	}

	return gen.V1CelDebug200JSONResponse(transformers.ToV1CELDebugResponse(
		err == nil,
		output,
		outputType,
		errorMessage,
	)), nil
}
