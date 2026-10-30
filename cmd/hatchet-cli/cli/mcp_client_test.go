package cli

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/mcp"
	profileconfig "github.com/hatchet-dev/hatchet/pkg/config/cli"
)

// hatchetClientEnvVars are the environment variables the SDK's config loader
// binds for connection settings. The MCP path must never read any of them.
var hatchetClientEnvVars = []string{
	"HATCHET_CLIENT_TOKEN",
	"HATCHET_CLIENT_TENANT_ID",
	"HATCHET_CLIENT_HOST_PORT",
	"HATCHET_CLIENT_SERVER_URL",
	"HATCHET_CLIENT_NAMESPACE",
	"HATCHET_CLIENT_TLS_STRATEGY",
	"HATCHET_CLIENT_TLS_ROOT_CA_FILE",
	"HATCHET_CLIENT_TLS_CERT_FILE",
	"HATCHET_CLIENT_TLS_KEY_FILE",
}

func clearHatchetClientEnv(t *testing.T) {
	t.Helper()
	for _, key := range hatchetClientEnvVars {
		t.Setenv(key, "")
	}
}

// syntheticProfileJWT builds an unsigned JWT with the claims the client
// config loader reads. No real credential material is involved.
func syntheticProfileJWT(t *testing.T, tenantID, serverURL, grpcAddr string) string {
	t.Helper()

	claims, err := json.Marshal(map[string]any{
		"sub":                    tenantID,
		"exp":                    time.Now().Add(time.Hour).Unix(),
		"server_url":             serverURL,
		"grpc_broadcast_address": grpcAddr,
	})
	require.NoError(t, err)

	return "e30." + base64.RawURLEncoding.EncodeToString(claims) + ".synthetic"
}

// recordingAPIServer is a loopback HTTP fixture that records every request's
// Authorization header and path and answers any GET with an empty row list.
type recordingAPIServer struct {
	srv *httptest.Server

	mu    sync.Mutex
	auths []string
	paths []string
}

func newRecordingAPIServer(t *testing.T) *recordingAPIServer {
	t.Helper()

	rec := &recordingAPIServer{}
	rec.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.mu.Lock()
		rec.auths = append(rec.auths, r.Header.Get("Authorization"))
		rec.paths = append(rec.paths, r.URL.Path)
		rec.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"rows":[]}`)
	}))
	t.Cleanup(rec.srv.Close)

	return rec
}

func (r *recordingAPIServer) requestCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.auths)
}

func (r *recordingAPIServer) lastRequest(t *testing.T) (auth, path string) {
	t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()
	require.NotEmpty(t, r.auths, "the profile's endpoint must receive the request")
	return r.auths[len(r.auths)-1], r.paths[len(r.paths)-1]
}

// TestMCPEngineFactoryIgnoresEnvironment is the regression test for the MCP
// grant boundary: once a profile is authorized, HATCHET_CLIENT_* environment
// variables must not redirect its credentials, tenant, or endpoints (each
// override is exercised individually and together).
func TestMCPEngineFactoryIgnoresEnvironment(t *testing.T) {
	const grantedTenant = "11111111-1111-4111-8111-111111111111"
	const envTenant = "22222222-2222-4222-8222-222222222222"

	cases := []struct {
		name string
		env  func(t *testing.T, envServerURL, envToken string)
	}{
		{
			name: "token override",
			env: func(t *testing.T, _, envToken string) {
				t.Setenv("HATCHET_CLIENT_TOKEN", envToken)
			},
		},
		{
			name: "tenant id override",
			env: func(t *testing.T, _, _ string) {
				t.Setenv("HATCHET_CLIENT_TENANT_ID", envTenant)
			},
		},
		{
			name: "server url override",
			env: func(t *testing.T, envServerURL, _ string) {
				t.Setenv("HATCHET_CLIENT_SERVER_URL", envServerURL)
			},
		},
		{
			name: "host port override",
			env: func(t *testing.T, _, _ string) {
				t.Setenv("HATCHET_CLIENT_HOST_PORT", "127.0.0.1:2")
			},
		},
		{
			name: "all overrides together",
			env: func(t *testing.T, envServerURL, envToken string) {
				t.Setenv("HATCHET_CLIENT_TOKEN", envToken)
				t.Setenv("HATCHET_CLIENT_TENANT_ID", envTenant)
				t.Setenv("HATCHET_CLIENT_SERVER_URL", envServerURL)
				t.Setenv("HATCHET_CLIENT_HOST_PORT", "127.0.0.1:2")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clearHatchetClientEnv(t)

			profileSrv := newRecordingAPIServer(t)
			envSrv := newRecordingAPIServer(t)

			profileToken := syntheticProfileJWT(t, grantedTenant, profileSrv.srv.URL, "127.0.0.1:1")
			envToken := syntheticProfileJWT(t, envTenant, envSrv.srv.URL, "127.0.0.1:2")
			tc.env(t, envSrv.srv.URL, envToken)

			profile := &profileconfig.Profile{
				Name:         "allowed",
				TenantId:     grantedTenant,
				Token:        profileToken,
				ApiServerURL: profileSrv.srv.URL,
				GrpcHostPort: "127.0.0.1:1",
				TLSStrategy:  "none",
			}

			engine, err := mcpEngineFactory(profile)
			require.NoError(t, err)
			require.Equal(t, grantedTenant, engine.TenantID(), "the client tenant must be the granted profile's tenant")

			_, err = engine.ListWorkers(context.Background())
			require.NoError(t, err)

			auth, path := profileSrv.lastRequest(t)
			assert.Equal(t, "Bearer "+profileToken, auth, "the request must carry the profile token, not the environment token")
			assert.Contains(t, path, grantedTenant, "the request must target the granted profile's tenant")
			assert.Zero(t, envSrv.requestCount(), "the environment-configured endpoint must never be contacted")
		})
	}
}

// TestMCPEngineFactoryMalformedToken is the regression test for the MCP
// stdio server's resilience: a granted profile with a malformed stored token
// must produce a tool error, not a panic that kills the whole server, and the
// error must not echo the stored token.
func TestMCPEngineFactoryMalformedToken(t *testing.T) {
	clearHatchetClientEnv(t)

	profile := &profileconfig.Profile{
		Name:         "allowed",
		TenantId:     "11111111-1111-4111-8111-111111111111",
		Token:        "synthetic-invalid-token",
		ApiServerURL: "http://127.0.0.1:1",
		GrpcHostPort: "127.0.0.1:1",
		TLSStrategy:  "none",
	}

	var engine mcp.Engine
	var err error
	require.NotPanics(t, func() {
		engine, err = mcpEngineFactory(profile)
	}, "a malformed stored token must not panic the MCP server")
	require.Error(t, err)
	assert.Nil(t, engine)
	assert.NotContains(t, err.Error(), "synthetic-invalid-token", "the error must not echo the stored token")
}
