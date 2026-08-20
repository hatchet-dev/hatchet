package operatorsv1

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *V1OperatorsService) V1HttpOperatorGet(ctx echo.Context, request gen.V1HttpOperatorGetRequestObject) (gen.V1HttpOperatorGetResponseObject, error) {
	operator := ctx.Get("v1-http-operator").(*sqlcv1.V1Operator)

	transformed, err := transformers.ToV1HTTPOperator(operator)

	if err != nil {
		return nil, fmt.Errorf("failed to transform operator: %w", err)
	}

	return gen.V1HttpOperatorGet200JSONResponse(transformed), nil
}
