package authz

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/auth/rbac"
)

func TestBearerPolicyValidateSpec(t *testing.T) {
	_, err := newBearerPolicy()
	assert.Nil(t, err)
}

func TestBearerPolicyRejectsUnclassifiedOperation(t *testing.T) {
	policy, err := rbac.LoadBearerPolicy([]byte("operations:\n  denied: []\n  allowed: []\n"))
	require.NoError(t, err)

	spec, err := gen.GetSwagger()
	require.NoError(t, err)

	err = policy.ValidateSpec(*spec)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "exists in openapi specs but not in bearer.yaml")
}

func TestBearerPolicyRejectsUnknownOperation(t *testing.T) {
	policy, err := rbac.LoadBearerPolicy([]byte("operations:\n  denied:\n    - NotARealOperation\n  allowed: []\n"))
	require.NoError(t, err)

	spec, err := gen.GetSwagger()
	require.NoError(t, err)

	err = policy.ValidateSpec(*spec)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "NotARealOperation exists in bearer.yaml but not in specs")
}

func TestBearerPolicyRejectsDuplicateOperation(t *testing.T) {
	policy, err := rbac.LoadBearerPolicy([]byte("operations:\n  denied:\n    - TenantGet\n  allowed:\n    - TenantGet\n"))
	require.NoError(t, err)

	spec, err := gen.GetSwagger()
	require.NoError(t, err)

	err = policy.ValidateSpec(*spec)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "TenantGet is listed more than once in bearer.yaml")
}

func newBearerContext() echo.Context {
	e := echo.New()
	return e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
}

func TestHandleBearerAuthEnforcesPolicy(t *testing.T) {
	policy, err := newBearerPolicy()
	require.NoError(t, err)

	a := &AuthZ{bearer: policy}

	require.NotEmpty(t, policy.Operations.Denied)
	require.NotEmpty(t, policy.Operations.Allowed)

	for _, operationId := range policy.Operations.Denied {
		t.Run("denied/"+operationId, func(t *testing.T) {
			err := a.handleBearerAuth(newBearerContext(), &middleware.RouteInfo{OperationID: operationId})

			var httpErr *echo.HTTPError

			require.ErrorAs(t, err, &httpErr)
			assert.Equal(t, http.StatusForbidden, httpErr.Code)
		})
	}

	for _, operationId := range policy.Operations.Allowed {
		t.Run("allowed/"+operationId, func(t *testing.T) {
			assert.NoError(t, a.handleBearerAuth(newBearerContext(), &middleware.RouteInfo{OperationID: operationId}))
		})
	}
}
