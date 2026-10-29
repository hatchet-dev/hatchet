package authz

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

type bearerTokenRepo struct {
	repository.APITokenRepository
	readOnly bool
}

func (r bearerTokenRepo) GetAPITokenById(_ context.Context, id uuid.UUID) (*sqlcv1.APIToken, error) {
	return &sqlcv1.APIToken{ID: id, ReadOnly: r.readOnly}, nil
}

type bearerRepo struct {
	repository.Repository
	tokens bearerTokenRepo
}

func (r bearerRepo) APIToken() repository.APITokenRepository { return r.tokens }

func TestBearerTokenPermissions(t *testing.T) {
	for _, readOnly := range []bool{false, true} {
		config := &server.ServerConfig{Layer: &database.Layer{V1: bearerRepo{tokens: bearerTokenRepo{readOnly: readOnly}}}}
		config.Auth.AllowedWriteOperations = []string{"WorkflowRunCreate"}
		auth, err := NewAuthZ(config)
		require.NoError(t, err)
		for _, op := range append([]string{"WorkflowRunList", "V1CelDebug", "WorkflowRunCreate", "V1TaskCancel", "EventCreate"}, restrictedWithBearerToken...) {
			t.Run(op+map[bool]string{true: "/read-only", false: "/unrestricted"}[readOnly], func(t *testing.T) {
				c := echo.New().NewContext(httptest.NewRequest("GET", "/", nil), httptest.NewRecorder())
				c.Set(string(analytics.APITokenIDKey), uuid.New())
				err := auth.handleBearerAuth(c, &middleware.RouteInfo{OperationID: op})
				banned := false
				for _, restricted := range restrictedWithBearerToken {
					banned = banned || op == restricted
				}
				if banned || (readOnly && op != "WorkflowRunList" && op != "V1CelDebug") {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
				require.True(t, CanViewPayloads(c))
			})
		}
	}
}
