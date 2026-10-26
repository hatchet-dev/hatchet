package ratelimit

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/apierrors"
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

// storeConfig gives each identifier a bucket of `limit` requests that refills over `window`.
// Echo's Rate is per second, so the per-window limit has to be converted. A bucket idle for a
// full window is full again, so expiring it after `window` drops no state.
func storeConfig(limit int, window time.Duration) echoMiddleware.RateLimiterMemoryStoreConfig {
	limit = max(limit, 1)
	window = max(window, time.Second)

	return echoMiddleware.RateLimiterMemoryStoreConfig{
		Rate:      rate.Limit(float64(limit) / window.Seconds()),
		Burst:     limit,
		ExpiresIn: window,
	}
}

func (m *RateLimitMiddleware) Middleware() echo.MiddlewareFunc {
	router, err := gorillamux.NewRouter(m.swagger)
	if err != nil {
		return nil
	}

	storeCfg := storeConfig(m.config.Runtime.APIRateLimit, m.config.Runtime.APIRateLimitWindow)
	retryAfter := strconv.Itoa(int(math.Ceil(storeCfg.ExpiresIn.Seconds() / float64(storeCfg.Burst))))

	rateLimiterConfig := echoMiddleware.RateLimiterConfig{
		Skipper: func(c echo.Context) bool {
			route, _, err := router.FindRoute(c.Request())
			if err != nil {
				m.config.Logger.Info().Err(err).Msg("error finding route")
				return false
			}

			enableRateLimitingInt := route.Operation.Extensions["x-enable-rate-limiting"]
			if enableRateLimitingInt == nil {
				return true
			}

			enableRateLimiting := enableRateLimitingInt.(bool)

			return !enableRateLimiting
		},
		Store: echoMiddleware.NewRateLimiterMemoryStoreWithConfig(storeCfg),
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
			context.Response().Header().Set(echo.HeaderRetryAfter, retryAfter)

			return context.JSON(http.StatusTooManyRequests, apierrors.NewAPIErrors("too many requests, try again later"))
		},
	}

	return echoMiddleware.RateLimiterWithConfig(rateLimiterConfig)
}
