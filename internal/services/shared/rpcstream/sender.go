// Package rpcstream holds helpers shared by the engine's streaming RPC handlers.
package rpcstream

import (
	"errors"
	"sync"
)

// ErrClosed is returned by Send once the handler that owns the stream has returned.
var ErrClosed = errors.New("stream is closed")

// Stream is the sending half of a server stream, satisfied by *connect.ServerStream[T] and
// *connect.BidiStream[Req, T].
type Stream[T any] interface {
	Send(*T) error
}

// Sender serializes sends on a server stream and rejects sends once the owning handler has
// returned.
//
// Handlers hand their stream to other goroutines (the dispatcher sends assigned actions from
// message queue consumers, subscriptions send from fan-out goroutines). The HTTP/2 server
// panics on a write that arrives after the handler has returned, so every handler that shares
// its stream must wrap it in a Sender and defer Close: Close waits for an in-flight Send and
// turns every later Send into ErrClosed.
type Sender[T any] struct {
	stream Stream[T]
	mu     sync.Mutex
	closed bool
}

func NewSender[T any](stream Stream[T]) *Sender[T] {
	return &Sender[T]{stream: stream}
}

func (s *Sender[T]) Send(msg *T) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return ErrClosed
	}

	return s.stream.Send(msg)
}

// Close marks the stream as finished. It must be called before the handler returns, normally
// with defer.
func (s *Sender[T]) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.closed = true
}
