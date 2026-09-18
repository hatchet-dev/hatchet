package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cliconfig "github.com/hatchet-dev/hatchet/pkg/config/cli"
)

func TestRenderProfileEnv(t *testing.T) {
	tests := []struct {
		name        string
		profileName string
		profile     cliconfig.Profile
		noExport    bool
		wantErr     bool
		wantLines   []string
		notContains []string
	}{
		{
			name:        "tls enabled emits tls strategy",
			profileName: "prod",
			profile: cliconfig.Profile{
				Token:        "tok",
				TLSStrategy:  "tls",
				GrpcHostPort: "host:443",
				ApiServerURL: "https://api",
				TenantId:     "tenant-1",
			},
			wantLines: []string{
				"export HATCHET_CLIENT_TOKEN='tok'",
				"export HATCHET_CLIENT_TLS_STRATEGY='tls'",
				"export HATCHET_CLIENT_HOST_PORT='host:443'",
				"export HATCHET_CLIENT_SERVER_URL='https://api'",
				"export HATCHET_CLIENT_API_URL='https://api'",
				"export HATCHET_CLIENT_TENANT_ID='tenant-1'",
			},
		},
		{
			name:        "tls disabled self hosted emits none and unsets absent fields",
			profileName: "local",
			profile: cliconfig.Profile{
				Token:        "tok",
				TLSStrategy:  "none",
				GrpcHostPort: "localhost:7070",
			},
			wantLines: []string{
				"export HATCHET_CLIENT_TOKEN='tok'",
				"export HATCHET_CLIENT_TLS_STRATEGY='none'",
				"export HATCHET_CLIENT_HOST_PORT='localhost:7070'",
				// Absent optional fields are unset in export mode so a prior
				// profile's endpoint and tenant cannot linger in the shell.
				"unset HATCHET_CLIENT_SERVER_URL",
				"unset HATCHET_CLIENT_API_URL",
				"unset HATCHET_CLIENT_TENANT_ID",
			},
			notContains: []string{
				"export HATCHET_CLIENT_SERVER_URL",
				"export HATCHET_CLIENT_API_URL",
				"export HATCHET_CLIENT_TENANT_ID",
			},
		},
		{
			name:        "empty tls strategy falls back to tls",
			profileName: "prod",
			profile: cliconfig.Profile{
				Token:       "tok",
				TLSStrategy: "",
			},
			wantLines: []string{
				"export HATCHET_CLIENT_TOKEN='tok'",
				"export HATCHET_CLIENT_TLS_STRATEGY='tls'",
			},
		},
		{
			name:        "optional fields empty are unset but token and tls remain",
			profileName: "prod",
			profile: cliconfig.Profile{
				Token:       "tok",
				TLSStrategy: "tls",
			},
			wantLines: []string{
				"export HATCHET_CLIENT_TOKEN='tok'",
				"export HATCHET_CLIENT_TLS_STRATEGY='tls'",
				"unset HATCHET_CLIENT_HOST_PORT",
				"unset HATCHET_CLIENT_SERVER_URL",
				"unset HATCHET_CLIENT_API_URL",
				"unset HATCHET_CLIENT_TENANT_ID",
			},
			notContains: []string{
				"export HATCHET_CLIENT_HOST_PORT",
				"export HATCHET_CLIENT_SERVER_URL",
				"export HATCHET_CLIENT_TENANT_ID",
			},
		},
		{
			name:        "empty token errors",
			profileName: "prod",
			profile: cliconfig.Profile{
				Token:       "",
				TLSStrategy: "tls",
			},
			wantErr: true,
		},
		{
			name:        "no export drops the export prefix",
			profileName: "prod",
			profile: cliconfig.Profile{
				Token:       "tok",
				TLSStrategy: "tls",
			},
			noExport: true,
			wantLines: []string{
				"HATCHET_CLIENT_TOKEN='tok'",
				"HATCHET_CLIENT_TLS_STRATEGY='tls'",
			},
			// A file snapshot has nothing to clear, so empty fields are omitted
			// rather than unset, and there is no export prefix.
			notContains: []string{"export ", "unset "},
		},
		{
			name:        "single quote in value is escaped",
			profileName: "prod",
			profile: cliconfig.Profile{
				Token:       "a'b",
				TLSStrategy: "tls",
			},
			wantLines: []string{
				`export HATCHET_CLIENT_TOKEN='a'\''b'`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := renderProfileEnv(tt.profileName, tt.profile, tt.noExport)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			for _, line := range tt.wantLines {
				assert.Contains(t, out, line)
			}
			for _, absent := range tt.notContains {
				assert.NotContains(t, out, absent)
			}
		})
	}
}

// TestRenderProfileEnvOrdering pins the deterministic emission order.
func TestRenderProfileEnvOrdering(t *testing.T) {
	out, err := renderProfileEnv("prod", cliconfig.Profile{
		Token:        "tok",
		TLSStrategy:  "none",
		GrpcHostPort: "host:443",
		ApiServerURL: "https://api",
		TenantId:     "tenant-1",
	}, false)
	require.NoError(t, err)

	got := indexOrder(out, []string{
		"HATCHET_CLIENT_TOKEN",
		"HATCHET_CLIENT_TLS_STRATEGY",
		"HATCHET_CLIENT_HOST_PORT",
		"HATCHET_CLIENT_SERVER_URL",
		"HATCHET_CLIENT_API_URL",
		"HATCHET_CLIENT_TENANT_ID",
	})

	assert.Equal(t, []string{
		"HATCHET_CLIENT_TOKEN",
		"HATCHET_CLIENT_TLS_STRATEGY",
		"HATCHET_CLIENT_HOST_PORT",
		"HATCHET_CLIENT_SERVER_URL",
		"HATCHET_CLIENT_API_URL",
		"HATCHET_CLIENT_TENANT_ID",
	}, got)
}

// indexOrder returns the keys sorted by where they first appear in out.
func indexOrder(out string, keys []string) []string {
	type pos struct {
		key string
		at  int
	}
	var found []pos
	for _, k := range keys {
		if i := strings.Index(out, k); i >= 0 {
			found = append(found, pos{k, i})
		}
	}
	for i := 1; i < len(found); i++ {
		for j := i; j > 0 && found[j-1].at > found[j].at; j-- {
			found[j-1], found[j] = found[j], found[j-1]
		}
	}
	ordered := make([]string, 0, len(found))
	for _, p := range found {
		ordered = append(ordered, p.key)
	}
	return ordered
}

// TestShellSingleQuote checks that escaped values remain valid single-quoted
// shell strings (balanced quotes) even when they contain a single quote.
func TestShellSingleQuote(t *testing.T) {
	cases := []string{"plain", "a'b", "'", "''", "no'thing'here"}
	for _, c := range cases {
		got := shellSingleQuote(c)
		assert.True(t, strings.HasPrefix(got, "'"), "must start with a quote: %q", got)
		assert.True(t, strings.HasSuffix(got, "'"), "must end with a quote: %q", got)
		// A valid single-quoted string escapes every literal quote as '\'' so no
		// bare quote survives inside the wrapper.
		inner := got[1 : len(got)-1]
		assert.NotContains(t, strings.ReplaceAll(inner, `'\''`, ""), "'",
			"unbalanced quote in %q", got)
	}
}
