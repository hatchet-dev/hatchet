package apitokens

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/enumerables"
	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
)

func (a *APITokenService) ApiTokenCapabilityList(ctx echo.Context, request gen.ApiTokenCapabilityListRequestObject) (gen.ApiTokenCapabilityListResponseObject, error) {
	_, tokens, err := a.permissions()
	if err != nil {
		return nil, err
	}

	rows := enumerables.Map(tokens.Capabilities(), func(c pkgauthz.CapabilityInfo) gen.TokenCapability {
		row := gen.TokenCapability{
			Name:        c.Name,
			Description: c.Description,
			Bundle:      c.Bundle,
			Includes:    c.Includes,
		}

		if c.Label != "" {
			row.Label = &c.Label
		}

		if c.Group != "" {
			row.Group = &c.Group
		}

		return row
	})

	return gen.ApiTokenCapabilityList200JSONResponse(gen.TokenCapabilityList{Rows: rows}), nil
}
