package apitokens

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

func (a *APITokenService) ApiTokenCreate(ctx echo.Context, request gen.ApiTokenCreateRequestObject) (gen.ApiTokenCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*db.TenantModel)

	// validate the request
	if apiErrors, err := a.config.Validator.ValidateAPI(request.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.ApiTokenCreate400JSONResponse(*apiErrors), nil
	}

	token, err := a.config.Auth.JWTManager.GenerateTenantToken(tenant.ID, request.Body.Name)

	if err != nil {
		return nil, err
	}

	// This is the only time the token is sent over the API
	return gen.ApiTokenCreate200JSONResponse{
		Token: token,
	}, nil
}
