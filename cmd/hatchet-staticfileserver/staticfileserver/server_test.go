//go:build !e2e && !load && !rampup && !integration

package staticfileserver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestStaticDir lays out a minimal dashboard build and, next to it, a file outside the static
// root that no request may reach. It returns the static root.
func newTestStaticDir(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	dir := filepath.Join(root, "static")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "assets"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte(`<base href="{{ .BasePath }}"><div id="root"></div>`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "assets", "index-abc123.js"), []byte("console.log(1)"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "env.js"), []byte("window.env={}"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "secret.txt"), []byte("outside the static root"), 0o600))

	return dir
}

// wantSecurityHeaders pins the exact values: each one is a security decision, so a change should
// show up as a diff here.
var wantSecurityHeaders = map[string]string{
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

// TestDashboardResponses walks every branch of the dashboard handler and checks that each one
// carries the full security header set exactly once, alongside the routing and caching outcome
// specific to that branch.
func TestDashboardResponses(t *testing.T) {
	dir := newTestStaticDir(t)
	future := time.Now().Add(time.Hour).UTC().Format(http.TimeFormat)

	const (
		noCache   = "no-cache"
		immutable = "public, max-age=31536000, immutable"
	)

	cases := []struct {
		name            string
		basePath        string
		path            string
		ifModifiedSince string
		status          int
		cacheControl    string
		location        string
		// body is the exact response body; empty skips the check.
		body string
	}{
		{name: "index", basePath: "/", path: "/", status: http.StatusOK, cacheControl: noCache, body: `<base href="/"><div id="root"></div>`},
		{name: "spa fallback", basePath: "/", path: "/tenants/some-tenant/runs", status: http.StatusOK, cacheControl: noCache, body: `<base href="/"><div id="root"></div>`},
		{name: "index.html redirects to the rendered index", basePath: "/", path: "/index.html", status: http.StatusMovedPermanently, cacheControl: noCache, location: "./"},
		{name: "hashed asset", basePath: "/", path: "/assets/index-abc123.js", status: http.StatusOK, cacheControl: immutable, body: "console.log(1)"},
		{name: "hashed asset not modified", basePath: "/", path: "/assets/index-abc123.js", ifModifiedSince: future, status: http.StatusNotModified, cacheControl: immutable},
		{name: "unhashed js", basePath: "/", path: "/env.js", status: http.StatusOK, cacheControl: noCache, body: "window.env={}"},
		// The path guard turns a probe outside the root into the SPA fallback, so existence of the
		// file can't be told apart from any other unknown route.
		{name: "traversal outside root", basePath: "/", path: "/../secret.txt", status: http.StatusOK, cacheControl: noCache, body: `<base href="/"><div id="root"></div>`},
		{name: "base path redirect", basePath: "/hatchet", path: "/hatchet", status: http.StatusFound, location: "/hatchet/"},
		{name: "base path index", basePath: "/hatchet", path: "/hatchet/", status: http.StatusOK, cacheControl: noCache, body: `<base href="/hatchet/"><div id="root"></div>`},
		{name: "base path without slashes", basePath: "hatchet", path: "/hatchet/", status: http.StatusOK, cacheControl: noCache, body: `<base href="/hatchet/"><div id="root"></div>`},
		{name: "base path spa fallback", basePath: "/hatchet/", path: "/hatchet/tenants/some-tenant/runs", status: http.StatusOK, cacheControl: noCache, body: `<base href="/hatchet/"><div id="root"></div>`},
		{name: "base path hashed asset", basePath: "/hatchet", path: "/hatchet/assets/index-abc123.js", status: http.StatusOK, cacheControl: immutable, body: "console.log(1)"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := NewStaticFileServer(dir, tc.basePath, DefaultContentSecurityPolicy)

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			if tc.ifModifiedSince != "" {
				req.Header.Set("If-Modified-Since", tc.ifModifiedSince)
			}
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)

			require.Equal(t, tc.status, rec.Code)
			assert.Equal(t, tc.cacheControl, rec.Header().Get("Cache-Control"))
			assert.Equal(t, tc.location, rec.Header().Get("Location"))
			if tc.body != "" {
				assert.Equal(t, tc.body, rec.Body.String())
			}
			for name, value := range wantSecurityHeaders {
				assert.Equal(t, []string{value}, rec.Header().Values(name), name)
			}
		})
	}
}

func TestBasePathDoesNotServeRoot(t *testing.T) {
	srv := NewStaticFileServer(newTestStaticDir(t), "/hatchet", DefaultContentSecurityPolicy)

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/assets/index-abc123.js", nil))

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
