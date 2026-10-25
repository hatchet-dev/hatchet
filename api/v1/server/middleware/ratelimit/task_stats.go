package ratelimit

import (
	"math"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const taskStatsOperationID = "tenant:get:task-stats"

// NewTaskStatsMiddleware limits the task-stats endpoint per tenant. Autoscalers poll it, so it
// is kept off the per-IP limiter. It is a route-aware middleware so it can run after authz,
// where only callers authorized for a tenant can spend that tenant's bucket. A limit of zero or
// less disables it.
func NewTaskStatsMiddleware(limit rate.Limit, burst int) middleware.MiddlewareFunc {
	if limit <= 0 {
		return func(*middleware.RouteInfo) echo.HandlerFunc {
			return func(echo.Context) error { return nil }
		}
	}

	burst = max(burst, 1)

	store := echoMiddleware.NewRateLimiterMemoryStoreWithConfig(echoMiddleware.RateLimiterMemoryStoreConfig{
		Rate:  limit,
		Burst: burst,
		// An idle bucket is only dropped once it has fully refilled, so expiry loses no state.
		ExpiresIn: max(time.Minute, time.Duration(float64(burst)/float64(limit)*float64(time.Second))),
	})

	// time until one more token is available
	retryAfter := strconv.Itoa(int(math.Ceil(1 / float64(limit))))

	return func(r *middleware.RouteInfo) echo.HandlerFunc {
		return func(c echo.Context) error {
			if r.OperationID != taskStatsOperationID || r.Security.NoAuth() || r.Security.IsOptional() {
				return nil
			}

			tenant, ok := c.Get("tenant").(*sqlcv1.Tenant)
			if !ok || tenant == nil {
				return nil
			}

			// the in-memory store never returns an error
			if allowed, _ := store.Allow(tenant.ID.String()); allowed {
				return nil
			}

			return tooManyRequests(c, retryAfter)
		}
	}
}
