//go:build !authdisabled

package authz

import (
	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

// Extension hooks run last so the email check keeps its own error message.
func buildPolicies(config *server.ServerConfig, perms pkgauthz.RolePermissions) []pkgauthz.Policy {
	unverifiedUser := func(p pkgauthz.Principal) bool {
		u, ok := p.(pkgauthz.UserPrincipal)
		return ok && !u.EmailVerified
	}

	return append([]pkgauthz.Policy{
		pkgauthz.CapToRole(perms, roleUnverifiedEmail, pkgauthz.EmailUnverified, unverifiedUser),
	}, extensionPolicies(config)...)
}
