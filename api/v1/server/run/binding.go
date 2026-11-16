package run

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

// requestBinder keeps transport and decoder diagnostics out of generated handlers' responses.
type requestBinder struct {
	echo.Binder
	l *zerolog.Logger
}

func (b *requestBinder) Bind(value interface{}, c echo.Context) error {
	err := b.Binder.Bind(value, c)
	if err == nil {
		return nil
	}

	b.l.Warn().Ctx(c.Request().Context()).Err(err).Msg("could not bind API request")

	code := http.StatusBadRequest
	var httpErr *echo.HTTPError
	if errors.As(err, &httpErr) {
		code = httpErr.Code
	}

	return echo.NewHTTPError(code, http.StatusText(code))
}
