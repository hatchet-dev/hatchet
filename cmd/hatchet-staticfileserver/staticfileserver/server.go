package staticfileserver

import (
	"html/template"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func NewStaticFileServer(staticFilePath, basePath string) *chi.Mux {
	r := chi.NewRouter()

	fs := http.FileServer(http.Dir(staticFilePath))
	index := indexHandler(staticFilePath, basePath)

	r.Use(middleware.Logger)

	r.Get("/", index)

	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")

		// Clean and join under staticFilePath so a crafted "../" in the request path can't be
		// used to probe file existence outside the static root.
		requestedPath := filepath.Join(staticFilePath, path.Clean("/"+r.URL.Path))

		if _, err := os.Stat(requestedPath); os.IsNotExist(err) {
			// SPA fallback: render index.html through the template so the base
			// path is injected on deep links and refreshes, not just at the
			// exact root. Serving the file raw here would leak the unrendered
			// {{ .BasePath }} directive to the browser.
			index(w, r)
		} else {
			// Set static files involving html, js, or empty cache to "no-cache", which means they must be validated
			// for changes before the browser uses the cache
			if base := path.Base(r.URL.Path); strings.Contains(base, "html") || strings.Contains(base, "js") || base == "." || base == "/" {
				w.Header().Set("Cache-Control", "no-cache")
			}

			fs.ServeHTTP(w, r)
		}
	})

	return r
}

func indexHandler(staticFilePath, basePath string) http.HandlerFunc {
	// Normalize to a leading and trailing slash so the browser resolves the
	// app's relative asset URLs (./assets/...) under the subpath rather than the
	// host root, regardless of whether BASE_PATH was written as "hatchet",
	// "/hatchet", or "/hatchet/".
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
