package authz

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/internal/permissions"
	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/enums"
	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type AuthZ struct {
	authorizer pkgauthz.Authorizer
	roles      *tenantRoleResolver
	l          *zerolog.Logger
}

func NewAuthZ(config *server.ServerConfig) (*AuthZ, error) {
	perms, err := permissions.Users()
	if err != nil {
		return nil, err
	}

	// Extension operations are not in the OSS spec, so they are granted after validation.
	if err := perms.Grant(string(enums.TenantMemberRoleVIEWER), config.Auth.AllowedOperations...); err != nil {
		return nil, err
	}

	if err := perms.Grant(string(enums.TenantMemberRoleMEMBER), config.Auth.AllowedWriteOperations...); err != nil {
		return nil, err
	}

	tokens, err := permissions.Tokens()
	if err != nil {
		return nil, err
	}

	// read-only tokens keep the extension reads VIEWER gets; "all" already grants every operation
	if err := tokens.GrantREST(pkgauthz.ReadOnlyCapability, config.Auth.AllowedOperations...); err != nil {
		return nil, err
	}

	roles := &tenantRoleResolver{repo: config.V1}
	everyoneElse := func(pkgauthz.Request) bool { return true }

	authorizer, err := pkgauthz.New(
		// tokens first: they handle tenant API tokens, and roles handle users and management tokens
		pkgauthz.WithPermissions(tokens, pkgauthz.RoleBased(perms, roles, everyoneElse)),
		pkgauthz.WithPolicies(buildPolicies(config, perms)...),
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
	ctx := pkgauthz.WithRequestCache(c.Request().Context())
	c.SetRequest(c.Request().WithContext(ctx))

	tenantID := tenantFromEcho(c)

	req := pkgauthz.Request{
		Principal: pkgauthz.PrincipalFrom(ctx),
		Action:    pkgauthz.REST(r.OperationID, tenantID),
	}

	d, err := a.authorizer.Authorize(context.WithValue(ctx, echoRequestCtxKey{}, echoRequest{c: c, route: r}), req)
	if err != nil {
		var hookErr hookError
		if errors.As(err, &hookErr) {
			return hookErr.err
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

		return httpErrorFor(req.Principal, d.Reason)
	}

	return a.setTenantMember(c, req.Principal, tenantID)
}

// setTenantMember keeps the tenant-member context value that handlers and CanViewPayloads still read.
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

// echoRequest lets the extension hook policies reach the echo request they were written against.
type echoRequest struct {
	c     echo.Context
	route *middleware.RouteInfo
}

// hookError carries an extension hook's own HTTP error through Authorize unchanged.
type hookError struct {
	err error
}

func (e hookError) Error() string { return e.err.Error() }

func (e hookError) Unwrap() error { return e.err }

// tenantFromEcho reads the tenant the populator resolved from the route.
func tenantFromEcho(c echo.Context) *uuid.UUID {
	if tenant, ok := c.Get("tenant").(*sqlcv1.Tenant); ok && tenant != nil {
		id := tenant.ID
		return &id
	}

	return nil
}

// httpErrorFor keeps the status codes and messages the old cookie and bearer paths returned.
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
