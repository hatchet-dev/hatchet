package serverlessv1

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/authz"
	v1handlers "github.com/hatchet-dev/hatchet/api/v1/server/handlers/v1"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// V1ServerlessEndpointEventList returns the endpoint's most recent events within the tenant's
// retention window, newest first. The repository applies the default and maximum page size.
// Response excerpts are the endpoint's own output and may hold task data, so members who
// cannot view payloads do not get them.
func (t *V1ServerlessService) V1ServerlessEndpointEventList(ctx echo.Context, request gen.V1ServerlessEndpointEventListRequestObject) (gen.V1ServerlessEndpointEventListResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	endpoint := ctx.Get("v1-serverless-endpoint").(*sqlcv1.V1ServerlessEndpoint)

	var limit int64

	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	events, err := t.config.V1.Serverless().Events().List(ctx.Request().Context(), endpoint.TenantID, endpoint.ID, v1handlers.RetentionBoundary(tenant.DataRetentionPeriod), request.Params.Since, limit)

	if err != nil {
		return nil, fmt.Errorf("failed to list serverless endpoint events: %w", err)
	}

	return gen.V1ServerlessEndpointEventList200JSONResponse(transformers.ToV1ServerlessEndpointEventList(events, transformers.WithPayloads(authz.CanViewPayloads(ctx)))), nil
}
