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
	s, err := newServer(os.Args[1:], os.LookupEnv)
	if err != nil {
		log.Print(err.Error())
		os.Exit(1)
	}

	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("static file server failure: %s", err.Error())
		os.Exit(1)
	}
}

// newServer resolves flags and environment into the server main runs, so tests exercise the same
// wiring as production.
func newServer(args []string, lookupEnv func(string) (string, bool)) (*http.Server, error) {
	fs := flag.NewFlagSet("hatchet-staticfileserver", flag.ExitOnError)

	basePathDefault := "/"
	if v, ok := lookupEnv("BASE_PATH"); ok && v != "" {
		basePathDefault = v
	}

	// NOTE: an empty CONTENT_SECURITY_POLICY is honored (it disables the header), so only an unset
	// variable falls back to the default policy.
	cspDefault := staticfileserver.DefaultContentSecurityPolicy
	if v, ok := lookupEnv("CONTENT_SECURITY_POLICY"); ok {
		cspDefault = v
	}

	port := fs.String("port", "80", "port to listen on")
	staticAssetDir := fs.String("static-asset-dir", ".", "directory to serve static assets from")
	basePath := fs.String("base-path", basePathDefault, "base path the app is served under (e.g. /hatchet); defaults to $BASE_PATH, or / if unset")
	apiProxy := fs.String("api-proxy", "", "if set, reverse-proxy /api requests to this URL (e.g. http://localhost:8080)")
	contentSecurityPolicy := fs.String("content-security-policy", cspDefault, "Content-Security-Policy header for the dashboard; defaults to $CONTENT_SECURITY_POLICY, or a policy that restricts only framing, plugins and <base> if unset (empty value sends none)")
	_ = fs.Parse(args)

	c := staticfileserver.NewStaticFileServer(*staticAssetDir, *basePath, *contentSecurityPolicy)

	if *apiProxy != "" {
		target, err := url.Parse(*apiProxy)
		if err != nil {
			return nil, fmt.Errorf("invalid -api-proxy url: %w", err)
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

	return &http.Server{
		Addr:              fmt.Sprintf(":%s", *port),
		Handler:           c,
		ReadHeaderTimeout: 5 * time.Second,
	}, nil
}
