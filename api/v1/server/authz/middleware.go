package authz

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type AuthZ struct {
	authorizer pkgauthz.Authorizer
	roles      *tenantRoleResolver
	l          *zerolog.Logger
}

func NewAuthZ(config *server.ServerConfig) (*AuthZ, error) {
	perms, tokens, err := LoadPermissions(config)
	if err != nil {
		return nil, err
	}

	roles := &tenantRoleResolver{repo: config.V1}
	// role-based permissions take every request the token permissions did not handle
	everyoneElse := func(pkgauthz.Request) bool { return true }

	authorizer, err := pkgauthz.New(
		// the order here determines precedence. tokens first, then role based authentication
		pkgauthz.WithPermissions(tokens, pkgauthz.RoleBased(perms, roles, everyoneElse)),
		// supplied policies run after the built-in ones so their errors never mask an unverified email
		pkgauthz.WithPolicies(append(buildPolicies(config, perms), config.Auth.Policies...)...),
	)
	if err != nil {
		return nil, err
	}

	return &AuthZ{
		authorizer: authorizer,
		roles:      roles,
		l:          config.Logger,
	}, nil
}

func (a *AuthZ) Middleware(r *middleware.RouteInfo) echo.HandlerFunc {
	return func(c echo.Context) error {
		return a.authorize(c, r)
	}
}

func (a *AuthZ) authorize(c echo.Context, r *middleware.RouteInfo) error {
	if r.Security.IsOptional() || r.Security.NoAuth() {
		return nil
	}

	return a.authorizeSecured(c, r)
}

func (a *AuthZ) authorizeSecured(c echo.Context, r *middleware.RouteInfo) error {
	// Every check in this request (role resolution, setTenantMember below) shares loaded rows through
	// this cache; without it each pkgauthz.Fetch goes to the database.
	ctx := pkgauthz.WithRequestCache(c.Request().Context())
	c.SetRequest(c.Request().WithContext(ctx))

	tenantID := tenantFromEcho(c)

	req := pkgauthz.Request{
		Principal: pkgauthz.PrincipalFrom(ctx),
		Action:    pkgauthz.REST(r.OperationID, tenantID),
	}

	d, err := a.authorizer.Authorize(context.WithValue(ctx, echoRequestCtxKey{}, echoRequest{c: c, route: r}), req)
	if err != nil {
		// policies can choose the response by returning an HTTP error
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			return httpErr
		}

		var limitErr *fairpool.LimitError
		if errors.As(err, &limitErr) {
			return err
		}

		a.l.Debug().Ctx(ctx).Err(err).Msg("error authorizing request")

		return echo.NewHTTPError(http.StatusUnauthorized, "Not authorized to view this resource")
	}

	if !d.Allowed {
		a.l.Debug().Ctx(ctx).Msgf("denied %s: %s", r.OperationID, d.Reason)

		// keeps the status codes and messages the old cookie and bearer paths returned
		return httpErrorFor(req.Principal, d.Reason)
	}

	// Handlers and CanViewPayloads still read the tenant member from the echo context, so set it here.
	// The row is already cached from role resolution, so this does not hit the database again.
	return a.setTenantMember(c, req.Principal, tenantID)
}

func (a *AuthZ) setTenantMember(c echo.Context, p pkgauthz.Principal, tenantID *uuid.UUID) error {
	user, ok := p.(pkgauthz.UserPrincipal)
	if !ok || tenantID == nil {
		return nil
	}

	member, err := a.roles.member(c.Request().Context(), *tenantID, user.UserID)
	if err != nil {
		return err
	}

	if member != nil {
		c.Set("tenant-member", member)
	}

	return nil
}

type echoRequestCtxKey struct{}

type echoRequest struct {
	c     echo.Context
	route *middleware.RouteInfo
}

// EchoRequestFrom gives policies the echo request and route for REST checks; ok is false for any other caller.
func EchoRequestFrom(ctx context.Context) (echo.Context, *middleware.RouteInfo, bool) {
	er, ok := ctx.Value(echoRequestCtxKey{}).(echoRequest)
	return er.c, er.route, ok
}

func tenantFromEcho(c echo.Context) *uuid.UUID {
	if tenant, ok := c.Get("tenant").(*sqlcv1.Tenant); ok && tenant != nil {
		id := tenant.ID
		return &id
	}

	return nil
}

func httpErrorFor(p pkgauthz.Principal, reason pkgauthz.DenyReason) error {
	switch reason {
	case pkgauthz.EmailUnverified:
		return echo.NewHTTPError(http.StatusUnauthorized, "Please verify your email before continuing")
	case pkgauthz.TokenRestricted:
		return echo.NewHTTPError(http.StatusUnauthorized, "Not authorized to perform this operation")
	case pkgauthz.RoleInsufficient:
		if _, isToken := p.(pkgauthz.TokenPrincipal); isToken {
			return echo.NewHTTPError(http.StatusUnauthorized, "Not authorized to perform this operation")
		}
	case pkgauthz.MissingCapability:
		return echo.NewHTTPError(http.StatusUnauthorized, "Not authorized to perform this operation")
	case pkgauthz.NotMember:
	}

	return echo.NewHTTPError(http.StatusUnauthorized, "Not authorized to view this resource")
}
