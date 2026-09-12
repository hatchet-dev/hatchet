package mcp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cliconfig "github.com/hatchet-dev/hatchet/pkg/config/cli"
)

func testProfileSource(defaultName string, names ...string) ProfileSource {
	profiles := make(map[string]cliconfig.Profile, len(names))
	for _, name := range names {
		profiles[name] = cliconfig.Profile{Name: name, TenantId: "tenant-" + name}
	}

	return ProfileSource{
		Profiles:       func() map[string]cliconfig.Profile { return profiles },
		DefaultProfile: func() string { return defaultName },
	}
}

func grantsFor(names ...string) *Grants {
	grants := &Grants{}
	for _, name := range names {
		grants.Add(name)
	}
	return grants
}

func usableEmbedded() *EmbeddedDetection {
	return &EmbeddedDetection{
		Profile: &cliconfig.Profile{
			Name:         EmbeddedProfileName,
			TenantId:     "tenant-embedded",
			Token:        "token",
			ApiServerURL: "http://localhost:28243",
			GrpcHostPort: "127.0.0.1:12345",
			TLSStrategy:  "none",
		},
	}
}

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

	detection := detectEmbedded(context.Background(), source)

	require.True(t, detection.Detected())
	assert.Equal(t, server.URL, detection.Profile.ApiServerURL)
	assert.Equal(t, "token-registered", detection.Profile.Token)
	assert.Equal(t, "tenant-registered", detection.Profile.TenantId)
	assert.Equal(t, "127.0.0.1:50051", detection.Profile.GrpcHostPort)
	assert.Empty(t, detection.Note)
}

func TestDetectEmbeddedStaleRegistrationIsSkipped(t *testing.T) {
	// A registration left behind by a crashed engine: nothing answers at its
	// API URL, so it is treated as absent, with a note, and never deleted.
	source := storedProfiles(map[string]cliconfig.Profile{
		EmbeddedProfileName: registeredEmbedded("http://127.0.0.1:1"),
	})

	detection := detectEmbedded(context.Background(), source)

	assert.False(t, detection.Detected())
	assert.Contains(t, detection.Note, "stale embedded registration")
}

func TestDetectEmbeddedProfileForNonEmbeddedServerIsSkipped(t *testing.T) {
	// A hand-made "embedded" profile pointing at a non-embedded deployment
	// must not pass the implicit-grant gate: /api/v1/meta must report
	// embedded: true.
	server := newMetaServer(t, false)
	defer server.Close()

	source := storedProfiles(map[string]cliconfig.Profile{
		EmbeddedProfileName: registeredEmbedded(server.URL),
	})

	detection := detectEmbedded(context.Background(), source)

	assert.False(t, detection.Detected())
	assert.Contains(t, detection.Note, "stale embedded registration")
}

// TestDetectEmbeddedRejectsRedirectsAndNonLocalDestinations is the regression
// test for the embedded implicit grant: the registration only counts when both
// stored destinations are loopback and the probe answers directly, without
// redirects. Anything else is treated like a stale registration.
func TestDetectEmbeddedRejectsRedirectsAndNonLocalDestinations(t *testing.T) {
	meta := newMetaServer(t, true)
	defer meta.Close()

	redirecting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, meta.URL+"/api/v1/meta", http.StatusFound)
	}))
	defer redirecting.Close()

	cases := []struct {
		name   string
		mutate func(p *cliconfig.Profile)
	}{
		{
			name:   "api url answers with a redirect",
			mutate: func(p *cliconfig.Profile) { p.ApiServerURL = redirecting.URL },
		},
		{
			name:   "grpc address is not loopback",
			mutate: func(p *cliconfig.Profile) { p.GrpcHostPort = "unrelated.example.invalid:443" },
		},
		{
			name:   "grpc address is missing",
			mutate: func(p *cliconfig.Profile) { p.GrpcHostPort = "" },
		},
		{
			name:   "api url host is not loopback",
			mutate: func(p *cliconfig.Profile) { p.ApiServerURL = "http://unrelated.example.invalid:443" },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			profile := registeredEmbedded(meta.URL)
			tc.mutate(&profile)

			source := storedProfiles(map[string]cliconfig.Profile{EmbeddedProfileName: profile})
			detection := detectEmbedded(context.Background(), source)

			assert.False(t, detection.Detected(), "a registration failing the local-destination checks must not be implicitly granted")
			assert.Contains(t, detection.Note, "stale embedded registration")
		})
	}
}

func TestDetectEmbeddedNothingRegistered(t *testing.T) {
	detection := detectEmbedded(context.Background(), storedProfiles(nil))

	assert.False(t, detection.Detected())
	assert.Empty(t, detection.Note)

	detection = detectEmbedded(context.Background(), ProfileSource{})

	assert.False(t, detection.Detected())
	assert.Empty(t, detection.Note)
}

