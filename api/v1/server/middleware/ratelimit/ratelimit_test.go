package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

func TestStoreConfigAllowsLimitPerWindow(t *testing.T) {
	cfg := storeConfig(10, 5*time.Minute)
	// echo's memory store keeps one limiter like this per identifier
	limiter := rate.NewLimiter(cfg.Rate, cfg.Burst)
	now := time.Now()

	for i := range 10 {
		require.True(t, limiter.AllowN(now, 1), "request %d should be allowed", i+1)
	}

	require.False(t, limiter.AllowN(now, 1), "request beyond the limit should be denied")
	require.False(t, limiter.AllowN(now.Add(29*time.Second), 1), "no refill before window/limit")
	require.True(t, limiter.AllowN(now.Add(31*time.Second), 1), "one request refills every window/limit")
}

func TestStoreConfigClampsInvalidConfig(t *testing.T) {
	cfg := storeConfig(0, 0)

	require.Equal(t, 1, cfg.Burst)
	require.Equal(t, rate.Limit(1), cfg.Rate)
}

const testSpec = `
openapi: 3.0.0
info:
  title: test
  version: "1"
paths:
  /limited:
    post:
      operationId: limited:op
      x-enable-rate-limiting: true
      responses:
        "200":
          description: ok
  /other-limited:
    post:
      operationId: other-limited:op
      x-enable-rate-limiting: true
      responses:
        "200":
          description: ok
  /unlimited:
    post:
      operationId: unlimited:op
      responses:
        "200":
          description: ok
`

func TestMiddlewareEnforcesLimitPerOperation(t *testing.T) {
	swagger, err := openapi3.NewLoader().LoadFromData([]byte(testSpec))
	require.NoError(t, err)

	logger := zerolog.Nop()
	cfg := &server.ServerConfig{Logger: &logger}
	cfg.Runtime.APIRateLimit = 3
	cfg.Runtime.APIRateLimitWindow = 5 * time.Minute

	mw := NewRateLimitMiddleware(cfg, swagger).Middleware()
	require.NotNil(t, mw)

	e := echo.New()
	handler := mw(func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	serve := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		rec := httptest.NewRecorder()
		require.NoError(t, handler(e.NewContext(req, rec)))

		return rec
	}

	do := func(path string) int {
		return serve(path).Code
	}

	for range 3 {
		require.Equal(t, http.StatusOK, do("/limited"))
	}

	denied := serve("/limited")
	require.Equal(t, http.StatusTooManyRequests, denied.Code)
	require.Equal(t, "100", denied.Header().Get("Retry-After"), "one request refills every window/limit")
	require.JSONEq(t, `{"errors":[{"description":"too many requests, try again later"}]}`, denied.Body.String())

	// a bucket that refilled at the limit per second would allow this request
	time.Sleep(1100 * time.Millisecond)
	require.Equal(t, http.StatusTooManyRequests, do("/limited"), "bucket must not refill at the limit per second")

	require.Equal(t, http.StatusOK, do("/other-limited"), "operations have separate buckets")

	for range 10 {
		require.Equal(t, http.StatusOK, do("/unlimited"))
	}
}
