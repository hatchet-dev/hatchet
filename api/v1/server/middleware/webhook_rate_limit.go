package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"golang.org/x/time/rate"
)

func WebhookRateLimitMiddleware(rateLimit rate.Limit, burst int, l *zerolog.Logger) echo.MiddlewareFunc {
	config := middleware.RateLimiterConfig{
		Skipper: func(c echo.Context) bool {
			method := c.Request().Method

			if method != http.MethodPost {
				return true
			}

			tenantId := c.Param("tenant")
			webhookName := c.Param("v1-webhook")

			if tenantId == "" || webhookName == "" {
				return true
			}

			return c.Request().URL.Path != fmt.Sprintf("/api/v1/stable/tenants/%s/webhooks/%s", tenantId, webhookName)
		},

		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate:      rateLimit,
				Burst:     burst,
				ExpiresIn: 10 * time.Minute,
			},
		),
	}

	return middleware.RateLimiterWithConfig(config)
}
