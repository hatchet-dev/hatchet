package health

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
)

type Health struct {
	ready bool

	repository repository.Repository
	queue      taskqueue.TaskQueue
}

func New(prisma repository.Repository, queue taskqueue.TaskQueue) *Health {
	return &Health{
		repository: prisma,
		queue:      queue,
	}
}

func (h *Health) SetReady(ready bool) {
	h.ready = ready
}

func (h *Health) Start() func() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if !h.ready || !h.queue.IsReady() || !h.repository.Health().IsHealthy() || !h.repository.Health().IsHealthy() {
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

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
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

	return cleanup
}
