package authz

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/auth/exchangetoken"
	"github.com/hatchet-dev/hatchet/pkg/auth/rbac"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type AuthZ struct {
	config *server.ServerConfig
	rbac   *rbac.Authorizer
	l      *zerolog.Logger
}

func NewAuthZ(config *server.ServerConfig) (*AuthZ, error) {
	rbacAuthorizer, err := newHatchetAuthorizer()
	if err != nil {
		return nil, err
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

	// authPreflight only returns handled=true in authdisabled builds, where it authorizes the
	// request against the NOAUTH role and short-circuits the strategy switch below. In normal
	// builds it is a no-op (returns false), so authorization always proceeds. Do not invert this.
	if handled, err := a.authPreflight(c, r); handled {
		return err
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
	if err := a.ensureVerifiedEmail(c, r); err != nil {
		a.l.Debug().Ctx(c.Request().Context()).Err(err).Msgf("error ensuring verified email")
		return echo.NewHTTPError(http.StatusUnauthorized, "Please verify your email before continuing")
	}

	if err := a.validateUserTenantPermissions(c, r); err != nil {
		return err
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
	// tokens are not tied to a member role, so they cannot change Labs entitlements
	"TenantEntitlementsUpdate",
}

// Bearer tokens have tenant access checked by authentication and optional viewer restrictions here.
func (a *AuthZ) handleBearerAuth(c echo.Context, r *middleware.RouteInfo) error {
	// check for is_exchange_token set in the context, in which case we need to validate the user set in the context
	// exchange tokens are subject to the same RBAC restrictions as cookie auth, since they represent a user. only
	// regular bearer tokens should be subject to the additional restrictions in restrictedWithBearerToken
	if isExchangeToken, ok := c.Get(middleware.IsExchangeTokenContextKey).(bool); ok && isExchangeToken {
		isSupportToken, _ := c.Get(middleware.IsSupportTokenContextKey).(bool)

		if isSupportToken && a.config.Auth.SupportTokenClient == nil {
			a.l.Error().Msgf("support token client is not configured, but is_support_token is set in context")
			return echo.NewHTTPError(http.StatusInternalServerError, "Support token client is not configured")
		}

		if !isSupportToken && a.config.Auth.ExchangeTokenClient == nil {
			a.l.Error().Msgf("exchange token client is not configured, but is_exchange_token is set in context")
			return echo.NewHTTPError(http.StatusInternalServerError, "Exchange token client is not configured")
		}

		if err := a.ensureVerifiedEmail(c, r); err != nil {
			a.l.Debug().Ctx(c.Request().Context()).Err(err).Msgf("error ensuring verified email for exchange token user")
			return echo.NewHTTPError(http.StatusUnauthorized, "Please verify your email before continuing")
		}

		if err := a.validateUserTenantPermissions(c, r); err != nil {
			a.l.Debug().Ctx(c.Request().Context()).Err(err).Msgf("error validating user tenant permissions for exchange token user")
			return echo.NewHTTPError(http.StatusUnauthorized, "Not authorized to view this resource")
		}
	} else if !isExchangeToken {
		if rbac.OperationIn(r.OperationID, restrictedWithBearerToken) {
			return echo.NewHTTPError(http.StatusUnauthorized, "Not authorized to perform this operation")
		}
		tokenID, ok := c.Get(string(analytics.APITokenIDKey)).(uuid.UUID)
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "Missing API token")
		}
		token, err := a.config.V1.APIToken().GetAPITokenById(c.Request().Context(), tokenID)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid API token")
		}
		if token.ReadOnly {
			return a.authorizeTenantOperations(string(sqlcv1.TenantMemberRoleVIEWER), r)
		}
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

func (a *AuthZ) validateUserTenantPermissions(c echo.Context, r *middleware.RouteInfo) error {
	unauthorized := echo.NewHTTPError(http.StatusUnauthorized, "Not authorized to view this resource")
	ctx := c.Request().Context()

	// if tenant is set in the context, verify that the user is a member of the tenant
	if tenant, ok := c.Get("tenant").(*sqlcv1.Tenant); ok {
		user, ok := c.Get("user").(*sqlcv1.User)

		if !ok {
			a.l.Debug().Ctx(ctx).Msgf("user not found in context")

			return unauthorized
		}

		if isSupport, _ := c.Get(middleware.IsSupportTokenContextKey).(bool); isSupport {
			return a.validateSupportTenantPermissions(c, r, tenant, user)
		}

		// check if the user is a member of the tenant
		tenantMember, err := a.config.V1.Tenant().GetTenantMemberByUserID(c.Request().Context(), tenant.ID, user.ID)

		if err != nil {
			var limitErr *fairpool.LimitError
			if errors.As(err, &limitErr) {
				return err
			}

			a.l.Debug().Ctx(ctx).Err(err).Msgf("error getting tenant member")

			return unauthorized
		}

		if tenantMember == nil {
			a.l.Debug().Ctx(ctx).Msgf("user is not a member of the tenant")

			return unauthorized
		}

		// set the tenant member in the context
		c.Set("tenant-member", tenantMember)

		// authorize tenant operations
		if err := a.authorizeTenantOperations(string(tenantMember.Role), r); err != nil {
			a.l.Debug().Ctx(ctx).Err(err).Msgf("error authorizing tenant operations")

			return unauthorized
		}
	}

	return nil
}

// there is no TenantMember row for support access: a synthetic member carrying the token's role goes in the context
func (a *AuthZ) validateSupportTenantPermissions(c echo.Context, r *middleware.RouteInfo, tenant *sqlcv1.Tenant, user *sqlcv1.User) error {
	unauthorized := echo.NewHTTPError(http.StatusUnauthorized, "Not authorized to view this resource")
	ctx := c.Request().Context()

	claims, ok := c.Get(middleware.SupportClaimsContextKey).(*exchangetoken.SupportClaims)

	if !ok || claims == nil {
		a.l.Error().Ctx(ctx).Msg("support token request without support claims in context")

		return unauthorized
	}

	if claims.TenantID != tenant.ID || claims.UserID != user.ID {
		a.l.Error().Ctx(ctx).Msg("support claims do not match the tenant or user in context")

		return unauthorized
	}

	if !exchangetoken.IsSupportRole(string(claims.Role)) {
		a.l.Error().Ctx(ctx).Str("role", string(claims.Role)).Msg("support token carries an unknown role")

		return unauthorized
	}

	c.Set("tenant-member", syntheticSupportMember(tenant, user, claims))

	if err := a.authorizeTenantOperations(string(claims.Role), r); err != nil {
		a.l.Debug().Ctx(ctx).Err(err).Str("support_grant_id", claims.GrantID.String()).Msgf("error authorizing support operation")

		return unauthorized
	}

	return nil
}

func syntheticSupportMember(tenant *sqlcv1.Tenant, user *sqlcv1.User, claims *exchangetoken.SupportClaims) *sqlcv1.PopulateTenantMembersRow {
	return &sqlcv1.PopulateTenantMembersRow{
		ID:              claims.GrantID,
		TenantId:        tenant.ID,
		UserId:          user.ID,
		Role:            sqlcv1.TenantMemberRole(claims.Role),
		CanViewPayloads: claims.Role == exchangetoken.SupportRoleOperator,
		Email:           user.Email,
		Name:            user.Name,
		TenantId_2:      tenant.ID,
		TenantCreatedAt: tenant.CreatedAt,
		TenantUpdatedAt: tenant.UpdatedAt,
		TenantName:      tenant.Name,
		TenantSlug:      tenant.Slug,
	}
}

func (a *AuthZ) authorizeTenantOperations(roleName string, r *middleware.RouteInfo) error {
	// support roles never use the extension bypass lists: an extension operation must be granted to the role explicitly
	if exchangetoken.IsSupportRole(roleName) {
		if rbac.OperationIn(r.OperationID, a.config.Auth.SupportRoleOperations[roleName]) {
			return nil
		}

		if !a.rbac.IsAuthorized(roleName, r.OperationID) {
			return echo.NewHTTPError(http.StatusUnauthorized, "Not authorized to perform this operation")
		}

		return nil
	}

	// if the operation is in the allowed operations, skip the RBAC check this is needed for extensions
	if rbac.OperationIn(r.OperationID, a.config.Auth.AllowedOperations) {
		return nil
	}

	if rbac.OperationIn(r.OperationID, a.config.Auth.AllowedWriteOperations) &&
		roleName != string(sqlcv1.TenantMemberRoleVIEWER) {
		return nil
	}

	// at the moment, tenant members are only restricted from creating other tenant users.
	if !a.rbac.IsAuthorized(roleName, r.OperationID) {
		return echo.NewHTTPError(http.StatusUnauthorized, "Not authorized to perform this operation")
	}

	// NOTE(abelanger5): this should be default-deny, but there's not a strong use-case for restricting member
	// operations at the moment. If there is, we should modify this logic.
	return nil
}
