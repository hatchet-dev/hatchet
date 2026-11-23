//go:build authdisabled

package authz

import (
	"github.com/hatchet-dev/hatchet/internal/permissions"
	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

// authdisabled builds skip the email check, as the old preflight did, and cap every request at NOAUTH.
func buildPolicies(_ *server.ServerConfig, perms pkgauthz.RolePermissions) []pkgauthz.Policy {
	return []pkgauthz.Policy{
		pkgauthz.CapToRole(perms, permissions.RoleNoAuth, pkgauthz.RoleInsufficient, func(pkgauthz.Principal) bool { return true }),
	}
}
