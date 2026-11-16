package rate_limits

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type failingRateLimits struct {
	repository.RateLimitRepository
	err error
}

func (r failingRateLimits) DeleteRateLimits(context.Context, uuid.UUID, string) error { return r.err }

type rateLimitRepositoryStub struct {
	repository.Repository
	rateLimits repository.RateLimitRepository
}

func (r rateLimitRepositoryStub) RateLimit() repository.RateLimitRepository { return r.rateLimits }

func TestDeleteRateLimitPreservesStorageFailureForMiddleware(t *testing.T) {
	cause := errors.New("private database marker")
	svc := NewRateLimitService(&server.ServerConfig{
		Layer: &database.Layer{V1: rateLimitRepositoryStub{rateLimits: failingRateLimits{err: cause}}},
	})
	e := echo.New()
	out := httptest.NewRecorder()
	ctx := e.NewContext(httptest.NewRequest(http.MethodDelete, "/", nil), out)
	ctx.Set("tenant", &sqlcv1.Tenant{ID: uuid.New()})
	resp, err := svc.RateLimitDelete(ctx, gen.RateLimitDeleteRequestObject{})
	require.Nil(t, resp)
	require.ErrorIs(t, err, cause)
	e.HTTPErrorHandler(err, ctx)
	require.Equal(t, http.StatusInternalServerError, out.Code)
	require.NotContains(t, out.Body.String(), "private database marker")
}
