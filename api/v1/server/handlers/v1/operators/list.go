package operatorsv1

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const (
	defaultOperatorListLimit int64 = 50
)

func (t *V1OperatorsService) V1HttpOperatorList(ctx echo.Context, request gen.V1HttpOperatorListRequestObject) (gen.V1HttpOperatorListResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	limit := defaultOperatorListLimit
	offset := int64(0)

	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	if request.Params.Offset != nil {
		offset = *request.Params.Offset
	}

	kind := sqlcv1.V1OperatorKindHTTPAPI

	operators, total, err := t.config.V1.Operators().ListOperators(
		ctx.Request().Context(),
		tenant.ID,
		v1.ListOperatorsOpts{
			Kind:   &kind,
			Limit:  limit,
			Offset: offset,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to list operators: %w", err)
	}

	transformed, err := transformers.ToV1HTTPOperatorList(operators, total, limit, offset)

	if err != nil {
		return nil, fmt.Errorf("failed to transform operators: %w", err)
	}

	return gen.V1HttpOperatorList200JSONResponse(transformed), nil
}
