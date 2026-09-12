package serverlessv1

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *V1ServerlessService) V1ServerlessEndpointDelete(ctx echo.Context, request gen.V1ServerlessEndpointDeleteRequestObject) (gen.V1ServerlessEndpointDeleteResponseObject, error) {
	endpoint := ctx.Get("v1-serverless-endpoint").(*sqlcv1.V1ServerlessEndpoint)

	deleted, err := t.config.V1.Serverless().Endpoints().Delete(ctx.Request().Context(), endpoint.TenantID, endpoint.ID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return gen.V1ServerlessEndpointDelete404JSONResponse(apierrors.NewAPIErrors("serverless endpoint not found")), nil
		}

		return nil, fmt.Errorf("failed to delete serverless endpoint: %w", err)
	}

	return gen.V1ServerlessEndpointDelete200JSONResponse(transformers.ToV1ServerlessEndpoint(deleted)), nil
}
