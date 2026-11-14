package metadata

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

func TestMetadataOmitsUserAuthSchemesWhenExchangeTokenEnabled(t *testing.T) {
	svc := NewMetadataService(metadataConfig(true))

	res, err := svc.MetadataGet(echoContext(t), gen.MetadataGetRequestObject{})
	require.NoError(t, err)

	meta := gen.APIMeta(res.(gen.MetadataGet200JSONResponse))
	require.NotNil(t, meta.Auth)
	require.NotNil(t, meta.Auth.Schemes)
	require.Empty(t, *meta.Auth.Schemes)
}

func TestMetadataKeepsUserAuthSchemesWithoutExchangeToken(t *testing.T) {
	svc := NewMetadataService(metadataConfig(false))

	res, err := svc.MetadataGet(echoContext(t), gen.MetadataGetRequestObject{})
	require.NoError(t, err)

	meta := gen.APIMeta(res.(gen.MetadataGet200JSONResponse))
	require.NotNil(t, meta.Auth)
	require.NotNil(t, meta.Auth.Schemes)
	require.ElementsMatch(t, []string{"basic", "google", "github"}, *meta.Auth.Schemes)
}

func metadataConfig(exchangeTokenEnabled bool) *server.ServerConfig {
	return &server.ServerConfig{
		Pylon: &server.PylonConfig{},
		Auth: server.AuthConfig{
			ConfigFile: server.ConfigFileAuth{
				BasicAuthEnabled: true,
				Google:           server.ConfigFileAuthGoogle{Enabled: true},
				Github:           server.ConfigFileAuthGithub{Enabled: true},
				ControlPlaneExchangeTokenConfig: server.ConfigFileAuthControlPlaneExchangeToken{
					Enabled: exchangeTokenEnabled,
				},
			},
		},
	}
}

func echoContext(t *testing.T) echo.Context {
	t.Helper()
	return echo.New().NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
}
