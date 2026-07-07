//go:build authdisabled

package authz

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/pkg/auth/rbac"
)

func (a *AuthZ) authPreflight(c echo.Context, r *middleware.RouteInfo) (handled bool, err error) {
	if rbac.OperationIn(r.OperationID, authDisabledDeniedOperations) {
		return true, echo.NewHTTPError(http.StatusForbidden, "This operation is disabled while authentication is disabled")
	}

	return true, a.validateUserTenantPermissions(c, r)
}

var authDisabledDeniedOperations = []string{
	"TenantInviteAccept",
	"TenantInviteReject",
	"TenantInviteUpdate",
	"TenantInviteDelete",
	"TenantMemberUpdate",
	"TenantMemberDelete",
	"ApiTokenCreate",
	"ApiTokenList",
	"ApiTokenUpdateRevoke",
}
