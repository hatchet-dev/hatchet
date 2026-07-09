package authz

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
)

// authorizeAuthDisabled authorizes the request against the NOAUTH role and validates tenant
// membership. It backs both the authdisabled build tag and the runtime IsAuthDisabled config used by
// embedded/library mode (github.com/hatchet-dev/hatchet/hatchetembed). It always reports
// handled=true so the caller short-circuits the normal authorization strategies.
func (a *AuthZ) authorizeAuthDisabled(c echo.Context, r *middleware.RouteInfo) (bool, error) {
	if err := a.authorizeTenantOperations("NOAUTH", r); err != nil {
		return true, err
	}

	return true, a.validateUserTenantPermissions(c, r)
}
