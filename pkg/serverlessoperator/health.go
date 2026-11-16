package serverlessoperator

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

// healthServer serves /healthz (process up), /readyz (first rebalance tick done) and
// /metrics on the health port.
type healthServer struct {
	srv   *http.Server
	l     *zerolog.Logger
	ready atomic.Bool
}

func newHealthServer(port int, l *zerolog.Logger) *healthServer {
	h := &healthServer{l: l}

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if !h.ready.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("not ready"))

			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})

	mux.Handle("/metrics", promhttp.Handler())

	h.srv = &http.Server{
		Addr:              net.JoinHostPort("", fmt.Sprint(port)),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return h
}

func (h *healthServer) setReady() {
	h.ready.Store(true)
}

// serve blocks until the server stops. A closed server is not an error.
func (h *healthServer) serve() error {
	err := h.srv.ListenAndServe()

	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return err
}

func (h *healthServer) shutdown(ctx context.Context) {
	if err := h.srv.Shutdown(ctx); err != nil {
		h.l.Warn().Err(err).Msg("health server shutdown")
	}
}
