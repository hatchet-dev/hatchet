//go:build !e2e && !load && !rampup && !integration

package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-staticfileserver/staticfileserver"
)

func newTestStaticDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte(`<base href="{{ .BasePath }}">`), 0o600))

	return dir
}

func env(vars map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		v, ok := vars[key]
		return v, ok
	}
}

func serve(t *testing.T, s *http.Server, path string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	s.Handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestContentSecurityPolicyResolution(t *testing.T) {
	const custom = "default-src 'self'"

	cases := []struct {
		name string
		env  map[string]string
		args []string
		want []string
	}{
		{name: "unset env uses the default", want: []string{staticfileserver.DefaultContentSecurityPolicy}},
		{name: "empty env disables the header", env: map[string]string{"CONTENT_SECURITY_POLICY": ""}, want: nil},
		{name: "env sets the policy", env: map[string]string{"CONTENT_SECURITY_POLICY": custom}, want: []string{custom}},
		{name: "flag overrides env", env: map[string]string{"CONTENT_SECURITY_POLICY": custom}, args: []string{"-content-security-policy", "object-src 'none'"}, want: []string{"object-src 'none'"}},
		{name: "empty flag disables the header", args: []string{"-content-security-policy="}, want: nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := newServer(append([]string{"-static-asset-dir", newTestStaticDir(t)}, tc.args...), env(tc.env))
			require.NoError(t, err)

			rec := serve(t, s, "/")

			require.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, tc.want, rec.Header().Values("Content-Security-Policy"))
		})
	}
}

func TestBasePathResolution(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		args []string
		path string
		want string
	}{
		{name: "unset env serves at root", path: "/", want: `<base href="/">`},
		{name: "empty env serves at root", env: map[string]string{"BASE_PATH": ""}, path: "/", want: `<base href="/">`},
		{name: "env sets the base path", env: map[string]string{"BASE_PATH": "/hatchet"}, path: "/hatchet/", want: `<base href="/hatchet/">`},
		{name: "flag overrides env", env: map[string]string{"BASE_PATH": "/hatchet"}, args: []string{"-base-path", "/other"}, path: "/other/", want: `<base href="/other/">`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := newServer(append([]string{"-static-asset-dir", newTestStaticDir(t)}, tc.args...), env(tc.env))
			require.NoError(t, err)

			rec := serve(t, s, tc.path)

			require.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, tc.want, rec.Body.String())
		})
	}
}

// TestAPIProxy checks that /api stays at the root even under a base path, and that the proxied
// responses keep the API's own security headers instead of the dashboard's.
func TestAPIProxy(t *testing.T) {
	const upstreamCSP = "script-src 'self'; frame-ancestors 'self'; object-src 'none'; base-uri 'self'"

	var gotPath, gotHost, gotForwardedHost string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotHost, gotForwardedHost = r.URL.Path, r.Host, r.Header.Get("X-Forwarded-Host")
		w.Header().Set("Content-Security-Policy", upstreamCSP)
		w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	s, err := newServer(
		[]string{"-static-asset-dir", newTestStaticDir(t), "-api-proxy", upstream.URL},
		env(map[string]string{"BASE_PATH": "/hatchet"}),
	)
	require.NoError(t, err)

	for _, path := range []string{"/api", "/api/v1/meta"} {
		t.Run(path, func(t *testing.T) {
			rec := serve(t, s, path)

			require.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, path, gotPath)
			assert.Equal(t, "example.com", gotHost, "the dashboard's Host is forwarded so the API builds its own URLs")
			assert.Equal(t, "example.com", gotForwardedHost)
			assert.Equal(t, []string{upstreamCSP}, rec.Header().Values("Content-Security-Policy"))
			assert.Equal(t, []string{"require-corp"}, rec.Header().Values("Cross-Origin-Embedder-Policy"))
			assert.Empty(t, rec.Header().Values("Cross-Origin-Opener-Policy"))
			assert.Empty(t, rec.Header().Values("Strict-Transport-Security"))
		})
	}
}
