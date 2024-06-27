package apitokens

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func (a *APITokenService) ApiTokenCreate(ctx echo.Context, request gen.ApiTokenCreateRequestObject) (gen.ApiTokenCreateResponseObject, error) {
	return gen.ApiTokenCreate403JSONResponse(apierrors.NewAPIErrors("this is disabled for the demo")), nil
}
