package serverlessv1

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *V1ServerlessService) V1ServerlessEndpointGet(ctx echo.Context, request gen.V1ServerlessEndpointGetRequestObject) (gen.V1ServerlessEndpointGetResponseObject, error) {
	endpoint := ctx.Get("v1-serverless-endpoint").(*sqlcv1.V1ServerlessEndpoint)

	return gen.V1ServerlessEndpointGet200JSONResponse(transformers.ToV1ServerlessEndpoint(endpoint)), nil
}
