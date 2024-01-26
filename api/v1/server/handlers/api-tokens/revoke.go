package apitokens

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

func (a *APITokenService) ApiTokenUpdateRevoke(ctx echo.Context, request gen.ApiTokenUpdateRevokeRequestObject) (gen.ApiTokenUpdateRevokeResponseObject, error) {
	apiToken := ctx.Get("api-token").(*db.APITokenModel)

	err := a.config.Repository.APIToken().RevokeAPIToken(apiToken.ID)

	if err != nil {
		return nil, err
	}
	// This is the only time the token is sent over the API
	return gen.ApiTokenUpdateRevoke204Response{}, nil
}
