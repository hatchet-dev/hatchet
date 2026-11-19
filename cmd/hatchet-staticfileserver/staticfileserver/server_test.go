//go:build !e2e && !load && !rampup && !integration

package staticfileserver

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
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
		"Permissions-Policy":           "geolocation=(), midi=(), sync-xhr=(), microphone=(), camera=(), magnetometer=(), gyroscope=(), fullscreen=(self), payment=()",
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

func TestAPIProxyPreservesUpstreamHeaders(t *testing.T) {
	dir := newTestStaticDir(t)

	const upstreamCSP = "script-src 'self'; frame-ancestors 'self'; object-src 'none'; base-uri 'self'"
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", upstreamCSP)
		w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")
		w.Header().Set("X-Frame-Options", "DENY")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	target, err := url.Parse(upstream.URL)
	require.NoError(t, err)

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
		},
	}

	srv := NewStaticFileServer(dir, "/", DefaultContentSecurityPolicy)
	srv.Handle("/api", proxy)
	srv.Handle("/api/*", proxy)

	for _, reqPath := range []string{"/api", "/api/v1/meta"} {
		t.Run(reqPath, func(t *testing.T) {
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, reqPath, nil))

			require.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, []string{upstreamCSP}, rec.Header().Values("Content-Security-Policy"))
			assert.Equal(t, []string{"require-corp"}, rec.Header().Values("Cross-Origin-Embedder-Policy"))
			assert.Equal(t, []string{"DENY"}, rec.Header().Values("X-Frame-Options"))
			assert.Empty(t, rec.Header().Values("Cross-Origin-Opener-Policy"))
		})
	}
}
