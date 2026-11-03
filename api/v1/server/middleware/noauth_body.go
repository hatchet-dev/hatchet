package middleware

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

const (
	noAuthBodyLimit       = 1 << 20
	noAuthBodyReadTimeout = 30 * time.Second
	maxBodyBytesExtension = "x-max-body-bytes"
)

// NoAuthBodyLimit bounds what an unauthenticated caller can make the server buffer. Authenticated
// operations reject a bad caller before reading the body; these read it before knowing who sent it.
// An operation that needs more than the default declares x-max-body-bytes in the spec.
func NoAuthBodyLimit(r *RouteInfo) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !r.Security.NoAuth() {
			return nil
		}

		limit := int64(noAuthBodyLimit)

		if r.Route != nil && r.Route.Operation != nil {
			if v, ok := r.Route.Operation.Extensions[maxBodyBytesExtension].(float64); ok && v > 0 {
				limit = int64(v)
			}
		}

		req := c.Request()

		if req.ContentLength > limit {
			return echo.NewHTTPError(http.StatusRequestEntityTooLarge)
		}

		req.Body = http.MaxBytesReader(c.Response(), req.Body, limit)

		// net/http lifts the read deadline once the body has been read, so it bounds only the
		// body. A request without a body is already watched for the caller going away, and an
		// expired deadline there would cancel the request.
		if req.ProtoMajor == 1 && req.ContentLength != 0 {
			_ = http.NewResponseController(c.Response()).SetReadDeadline(time.Now().Add(noAuthBodyReadTimeout))
		}

		return nil
	}
}
