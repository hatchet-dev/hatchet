package apitokens

import (
	"errors"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	authtoken "github.com/hatchet-dev/hatchet/pkg/auth/token"
	"github.com/hatchet-dev/hatchet/pkg/constants"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (a *APITokenService) ApiTokenCreate(ctx echo.Context, request gen.ApiTokenCreateRequestObject) (gen.ApiTokenCreateResponseObject, error) {
	tenant := ctx.Get("tenant").(*sqlcv1.Tenant)

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

	readOnly := request.Body.ReadOnly != nil && *request.Body.ReadOnly

	// readOnly keeps working for existing clients; combined with capabilities it would restrict gRPC
	// while REST followed the list, so the two are exclusive.
	if readOnly && request.Body.Capabilities != nil {
		return gen.ApiTokenCreate400JSONResponse(apierrors.NewAPIErrors("readOnly cannot be combined with capabilities; pass [\"read-only\"] instead")), nil
	}

	// Omitted capabilities fall through to the engine's default of "all".
	var opts []authtoken.GenerateOption

	if request.Body.Capabilities != nil {
		// GenerateTenantToken rejects the same lists, but only as an internal error; checking first makes it a 400.
		if apiErr, err := validateCapabilities(*request.Body.Capabilities); err != nil {
			return nil, err
		} else if apiErr != nil {
			return gen.ApiTokenCreate400JSONResponse(*apiErr), nil
		}

		opts = append(opts, authtoken.WithCapabilities(*request.Body.Capabilities))
	}

	token, err := a.config.Auth.JWTManager.GenerateTenantToken(ctx.Request().Context(), tenantId, request.Body.Name, false, readOnly, expiresAt, opts...)

	if err != nil {
		return nil, err
	}

	ctx.Set(constants.ResourceIdKey.String(), token.TokenId.String())
	ctx.Set(constants.ResourceTypeKey.String(), constants.ResourceTypeApiToken.String())

	a.config.Analytics.Enqueue(
		ctx.Request().Context(),
		analytics.Token, analytics.Create,
		token.TokenId.String(),
		map[string]interface{}{
			"name":         request.Body.Name,
			"expires_at":   expiresAt,
			"capabilities": request.Body.Capabilities,
		},
	)

	// This is the only time the token is sent over the API
	return gen.ApiTokenCreate200JSONResponse{
		Token: token.Token,
	}, nil
}

func validateCapabilities(capabilities []string) (*gen.APIErrors, error) {
	err := authtoken.ValidateCapabilities(capabilities)
	if errors.Is(err, authtoken.ErrInvalidCapabilities) {
		e := apierrors.NewAPIErrors(strings.TrimPrefix(err.Error(), authtoken.ErrInvalidCapabilities.Error()+": "))
		return &e, nil
	}

	return nil, err
}
