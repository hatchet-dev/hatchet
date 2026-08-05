package authz

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

func newBearerContext() echo.Context {
	e := echo.New()
	return e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
}

func TestHandleBearerAuthEnforcesBearerTokenRole(t *testing.T) {
	authorizer, err := newHatchetAuthorizer()
	require.NoError(t, err)

	a := &AuthZ{config: &server.ServerConfig{}, rbac: authorizer}

	denied := []string{
		"ApiTokenCreate",
		"ApiTokenList",
		"ApiTokenUpdateRevoke",
		"TenantCreate",
		"TenantInviteAccept",
		"TenantInviteCreate",
		"TenantInviteDelete",
		"TenantInviteList",
		"TenantInviteReject",
		"TenantInviteUpdate",
		"TenantMemberDelete",
		"TenantMemberList",
		"TenantMemberUpdate",
		"TenantMembershipsList",
		"UserGetCurrent",
		"UserListTenantInvites",
		"UserUpdateLogout",
		"UserUpdatePassword",
	}

	for _, operationId := range denied {
		t.Run("denied/"+operationId, func(t *testing.T) {
			err := a.handleBearerAuth(newBearerContext(), &middleware.RouteInfo{OperationID: operationId})

			var httpErr *echo.HTTPError

			require.ErrorAs(t, err, &httpErr)
			assert.Equal(t, http.StatusForbidden, httpErr.Code)
		})
	}

	allowed := []string{
		"TenantGet",
		"WorkflowRunCreate",
		"V1WorkflowRunCreate",
		"WorkerList",
		"EventCreate",
	}

	for _, operationId := range allowed {
		t.Run("allowed/"+operationId, func(t *testing.T) {
			assert.NoError(t, a.handleBearerAuth(newBearerContext(), &middleware.RouteInfo{OperationID: operationId}))
		})
	}
}

func TestHandleBearerAuthAllowedOperationsBypass(t *testing.T) {
	authorizer, err := newHatchetAuthorizer()
	require.NoError(t, err)

	config := &server.ServerConfig{}
	config.Auth.AllowedOperations = []string{"ApiTokenCreate"}

	a := &AuthZ{config: config, rbac: authorizer}

	assert.NoError(t, a.handleBearerAuth(newBearerContext(), &middleware.RouteInfo{OperationID: "ApiTokenCreate"}))
}
