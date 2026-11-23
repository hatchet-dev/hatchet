//go:build authdisabled

package authz

import (
	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

// authdisabled builds skip the email check, as the old preflight did, and cap every request at NOAUTH.
func buildPolicies(_ *server.ServerConfig, perms pkgauthz.RolePermissions) []pkgauthz.Policy {
	return []pkgauthz.Policy{
		pkgauthz.CapToRole(perms, roleNoAuth, pkgauthz.RoleInsufficient, func(pkgauthz.Principal) bool { return true }),
	}
}
