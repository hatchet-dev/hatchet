package users

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware/redirect"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

func exchangeTokenConfig() *server.ServerConfig {
	logger := zerolog.Nop()

	return &server.ServerConfig{
		Logger: &logger,
		Auth: server.AuthConfig{
			ConfigFile: server.ConfigFileAuth{
				BasicAuthEnabled: true,
				Google:           server.ConfigFileAuthGoogle{Enabled: true},
				Github:           server.ConfigFileAuthGithub{Enabled: true},
				ControlPlaneExchangeTokenConfig: server.ConfigFileAuthControlPlaneExchangeToken{
					Enabled: true,
				},
			},
		},
	}
}

func TestUserUpdateLoginRejectedWhenExchangeTokenEnabled(t *testing.T) {
	svc := NewUserService(exchangeTokenConfig())
	c := echo.New().NewContext(httptest.NewRequest(http.MethodPost, "/", nil), httptest.NewRecorder())

	res, err := svc.UserUpdateLogin(c, gen.UserUpdateLoginRequestObject{})
	require.NoError(t, err)
	_, ok := res.(gen.UserUpdateLogin405JSONResponse)
	require.True(t, ok)
}

func TestUserCreateRejectedWhenExchangeTokenEnabled(t *testing.T) {
	svc := NewUserService(exchangeTokenConfig())
	c := echo.New().NewContext(httptest.NewRequest(http.MethodPost, "/", nil), httptest.NewRecorder())

	res, err := svc.UserCreate(c, gen.UserCreateRequestObject{})
	require.NoError(t, err)
	_, ok := res.(gen.UserCreate405JSONResponse)
	require.True(t, ok)
}

func TestGoogleOauthStartRejectedWhenExchangeTokenEnabled(t *testing.T) {
	svc := NewUserService(exchangeTokenConfig())
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)

	_, err := svc.UserUpdateGoogleOauthStart(c, gen.UserUpdateGoogleOauthStartRequestObject{})
	require.ErrorIs(t, err, redirect.ErrRedirect)
	require.Contains(t, rec.Header().Get("Location"), "User signup is disabled.")
}
