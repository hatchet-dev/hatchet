package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-staticfileserver/staticfileserver"
)

func main() {
	port := flag.String("port", "80", "port to listen on")
	staticAssetDir := flag.String("static-asset-dir", ".", "directory to serve static assets from")
	basePath := flag.String("base-path", envOrDefault("BASE_PATH", "/"), "base path the app is served under (e.g. /hatchet); defaults to $BASE_PATH, or / if unset")
	apiProxy := flag.String("api-proxy", "", "if set, reverse-proxy /api requests to this URL (e.g. http://localhost:8080)")
	flag.Parse()

	c := staticfileserver.NewStaticFileServer(*staticAssetDir, *basePath)

	if *apiProxy != "" {
		target, err := url.Parse(*apiProxy)
		if err != nil {
			log.Printf("invalid -api-proxy url: %s", err.Error())
			os.Exit(1)
		}

		proxy := &httputil.ReverseProxy{
			Rewrite: func(r *httputil.ProxyRequest) {
				r.SetURL(target)
				r.Out.Host = r.In.Host
				r.SetXForwarded()
			},
		}

		c.Handle("/api", proxy)
		c.Handle("/api/*", proxy)
	}

	s := &http.Server{
		Addr:              fmt.Sprintf(":%s", *port),
		Handler:           c,
		ReadHeaderTimeout: 5 * time.Second,
	}

	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("static file server failure: %s", err.Error())
		os.Exit(1)
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
