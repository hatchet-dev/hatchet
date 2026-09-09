package mcp

import (
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
		Detected: true,
		APIURL:   "http://localhost:28243",
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
		detection.Source = EmbeddedSourceProfile

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

	t.Run("detected without token explains the limitation", func(t *testing.T) {
		detection := &EmbeddedDetection{Detected: true, APIURL: "http://localhost:28243", Note: "export HATCHET_CLIENT_TOKEN"}
		_, err := resolveProfile(EmbeddedProfileName, source, grantsFor(), detection)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no client token")
		assert.Contains(t, err.Error(), "HATCHET_CLIENT_TOKEN")
	})

	t.Run("not detected", func(t *testing.T) {
		_, err := resolveProfile(EmbeddedProfileName, source, grantsFor(), &EmbeddedDetection{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no running embedded Hatchet instance")
	})
}
