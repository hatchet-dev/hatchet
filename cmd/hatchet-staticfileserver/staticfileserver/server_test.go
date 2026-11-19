//go:build !e2e && !load && !rampup && !integration

package staticfileserver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStaticDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte(`<base href="{{ .BasePath }}"><div id="root"></div>`), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "assets"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "assets", "index-abc123.js"), []byte("console.log(1)"), 0o600))

	return dir
}

func TestSecurityHeaders(t *testing.T) {
	dir := newTestStaticDir(t)

	want := map[string]string{
		"Strict-Transport-Security":    "max-age=776000; includeSubDomains; preload",
		"Content-Security-Policy":      DefaultContentSecurityPolicy,
		"Referrer-Policy":              "strict-origin-when-cross-origin",
		"X-Frame-Options":              "DENY",
		"X-Content-Type-Options":       "nosniff",
		"Permissions-Policy":           "geolocation=(), midi=(), notifications=(), push=(), sync-xhr=(), microphone=(), camera=(), magnetometer=(), gyroscope=(), speaker=(), vibrate=(), fullscreen=(self), payment=()",
		"Cross-Origin-Embedder-Policy": "credentialless",
		"Cross-Origin-Opener-Policy":   "same-origin",
		"Cross-Origin-Resource-Policy": "same-origin",
	}

	cases := []struct {
		name     string
		basePath string
		path     string
		status   int
	}{
		{name: "index", basePath: "/", path: "/", status: http.StatusOK},
		{name: "asset", basePath: "/", path: "/assets/index-abc123.js", status: http.StatusOK},
		{name: "spa fallback", basePath: "/", path: "/tenants/some-tenant/runs", status: http.StatusOK},
		{name: "base path redirect", basePath: "/hatchet", path: "/hatchet", status: http.StatusFound},
		{name: "base path asset", basePath: "/hatchet", path: "/hatchet/assets/index-abc123.js", status: http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := NewStaticFileServer(dir, tc.basePath, DefaultContentSecurityPolicy)

			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))

			require.Equal(t, tc.status, rec.Code)
			for name, value := range want {
				assert.Equal(t, []string{value}, rec.Header().Values(name), name)
			}
		})
	}
}

func TestContentSecurityPolicyOverride(t *testing.T) {
	dir := newTestStaticDir(t)
	policy := "default-src 'self'; script-src 'self' 'unsafe-inline'"

	srv := NewStaticFileServer(dir, "/", policy)

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, []string{policy}, rec.Header().Values("Content-Security-Policy"))

	srv = NewStaticFileServer(dir, "/", "")

	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Empty(t, rec.Header().Values("Content-Security-Policy"))
}
