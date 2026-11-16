//go:build !authdisabled

package authn

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

func TestCookieAuthRejectedWhenExchangeTokenEnabled(t *testing.T) {
	logger := zerolog.Nop()
	a := &AuthN{
		l: &logger,
		config: &server.ServerConfig{
			Auth: server.AuthConfig{
				ConfigFile: server.ConfigFileAuth{
					ControlPlaneExchangeTokenConfig: server.ConfigFileAuthControlPlaneExchangeToken{
						Enabled: true,
					},
				},
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := echo.New().NewContext(req, httptest.NewRecorder())

	err := a.handleCookieAuth(c)

	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 without reading the session, got %v", err)
	}
}
