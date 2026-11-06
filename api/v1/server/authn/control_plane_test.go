package authn

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

type controlPlaneTokenClient struct{}

func (controlPlaneTokenClient) ValidateExchangeToken(context.Context, string) (*uuid.UUID, *uuid.UUID, error) {
	return nil, nil, nil
}
func TestControlPlaneShardsRejectLocalLoginAndCookies(t *testing.T) {
	config := &server.ServerConfig{}
	config.Auth.ExchangeTokenClient = controlPlaneTokenClient{}
	auth := NewAuthN(config)
	spec, err := gen.GetSwagger()
	require.NoError(t, err)
	for _, path := range []string{"/api/v1/users/register", "/api/v1/users/login", "/api/v1/users/google/start", "/api/v1/users/google/callback", "/api/v1/users/github/start", "/api/v1/users/github/callback"} {
		t.Run(path, func(t *testing.T) {
			item := spec.Paths.Value(path)
			require.NotNil(t, item)
			for method, operation := range item.Operations() {
				c := echo.New().NewContext(httptest.NewRequest(method, path, nil), httptest.NewRecorder())
				route := &middleware.RouteInfo{Route: &routers.Route{Operation: operation}}
				err := auth.authenticate(c, route)
				var httpErr *echo.HTTPError
				require.ErrorAs(t, err, &httpErr)
				require.Equal(t, http.StatusForbidden, httpErr.Code)
			}
		})
	}

	c := echo.New().NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
	err = auth.handleCookieAuth(c)
	var httpErr *echo.HTTPError
	require.ErrorAs(t, err, &httpErr)
	require.Equal(t, http.StatusForbidden, httpErr.Code)
	require.False(t, isLocalLoginRoute(&middleware.RouteInfo{Route: &routers.Route{Operation: &openapi3.Operation{OperationID: "TenantGet"}}}))
}
