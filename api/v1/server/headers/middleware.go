package headers

import (
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
)

func Middleware() middleware.MiddlewareFunc {
	return func(r *middleware.RouteInfo) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("Strict-Transport-Security", "max-age=776000; includeSubDomains;  preload")
			c.Response().Header().Set("Content-Security-Policy", "script-src 'self'; frame-ancestors 'self'; object-src 'none'; base-uri 'self'")
			c.Response().Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			c.Response().Header().Set("X-Frame-Options", "DENY")
			c.Response().Header().Set("X-Content-Type-Options", "nosniff")
			c.Response().Header().Set("Permissions-Policy", "geolocation=(), midi=(), notifications=(), push=(), sync-xhr=(), microphone=(), camera=(), magnetometer=(), gyroscope=(), speaker=(), vibrate=(), fullscreen=(self), payment=()")
			c.Response().Header().Set("Cross-Origin-Embedder-Policy", "require-corp")
			c.Response().Header().Set("Cross-Origin-Opener-Policy", "same-origin")
			c.Response().Header().Set("Cross-Origin-Resource-Policy", "same-origin")
			return nil
		}
	}
}
