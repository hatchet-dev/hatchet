package serverlessv1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// entitlementStub answers the serverless operator entitlement check with a fixed value.
type entitlementStub struct {
	repository.TenantEntitlementRepository
	enabled bool
}

func (s entitlementStub) IsServerlessOperatorEnabled(context.Context, uuid.UUID) (bool, error) {
	return s.enabled, nil
}

// repositoryStub exposes only the entitlement repository; the handler must not reach any
// other repository before the entitlement check.
type repositoryStub struct {
	repository.Repository
	entitlement repository.TenantEntitlementRepository
}

func (s repositoryStub) TenantEntitlement() repository.TenantEntitlementRepository {
	return s.entitlement
}

func newCreateContext(t *testing.T) echo.Context {
	t.Helper()

	ctx := echo.New().NewContext(httptest.NewRequest(http.MethodPost, "/", nil), httptest.NewRecorder())
	ctx.Set("tenant", &sqlcv1.Tenant{ID: uuid.New()})

	return ctx
}

func newCreateService(enabled bool) *V1ServerlessService {
	return NewV1ServerlessService(&server.ServerConfig{
		Layer: &database.Layer{V1: repositoryStub{entitlement: entitlementStub{enabled: enabled}}},
	})
}

var createBody = gen.V1CreateServerlessEndpointRequest{
	Name:           "endpoint",
	HealthcheckUrl: "https://example.com/health",
	TriggerUrl:     "https://example.com/trigger",
	SigningSecret:  "0123456789abcdef0123456789abcdef",
}

// A tenant without the serverless operator entitlement is refused with 403 before the request
// is validated or anything is written.
func TestCreateRefusesATenantWithoutTheEntitlement(t *testing.T) {
	body := createBody

	resp, err := newCreateService(false).V1ServerlessEndpointCreate(newCreateContext(t), gen.V1ServerlessEndpointCreateRequestObject{Body: &body})
	require.NoError(t, err)

	forbidden, ok := resp.(gen.V1ServerlessEndpointCreate403JSONResponse)
	require.True(t, ok, "expected a 403 response, got %T", resp)
	require.Len(t, forbidden.Errors, 1)
	assert.Equal(t, serverlessNotEntitledMessage, forbidden.Errors[0].Description)
}

// An entitled tenant passes the gate: the request goes on to validation, which is what
// answers here since the trigger URL is not https.
func TestCreateAdmitsAnEntitledTenant(t *testing.T) {
	body := createBody
	body.TriggerUrl = "http://example.com/trigger"

	resp, err := newCreateService(true).V1ServerlessEndpointCreate(newCreateContext(t), gen.V1ServerlessEndpointCreateRequestObject{Body: &body})
	require.NoError(t, err)

	_, ok := resp.(gen.V1ServerlessEndpointCreate400JSONResponse)
	require.True(t, ok, "expected a 400 response from validation, got %T", resp)
}
