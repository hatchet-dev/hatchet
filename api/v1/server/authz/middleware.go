package authz

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"go.yaml.in/yaml/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type AuthZ struct {
	config *server.ServerConfig
	rbac   *Authorizer
	l      *zerolog.Logger
}

func NewAuthZ(config *server.ServerConfig) *AuthZ {
	rbacAuthorizer, err := NewAuthorizer()
	if err != nil {
		panic(err)
	}
	return &AuthZ{
		config: config,
		l:      config.Logger,
		rbac:   rbacAuthorizer,
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

	if operationIn(r.OperationID, permittedWithUnverifiedEmail) {
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

func NewAuthorizer() (*Authorizer, error) {
	permMap, err := LoadYaml()
	if err != nil {
		return nil, err
	}
	return &Authorizer{
		permissionMap: *permMap,
	}, nil
}

func (a *Authorizer) IsAuthorized(role sqlcv1.TenantMemberRole, operation string) bool {
	return a.permissionMap.HasPermission(string(role), operation)
}

type Role struct {
	Inherits    *[]string
	Permissions *[]string
}

type PermissionError struct {
	Message string
}

func (e *PermissionError) Error() string {
	return e.Message
}

type PermissionMap struct {
	Roles map[string]*Role
}

func (p *PermissionMap) HasPermission(roleName string, operation string) bool {
	curRole := p.Roles[roleName]
	if curRole.Permissions != nil {
		inRole := operationIn(operation, *curRole.Permissions)
		if inRole {
			return true
		}
	}
	if curRole.Inherits != nil {
		for _, inheritedRoleName := range *curRole.Inherits {
			if p.HasPermission(inheritedRoleName, operation) {
				return true
			}
		}
	}
	return false
}

func (p *PermissionMap) ValidInheritance(roleName string) error {
	if p.Roles[roleName].Inherits == nil {
		return nil
	}
	for _, inheritedRole := range *p.Roles[roleName].Inherits {
		_, ok := p.Roles[inheritedRole]
		if !ok {
			return &PermissionError{
				Message: fmt.Sprintf("%s inherits from %s which does not exist", roleName, inheritedRole),
			}
		}
	}
	return nil
}

func (p *PermissionMap) Validate() error {
	for roleName := range p.Roles {
		if err := p.ValidInheritance(roleName); err != nil {
			return err
		}
	}
	return nil
}

func LoadYaml() (*PermissionMap, error) {
	_, yamlPath, _, _ := runtime.Caller(0)
	yamlFile, err := os.ReadFile(filepath.Join(filepath.Dir(yamlPath), "rbac.yaml"))
	if err != nil {
		return nil, err
	}
	var yamlContents PermissionMap
	err = yaml.Unmarshal(yamlFile, &yamlContents)
	if err != nil {
		return nil, err
	}
	if err := yamlContents.Validate(); err != nil {
		return nil, err
	}
	return &yamlContents, nil
}
