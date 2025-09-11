package ratelimit

import (
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"

	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type RateLimitMiddleware struct {
	config  *server.ServerConfig
	swagger *openapi3.T
}

func NewRateLimitMiddleware(config *server.ServerConfig, swagger *openapi3.T) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		config:  config,
		swagger: swagger,
	}
}

func (m *RateLimitMiddleware) Middleware() echo.MiddlewareFunc {
	router, err := gorillamux.NewRouter(m.swagger)
	if err != nil {
		return nil
	}

	rateLimiterConfig := echoMiddleware.RateLimiterConfig{
		Skipper: func(c echo.Context) bool {
			route, _, err := router.FindRoute(c.Request())
			if err != nil {
				c.Logger().Error(err)
				return false
			}

			enableRateLimitingInt := route.Operation.Extensions["x-enable-rate-limiting"]
			if enableRateLimitingInt == nil {
				return true
			}

			enableRateLimiting := enableRateLimitingInt.(bool)

			return !enableRateLimiting
		},
		Store: echoMiddleware.NewRateLimiterMemoryStoreWithConfig(
			echoMiddleware.RateLimiterMemoryStoreConfig{Rate: rate.Limit(m.config.Runtime.APIRateLimit), ExpiresIn: m.config.Runtime.APIRateLimitWindow},
		),
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			route, _, err := router.FindRoute(ctx.Request())
			if err != nil {
				return "", err
			}

			id := ctx.RealIP() + ":" + route.Operation.OperationID

			return id, nil
		},
		ErrorHandler: func(context echo.Context, err error) error {
			return context.JSON(http.StatusForbidden, nil)
		},
		DenyHandler: func(context echo.Context, identifier string, err error) error {
			return context.JSON(http.StatusTooManyRequests, nil)
		},
	}

	return echoMiddleware.RateLimiterWithConfig(rateLimiterConfig)
}
