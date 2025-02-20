package apitokens

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func (a *APITokenService) ApiTokenUpdateRevoke(ctx echo.Context, request gen.ApiTokenUpdateRevokeRequestObject) (gen.ApiTokenUpdateRevokeResponseObject, error) {
	apiToken := ctx.Get("api-token").(*db.APITokenModel)

	if apiToken.Internal {
		return gen.ApiTokenUpdateRevoke403JSONResponse(
			apierrors.NewAPIErrors("Cannot revoke internal API tokens"),
		), nil
	}

	err := a.config.APIRepository.APIToken().RevokeAPIToken(apiToken.ID)

	if err != nil {
		return nil, err
	}

	return gen.ApiTokenUpdateRevoke204Response{}, nil
}