func TestResolveProfileExplicit(t *testing.T) {
	source := testProfileSource("local", "local", "prod")

	t.Run("granted profile resolves", func(t *testing.T) {
		rp, err := resolveProfile("local", source, grantsFor("local"), &EmbeddedDetection{})
		require.NoError(t, err)
		assert.Equal(t, "local", rp.Name)
		assert.Equal(t, "tenant-local", rp.Profile.TenantId)
		assert.False(t, rp.Embedded)
	})

	t.Run("ungranted profile is denied with actionable error", func(t *testing.T) {
		_, err := resolveProfile("prod", source, grantsFor("local"), &EmbeddedDetection{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), `profile "prod" is not authorized for MCP use`)
		assert.Contains(t, err.Error(), "hatchet mcp auth")
	})

	t.Run("nonexistent profile gets the same denial as an ungranted one", func(t *testing.T) {
		existsErr, err1 := resolveProfileErr("prod", source, grantsFor("local"))
		missingErr, err2 := resolveProfileErr("nope", source, grantsFor("local"))
		require.Error(t, err1)
		require.Error(t, err2)

		// same shape, so an agent cannot probe whether an ungranted name exists
		assert.Equal(t,
			replaceName(existsErr, "prod"),
			replaceName(missingErr, "nope"),
		)
	})

	t.Run("wildcard grants any existing profile", func(t *testing.T) {
		rp, err := resolveProfile("prod", source, grantsFor(GrantWildcard), &EmbeddedDetection{})
		require.NoError(t, err)
		assert.Equal(t, "prod", rp.Name)
	})

	t.Run("denial only lists granted names", func(t *testing.T) {
		_, err := resolveProfile("nope", source, grantsFor("local"), &EmbeddedDetection{})
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "prod")
	})
}

func resolveProfileErr(requested string, source ProfileSource, grants *Grants) (string, error) {
	_, err := resolveProfile(requested, source, grants, &EmbeddedDetection{})
	if err == nil {
		return "", nil
	}
	return err.Error(), err
}

func replaceName(msg, name string) string {
	out := ""
	for i := 0; i+len(name) <= len(msg); i++ {
		if msg[i:i+len(name)] == name {
			out = msg[:i] + "<name>" + msg[i+len(name):]
			break
		}
	}
	if out == "" {
		return msg
	}
	return out
}

func TestResolveProfileDefaulting(t *testing.T) {
	t.Run("granted default profile is used", func(t *testing.T) {
		source := testProfileSource("prod", "local", "prod")
		rp, err := resolveProfile("", source, grantsFor("prod"), &EmbeddedDetection{})
		require.NoError(t, err)
		assert.Equal(t, "prod", rp.Name)
	})

	t.Run("sole profile acts as default", func(t *testing.T) {
		source := testProfileSource("", "local")
		rp, err := resolveProfile("", source, grantsFor("local"), &EmbeddedDetection{})
		require.NoError(t, err)
		assert.Equal(t, "local", rp.Name)
	})

	t.Run("ungranted default falls through to embedded", func(t *testing.T) {
		source := testProfileSource("prod", "local", "prod")
		rp, err := resolveProfile("", source, grantsFor("local"), usableEmbedded())
		require.NoError(t, err)
		assert.Equal(t, EmbeddedProfileName, rp.Name)
		assert.True(t, rp.Embedded)
	})

	t.Run("no default, granted profiles listed in error", func(t *testing.T) {
		source := testProfileSource("", "local", "prod")
		_, err := resolveProfile("", source, grantsFor("local"), &EmbeddedDetection{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "local")
		assert.NotContains(t, err.Error(), "prod", "ungranted names must not leak")
	})

	t.Run("nothing granted, no embedded", func(t *testing.T) {
		source := testProfileSource("", "local")
		_, err := resolveProfile("", source, grantsFor(), &EmbeddedDetection{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "hatchet mcp auth")
		assert.NotContains(t, err.Error(), "local")
	})
}

func TestResolveProfileEmbedded(t *testing.T) {
	source := testProfileSource("", "local", "prod")

	t.Run("embedded is implicitly granted", func(t *testing.T) {
		rp, err := resolveProfile(EmbeddedProfileName, source, grantsFor(), usableEmbedded())
		require.NoError(t, err)
		assert.Equal(t, EmbeddedProfileName, rp.Name)
		assert.True(t, rp.Embedded)
	})

	t.Run("live registration resolves via detection without a grant", func(t *testing.T) {
		// The engine's self-registration in the profile store arrives here as
		// the (live-verified) detection result, implicitly granted.
		sourceWithEmbedded := testProfileSource("", "embedded")
		detection := usableEmbedded()

		rp, err := resolveProfile("embedded", sourceWithEmbedded, grantsFor(), detection)
		require.NoError(t, err)
		assert.True(t, rp.Embedded)
		assert.Equal(t, "tenant-embedded", rp.Profile.TenantId)
	})

	t.Run("stale registration is treated as absent", func(t *testing.T) {
		// The profile exists in the store but its engine is dead: detection
		// skipped it (with a note), and even a grant must not resurrect it.
		sourceWithEmbedded := testProfileSource("", "embedded")
		detection := &EmbeddedDetection{Note: "stale embedded registration: the engine at http://localhost:28243 is gone"}

		_, err := resolveProfile("embedded", sourceWithEmbedded, grantsFor("embedded"), detection)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no running embedded Hatchet instance")
		assert.Contains(t, err.Error(), "stale embedded registration")
	})

	t.Run("stale registration is never the default or sole profile", func(t *testing.T) {
		sourceWithEmbedded := testProfileSource("embedded", "embedded")
		detection := &EmbeddedDetection{Note: "stale embedded registration: the engine is gone"}

		_, err := resolveProfile("", sourceWithEmbedded, grantsFor(GrantWildcard), detection)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "stale embedded registration")
	})

	t.Run("not detected", func(t *testing.T) {
		_, err := resolveProfile(EmbeddedProfileName, source, grantsFor(), &EmbeddedDetection{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no running embedded Hatchet instance")
	})
}
