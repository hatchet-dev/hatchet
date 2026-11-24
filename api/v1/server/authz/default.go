//go:build !authdisabled

package authz

import (
	"github.com/hatchet-dev/hatchet/internal/permissions"
	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

func buildPolicies(config *server.ServerConfig, perms pkgauthz.RolePermissions) []pkgauthz.Policy {
	unverifiedUser := func(p pkgauthz.Principal) bool {
		u, ok := p.(pkgauthz.UserPrincipal)
		return ok && !u.EmailVerified
	}

	// Custom authenticator hooks run last so the email check keeps its own error message.
	return append([]pkgauthz.Policy{
		// unverified users are denied anything UNVERIFIED_EMAIL does not also permit, on top of their own role
		pkgauthz.CapToRole(perms, permissions.RoleUnverifiedEmail, pkgauthz.EmailUnverified, unverifiedUser),
	}, customAuthenticatorPolicies(config)...)
}
