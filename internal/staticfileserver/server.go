package staticfileserver

import (
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func NewStaticFileServer(staticFilePath string) *chi.Mux {
	r := chi.NewRouter()

	fs := http.FileServer(http.Dir(staticFilePath))

	r.Use(middleware.Logger)

	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")

		if _, err := os.Stat(staticFilePath + r.RequestURI); os.IsNotExist(err) {
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
