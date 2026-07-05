package hatchetembed

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// dashboardFS holds the built frontend dashboard, embedded into the binary. The contents of the
// dashboard/ directory are generated from frontend/app by `task dashboard:embed`; a placeholder
// index.html is committed so the package always compiles.
//
//go:embed all:dashboard
var dashboardFS embed.FS

// dashboardHandler returns an http.Handler serving the embedded dashboard single-page app. Requests
// for paths that don't map to a real asset fall back to index.html so client-side routes resolve.
func dashboardHandler() (http.Handler, error) {
	sub, err := fs.Sub(dashboardFS, "dashboard")
	if err != nil {
		return nil, fmt.Errorf("could not open embedded dashboard: %w", err)
	}

	indexBytes, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		return nil, fmt.Errorf("embedded dashboard is missing index.html (build it with `task dashboard:embed`): %w", err)
	}

	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")

		if reqPath != "" {
			if _, statErr := fs.Stat(sub, reqPath); statErr == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// SPA fallback for the index and client-side routes.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexBytes)
	}), nil
}
