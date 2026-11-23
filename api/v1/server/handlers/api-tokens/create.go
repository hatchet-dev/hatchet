package apitokens

import (
	"fmt"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/tokenauthz"
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

	var opts []authtoken.GenerateOption

	if request.Body.Capabilities != nil {
		if apiErr, err := validateCapabilities(*request.Body.Capabilities, readOnly); err != nil {
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

// validateCapabilities rejects names tokens.yaml does not define, since an unknown capability would silently grant nothing.
func validateCapabilities(capabilities []string, readOnly bool) (*gen.APIErrors, error) {
	if readOnly {
		e := apierrors.NewAPIErrors("read-only tokens already hold the read-only capability; omit readOnly to choose capabilities")
		return &e, nil
	}

	if len(capabilities) == 0 {
		e := apierrors.NewAPIErrors("capabilities cannot be empty; omit the field to grant all")
		return &e, nil
	}

	for _, c := range capabilities {
		ok, err := tokenauthz.Defines(c)
		if err != nil {
			return nil, err
		}

		if !ok {
			e := apierrors.NewAPIErrors(fmt.Sprintf("unknown capability %q", c))
			return &e, nil
		}
	}

	return nil, nil
}
