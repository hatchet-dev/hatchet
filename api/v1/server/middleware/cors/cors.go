package cors

import (
	"net/http"
	"path"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

// Middleware returns an echo middleware that handles CORS for all requests.
//
// If AllowedOrigins is configured in the server runtime config, the request Origin header is
// matched against each pattern using wildcard glob syntax (e.g. "https://*.example.com").
// On a match, Access-Control-Allow-Origin is set to the exact origin value from the request.
// On no match for an OPTIONS preflight, 403 is returned. For other methods the request
// proceeds without the header, leaving the browser to enforce the same-origin policy.
//
// If AllowedOrigins is empty (not configured), Access-Control-Allow-Origin is set to "*",
// preserving the previous open behaviour.
func Middleware(config *server.ServerConfig) echo.MiddlewareFunc {
	allowedOrigins := config.Runtime.AllowedOrigins

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			origin := c.Request().Header.Get("Origin")
			allowOrigin, matched := resolveAllowOrigin(origin, allowedOrigins)

			if c.Request().Method == http.MethodOptions {
				if !matched {
					return c.NoContent(http.StatusForbidden)
				}

				c.Response().Header().Set("Access-Control-Allow-Origin", allowOrigin)
				c.Response().Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
				c.Response().Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-exchange-token")

				return c.NoContent(http.StatusOK)
			}

			if matched {
				c.Response().Header().Set("Access-Control-Allow-Origin", allowOrigin)
			}

			return next(c)
		}
	}
}

// resolveAllowOrigin returns the value to use for Access-Control-Allow-Origin and whether
// the request origin is permitted.
//
//   - No configured origins → ("*", true) — open / backwards-compatible.
//   - Configured origins, origin matches a pattern → (origin, true).
//   - Configured origins, no match → ("", false).
func resolveAllowOrigin(origin string, allowedOrigins []string) (string, bool) {
	if len(allowedOrigins) == 0 {
		return "*", true
	}

	if origin == "" {
		return "", false
	}

	for _, pattern := range allowedOrigins {
		matched, err := path.Match(pattern, origin)
		if err == nil && matched {
			return origin, true
		}
	}

	return "", false
}
