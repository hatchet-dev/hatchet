package redirect

import (
	"errors"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

var ErrRedirect = errors.New("redirecting")

func GetRedirectWithError(ctx echo.Context, l *zerolog.Logger, internalErr error, userErr string) error {
	l.Err(internalErr).Msgf("redirecting with error")
	err := ctx.Redirect(302, "/auth/login?error="+userErr)

	if err != nil {
		return err
	}

	return ErrRedirect
}

func GetRedirectNoError(ctx echo.Context, serverURL string) error {
	err := ctx.Redirect(302, serverURL)

	if err != nil {
		return err
	}

	return ErrRedirect
}
