package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const taskStatsTestSpec = `
openapi: 3.0.0
info:
  title: test
  version: "1"
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
security:
  - bearerAuth: []
paths:
  /api/v1/tenants/{tenant}/task-stats:
    get:
      operationId: tenant:get:task-stats
      responses:
        "200":
          description: ok
  /api/v1/tenants/{tenant}/other:
    get:
      operationId: tenant:get:other
      responses:
        "200":
          description: ok
`

// The harness stands in for the populator, authn and authz: a token maps to the one tenant it
// may access, and anything else is rejected before the limiter runs.
type taskStatsHarness struct {
	e       *echo.Echo
	handled int
	tenantA uuid.UUID
	tenantB uuid.UUID
}

func newTaskStatsHarness(t *testing.T, limit rate.Limit, burst int) *taskStatsHarness {
	t.Helper()

	swagger, err := openapi3.NewLoader().LoadFromData([]byte(taskStatsTestSpec))
	require.NoError(t, err)

	h := &taskStatsHarness{e: echo.New(), tenantA: uuid.New(), tenantB: uuid.New()}
	tokenFor := map[string]uuid.UUID{"token-a": h.tenantA, "token-b": h.tenantB}

	chain, err := middleware.NewMiddlewareHandler(swagger)
	require.NoError(t, err)

	chain.Use(func(*middleware.RouteInfo) echo.HandlerFunc { // populator
		return func(c echo.Context) error {
			id, err := uuid.Parse(c.Param("tenant"))
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "bad tenant")
			}

			c.Set("tenant", &sqlcv1.Tenant{ID: id})

			return nil
		}
	})
	chain.Use(func(*middleware.RouteInfo) echo.HandlerFunc { // authn and authz
		return func(c echo.Context) error {
			tokenTenant, ok := tokenFor[c.Request().Header.Get("Authorization")]
			if !ok {
				return echo.NewHTTPError(http.StatusForbidden, "unauthenticated")
			}

			if tokenTenant != c.Get("tenant").(*sqlcv1.Tenant).ID {
				return echo.NewHTTPError(http.StatusUnauthorized, "not authorized for tenant")
			}

			return nil
		}
	})
	chain.Use(NewTaskStatsMiddleware(limit, burst))

	mw, err := chain.Middleware()
	require.NoError(t, err)

	handler := func(c echo.Context) error {
		h.handled++

		return c.NoContent(http.StatusOK)
	}
	h.e.GET("/api/v1/tenants/:tenant/task-stats", handler, mw)
	h.e.GET("/api/v1/tenants/:tenant/other", handler, mw)

	return h
}

func (h *taskStatsHarness) get(path string, tenant uuid.UUID, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/"+tenant.String()+path, nil)
	req.Header.Set("Authorization", token)
	rec := httptest.NewRecorder()
	h.e.ServeHTTP(rec, req)

	return rec
}

func TestTaskStatsLimitsPerTenant(t *testing.T) {
	h := newTaskStatsHarness(t, 0.01, 3)

	for i := range 3 {
		require.Equal(t, http.StatusOK, h.get("/task-stats", h.tenantA, "token-a").Code, "request %d", i+1)
	}

	denied := h.get("/task-stats", h.tenantA, "token-a")
	require.Equal(t, http.StatusTooManyRequests, denied.Code)
	require.Equal(t, "100", denied.Header().Get("Retry-After"))
	require.JSONEq(t, `{"errors":[{"description":"too many requests, try again later"}]}`, denied.Body.String())
	require.Equal(t, 3, h.handled, "a denied request must not reach the handler")

	require.Equal(t, http.StatusOK, h.get("/task-stats", h.tenantB, "token-b").Code, "other tenants have their own bucket")

	require.Equal(t, http.StatusOK, h.get("/other", h.tenantA, "token-a").Code, "other operations are not limited")
}

func TestTaskStatsOnlyCountsAuthorizedRequests(t *testing.T) {
	h := newTaskStatsHarness(t, 0.01, 2)

	for range 10 {
		require.Equal(t, http.StatusForbidden, h.get("/task-stats", h.tenantA, "no-such-token").Code)
		// a member of another tenant is authenticated but not authorized for tenant A
		require.Equal(t, http.StatusUnauthorized, h.get("/task-stats", h.tenantA, "token-b").Code)
	}

	for i := range 2 {
		require.Equal(t, http.StatusOK, h.get("/task-stats", h.tenantA, "token-a").Code, "request %d", i+1)
	}

	require.Equal(t, http.StatusTooManyRequests, h.get("/task-stats", h.tenantA, "token-a").Code)
}

func TestTaskStatsDisabledByZeroLimit(t *testing.T) {
	h := newTaskStatsHarness(t, 0, 10)

	for range 20 {
		require.Equal(t, http.StatusOK, h.get("/task-stats", h.tenantA, "token-a").Code)
	}
}

func TestTaskStatsZeroBurstStillLimits(t *testing.T) {
	h := newTaskStatsHarness(t, 0.01, 0)

	require.Equal(t, http.StatusOK, h.get("/task-stats", h.tenantA, "token-a").Code)
	require.Equal(t, http.StatusTooManyRequests, h.get("/task-stats", h.tenantA, "token-a").Code)
}
