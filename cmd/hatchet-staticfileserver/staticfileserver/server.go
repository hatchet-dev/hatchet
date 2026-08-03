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

	r.Use(middleware.Logger)

	fs := http.FileServer(http.Dir(staticFilePath))
	index := indexHandler(staticFilePath, basePath)

	content := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "" {
			index(w, r)
			return
		}

		w.Header().Set("X-Frame-Options", "DENY")

		// Clean and join under staticFilePath so a crafted "../" in the request path can't be
		// used to probe file existence outside the static root.
		requestedPath := filepath.Join(staticFilePath, path.Clean("/"+r.URL.Path))

		if _, err := os.Stat(requestedPath); os.IsNotExist(err) {
			index(w, r)
			return
		}

		// Set static files involving html, js, or empty cache to "no-cache", which means they must be validated
		// for changes before the browser uses the cache
		if base := path.Base(r.URL.Path); strings.Contains(base, "html") || strings.Contains(base, "js") || base == "." || base == "/" {
			w.Header().Set("Cache-Control", "no-cache")
		}

		fs.ServeHTTP(w, r)
	})

	if prefix := routePrefix(basePath); prefix != "" {
		r.Get(prefix, http.RedirectHandler(prefix+"/", http.StatusFound).ServeHTTP)
		r.Handle(prefix+"/*", http.StripPrefix(prefix, content))
	} else {
		r.Handle("/*", content)
	}

	return r
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
