package authn

import (
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

func GetRedirectWithError(ctx echo.Context, l *zerolog.Logger, internalErr error, userErr string) error {
	l.Err(internalErr).Msgf("redirecting with error")
	return ctx.Redirect(302, "/auth/login?error="+userErr)
}

func GetRedirectNoError(ctx echo.Context, serverURL string) error {
	return ctx.Redirect(302, serverURL)
}
