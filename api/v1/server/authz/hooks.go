//go:build !authdisabled

package authz

import (
	"context"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

// managementTokenHookPolicy defers management tokens to the extension that issued them, since no tenant role applies.
type managementTokenHookPolicy struct {
	auth server.CustomAuthenticator
}

func (p managementTokenHookPolicy) Evaluate(ctx context.Context, req pkgauthz.Request) (pkgauthz.Decision, error) {
	t, ok := req.Principal.(pkgauthz.TokenPrincipal)
	if !ok || t.Kind != pkgauthz.ManagementToken {
		return pkgauthz.Allow(), nil
	}

	return runHook(ctx, req, p.auth.Authorize)
}

// cookieSessionHookPolicy lets the extension add checks for cookie sessions, which cloud uses for organization-owner access.
type cookieSessionHookPolicy struct {
	auth server.CustomAuthenticator
}

func (p cookieSessionHookPolicy) Evaluate(ctx context.Context, req pkgauthz.Request) (pkgauthz.Decision, error) {
	u, ok := req.Principal.(pkgauthz.UserPrincipal)
	if !ok || u.Via != pkgauthz.ViaCookie {
		return pkgauthz.Allow(), nil
	}

	return runHook(ctx, req, p.auth.CookieAuthorizerHook)
}

func customAuthenticatorPolicies(config *server.ServerConfig) []pkgauthz.Policy {
	if config.Auth.CustomAuthenticator == nil {
		return nil
	}

	return []pkgauthz.Policy{
		managementTokenHookPolicy{auth: config.Auth.CustomAuthenticator},
		cookieSessionHookPolicy{auth: config.Auth.CustomAuthenticator},
	}
}

func runHook(ctx context.Context, req pkgauthz.Request, hook func(echo.Context, *middleware.RouteInfo) error) (pkgauthz.Decision, error) {
	c, route, ok := EchoRequestFrom(ctx)
	if !ok || req.Action.Type() != pkgauthz.RESTEndpoint {
		return pkgauthz.Allow(), nil
	}

	if err := hook(c, route); err != nil {
		return pkgauthz.Decision{}, err
	}

	return pkgauthz.Allow(), nil
}
