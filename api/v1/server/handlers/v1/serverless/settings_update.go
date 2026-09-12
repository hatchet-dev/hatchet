package serverlessv1

import (
	"fmt"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	transformers "github.com/hatchet-dev/hatchet/api/v1/server/oas/transformers/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const (
	// Shard count bounds match the OpenAPI schema. Each shard is a lease unit, so the ceiling
	// caps how many lease rows a single tenant can spread over.
	minShardCount int32 = 1
	maxShardCount int32 = 64
)

func (t *V1ServerlessService) V1ServerlessTenantUpdate(ctx echo.Context, request gen.V1ServerlessTenantUpdateRequestObject) (gen.V1ServerlessTenantUpdateResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	shardCount := request.Body.ShardCount

	if shardCount < minShardCount || shardCount > maxShardCount {
		return gen.V1ServerlessTenantUpdate400JSONResponse(apierrors.NewAPIErrors(
			fmt.Sprintf("shardCount must be between %d and %d", minShardCount, maxShardCount),
		)), nil
	}

	settings, err := t.config.V1.Serverless().Tenants().UpdateShardCount(ctx.Request().Context(), tenant.ID, shardCount)

	if err != nil {
		return nil, fmt.Errorf("failed to update serverless tenant settings: %w", err)
	}

	return gen.V1ServerlessTenantUpdate200JSONResponse(transformers.ToV1ServerlessTenantSettings(settings)), nil
}
