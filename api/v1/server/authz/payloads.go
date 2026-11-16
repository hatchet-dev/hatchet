package authz

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// CanViewPayloads reports whether the current request may see task and event payloads.
// API tokens (no tenant-member in context) are unrestricted. OWNER and ADMIN always can.
func CanViewPayloads(c echo.Context) bool {
	member, ok := c.Get("tenant-member").(*sqlcv1.PopulateTenantMembersRow)
	// NOTE: this is the api token case, not the user case
	if !ok || member == nil {
		return true
	}

	if member.Role == sqlcv1.TenantMemberRoleOWNER || member.Role == sqlcv1.TenantMemberRoleADMIN {
		return true
	}

	return member.CanViewPayloads
}
