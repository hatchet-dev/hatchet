package health

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
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

	checksMu sync.RWMutex
	checks   []readinessCheck
}

// readinessCheck is a named condition the readiness probe requires in addition to the queue
// and the repository.
type readinessCheck struct {
	name  string
	ready func() bool
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

// AddReadinessCheck adds a named condition /ready requires, for a service that starts after
// the health server (the in-engine serverless operator). While the check reports false the
// probe answers 503 and logs the check's name; /live is not affected.
func (h *Health) AddReadinessCheck(name string, ready func() bool) {
	h.checksMu.Lock()
	defer h.checksMu.Unlock()

	h.checks = append(h.checks, readinessCheck{name: name, ready: ready})
}

// failedReadinessCheck returns the name of the first added check that reports not ready.
func (h *Health) failedReadinessCheck() (string, bool) {
	h.checksMu.RLock()
	defer h.checksMu.RUnlock()

	for _, check := range h.checks {
		if !check.ready() {
			return check.name, true
		}
	}

	return "", false
}

func (h *Health) Start(port int) (func() error, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		queueReady := h.queue.IsReady()
		repositoryReady := h.repository.IsHealthy(ctx)

		if !queueReady || !repositoryReady {
			h.l.Error().Ctx(ctx).Msgf("liveness check failed - queue ready: %t, repository ready: %t", queueReady, repositoryReady)
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
		failedCheck, checkFailed := h.failedReadinessCheck()

		if h.shuttingDown || !queueReady || !repositoryReady || checkFailed {
			if !h.shuttingDown {
				h.l.Error().Ctx(ctx).Msgf("readiness check failed - queue ready: %t, repository ready: %t, failed check: %q", queueReady, repositoryReady, failedCheck)
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
