//go:build !authdisabled

package authz

import (
	"context"

	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

// Extension hooks run last so the email and token checks keep their own error messages.
func buildPolicies(config *server.ServerConfig, _ pkgauthz.RolePermissions) []pkgauthz.Policy {
	return append([]pkgauthz.Policy{emailVerifiedPolicy{}, apiTokenRestrictionPolicy{}}, extensionPolicies(config)...)
}

var restrictedWithBearerToken = []string{
	// bearer tokens cannot read, list, or write other bearer tokens
	"ApiTokenList",
	"ApiTokenCreate",
	"ApiTokenUpdateRevoke",
	// tokens are not tied to a member role, so they cannot change Labs entitlements
	"TenantEntitlementsUpdate",
}

var permittedWithUnverifiedEmail = []string{
	"UserGetCurrent",
	"UserUpdateLogout",
}

type emailVerifiedPolicy struct{}

func (emailVerifiedPolicy) Evaluate(_ context.Context, req pkgauthz.Request) (pkgauthz.Decision, error) {
	u, ok := req.Principal.(pkgauthz.UserPrincipal)
	if !ok || u.EmailVerified || pkgauthz.OperationIn(req.Action.Name(), permittedWithUnverifiedEmail) {
		return pkgauthz.Allow(), nil
	}

	return pkgauthz.Deny(pkgauthz.EmailUnverified), nil
}

type apiTokenRestrictionPolicy struct{}

func (apiTokenRestrictionPolicy) Evaluate(_ context.Context, req pkgauthz.Request) (pkgauthz.Decision, error) {
	t, ok := req.Principal.(pkgauthz.TokenPrincipal)
	if !ok || t.Kind != pkgauthz.TenantAPIToken || !pkgauthz.OperationIn(req.Action.Name(), restrictedWithBearerToken) {
		return pkgauthz.Allow(), nil
	}

	return pkgauthz.Deny(pkgauthz.TokenRestricted), nil
}
