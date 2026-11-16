// Package rpcstream holds helpers shared by the engine's streaming RPC handlers.
package rpcstream

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// ErrClosed is returned by Send once the handler that owns the stream has returned.
var ErrClosed = errors.New("stream is closed")

// closeGrace is how long Close lets an in-flight Send finish on its own before interrupting it,
// so a stream that ends while its last message is still being written ends cleanly.
const closeGrace = time.Second

// Stream is the sending half of a server stream, satisfied by *connect.ServerStream[T] and
// *connect.BidiStream[Req, T].
type Stream[T any] interface {
	Send(*T) error
}

type abortKey struct{}

// WithAbort returns a context carrying a function that interrupts a write blocked on the
// stream's transport, for example by expiring its write deadline. The server sets it for every
// request; NewSender picks it up.
func WithAbort(ctx context.Context, abort func()) context.Context {
	return context.WithValue(ctx, abortKey{}, abort)
}

// Sender serializes sends on a server stream and rejects sends once the owning handler has
// returned.
//
// Handlers hand their stream to other goroutines (the dispatcher sends assigned actions from
// message queue consumers, subscriptions send from fan-out goroutines). The HTTP/2 server
// panics on a write that arrives after the handler has returned, so every handler that shares
// its stream must wrap it in a Sender and defer Close: Close turns every later Send into
// ErrClosed and does not return while a Send is still writing. A Send to a peer that has
// stopped reading is interrupted after a short grace period, so a handler can always return.
type Sender[T any] struct {
	stream  Stream[T]
	abort   func()
	mu      sync.Mutex
	closing atomic.Bool
}

// NewSender wraps stream. ctx is the handler's context.
func NewSender[T any](ctx context.Context, stream Stream[T]) *Sender[T] {
	abort, _ := ctx.Value(abortKey{}).(func())

	return &Sender[T]{stream: stream, abort: abort}
}

func (s *Sender[T]) Send(msg *T) error {
	if s.closing.Load() {
		return ErrClosed
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closing.Load() {
		return ErrClosed
	}

	return s.stream.Send(msg)
}

// Close marks the stream as finished. It must be called before the handler returns, normally
// with defer.
func (s *Sender[T]) Close() {
	s.closing.Store(true)

	if s.mu.TryLock() {
		s.mu.Unlock() // nolint:staticcheck
		return
	}

	// a Send is writing: sends queued behind it fail as soon as they get the lock
	idle := make(chan struct{})

	go func() {
		s.mu.Lock()
		s.mu.Unlock() // nolint:staticcheck
		close(idle)
	}()

	select {
	case <-idle:
		return
	case <-time.After(closeGrace):
	}

	if s.abort != nil {
		s.abort()
	}

	<-idle
}
