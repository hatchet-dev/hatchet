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

	r.Use(middleware.Logger)

	r.Get("/", indexHandler(staticFilePath, basePath))

	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")

		if _, err := os.Stat(staticFilePath + r.RequestURI); os.IsNotExist(err) { //nolint gosec
			w.Header().Set("Cache-Control", "no-cache")

			http.StripPrefix(r.URL.Path, fs).ServeHTTP(w, r)
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
