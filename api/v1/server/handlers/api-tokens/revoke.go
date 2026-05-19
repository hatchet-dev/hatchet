package apitokens

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/constants"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (a *APITokenService) ApiTokenUpdateRevoke(ctx echo.Context, request gen.ApiTokenUpdateRevokeRequestObject) (gen.ApiTokenUpdateRevokeResponseObject, error) {
	apiToken := ctx.Get("api-token").(*sqlcv1.APIToken)

	if apiToken.Internal {
		return gen.ApiTokenUpdateRevoke403JSONResponse(
			apierrors.NewAPIErrors("Cannot revoke internal API tokens"),
		), nil
	}

	err := a.config.V1.APIToken().RevokeAPIToken(ctx.Request().Context(), apiToken.ID)

	if err != nil {
		return nil, err
	}

	ctx.Set(constants.ResourceIdKey.String(), apiToken.ID.String())
	ctx.Set(constants.ResourceTypeKey.String(), constants.ResourceTypeApiToken.String())

	a.config.Analytics.Enqueue(
		ctx.Request().Context(),
		analytics.Token, analytics.Revoke,
		apiToken.ID.String(),
		nil,
	)
	return gen.ApiTokenUpdateRevoke204Response{}, nil
}
