package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-api/api"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-engine/engine"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-lite/staticfileserver"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
)

var printVersion bool
var configDirectory string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hatchet-lite",
	Short: "hatchet-lite runs a Hatchet instance with static files, API and engine all served on the same instance.",
	Run: func(cmd *cobra.Command, args []string) {
		if printVersion {
			fmt.Println(Version)
			os.Exit(0)
		}

		cf := loader.NewConfigLoader(configDirectory)
		interruptChan := cmdutils.InterruptChan()

		if err := start(cf, interruptChan, Version); err != nil {
			log.Println("error starting API:", err)
			os.Exit(1)
		}
	},
}

// Version will be linked by an ldflag during build
var Version = "v0.1.0-alpha.0"

func main() {
	rootCmd.PersistentFlags().BoolVar(
		&printVersion,
		"version",
		false,
		"print version and exit.",
	)

	rootCmd.PersistentFlags().StringVar(
		&configDirectory,
		"config",
		"",
		"The path the config folder.",
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// runs a static file server, api and engine in the same process.
func start(cf *loader.ConfigLoader, interruptCh <-chan interface{}, version string) error {
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
