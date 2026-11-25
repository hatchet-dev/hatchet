package apitokens

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (a *APITokenService) ApiTokenCapabilityList(ctx echo.Context, request gen.ApiTokenCapabilityListRequestObject) (gen.ApiTokenCapabilityListResponseObject, error) {
	_, tokens, err := a.permissions()
	if err != nil {
		return nil, err
	}

	caps := tokens.Capabilities()
	rows := make([]gen.TokenCapability, 0, len(caps))

	for _, c := range caps {
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

		rows = append(rows, row)
	}

	return gen.ApiTokenCapabilityList200JSONResponse(gen.TokenCapabilityList{Rows: rows}), nil
}
