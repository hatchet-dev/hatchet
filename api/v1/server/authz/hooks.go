//go:build !authdisabled

package authz

import (
	"context"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

// customAuthorizePolicy runs the extension's Authorize hook for principals its own authenticator created.
type customAuthorizePolicy struct {
	auth server.CustomAuthenticator
}

func (p customAuthorizePolicy) Evaluate(ctx context.Context, req pkgauthz.Request) (pkgauthz.Decision, error) {
	t, ok := req.Principal.(pkgauthz.TokenPrincipal)
	if !ok || t.Kind != pkgauthz.ManagementToken {
		return pkgauthz.Allow(), nil
	}

	return runHook(ctx, req, p.auth.Authorize)
}

// cookieHookPolicy runs the extension's cookie hook, which cloud uses for organization-owner checks.
type cookieHookPolicy struct {
	auth server.CustomAuthenticator
}

func (p cookieHookPolicy) Evaluate(ctx context.Context, req pkgauthz.Request) (pkgauthz.Decision, error) {
	u, ok := req.Principal.(pkgauthz.UserPrincipal)
	if !ok || u.Via != pkgauthz.ViaCookie {
		return pkgauthz.Allow(), nil
	}

	return runHook(ctx, req, p.auth.CookieAuthorizerHook)
}

func extensionPolicies(config *server.ServerConfig) []pkgauthz.Policy {
	if config.Auth.CustomAuthenticator == nil {
		return nil
	}

	return []pkgauthz.Policy{
		customAuthorizePolicy{auth: config.Auth.CustomAuthenticator},
		cookieHookPolicy{auth: config.Auth.CustomAuthenticator},
	}
}

// runHook only fires for the route-level check, which is the only caller that attaches the echo request.
func runHook(ctx context.Context, req pkgauthz.Request, hook func(echo.Context, *middleware.RouteInfo) error) (pkgauthz.Decision, error) {
	er, ok := ctx.Value(echoRequestCtxKey{}).(echoRequest)
	if !ok || req.Action.Type() != pkgauthz.RESTEndpoint {
		return pkgauthz.Allow(), nil
	}

	if err := hook(er.c, er.route); err != nil {
		return pkgauthz.Decision{}, hookError{err: err}
	}

	return pkgauthz.Allow(), nil
}
