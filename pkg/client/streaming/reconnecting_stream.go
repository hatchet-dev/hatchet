// Package streaming holds the machinery every long-lived gRPC stream client in
// this module shares: a stream whose underlying client is replaced on
// reconnect, the receive loop that drives it with backoff, and the error
// classification that decides between reconnecting and stopping.
package streaming

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/rs/zerolog"
	"golang.org/x/sync/singleflight"

	"github.com/hatchet-dev/hatchet/pkg/client/retry"
)

// ErrStreamNotConnected is returned by a send on a stream that has no client
// installed.
var ErrStreamNotConnected = errors.New("client is not connected")

// ReconnectingStream is one logical gRPC stream whose underlying client can be
// replaced on reconnect. connectGroup (singleflight) coalesces concurrent
// connect attempts so only one replacement stream opens. Connect runs outside
// mu so snapshots do not block behind network I/O. replay re-sends current
// subscriptions on a fresh stream before install. lifecycleCtx lasts until
// Close; caller contexts must not kill shared listeners.
// NOTE: field order follows govet fieldalignment (enforced by the pre-commit
// autofixer); mu guards client, generation, hasClient, and closed.
type ReconnectingStream[C any] struct {
	lifecycleCtx context.Context
	client       C
	connectGroup singleflight.Group

	// sleep waits out the backoff delay for the given attempt. Tests inject a
	// no-op through SetSleep; production uses retry.SleepStreamBackoff.
	sleep func(ctx context.Context, attempt int) error

	constructor     func(context.Context) (C, error)
	lifecycleCancel context.CancelFunc
	replay          func(context.Context, C) error
	closeSend       func(C) error
	l               *zerolog.Logger

	// name identifies the stream in log messages ("workflow run listener", …).
	name string

	generation uint64
	mu         sync.Mutex
	// sendMu serializes SendMsg and CloseSend on published clients: grpc-go
	// allows only one concurrent sender, and CloseSend must not run
	// concurrently with SendMsg. It also makes replay, publication, and
	// retirement one atomic handoff. Never held during construction or
	// backoff. Lock order is sendMu → mu, never reverse.
	sendMu    sync.Mutex
	hasClient bool
	closed    bool
}

// NewReconnectingStream builds a stream whose lifecycle ends only with Close.
// constructor opens a new client, closeSend half-closes a retired one, and
// replay (optional) brings a fresh client up to date before it is published.
func NewReconnectingStream[C any](
	l *zerolog.Logger,
	name string,
	constructor func(context.Context) (C, error),
	closeSend func(C) error,
	replay func(context.Context, C) error,
) *ReconnectingStream[C] {
	return NewReconnectingStreamWithLifecycle(context.Background(), l, name, constructor, closeSend, replay)
}

// NewReconnectingStreamWithLifecycle is NewReconnectingStream with a parent
// for the lifecycle context, so cancelling parent also ends the stream.
func NewReconnectingStreamWithLifecycle[C any](
	parent context.Context,
	l *zerolog.Logger,
	name string,
	constructor func(context.Context) (C, error),
	closeSend func(C) error,
	replay func(context.Context, C) error,
) *ReconnectingStream[C] {
	lifecycleCtx, lifecycleCancel := context.WithCancel(parent) // nolint: gosec // lifecycleCancel is stored on the struct and called by Close

	return &ReconnectingStream[C]{
		constructor:     constructor,
		closeSend:       closeSend,
		replay:          replay,
		sleep:           retry.SleepStreamBackoff,
		lifecycleCtx:    lifecycleCtx,
		lifecycleCancel: lifecycleCancel,
		name:            name,
		l:               l,
	}
}

// SetSleep replaces the backoff sleep used between reconnect and send
// attempts. Tests use it to disable backoff; it is not safe to call once the
// stream is in use.
func (s *ReconnectingStream[C]) SetSleep(sleep func(ctx context.Context, attempt int) error) {
	s.sleep = sleep
}

// Name is the stream's name in log messages.
func (s *ReconnectingStream[C]) Name() string {
	return s.name
}

// Snapshot returns the current client, its generation, and whether one is
// installed. It never blocks behind network I/O.
func (s *ReconnectingStream[C]) Snapshot() (client C, generation uint64, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.client, s.generation, s.hasClient
}

// SetInitialClient installs client without a connect when none is installed
// yet; a later ConnectOnce replaces it like any other.
func (s *ReconnectingStream[C]) SetInitialClient(client C) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hasClient {
		return
	}

	s.client = client
	s.hasClient = true
}

// installClientLocked publishes client and retires the previous client. The
// caller must hold sendMu.
func (s *ReconnectingStream[C]) installClientLocked(client C) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return ErrListenerClosed
	}

	oldClient := s.client
	hadOldClient := s.hasClient
	s.client = client
	s.hasClient = true
	s.generation++
	s.mu.Unlock()

	if hadOldClient && s.closeSend != nil {
		err := s.closeSend(oldClient)
		if err != nil {
			s.l.Warn().Err(err).Str("stream", s.name).Msg("failed to close replaced stream client")
		}
	}

	return nil
}

