package staticfileserver

import (
	"encoding/json"
	"fmt"
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

	basePath = strings.TrimRight(basePath, "/")
	if basePath != "" {
		// Dynamcally build and serve the index.html and config.js when we have a custom basePath
		r.Get(basePath, handleIndex(staticFilePath, basePath))
		r.Get(basePath+"/config.js", handleJsConfig(basePath))
	}

	r.Get(basePath+"/*", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		assetPath := strings.TrimPrefix(r.URL.Path, basePath)
		localAssetPath := filepath.Join(staticFilePath, filepath.FromSlash(assetPath))
		if _, err := os.Stat(localAssetPath); os.IsNotExist(err) { //nolint gosec
			w.Header().Set("Cache-Control", "no-cache")
			handleIndex(staticFilePath, basePath)(w, r)
		} else {
			// Set static files involving html, js, or empty cache to "no-cache", which means they must be validated
			// for changes before the browser uses the cache
			if base := path.Base(r.URL.Path); strings.Contains(base, "html") || strings.Contains(base, "js") || base == "." || base == "/" {
				w.Header().Set("Cache-Control", "no-cache")
			}
			http.StripPrefix(basePath, fs).ServeHTTP(w, r)
		}
	})

	return r
}

// handleJsConfig serves a dynamic config.js that sets window.__CONFIG__ with runtime values
// allowing the frontend to read deployment configuration at startup.
func handleJsConfig(basePath string) http.HandlerFunc {
	var conf struct {
		BasePath string `json:"BASE_PATH"`
	}

	conf.BasePath = basePath
	return func(w http.ResponseWriter, r *http.Request) {
		contents, err := json.Marshal(conf)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/javascript")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Frame-Options", "DENY")
		fmt.Fprintf(w, "window.__CONFIG__ = %s;\n", contents)
	}
}

// handleIndex serves index.html with the <base href> tag rewritten to basePath,
// enabling the router to resolve routes correctly when hosted under a sub-path.
func handleIndex(staticFilePath, basePath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := os.ReadFile(filepath.Join(staticFilePath, "index.html"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		content := strings.ReplaceAll(string(b), `<base href="/">`, `<base href="`+basePath+`/">`)
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write([]byte(content))
	}
}
