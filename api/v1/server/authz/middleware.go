package authz

import (
	"net/http"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/internal/config/server"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

type AuthZ struct {
	config *server.ServerConfig

	l *zerolog.Logger
}

func NewAuthZ(config *server.ServerConfig) *AuthZ {
	return &AuthZ{
		config: config,
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

	unauthorized := echo.NewHTTPError(http.StatusUnauthorized, "Not authorized to view this resource")

	// if tenant is set in the context, verify that the user is a member of the tenant
	if tenant, ok := c.Get("tenant").(*db.TenantModel); ok {
		user, ok := c.Get("user").(*db.UserModel)

		if !ok {
			a.l.Debug().Msgf("user not found in context")

			return unauthorized
		}

		// check if the user is a member of the tenant
		tenantMember, err := a.config.Repository.Tenant().GetTenantMemberByUserID(tenant.ID, user.ID)

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
	}

	return nil
}
