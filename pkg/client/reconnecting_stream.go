// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/rs/zerolog"
	"golang.org/x/sync/singleflight"

	"github.com/hatchet-dev/hatchet/pkg/client/retry"
)

var errStreamNotConnected = errors.New("client is not connected")

// reconnectingStream is one logical gRPC stream whose underlying client can be
// replaced on reconnect. connectGroup (singleflight) coalesces concurrent
// connect attempts so only one replacement stream opens. Connect runs outside
// mu so snapshots do not block behind network I/O. replay re-sends current
// subscriptions on a fresh stream before install. lifecycleCtx lasts until
// Close; caller contexts must not kill shared listeners.
// NOTE: field order follows govet fieldalignment (enforced by the pre-commit
// autofixer); mu guards client, generation, hasClient, and closed.
type reconnectingStream[C any] struct {
	lifecycleCtx context.Context
	client       C
	connectGroup singleflight.Group

	// sleep waits out the backoff delay for the given attempt. Tests inject a
	// no-op; production uses retry.SleepStreamBackoff.
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
	// concurrently with SendMsg. Held only around those calls, never across
	// reconnect or backoff. Lock order is sendMu → mu, never reverse.
	sendMu    sync.Mutex
	hasClient bool
	closed    bool
}

func newReconnectingStream[C any](
	l *zerolog.Logger,
	name string,
	constructor func(context.Context) (C, error),
	closeSend func(C) error,
	replay func(context.Context, C) error,
) *reconnectingStream[C] {
	return newReconnectingStreamWithLifecycle(context.Background(), l, name, constructor, closeSend, replay)
}

func newReconnectingStreamWithLifecycle[C any](
	parent context.Context,
	l *zerolog.Logger,
	name string,
	constructor func(context.Context) (C, error),
	closeSend func(C) error,
	replay func(context.Context, C) error,
) *reconnectingStream[C] {
	lifecycleCtx, lifecycleCancel := context.WithCancel(parent) // nolint: gosec // lifecycleCancel is stored on the struct and called by Close

	return &reconnectingStream[C]{
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

func (s *reconnectingStream[C]) snapshot() (client C, generation uint64, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.client, s.generation, s.hasClient
}

func (s *reconnectingStream[C]) setInitialClient(client C) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hasClient {
		return
	}

	s.client = client
	s.hasClient = true
}

func (s *reconnectingStream[C]) installClient(client C) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return errListenerClosed
	}

	oldClient := s.client
	hadOldClient := s.hasClient
	s.client = client
	s.hasClient = true
	s.generation++
	s.mu.Unlock()

	if hadOldClient && s.closeSend != nil {
		s.sendMu.Lock()
		err := s.closeSend(oldClient)
		s.sendMu.Unlock()
		if err != nil {
			s.l.Warn().Err(err).Str("stream", s.name).Msg("failed to close replaced stream client")
		}
	}

	return nil
}

func (s *reconnectingStream[C]) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

func (s *reconnectingStream[C]) lifecycleContext() context.Context {
	return s.lifecycleCtx
}

func (s *reconnectingStream[C]) connectOnce(ctx context.Context) error {
	_, err, _ := s.connectGroup.Do("connect", func() (interface{}, error) {
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			return nil, errListenerClosed
		}
		s.mu.Unlock()

		client, err := s.constructor(ctx)
		if err != nil {
			return nil, err
		}

		if s.replay != nil {
			if err := s.replay(ctx, client); err != nil {
				if s.closeSend != nil {
					_ = s.closeSend(client)
				}
				return nil, err
			}
		}

		if err := s.installClient(client); err != nil {
			if s.closeSend != nil {
				_ = s.closeSend(client)
			}
			return nil, err
		}

		return nil, nil
	})
	return err
}

func (s *reconnectingStream[C]) connectSync(ctx context.Context) error {
	if s.isClosed() {
		return errListenerClosed
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

		if s.isClosed() {
			return errListenerClosed
		}

		if err := ctx.Err(); err != nil {
			return err
		}

		err := s.connectOnce(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		if errors.Is(err, errListenerClosed) || retry.ClassifyStreamError(ctx, err) == retry.StreamDecisionStop {
			return err
		}

		s.l.Error().Err(err).Str("stream", s.name).Int("attempt", attempt+1).
			Msg("stream connect attempt failed")
	}

	return fmt.Errorf("could not connect to %s after %d attempts: %w", s.name, retry.StreamSyncMaxAttempts, lastErr)
}

// retrySend sends with bounded retries. Each failed attempt makes at most one
// reconnect attempt (connectOnce, coalesced with any concurrent reconnect via
// singleflight) before backing off, so the total budget is
// StreamSyncMaxAttempts sends, at most StreamSyncMaxAttempts reconnects, and
// at most StreamSyncMaxAttempts-1 backoff sleeps.
// A reconnect failure that is permanent (errListenerClosed or classified
// StreamDecisionStop) short-circuits immediately.
func (s *reconnectingStream[C]) retrySend(ctx context.Context, send func(C) error) error {
	var lastErr error
	for attempt := 0; attempt < retry.StreamSyncMaxAttempts; attempt++ {
		var gen uint64
		err := func() error {
			s.sendMu.Lock()
			defer s.sendMu.Unlock()

			client, g, ok := s.snapshot()
			if !ok {
				return errStreamNotConnected
			}
			gen = g
			return send(client)
		}()
		if err == nil {
			return nil
		}
		if errors.Is(err, errStreamNotConnected) {
			return err
		}

		lastErr = err
		s.l.Warn().Err(err).Str("stream", s.name).Int("attempt", attempt+1).Msg("stream send failed")

		if _, genAfter, _ := s.snapshot(); genAfter != gen {
			continue
		}

		if rerr := s.connectOnce(ctx); rerr != nil {
			if errors.Is(rerr, errListenerClosed) || retry.ClassifyStreamError(ctx, rerr) == retry.StreamDecisionStop {
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

func (s *reconnectingStream[C]) closeStream() error {
	client, _, ok := s.snapshot()
	if !ok || s.closeSend == nil {
		return nil
	}

	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	return s.closeSend(client)
}

func (s *reconnectingStream[C]) Close() error {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()

	if s.lifecycleCancel != nil {
		s.lifecycleCancel()
	}

	return s.closeStream()
}
