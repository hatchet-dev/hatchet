package staticfileserver

import (
	"html/template"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// DefaultContentSecurityPolicy restricts only what is the same for every deployment: framing,
// plugins and <base>. Which script, style and connect sources the dashboard needs depends on the
// deployment (analytics, support widget, error reporting), so those are left to an override.
const DefaultContentSecurityPolicy = "frame-ancestors 'none'; object-src 'none'; base-uri 'self'"

// NewStaticFileServer serves the dashboard. contentSecurityPolicy is sent as the
// Content-Security-Policy header; an empty value sends none.
func NewStaticFileServer(staticFilePath, basePath, contentSecurityPolicy string) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)

	fs := http.FileServer(http.Dir(staticFilePath))
	index := indexHandler(staticFilePath, basePath)
	withHeaders := securityHeaders(contentSecurityPolicy)

	content := withHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "" {
			index(w, r)
			return
		}

		// Clean and join under staticFilePath so a crafted "../" in the request path can't be
		// used to probe file existence outside the static root.
		requestedPath := filepath.Join(staticFilePath, path.Clean("/"+r.URL.Path))

		if _, err := os.Stat(requestedPath); os.IsNotExist(err) {
			index(w, r)
			return
		}

		// Vite emits content-hashed filenames under /assets, so those bytes never change for a
		// given URL. Marking them immutable lets browsers and CDNs (e.g. Cloudflare) serve them
		// from cache without revalidating against the origin on every request. Everything else
		// involving html or js stays "no-cache" so new deploys are picked up via index.html.
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else if base := path.Base(r.URL.Path); strings.Contains(base, "html") || strings.Contains(base, "js") || base == "." || base == "/" {
			w.Header().Set("Cache-Control", "no-cache")
		}

		fs.ServeHTTP(w, r)
	}))

	if prefix := routePrefix(basePath); prefix != "" {
		r.Get(prefix, withHeaders(http.RedirectHandler(prefix+"/", http.StatusFound)).ServeHTTP)
		r.Handle(prefix+"/*", http.StripPrefix(prefix, content))
	} else {
		r.Handle("/*", content)
	}

	return r
}

// securityHeaders sets the browser security headers on the dashboard's responses. They match the
// API's, except Cross-Origin-Embedder-Policy is credentialless rather than require-corp: the
// dashboard document loads cross-origin scripts and styles, some only when a deployment enables
// them, and require-corp blocks any of those that do not send Cross-Origin-Resource-Policy.
func securityHeaders(contentSecurityPolicy string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("Strict-Transport-Security", "max-age=776000; includeSubDomains; preload")
			if contentSecurityPolicy != "" {
				h.Set("Content-Security-Policy", contentSecurityPolicy)
			}
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("X-Frame-Options", "DENY")
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("Permissions-Policy", "geolocation=(), midi=(), notifications=(), push=(), sync-xhr=(), microphone=(), camera=(), magnetometer=(), gyroscope=(), speaker=(), vibrate=(), fullscreen=(self), payment=()")
			h.Set("Cross-Origin-Embedder-Policy", "credentialless")
			h.Set("Cross-Origin-Opener-Policy", "same-origin")
			h.Set("Cross-Origin-Resource-Policy", "same-origin")

			next.ServeHTTP(w, r)
		})
	}
}

func routePrefix(basePath string) string {
	trimmed := strings.Trim(basePath, "/")
	if trimmed == "" {
		return ""
	}
	return "/" + trimmed
}

func indexHandler(staticFilePath, basePath string) http.HandlerFunc {
	if basePath == "" {
		basePath = "/"
	}
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}
	if !strings.HasSuffix(basePath, "/") {
		basePath += "/"
	}

	t := template.Must(template.ParseFiles(filepath.Join(staticFilePath, "index.html")))
	data := struct{ BasePath string }{basePath}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		if err := t.Execute(w, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
