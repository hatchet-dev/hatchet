package serverlessv1

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const defaultEndpointListLimit int64 = 50

func (t *V1ServerlessService) V1ServerlessEndpointList(ctx echo.Context, request gen.V1ServerlessEndpointListRequestObject) (gen.V1ServerlessEndpointListResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	limit := defaultEndpointListLimit
	offset := int64(0)

	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	if request.Params.Offset != nil {
		offset = *request.Params.Offset
	}

	endpoints, total, err := t.config.V1.Serverless().Endpoints().List(
		ctx.Request().Context(),
		tenant.ID,
		v1.ListServerlessEndpointsOpts{
			Limit:  limit,
			Offset: offset,
		},
	)

	if err != nil {
		if msg, ok := badRequestMessage(err); ok {
			return gen.V1ServerlessEndpointList400JSONResponse(apierrors.NewAPIErrors(msg)), nil
		}

		return nil, fmt.Errorf("failed to list serverless endpoints: %w", err)
	}

	return gen.V1ServerlessEndpointList200JSONResponse(transformers.ToV1ServerlessEndpointList(endpoints, total, limit, offset)), nil
}
