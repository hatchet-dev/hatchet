//go:build authdisabled

package authz

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
)

func (a *AuthZ) authPreflight(c echo.Context, r *middleware.RouteInfo) (handled bool, err error) {
	if err := a.authorizeTenantOperations("NOAUTH", r); err != nil {
		return true, err
	}

	return true, a.validateUserTenantPermissions(c, r)
}
