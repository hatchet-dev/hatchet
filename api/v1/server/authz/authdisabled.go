//go:build authdisabled

package authz

import (
	"context"

	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

// authdisabled builds skip email and token checks, as the old preflight did, and cap requests at NOAUTH.
func buildPolicies(_ *server.ServerConfig, perms pkgauthz.RolePermissions) []pkgauthz.Policy {
	return []pkgauthz.Policy{noAuthRolePolicy{perms: perms}}
}

// noAuthRolePolicy caps authdisabled builds at the NOAUTH role on top of the member's own role.
type noAuthRolePolicy struct {
	perms pkgauthz.RolePermissions
}

func (p noAuthRolePolicy) Evaluate(_ context.Context, req pkgauthz.Request) (pkgauthz.Decision, error) {
	if p.perms.HasPermission(roleNoAuth, req.Action) {
		return pkgauthz.Allow(), nil
	}

	return pkgauthz.Deny(pkgauthz.RoleInsufficient), nil
}
