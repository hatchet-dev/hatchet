package internal

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-api/api"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-engine/engine"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-lite/staticfileserver"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
)

// runs a static file server, api and engine in the same process.
func StartLite(cf *loader.ConfigLoader, interruptCh <-chan interface{}, version string) error {
	// read static asset directory and frontend URL from the environment
	staticAssetDir := os.Getenv("LITE_STATIC_ASSET_DIR")
	frontendPort := os.Getenv("LITE_FRONTEND_PORT")
	runtimePort := os.Getenv("LITE_RUNTIME_PORT")

	if staticAssetDir == "" {
		return fmt.Errorf("LITE_STATIC_ASSET_DIR environment variable is required")
	}

	if frontendPort == "" {
		return fmt.Errorf("LITE_FRONTEND_PORT environment variable is required")
	}

	if runtimePort == "" {
		runtimePort = "8082"
	}

	// we hard code the msg queue kind to postgres
	err := os.Setenv("SERVER_MSGQUEUE_KIND", "postgres")

	if err != nil {
		return fmt.Errorf("error setting SERVER_MSGQUEUE_KIND to postgres: %w", err)
	}

	feURL, err := url.Parse(fmt.Sprintf("http://localhost:%s", frontendPort))

	if err != nil {
		return fmt.Errorf("error parsing frontend URL: %w", err)
	}

	_, sc, err := cf.CreateServerFromConfig(version)

	if err != nil {
		return fmt.Errorf("error loading server config: %w", err)
	}

	apiURL, err := url.Parse(fmt.Sprintf("http://localhost:%d", sc.Runtime.Port))

	if err != nil {
		return fmt.Errorf("error parsing API URL: %w", err)
	}

	// api process
	go func() {
		api.Start(cf, interruptCh, version) // nolint:errcheck
	}()

	// static file server
	go func() {
		c := staticfileserver.NewStaticFileServer(staticAssetDir)

		s := &http.Server{
			Addr:              fmt.Sprintf(":%s", frontendPort),
			Handler:           c,
			ReadHeaderTimeout: 5 * time.Second,
		}

		if err := s.ListenAndServe(); err != nil {
			log.Printf("static file server failure: %s", err.Error())
			os.Exit(1)
		}
	}()

	ctx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	go func() {
		if err := engine.Run(ctx, cf, version); err != nil {
			log.Printf("engine failure: %s", err.Error())
			os.Exit(1)
		}
	}()

	s := &http.Server{
		Addr:              fmt.Sprintf(":%s", runtimePort),
		ReadHeaderTimeout: 5 * time.Second,
		Handler: &httputil.ReverseProxy{
			Rewrite: func(r *httputil.ProxyRequest) {
				if strings.HasPrefix(r.In.URL.Path, "/api") {
					r.SetURL(apiURL)
				} else {
					r.SetURL(feURL)
				}
			},
		},
	}

	go func() {
		if err := s.ListenAndServe(); err != nil {
			log.Printf("reverse proxy failure: %s", err.Error())
			os.Exit(1)
		}
	}()

	<-interruptCh

	return nil
}
