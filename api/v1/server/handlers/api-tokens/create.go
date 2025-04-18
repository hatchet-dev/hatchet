package apitokens

import (
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/populator"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func (a *APITokenService) ApiTokenCreate(ctx echo.Context, request gen.ApiTokenCreateRequestObject) (gen.ApiTokenCreateResponseObject, error) {
	tenant, err := populator.FromContext(ctx).GetTenant()
	if err != nil {
		return nil, err
	}

	tenantId := sqlchelpers.UUIDToStr(tenant.ID)

	// validate the request
	if apiErrors, err := a.config.Validator.ValidateAPI(request.Body); err != nil {
		return nil, err
	} else if apiErrors != nil {
		return gen.ApiTokenCreate400JSONResponse(*apiErrors), nil
	}

	var expiresAt *time.Time

	if request.Body.ExpiresIn != nil {
		expiresIn, err := time.ParseDuration(*request.Body.ExpiresIn)

		if err != nil {
			return gen.ApiTokenCreate400JSONResponse(apierrors.NewAPIErrors("invalid expiration duration")), nil
		}

		e := time.Now().UTC().Add(expiresIn)

		expiresAt = &e
	}

	token, err := a.config.Auth.JWTManager.GenerateTenantToken(ctx.Request().Context(), tenantId, request.Body.Name, false, expiresAt)

	if err != nil {
		return nil, err
	}

	// This is the only time the token is sent over the API
	return gen.ApiTokenCreate200JSONResponse{
		Token: token.Token,
	}, nil
}
