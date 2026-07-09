//go:build !authdisabled

package authn

import "github.com/labstack/echo/v4"

func (a *AuthN) authPreflight(c echo.Context) (handled bool, err error) {
	return false, nil
}
