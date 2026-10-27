package ratelimit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

const testSpec = `{
  "openapi": "3.0.0",
  "info": {"title": "test", "version": "1"},
  "paths": {
    "/limited": {
      "get": {
        "operationId": "limited",
        "x-enable-rate-limiting": true,
        "responses": {"200": {"description": "ok"}}
      }
    }
  }
}`

func TestDenyHandlerReturnsAPIErrors(t *testing.T) {
	spec, err := openapi3.NewLoader().LoadFromData([]byte(testSpec))
	require.NoError(t, err)

	logger := zerolog.Nop()
	config := &server.ServerConfig{Logger: &logger}
	config.Runtime.APIRateLimit = 1
	config.Runtime.APIRateLimitWindow = time.Minute

	e := echo.New()
	e.Use(NewRateLimitMiddleware(config, spec).Middleware())
	e.GET("/limited", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	var rec *httptest.ResponseRecorder

	// The burst is one request, so the second request in the window is denied.
	for range 2 {
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/limited", nil))
	}

	require.Equal(t, http.StatusTooManyRequests, rec.Code)

	var body gen.APIErrors
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Errors, 1)
	require.NotEmpty(t, body.Errors[0].Description)
}
