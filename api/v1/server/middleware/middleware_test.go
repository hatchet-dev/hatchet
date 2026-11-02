package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func TestRouteInfosCoverRegisteredRoutes(t *testing.T) {
	spec, err := gen.GetSwagger()
	require.NoError(t, err)

	routes, err := buildRouteInfos(spec)
	require.NoError(t, err)

	e := echo.New()
	gen.RegisterHandlers(e.Group(""), gen.NewStrictHandler(nil))

	for _, route := range e.Routes() {
		require.Contains(t, routes, routeKey(route.Method, route.Path))
	}
}

func TestMiddlewareUsesRoutedTemplate(t *testing.T) {
	spec, err := gen.GetSwagger()
	require.NoError(t, err)

	h, err := NewMiddlewareHandler(spec)
	require.NoError(t, err)

	var operations []string

	h.Use(func(r *RouteInfo) echo.HandlerFunc {
		return func(c echo.Context) error {
			operations = append(operations, r.OperationID)
			return nil
		}
	})

	mw, err := h.Middleware()
	require.NoError(t, err)

	e := echo.New()
	g := e.Group("")
	g.Use(mw)

	handler := func(c echo.Context) error { return c.NoContent(http.StatusOK) }
	g.GET("/api/v1/tenants/:tenant", handler)
	g.GET("/api/v1/tenants/:tenant/members", handler)

	serve := func(path string) int {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec.Code
	}

	require.Equal(t, http.StatusOK, serve("/api/v1/tenants/t%2Fmembers"))
	require.Equal(t, http.StatusOK, serve("/api/v1/tenants/t/members"))
	require.Equal(t, []string{"TenantGet", "TenantMemberList"}, operations)

	operations = nil
	require.Equal(t, http.StatusNotFound, serve("/api/v1/not-a-route"))
	require.Empty(t, operations)
}