// IsClosed reports whether Close has been called.
func (s *ReconnectingStream[C]) IsClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// LifecycleContext is the context that only Close cancels. Reconnects and
// background loops that must outlive any one caller use it.
func (s *ReconnectingStream[C]) LifecycleContext() context.Context {
	return s.lifecycleCtx
}

// ConnectOnce makes one connect attempt, coalesced with any concurrent one,
// and publishes the new client after replay.
func (s *ReconnectingStream[C]) ConnectOnce(ctx context.Context) error {
	_, err, _ := s.connectGroup.Do("connect", func() (interface{}, error) {
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			return nil, ErrListenerClosed
		}
		s.mu.Unlock()

		client, err := s.constructor(ctx)
		if err != nil {
			return nil, err
		}

		s.sendMu.Lock()
		defer s.sendMu.Unlock()

		if s.replay != nil {
			if err := s.replay(ctx, client); err != nil {
				if s.closeSend != nil {
					_ = s.closeSend(client)
				}
				return nil, err
			}
		}

		if err := s.installClientLocked(client); err != nil {
			if s.closeSend != nil {
				_ = s.closeSend(client)
			}
			return nil, err
		}

		return nil, nil
	})
	return err
}

// ConnectSync connects with bounded retries and backoff, returning the last
// error when every attempt fails.
func (s *ReconnectingStream[C]) ConnectSync(ctx context.Context) error {
	if s.IsClosed() {
		return ErrListenerClosed
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	var lastErr error
	for attempt := 0; attempt < retry.StreamSyncMaxAttempts; attempt++ {
		if attempt > 0 {
			if err := s.sleep(ctx, attempt-1); err != nil {
				return err
			}
		}

		if s.IsClosed() {
			return ErrListenerClosed
		}

		if err := ctx.Err(); err != nil {
			return err
		}

		err := s.ConnectOnce(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		if errors.Is(err, ErrListenerClosed) || retry.ClassifyStreamError(ctx, err) == retry.StreamDecisionStop {
			return err
		}

		s.l.Error().Err(err).Str("stream", s.name).Int("attempt", attempt+1).
			Msg("stream connect attempt failed")
	}

	return fmt.Errorf("could not connect to %s after %d attempts: %w", s.name, retry.StreamSyncMaxAttempts, lastErr)
}

// RetrySend sends with bounded retries. Each failed attempt makes at most one
// reconnect attempt (ConnectOnce, coalesced with any concurrent reconnect via
// singleflight) before backing off, so the total budget is
// StreamSyncMaxAttempts sends, at most StreamSyncMaxAttempts reconnects, and
// at most StreamSyncMaxAttempts-1 backoff sleeps.
// A reconnect failure that is permanent (ErrListenerClosed or classified
// StreamDecisionStop) short-circuits immediately.
func (s *ReconnectingStream[C]) RetrySend(ctx context.Context, send func(C) error) error {
	var lastErr error
	for attempt := 0; attempt < retry.StreamSyncMaxAttempts; attempt++ {
		var gen uint64
		err := func() error {
			s.sendMu.Lock()
			defer s.sendMu.Unlock()

			client, g, ok := s.Snapshot()
			if !ok {
				return ErrStreamNotConnected
			}
			gen = g
			return send(client)
		}()
		if err == nil {
			return nil
		}
		if errors.Is(err, ErrStreamNotConnected) {
			return err
		}

		lastErr = err
		s.l.Warn().Err(err).Str("stream", s.name).Int("attempt", attempt+1).Msg("stream send failed")

		if _, genAfter, _ := s.Snapshot(); genAfter != gen {
			continue
		}

		if rerr := s.ConnectOnce(ctx); rerr != nil {
			if errors.Is(rerr, ErrListenerClosed) || retry.ClassifyStreamError(ctx, rerr) == retry.StreamDecisionStop {
				return fmt.Errorf("could not reconnect %s to retry send: %w", s.name, rerr)
			}
			s.l.Error().Err(rerr).Str("stream", s.name).Msg("stream reconnect after send failure failed")
		}

		if attempt < retry.StreamSyncMaxAttempts-1 {
			if serr := s.sleep(ctx, attempt); serr != nil {
				return serr
			}
		}
	}

	return fmt.Errorf("could not send to %s after %d attempts: %w", s.name, retry.StreamSyncMaxAttempts, lastErr)
}

// SendOnce performs one send on the current client under sendMu and never
// reconnects. Callers that keep their own record of what was sent use it so
// a reconnect's replay is the only path that sends the same message again.
func (s *ReconnectingStream[C]) SendOnce(send func(C) error) error {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()

	client, _, ok := s.Snapshot()
	if !ok {
		return ErrStreamNotConnected
	}

	return send(client)
}

// CloseStream half-closes the current client without closing the stream, so
// the receive loop reconnects.
func (s *ReconnectingStream[C]) CloseStream() error {
	client, _, ok := s.Snapshot()
	if !ok || s.closeSend == nil {
		return nil
	}

	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	return s.closeSend(client)
}

// Close ends the stream for good: the lifecycle context is cancelled and the
// current client is half-closed.
func (s *ReconnectingStream[C]) Close() error {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()

	if s.lifecycleCancel != nil {
		s.lifecycleCancel()
	}

	return s.CloseStream()
}
