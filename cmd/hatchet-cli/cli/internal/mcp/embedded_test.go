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

	cliconfig "github.com/hatchet-dev/hatchet/pkg/config/cli"
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

// storedProfiles builds a ProfileSource over the given profile map, mirroring
// what the CLI profile store returns.
func storedProfiles(profiles map[string]cliconfig.Profile) ProfileSource {
	return ProfileSource{
		Profiles:       func() map[string]cliconfig.Profile { return profiles },
		DefaultProfile: func() string { return "" },
	}
}

// registeredEmbedded is the profile the embedded engine registers on ready.
func registeredEmbedded(apiURL string) cliconfig.Profile {
	return cliconfig.Profile{
		TenantId:     "tenant-registered",
		Name:         EmbeddedProfileName,
		Token:        "token-registered",
		ApiServerURL: apiURL,
		GrpcHostPort: "127.0.0.1:50051",
		TLSStrategy:  "none",
	}
}

func TestDetectEmbeddedViaProfileRegistration(t *testing.T) {
	server := newMetaServer(t, true)
	defer server.Close()

	source := storedProfiles(map[string]cliconfig.Profile{
		EmbeddedProfileName: registeredEmbedded(server.URL),
	})

	detection := detectEmbedded(context.Background(), "http://127.0.0.1:1", env(nil), source)

	require.True(t, detection.Usable())
	assert.Equal(t, EmbeddedSourceProfile, detection.Source)
	assert.Equal(t, server.URL, detection.APIURL)
	assert.Equal(t, "token-registered", detection.Profile.Token)
	assert.Equal(t, "tenant-registered", detection.Profile.TenantId)
	assert.Equal(t, "127.0.0.1:50051", detection.Profile.GrpcHostPort)
	assert.Empty(t, detection.Note)
}

func TestDetectEmbeddedProfileTakesPrecedenceOverEnv(t *testing.T) {
	server := newMetaServer(t, true)
	defer server.Close()

	handshake, err := json.Marshal(embeddedHandshake{
		Token:       "token-env",
		TenantID:    "tenant-env",
		GRPCAddress: "127.0.0.1:54321",
		APIURL:      server.URL,
	})
	require.NoError(t, err)

	source := storedProfiles(map[string]cliconfig.Profile{
		EmbeddedProfileName: registeredEmbedded(server.URL),
	})

	detection := detectEmbedded(context.Background(), "http://127.0.0.1:1", env(map[string]string{
		handshakeEnv: string(handshake),
	}), source)

	require.True(t, detection.Usable())
	assert.Equal(t, EmbeddedSourceProfile, detection.Source)
	assert.Equal(t, "token-registered", detection.Profile.Token, "the profile registration wins over env detection")
}

func TestDetectEmbeddedStaleProfileRegistrationIsSkipped(t *testing.T) {
	// A registration left behind by a crashed engine: nothing answers at its
	// API URL, so it is treated as absent, with a note, and never deleted.
	source := storedProfiles(map[string]cliconfig.Profile{
		EmbeddedProfileName: registeredEmbedded("http://127.0.0.1:1"),
	})

	detection := detectEmbedded(context.Background(), "http://127.0.0.1:1", env(nil), source)

	assert.False(t, detection.Detected)
	assert.False(t, detection.Usable())
	assert.Contains(t, detection.Note, "stale embedded registration")
}

func TestDetectEmbeddedStaleProfileFallsBackToHandshakeEnv(t *testing.T) {
	server := newMetaServer(t, true)
	defer server.Close()

	handshake, err := json.Marshal(embeddedHandshake{
		Token:       "token-env",
		TenantID:    "tenant-env",
		GRPCAddress: "127.0.0.1:54321",
		APIURL:      server.URL,
	})
	require.NoError(t, err)

	source := storedProfiles(map[string]cliconfig.Profile{
		EmbeddedProfileName: registeredEmbedded("http://127.0.0.1:1"),
	})

	detection := detectEmbedded(context.Background(), "http://127.0.0.1:1", env(map[string]string{
		handshakeEnv: string(handshake),
	}), source)

	require.True(t, detection.Usable())
	assert.Equal(t, EmbeddedSourceHandshakeEnv, detection.Source)
	assert.Equal(t, "token-env", detection.Profile.Token)
	assert.Contains(t, detection.Note, "stale embedded registration", "the skipped registration is still surfaced")
}

func TestDetectEmbeddedProfileForNonEmbeddedServerIsSkipped(t *testing.T) {
	// A hand-made "embedded" profile pointing at a non-embedded deployment
	// must not pass the implicit-grant gate.
	server := newMetaServer(t, false)
	defer server.Close()

	source := storedProfiles(map[string]cliconfig.Profile{
		EmbeddedProfileName: registeredEmbedded(server.URL),
	})

	detection := detectEmbedded(context.Background(), "http://127.0.0.1:1", env(nil), source)

	assert.False(t, detection.Detected)
	assert.Contains(t, detection.Note, "stale embedded registration")
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
	}), ProfileSource{})

	require.True(t, detection.Usable())
	assert.Equal(t, EmbeddedSourceHandshakeEnv, detection.Source)
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
	}), ProfileSource{})

	require.True(t, detection.Usable())
	assert.Equal(t, EmbeddedSourceTokenEnv, detection.Source)
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
	}), ProfileSource{})

	assert.False(t, detection.Detected)
	assert.False(t, detection.Usable())
}

func TestDetectEmbeddedDefaultPortProbeWithoutToken(t *testing.T) {
	server := newMetaServer(t, true)
	defer server.Close()

	detection := detectEmbedded(context.Background(), server.URL, env(nil), ProfileSource{})

	assert.True(t, detection.Detected)
	assert.Equal(t, EmbeddedSourcePortProbe, detection.Source)
	assert.False(t, detection.Usable(), "port probe alone yields no token")
	assert.Contains(t, detection.Note, "HATCHET_CLIENT_TOKEN")
}

func TestDetectEmbeddedNothingRunning(t *testing.T) {
	detection := detectEmbedded(context.Background(), "http://127.0.0.1:1", env(nil), ProfileSource{})

	assert.False(t, detection.Detected)
	assert.False(t, detection.Usable())
	assert.Empty(t, detection.Note)
}
