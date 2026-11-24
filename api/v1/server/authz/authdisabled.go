//go:build authdisabled

package authz

import (
	"github.com/hatchet-dev/hatchet/internal/permissions"
	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

func buildPolicies(_ *server.ServerConfig, perms pkgauthz.RolePermissions) []pkgauthz.Policy {
	return []pkgauthz.Policy{
		// every request is denied anything NOAUTH does not also permit, on top of the principal's own role
		pkgauthz.CapToRole(perms, permissions.RoleNoAuth, pkgauthz.RoleInsufficient, func(pkgauthz.Principal) bool { return true }),
	}
}
