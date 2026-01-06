package health

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

type Health struct {
	shuttingDown bool
	version      string

	repository v1.HealthRepository
	queue      msgqueue.MessageQueue
	l          *zerolog.Logger
}

func New(repo v1.HealthRepository, queue msgqueue.MessageQueue, version string, l *zerolog.Logger) *Health {
	return &Health{
		version:    version,
		repository: repo,
		queue:      queue,
		l:          l,
	}
}

func (h *Health) SetShuttingDown(shuttingDown bool) {
	h.shuttingDown = shuttingDown
}

func (h *Health) Start(port int) (func() error, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		queueReady := h.queue.IsReady()
		repositoryReady := h.repository.IsHealthy(ctx)

		if !queueReady || !repositoryReady {
			h.l.Error().Msgf("liveness check failed - queue ready: %t, repository ready: %t", queueReady, repositoryReady)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		queueReady := h.queue.IsReady()
		repositoryReady := h.repository.IsHealthy(ctx)

		if h.shuttingDown || !queueReady || !repositoryReady {
			if !h.shuttingDown {
				h.l.Error().Msgf("readiness check failed - queue ready: %t, repository ready: %t", queueReady, repositoryReady)
			}

			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {

		w.WriteHeader(http.StatusOK)
		e := json.NewEncoder(w).Encode(map[string]string{"version": h.version})
		if e != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	})
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	l, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return nil, fmt.Errorf("could not listen on %s: %w", server.Addr, err)
	}
	go func() {
		if err := server.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	cleanup := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			return fmt.Errorf("could not shutdown server: %w", err)
		}
		return nil
	}

	return cleanup, nil
}
