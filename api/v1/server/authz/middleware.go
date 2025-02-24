package authz

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

type AuthZ struct {
	config *server.ServerConfig

	l *zerolog.Logger
}

func NewAuthZ(config *server.ServerConfig) *AuthZ {
	return &AuthZ{
		config: config,
		l:      config.Logger,
	}
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
	if tenant, ok := c.Get("tenant").(*dbsqlc.Tenant); ok {
		user, ok := c.Get("user").(*dbsqlc.User)

		if !ok {
			a.l.Debug().Msgf("user not found in context")

			return unauthorized
		}

		// check if the user is a member of the tenant
		tenantMember, err := a.config.APIRepository.Tenant().GetTenantMemberByUserID(c.Request().Context(), sqlchelpers.UUIDToStr(tenant.ID), sqlchelpers.UUIDToStr(user.ID))

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
		if err := a.authorizeTenantOperations(tenant, tenantMember, r); err != nil {
			a.l.Debug().Err(err).Msgf("error authorizing tenant operations")

			return unauthorized
		}
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
	if operationIn(r.OperationID, restrictedWithBearerToken) {
		return echo.NewHTTPError(http.StatusUnauthorized, "Not authorized to perform this operation")
	}

	return nil
}

var permittedWithUnverifiedEmail = []string{
	"UserGetCurrent",
	"UserUpdateLogout",
}

func (a *AuthZ) ensureVerifiedEmail(c echo.Context, r *middleware.RouteInfo) error {
	user, ok := c.Get("user").(*dbsqlc.User)

	if !ok {
		return nil
	}

	if operationIn(r.OperationID, permittedWithUnverifiedEmail) {
		return nil
	}

	if !user.EmailVerified {
		return echo.NewHTTPError(http.StatusForbidden, "Please verify your email before continuing")
	}

	return nil
}

var adminAndOwnerOnly = []string{
	"TenantInviteList",
	"TenantInviteCreate",
	"TenantInviteUpdate",
	"TenantInviteDelete",
	"TenantMemberList",
	// members cannot create API tokens for a tenant, because they have admin permissions
	"ApiTokenList",
	"ApiTokenCreate",
	"ApiTokenUpdateRevoke",
}

func (a *AuthZ) authorizeTenantOperations(tenant *dbsqlc.Tenant, tenantMember *dbsqlc.PopulateTenantMembersRow, r *middleware.RouteInfo) error {
	// if the user is an owner, they can do anything
	if tenantMember.Role == dbsqlc.TenantMemberRoleOWNER {
		return nil
	}

	// if the user is an admin, they can do anything at the moment. Some downstream handlers will case on
	// admin roles, for example admins cannot mark users as owners.
	if tenantMember.Role == dbsqlc.TenantMemberRoleADMIN {
		return nil
	}

	// at the moment, tenant members are only restricted from creating other tenant users.
	if operationIn(r.OperationID, adminAndOwnerOnly) {
		return echo.NewHTTPError(http.StatusUnauthorized, "Not authorized to perform this operation")
	}

	// NOTE(abelanger5): this should be default-deny, but there's not a strong use-case for restricting member
	// operations at the moment. If there is, we should modify this logic.
	return nil
}

func operationIn(operationId string, operationIds []string) bool {
	for _, id := range operationIds {
		if strings.EqualFold(operationId, id) {
			return true
		}
	}

	return false
}
