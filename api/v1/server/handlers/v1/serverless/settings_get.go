package serverlessv1

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *V1ServerlessService) V1ServerlessTenantGet(ctx echo.Context, request gen.V1ServerlessTenantGetRequestObject) (gen.V1ServerlessTenantGetResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	// The settings row is materialized lazily with the database defaults, so a tenant that has
	// never touched serverless reads the same defaults its first endpoint will be created with.
	settings, err := t.config.V1.Serverless().Tenants().Upsert(ctx.Request().Context(), tenant.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to get serverless tenant settings: %w", err)
	}

	return gen.V1ServerlessTenantGet200JSONResponse(transformers.ToV1ServerlessTenantSettings(settings)), nil
}
