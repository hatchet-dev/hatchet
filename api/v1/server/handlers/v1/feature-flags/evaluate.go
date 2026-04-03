package featureflags

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (s *V1FeatureFlagsService) TenantFeatureFlagEvaluate(ctx echo.Context, request gen.TenantFeatureFlagEvaluateRequestObject) (gen.TenantFeatureFlagEvaluateResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	isEnabled, err := s.config.Analytics.IsFeatureEnabled(
		ctx.Request().Context(),
		string(request.Params.FeatureFlagId),
		tenant.ID,
		request.Params.IsEnabledIfNoPosthog,
	)

	if err != nil {
		return nil, err
	}

	return gen.TenantFeatureFlagEvaluate200JSONResponse(gen.FeatureFlagEvaluationResult{
		IsEnabled: isEnabled,
	}), nil
}
