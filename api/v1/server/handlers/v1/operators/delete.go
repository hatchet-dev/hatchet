package operatorsv1

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *V1OperatorsService) V1HttpOperatorDelete(ctx echo.Context, request gen.V1HttpOperatorDeleteRequestObject) (gen.V1HttpOperatorDeleteResponseObject, error) {
	operator := ctx.Get("v1-http-operator").(*sqlcv1.V1Operator)

	deleted, err := t.config.V1.Operators().DeleteOperator(
		ctx.Request().Context(),
		operator.TenantID,
		operator.ID,
	)

	if err != nil {
		return gen.V1HttpOperatorDelete400JSONResponse(apierrors.NewAPIErrors("failed to delete operator")), nil
	}

	transformed, err := transformers.ToV1HTTPOperator(deleted)

	if err != nil {
		return nil, fmt.Errorf("failed to transform operator: %w", err)
	}

	return gen.V1HttpOperatorDelete200JSONResponse(transformed), nil
}
