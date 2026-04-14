package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-staticfileserver/staticfileserver"
)

func main() {
	port := flag.String("port", "80", "port to listen on")
	staticAssetDir := flag.String("static-asset-dir", ".", "directory to serve static assets from")
	basePath := flag.String("base-path", "/", "base path the app is served under (e.g. /hatchet)")
	flag.Parse()

	c := staticfileserver.NewStaticFileServer(*staticAssetDir, *basePath)

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
