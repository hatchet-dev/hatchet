package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newMetaServer serves /api/v1/meta with the given embedded flag.
func newMetaServer(t *testing.T, embedded bool) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/meta" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"embedded": %v}`, embedded)
	}))
}

// fakeJWT builds an unsigned JWT with the claims the CLI reads.
func fakeJWT(t *testing.T, serverURL, grpcAddress, tenantID string) string {
	t.Helper()

	claims := map[string]any{
		"server_url":             serverURL,
		"grpc_broadcast_address": grpcAddress,
		"sub":                    tenantID,
		"exp":                    time.Now().Add(time.Hour).Unix(),
	}
	payload, err := json.Marshal(claims)
	require.NoError(t, err)

	encode := base64.RawURLEncoding.EncodeToString

	return encode([]byte(`{"alg":"none"}`)) + "." + encode(payload) + ".sig"
}

func env(vars map[string]string) func(string) string {
	return func(key string) string { return vars[key] }
}

func TestDetectEmbeddedViaHandshakeEnv(t *testing.T) {
	server := newMetaServer(t, true)
	defer server.Close()

	handshake, err := json.Marshal(embeddedHandshake{
		Token:       "token-123",
		TenantID:    "tenant-123",
		GRPCAddress: "127.0.0.1:54321",
		APIURL:      server.URL,
	})
	require.NoError(t, err)

	detection := detectEmbedded(context.Background(), "http://127.0.0.1:1", env(map[string]string{
		handshakeEnv: string(handshake),
	}))

	require.True(t, detection.Usable())
	assert.Equal(t, server.URL, detection.APIURL)
	assert.Equal(t, "token-123", detection.Profile.Token)
	assert.Equal(t, "tenant-123", detection.Profile.TenantId)
	assert.Equal(t, "127.0.0.1:54321", detection.Profile.GrpcHostPort)
	assert.Equal(t, "none", detection.Profile.TLSStrategy)
	assert.Equal(t, EmbeddedProfileName, detection.Profile.Name)
}

func TestDetectEmbeddedViaClientTokenEnv(t *testing.T) {
	server := newMetaServer(t, true)
	defer server.Close()

	token := fakeJWT(t, server.URL, "127.0.0.1:54321", "tenant-jwt")

	detection := detectEmbedded(context.Background(), "http://127.0.0.1:1", env(map[string]string{
		"HATCHET_CLIENT_TOKEN": token,
	}))

	require.True(t, detection.Usable())
	assert.Equal(t, token, detection.Profile.Token)
	assert.Equal(t, "tenant-jwt", detection.Profile.TenantId)
	assert.Equal(t, "127.0.0.1:54321", detection.Profile.GrpcHostPort)
}

func TestDetectEmbeddedTokenForNonEmbeddedServerIsIgnored(t *testing.T) {
	// A cloud/self-hosted token in the environment must not create an implicit
	// grant: the embedded carve-out requires the API to report embedded: true.
	server := newMetaServer(t, false)
	defer server.Close()

	token := fakeJWT(t, server.URL, "127.0.0.1:54321", "tenant-cloud")

	detection := detectEmbedded(context.Background(), "http://127.0.0.1:1", env(map[string]string{
		"HATCHET_CLIENT_TOKEN": token,
	}))

	assert.False(t, detection.Detected)
	assert.False(t, detection.Usable())
}

func TestDetectEmbeddedDefaultPortProbeWithoutToken(t *testing.T) {
	server := newMetaServer(t, true)
	defer server.Close()

	detection := detectEmbedded(context.Background(), server.URL, env(nil))

	assert.True(t, detection.Detected)
	assert.False(t, detection.Usable(), "port probe alone yields no token")
	assert.Contains(t, detection.Note, "HATCHET_CLIENT_TOKEN")
}

func TestDetectEmbeddedNothingRunning(t *testing.T) {
	detection := detectEmbedded(context.Background(), "http://127.0.0.1:1", env(nil))

	assert.False(t, detection.Detected)
	assert.False(t, detection.Usable())
}
