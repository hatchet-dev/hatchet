package prometheus

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

var (
	serversMu sync.Mutex
	servers   = map[string]*metricsServer{}
)

type metricsServer struct {
	srv  *http.Server
	refs int
}

// StartServer serves the default registry at path on address and returns a function that stops
// it. Every Hatchet service in a process shares that registry, so a process that runs several of
// them (hatchet-lite runs the API and the engine) serves each address once: later calls share the
// first server, which keeps its path, and the server stops when the last caller stops it.
func StartServer(l *zerolog.Logger, address, path string) func() error {
	serversMu.Lock()
	defer serversMu.Unlock()

	if s, ok := servers[address]; ok {
		s.refs++
		return stopServer(l, address, s)
	}

	mux := http.NewServeMux()
	mux.Handle(path, promhttp.Handler())

	s := &metricsServer{
		srv: &http.Server{
			Addr:              address,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
		refs: 1,
	}
	servers[address] = s

	go func() {
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			l.Error().Err(err).Msg("failed to start prometheus server")
		}
	}()

	l.Info().Msgf("Prometheus server started on %s", address)

	return stopServer(l, address, s)
}

func stopServer(l *zerolog.Logger, address string, s *metricsServer) func() error {
	var once sync.Once

	return func() error {
		var err error

		once.Do(func() {
			serversMu.Lock()
			s.refs--
			last := s.refs == 0
			if last {
				delete(servers, address)
			}
			serversMu.Unlock()

			if !last {
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if shutdownErr := s.srv.Shutdown(ctx); shutdownErr != nil {
				err = fmt.Errorf("failed to shutdown prometheus server: %w", shutdownErr)
				return
			}

			l.Info().Msg("Prometheus server shutdown gracefully")
		})

		return err
	}
}
