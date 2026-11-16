//go:build !authdisabled

package authn

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/pkg/auth/token"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type limitedJWTManager struct{ token.JWTManager }

func (limitedJWTManager) ValidateTenantToken(context.Context, string) (uuid.UUID, uuid.UUID, error) {
	return uuid.Nil, uuid.Nil, fmt.Errorf("%w: %w", token.ErrCouldNotReadTokenFromDatabase, &fairpool.LimitError{Key: "shared", Limit: 2})
}

type bearerOrCookie struct{ middleware.SecurityRequirement }

func (bearerOrCookie) IsOptional() bool { return false }
func (bearerOrCookie) NoAuth() bool     { return false }
func (bearerOrCookie) BearerAuth() bool { return true }
func (bearerOrCookie) CustomAuth() bool { return false }
func (bearerOrCookie) CookieAuth() bool { return true }

func TestBearerAuthKeepsLimitError(t *testing.T) {
	logger := zerolog.Nop()
	a := &AuthN{l: &logger, config: &server.ServerConfig{Auth: server.AuthConfig{JWTManager: limitedJWTManager{}}}}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer x")
	c := echo.New().NewContext(req, httptest.NewRecorder())
	c.Set("tenant", &sqlcv1.Tenant{})

	err := a.authenticate(c, &middleware.RouteInfo{Security: bearerOrCookie{}})

	var limitErr *fairpool.LimitError
	if !errors.As(err, &limitErr) {
		t.Fatalf("expected the fairpool rejection, got %v", err)
	}
}
