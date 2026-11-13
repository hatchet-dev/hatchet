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

// Operations that declare security: [] and must stay reachable when this shard
// only accepts control-plane exchange tokens. Every other explicit security: []
// operation is a direct user login and is rejected below.
var publicOpsKeptWithExchangeTokens = map[string]struct{}{
	"ReadinessGet":     {},
	"LivenessGet":      {},
	"MetadataGet":      {},
	"CloudMetadataGet": {},
	"InfoGetVersion":   {},
	"SnsUpdate":        {},
	"V1WebhookReceive": {},
}

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

func TestSpecRefusesDirectLoginWhenExchangeTokenEnabled(t *testing.T) {
	spec, err := gen.GetSwagger()
	require.NoError(t, err)

	svc := NewUserService(exchangeTokenConfig())
	rejected := 0
	foundPublic := map[string]struct{}{}

	for path, item := range spec.Paths.Map() {
		for method, op := range item.Operations() {
			if op.Security == nil || len(*op.Security) != 0 {
				continue
			}

			foundPublic[op.OperationID] = struct{}{}

			if _, keep := publicOpsKeptWithExchangeTokens[op.OperationID]; keep {
				continue
			}

			rejected++
			t.Run(method+" "+path, func(t *testing.T) {
				assertDirectLoginRejected(t, svc, op.OperationID)
			})
		}
	}

	require.NotZero(t, rejected)

	for id := range publicOpsKeptWithExchangeTokens {
		_, ok := foundPublic[id]
		require.Truef(t, ok, "allowlisted public operation %s is not security: []", id)
	}
}

func assertDirectLoginRejected(t *testing.T, svc *UserService, operationID string) {
	t.Helper()

	rec := httptest.NewRecorder()
	// state and code are present so a missing guard reaches the session store or
	// the OAuth provider instead of returning an earlier validation error.
	req := httptest.NewRequest(http.MethodGet, "/?state=state&code=code", nil)
	c := echo.New().NewContext(req, rec)

	switch operationID {
	case "UserUpdateLogin":
		res, err := svc.UserUpdateLogin(c, gen.UserUpdateLoginRequestObject{})
		require.NoError(t, err)
		_, ok := res.(gen.UserUpdateLogin405JSONResponse)
		require.True(t, ok)
	case "UserCreate":
		res, err := svc.UserCreate(c, gen.UserCreateRequestObject{})
		require.NoError(t, err)
		_, ok := res.(gen.UserCreate405JSONResponse)
		require.True(t, ok)
	case "UserUpdateGoogleOauthStart":
		_, err := svc.UserUpdateGoogleOauthStart(c, gen.UserUpdateGoogleOauthStartRequestObject{})
		requireSignupDisabledRedirect(t, err, rec)
	case "UserUpdateGoogleOauthCallback":
		_, err := svc.UserUpdateGoogleOauthCallback(c, gen.UserUpdateGoogleOauthCallbackRequestObject{})
		requireSignupDisabledRedirect(t, err, rec)
	case "UserUpdateGithubOauthStart":
		_, err := svc.UserUpdateGithubOauthStart(c, gen.UserUpdateGithubOauthStartRequestObject{})
		requireSignupDisabledRedirect(t, err, rec)
	case "UserUpdateGithubOauthCallback":
		_, err := svc.UserUpdateGithubOauthCallback(c, gen.UserUpdateGithubOauthCallbackRequestObject{})
		requireSignupDisabledRedirect(t, err, rec)
	default:
		t.Fatalf("public operation %s is not rejected when exchange tokens are enabled", operationID)
	}
}

func requireSignupDisabledRedirect(t *testing.T, err error, rec *httptest.ResponseRecorder) {
	t.Helper()
	require.ErrorIs(t, err, redirect.ErrRedirect)
	require.Contains(t, rec.Header().Get("Location"), "User signup is disabled.")
}
