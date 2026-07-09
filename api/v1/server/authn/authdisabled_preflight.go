package authn

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func (a *AuthN) resolveAuthDisabled(c echo.Context) (bool, error) {
	ctx := c.Request().Context()

	user, err := a.config.V1.User().GetUserByEmail(ctx, a.config.Seed.AdminEmail)
	if err != nil {
		a.l.Error().Ctx(ctx).Err(err).Msg("authdisabled: could not resolve the seeded admin user")

		return true, echo.NewHTTPError(http.StatusInternalServerError, "authdisabled: could not resolve the seeded admin user; ensure the database was seeded (ADMIN_EMAIL/ADMIN_PASSWORD)")
	}

	c.Set("user", user)
	c.Set("auth_strategy", "authdisabled")

	return true, nil
}
