//go:build !authdisabled

package authz

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
)

func (a *AuthZ) authPreflight(c echo.Context, r *middleware.RouteInfo) (handled bool, err error) {
	return false, nil
}
