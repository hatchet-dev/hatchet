package streams

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

// Registry tracks cancellation functions for long-lived gRPC streams (e.g. workflow
// run subscriptions and durable event listeners) so they can be hung up during
// graceful shutdown. grpc.Server.GracefulStop waits for open streams to finish;
// without an active hangup, subscriber streams would block shutdown until the
// process is killed.
type Registry struct {
	sessions map[uuid.UUID]context.CancelFunc
	mu       sync.Mutex
}

func NewRegistry() *Registry {
	return &Registry{
		sessions: make(map[uuid.UUID]context.CancelFunc),
	}
}

// Register adds a stream session's cancel function to the registry and returns a
// deregister function which the stream handler must defer.
func (r *Registry) Register(cancel context.CancelFunc) (deregister func()) {
	id := uuid.New()

	r.mu.Lock()
	r.sessions[id] = cancel
	r.mu.Unlock()

	return func() {
		r.mu.Lock()
		delete(r.sessions, id)
		r.mu.Unlock()
	}
}

// CancelAll cancels every registered session. It is called once during shutdown.
func (r *Registry) CancelAll() {
	r.mu.Lock()
	cancels := make([]context.CancelFunc, 0, len(r.sessions))

	for _, cancel := range r.sessions {
		cancels = append(cancels, cancel)
	}

	r.sessions = make(map[uuid.UUID]context.CancelFunc)
	r.mu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
}
