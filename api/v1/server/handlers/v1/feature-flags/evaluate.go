package featureflags

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (s *V1FeatureFlagsService) TenantFeatureFlagEvaluate(ctx echo.Context, request gen.TenantFeatureFlagEvaluateRequestObject) (gen.TenantFeatureFlagEvaluateResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

	var flagUser *analytics.FeatureFlagUser

	if user, ok := ctx.Get("user").(*sqlcv1.User); ok && user != nil {
		flagUser = &analytics.FeatureFlagUser{
			ID:    user.ID,
			Email: user.Email,
		}
	}

	isEnabled, err := s.config.Analytics.IsFeatureEnabled(
		ctx.Request().Context(),
		string(request.Params.FeatureFlagId),
		tenant.ID,
		flagUser,
		request.Params.IsEnabledIfNoPosthog,
	)

	if err != nil {
		return nil, err
	}

	return gen.TenantFeatureFlagEvaluate200JSONResponse(gen.FeatureFlagEvaluationResult{
		IsEnabled: isEnabled,
	}), nil
}
