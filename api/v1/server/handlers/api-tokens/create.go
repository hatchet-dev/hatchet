package apitokens

import (
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/constants"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (a *APITokenService) ApiTokenCreate(ctx echo.Context, request gen.ApiTokenCreateRequestObject) (gen.ApiTokenCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)
	user := ctx.Get("user").(*sqlcv1.User)

	tenantId := tenant.ID

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

	ctx.Set(constants.ResourceIdKey.String(), token.TokenId)
	ctx.Set(constants.ResourceTypeKey.String(), constants.ResourceTypeApiToken.String())

	tenantIdStr := tenantId.String()
	a.config.Analytics.Enqueue(
		"api-token:create",
		user.ID.String(),
		&tenantIdStr,
		nil,
		map[string]interface{}{
			"name":       request.Body.Name,
			"expires_at": expiresAt,
			"token_id":   token.TokenId,
		},
	)

	// This is the only time the token is sent over the API
	return gen.ApiTokenCreate200JSONResponse{
		Token: token.Token,
	}, nil
}
