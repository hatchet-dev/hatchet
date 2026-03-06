package authz

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type AuthZ struct {
	config *server.ServerConfig
	rbac   *Authorizer
	l      *zerolog.Logger
}

func NewAuthZ(config *server.ServerConfig, spec *openapi3.T) *AuthZ {
	return &AuthZ{
		config: config,
		l:      config.Logger,
		rbac:   NewAuthorizer(),
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

// At the moment, there's no further bearer auth because bearer tokens are admin-scoped
// and we check that the bearer token has access to the tenant in the authn step.
func (a *AuthZ) handleBearerAuth(c echo.Context, r *middleware.RouteInfo) error {
	if !a.rbac.IsAuthorized(sqlcv1.TenantMemberRoleBEARERTOKEN, r.OperationID) {
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

func (a *AuthZ) ensureVerifiedEmail(c echo.Context, r *middleware.RouteInfo) error {
	user, ok := c.Get("user").(*sqlcv1.User)

	if !ok {
		return nil
	}

	if a.rbac.IsAuthorized(sqlcv1.TenantMemberRoleUNVERIFIEDEMAIL, r.OperationID) {
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

func operationIn(operationId string, operationIds []string) bool {
	for _, id := range operationIds {
		if strings.EqualFold(operationId, id) {
			return true
		}
	}

	return false
}

type Authorizer struct {
	permissionMap PermissionMap
}

func NewAuthorizer() *Authorizer {
	permMap, err := LoadYaml()
	if err != nil {
		return nil
	}
	return &Authorizer{
		permissionMap: *permMap,
	}
}

func (a *Authorizer) IsAuthorized(role sqlcv1.TenantMemberRole, operation string) bool {
	return operationIn(operation, *a.permissionMap.Roles[string(role)].Permissions)
}

type Role struct {
	Inherits    *[]string
	Permissions *[]string
	propagated  bool
}
type PermissionMap struct {
	Roles map[string]*Role
}

func (b *PermissionMap) RecurseOnRole(role *Role) []string {
	if role.Inherits == nil || role.propagated {
		role.propagated = true
		return *role.Permissions
	}
	mergedPerms := make([]string, 0)
	for _, perm := range *role.Inherits {
		mergedPerms = append(mergedPerms, b.RecurseOnRole(b.Roles[perm])...)
	}
	if role.Permissions != nil {
		mergedPerms = append(mergedPerms, *role.Permissions...)
	}
	role.Permissions = &mergedPerms
	role.propagated = true
	return mergedPerms
}

func (b *PermissionMap) PropagatePerms() {
	for _, role := range b.Roles {
		b.RecurseOnRole(role)
	}
}

func LoadYaml() (*PermissionMap, error) {
	yamlFile, err := os.ReadFile("rbac.yaml")
	if err != nil {
		return nil, err
	}
	var yamlContents PermissionMap
	err = yaml.Unmarshal(yamlFile, &yamlContents)
	if err != nil {
		return nil, err
	}
	yamlContents.PropagatePerms()
	return &yamlContents, nil
}
