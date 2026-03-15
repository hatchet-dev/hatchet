package authz

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/api/v1/server/rbac"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type AuthZ struct {
	config *server.ServerConfig
	rbac   *rbac.Authorizer
	l      *zerolog.Logger
}

func NewAuthZ(config *server.ServerConfig) (*AuthZ, error) {
	rbacAuthorizer, err := rbac.NewAuthorizer()
	if err != nil {
		return nil, err
	}

	if len(config.Auth.AdditionalRBACYAML) > 0 {
		additional, err := rbac.LoadYamlFrom(config.Auth.AdditionalRBACYAML)
		if err != nil {
			return nil, fmt.Errorf("error loading additional RBAC YAML: %w", err)
		}
		rbacAuthorizer.MergePermissions(additional)
	}

	return &AuthZ{
		config: config,
		l:      config.Logger,
		rbac:   rbacAuthorizer,
	}, nil
}

func (a *AuthZ) Middleware(r *middleware.RouteInfo) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := a.authorize(c, r)
		if err != nil {
			return err
		}

		return nil
	}
}

func (a *AuthZ) authorize(c echo.Context, r *middleware.RouteInfo) error {
	if r.Security.IsOptional() || r.Security.NoAuth() {
		return nil
	}

	var err error

	switch c.Get("auth_strategy").(string) {
	case "cookie":
		err = a.handleCookieAuth(c, r)
	case "bearer":
		err = a.handleBearerAuth(c, r)
	case "custom":
		err = a.handleCustomAuth(c, r)
	default:
		return echo.NewHTTPError(http.StatusInternalServerError, "No authorization strategy was checked")
	}

	return err
}

func (a *AuthZ) handleCookieAuth(c echo.Context, r *middleware.RouteInfo) error {
	unauthorized := echo.NewHTTPError(http.StatusUnauthorized, "Not authorized to view this resource")

	if err := a.ensureVerifiedEmail(c, r); err != nil {
		a.l.Debug().Err(err).Msgf("error ensuring verified email")
		return echo.NewHTTPError(http.StatusUnauthorized, "Please verify your email before continuing")
	}

	// if tenant is set in the context, verify that the user is a member of the tenant
	if tenant, ok := c.Get("tenant").(*sqlcv1.Tenant); ok {
		user, ok := c.Get("user").(*sqlcv1.User)

		if !ok {
			a.l.Debug().Msgf("user not found in context")

			return unauthorized
		}

		// check if the user is a member of the tenant
		tenantMember, err := a.config.V1.Tenant().GetTenantMemberByUserID(c.Request().Context(), tenant.ID, user.ID)

		if err != nil {
			a.l.Debug().Err(err).Msgf("error getting tenant member")

			return unauthorized
		}

		if tenantMember == nil {
			a.l.Debug().Msgf("user is not a member of the tenant")

			return unauthorized
		}

		// set the tenant member in the context
		c.Set("tenant-member", tenantMember)

		// authorize tenant operations
		if err := a.authorizeTenantOperations(tenantMember.Role, r); err != nil {
			a.l.Debug().Err(err).Msgf("error authorizing tenant operations")

			return unauthorized
		}
	}

	if a.config.Auth.CustomAuthenticator != nil {
		return a.config.Auth.CustomAuthenticator.CookieAuthorizerHook(c, r)
	}

	return nil
}

var restrictedWithBearerToken = []string{
	// bearer tokens cannot read, list, or write other bearer tokens
	"ApiTokenList",
	"ApiTokenCreate",
	"ApiTokenUpdateRevoke",
}

// At the moment, there's no further bearer auth because bearer tokens are admin-scoped
// and we check that the bearer token has access to the tenant in the authn step.
func (a *AuthZ) handleBearerAuth(c echo.Context, r *middleware.RouteInfo) error {
	if rbac.OperationIn(r.OperationID, restrictedWithBearerToken) {
		return echo.NewHTTPError(http.StatusUnauthorized, "Not authorized to perform this operation")
	}

	return nil
}

func (a *AuthZ) handleCustomAuth(c echo.Context, r *middleware.RouteInfo) error {
	if a.config.Auth.CustomAuthenticator == nil {
		return fmt.Errorf("custom auth handler is not set")
	}

	return a.config.Auth.CustomAuthenticator.Authorize(c, r)
}

var permittedWithUnverifiedEmail = []string{
	"UserGetCurrent",
	"UserUpdateLogout",
}

func (a *AuthZ) ensureVerifiedEmail(c echo.Context, r *middleware.RouteInfo) error {
	user, ok := c.Get("user").(*sqlcv1.User)

	if !ok {
		return nil
	}

	if rbac.OperationIn(r.OperationID, permittedWithUnverifiedEmail) {
		return nil
	}

	if !user.EmailVerified {
		return echo.NewHTTPError(http.StatusForbidden, "Please verify your email before continuing")
	}

	return nil
}

func (a *AuthZ) authorizeTenantOperations(tenantMemberRole sqlcv1.TenantMemberRole, r *middleware.RouteInfo) error {

	// at the moment, tenant members are only restricted from creating other tenant users.
	if !a.rbac.IsAuthorized(tenantMemberRole, r.OperationID) {
		return echo.NewHTTPError(http.StatusUnauthorized, "Not authorized to perform this operation")
	}

	// NOTE(abelanger5): this should be default-deny, but there's not a strong use-case for restricting member
	// operations at the moment. If there is, we should modify this logic.
	return nil
}
