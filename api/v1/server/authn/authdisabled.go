//go:build authdisabled

package authn

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func (a *AuthN) authPreflight(c echo.Context) (handled bool, err error) {
	forbidden := echo.NewHTTPError(http.StatusForbidden, "Please provide valid credentials")

	ctx := c.Request().Context()

	user, err := a.config.V1.User().GetUserByEmail(ctx, a.config.Seed.AdminEmail)

	if err != nil {
		a.l.Error().Ctx(ctx).Err(err).Msg("authdisabled: could not resolve default user")

		return true, forbidden
	}

	c.Set("user", user)
	c.Set("auth_strategy", "authdisabled")

	return true, nil
}
