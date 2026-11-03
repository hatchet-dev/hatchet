package middleware

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func TestNoAuthBodyLimit(t *testing.T) {
	spec, err := gen.GetSwagger()
	require.NoError(t, err)

	routes, err := buildRouteInfos(spec)
	require.NoError(t, err)

	webhook := routes[routeKey(http.MethodPost, "/api/v1/stable/tenants/:tenant/webhooks/:v1-webhook")]
	require.NotNil(t, webhook)
	require.Equal(t, "V1WebhookReceive", webhook.OperationID)

	noAuth := &RouteInfo{Security: &securityRequirement{}, Route: &routers.Route{Operation: &openapi3.Operation{}}}
	bearer := &RouteInfo{Security: &securityRequirement{requirements: []openapi3.SecurityRequirement{{"bearerAuth": {}}}}}

	run := func(r *RouteInfo, size int, contentLength int64) (mwErr, readErr error) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("a", size)))
		req.ContentLength = contentLength
		c := echo.New().NewContext(req, httptest.NewRecorder())

		if mwErr = NoAuthBodyLimit(r)(c); mwErr != nil {
			return mwErr, nil
		}

		_, readErr = io.ReadAll(c.Request().Body)

		return nil, readErr
	}

	mwErr, _ := run(noAuth, 2, noAuthBodyLimit+1)
	var httpErr *echo.HTTPError
	require.ErrorAs(t, mwErr, &httpErr)
	require.Equal(t, http.StatusRequestEntityTooLarge, httpErr.Code)

	_, readErr := run(noAuth, noAuthBodyLimit+1, -1)
	var maxBytesErr *http.MaxBytesError
	require.True(t, errors.As(readErr, &maxBytesErr))

	_, readErr = run(webhook, 25<<20, -1)
	require.NoError(t, readErr)

	_, readErr = run(webhook, 25<<20+1, -1)
	require.True(t, errors.As(readErr, &maxBytesErr))

	_, readErr = run(bearer, 26<<20, -1)
	require.NoError(t, readErr)
}
