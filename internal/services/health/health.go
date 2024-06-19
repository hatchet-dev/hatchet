package health

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/pkg/repository"
)

type Health struct {
	ready bool

	repository repository.EngineRepository
	queue      msgqueue.MessageQueue
}

func New(prisma repository.EngineRepository, queue msgqueue.MessageQueue) *Health {
	return &Health{
		repository: prisma,
		queue:      queue,
	}
}

func (h *Health) SetReady(ready bool) {
	h.ready = ready
}

func (h *Health) Start() (func() error, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		if !h.ready || !h.queue.IsReady() || !h.repository.Health().IsHealthy() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if !h.ready || !h.queue.IsReady() || !h.repository.Health().IsHealthy() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:         ":8733",
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
