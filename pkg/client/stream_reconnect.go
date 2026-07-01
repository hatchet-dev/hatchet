// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/rs/zerolog"
	"golang.org/x/sync/singleflight"

	"github.com/hatchet-dev/hatchet/pkg/client/retry"
)

const maxConsecutiveStreamNoProgress = 10

var errListenerClosed = errors.New("listener is closed")

type streamListenLabels struct {
	streamName    string
	reconnectVerb string
}

type streamNoProgressPolicy int

const (
	streamNoProgressRetriesUntilCap streamNoProgressPolicy = iota
	streamNoProgressStopsImmediately
)

type streamListenConfig[C any, E any] struct {
	reconnectContext     context.Context
	recv                 func(C) (E, error)
	handle               func(E) error
	shouldReconnectOnEOF func(context.Context) bool
	noProgressPolicy     streamNoProgressPolicy
	l                    *zerolog.Logger
	labels               streamListenLabels
}

func listenReconnectingStream[C any, E any](
	ctx context.Context,
	stream *reconnectingStream[C],
	cfg streamListenConfig[C, E],
) error {
	consecutiveNoProgress := 0
	reconnectAttempt := 0

	client, generation, ok := stream.getClientSnapshot()
	if !ok {
		return fmt.Errorf("client is not connected")
	}
	defer func() {
		if stream.closeSend != nil {
			if closeErr := stream.closeSend(client); closeErr != nil {
				cfg.l.Warn().Err(closeErr).Msgf("failed to close %s after listen exit", cfg.labels.streamName)
			}
		}
	}()

	verb := cfg.labels.reconnectVerb
	noProgressFormat := fmt.Sprintf("could not %s after %%d consecutive no-progress errors: %%w", verb)

	for {
		event, err := cfg.recv(client)

		if err != nil {
			eofPolicy := streamEOFStops
			if cfg.shouldReconnectOnEOF(ctx) {
				eofPolicy = streamEOFRetries
			}

			decision := classifyStreamRecvError(ctx, err, eofPolicy)

			switch decision {
			case retry.StreamDecisionStop:
				return nil
			case retry.StreamDecisionNoProgress:
				consecutiveNoProgress++
				if consecutiveNoProgress >= maxConsecutiveStreamNoProgress {
					return fmt.Errorf("stream made no progress after %d consecutive errors: %w", consecutiveNoProgress, err)
				}
				if cfg.noProgressPolicy == streamNoProgressStopsImmediately {
					return err
				}
			default:
				consecutiveNoProgress++
			}

			if _, genAfter, _ := stream.getClientSnapshot(); genAfter != generation {
				client, generation, _ = stream.getClientSnapshot()
				consecutiveNoProgress = 0
				reconnectAttempt = 0
				continue
			}

			if reconnectAttempt > 0 {
				if sleepErr := retry.SleepStreamBackoff(ctx, reconnectAttempt-1); sleepErr != nil {
					return nil
				}
			}

			retryErr := stream.retryConnectBackground(
				cfg.reconnectContext,
				func(err error, attempt int) {
					cfg.l.Error().Err(err).Msgf("could not %s to the %s (background attempt %d)", verb, cfg.labels.streamName, attempt)
				},
				noProgressFormat,
			)

			if retryErr != nil {
				if errors.Is(retryErr, errListenerClosed) {
					return nil
				}

				retryDecision := retry.ClassifyStreamError(ctx, retryErr)
				if streamDecisionStopsReconnect(retryDecision) {
					return fmt.Errorf("failed to %s: %w", verb, retryErr)
				}

				cfg.l.Error().Err(retryErr).Msgf("failed to %s (consecutive no-progress: %d/%d)", verb, consecutiveNoProgress, maxConsecutiveStreamNoProgress)

				if consecutiveNoProgress >= maxConsecutiveStreamNoProgress {
					return fmt.Errorf("failed to %s after %d consecutive errors: %w", verb, consecutiveNoProgress, retryErr)
				}

				reconnectAttempt++
				continue
			}

			client, generation, _ = stream.getClientSnapshot()
			consecutiveNoProgress = 0
			reconnectAttempt = 0
			continue
		}

		consecutiveNoProgress = 0
		reconnectAttempt = 0

		if err := cfg.handle(event); err != nil {
			return err
		}
	}
}

type streamEOFPolicy int

const (
	streamEOFStops streamEOFPolicy = iota
	streamEOFRetries
)

func classifyStreamRecvError(ctx context.Context, err error, eofPolicy streamEOFPolicy) retry.StreamDecision {
	if errors.Is(err, io.EOF) {
		if eofPolicy == streamEOFStops || ctx.Err() != nil {
			return retry.StreamDecisionStop
		}

		return retry.StreamDecisionRetry
	}

	return retry.ClassifyStreamError(ctx, err)
}

func streamDecisionStopsReconnect(decision retry.StreamDecision) bool {
	return decision == retry.StreamDecisionStop || decision == retry.StreamDecisionNoProgress
}

func sendListenerError(ctx context.Context, errCh chan<- error, err error) {
	select {
	case errCh <- err:
	case <-ctx.Done():
	}
}

type reconnectingStream[C any] struct {
	reconnectBackgroundGroup singleflight.Group
	client                   C
	lifecycleCtx             context.Context
	reconnectSyncGroup       singleflight.Group
	closeSend                func(C) error
	replay                   func(context.Context, C) error
	lifecycleCancel          context.CancelFunc
	constructor              func(context.Context) (C, error)
	generation               uint64
	reconnectMu              sync.Mutex
	clientMu                 sync.Mutex
	closeMu                  sync.Mutex
	hasClient                bool
	closed                   bool
}

func newReconnectingStream[C any](
	constructor func(context.Context) (C, error),
	closeSend func(C) error,
	replay func(context.Context, C) error,
) *reconnectingStream[C] {
	lifecycleCtx, lifecycleCancel := context.WithCancel(context.Background()) //nolint:gosec // Close owns lifecycleCancel.

	return &reconnectingStream[C]{
		constructor:     constructor,
		closeSend:       closeSend,
		replay:          replay,
		lifecycleCtx:    lifecycleCtx,
		lifecycleCancel: lifecycleCancel,
	}
}

func (s *reconnectingStream[C]) setInitialClient(client C) {
	s.clientMu.Lock()
	defer s.clientMu.Unlock()

	if s.hasClient {
		return
	}

	s.client = client
	s.hasClient = true
}

func (s *reconnectingStream[C]) lifecycleContext() context.Context {
	return s.lifecycleCtx
}

func (s *reconnectingStream[C]) isClosed() bool {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()
	return s.closed
}

func (s *reconnectingStream[C]) getClientSnapshot() (C, uint64, bool) {
	s.clientMu.Lock()
	defer s.clientMu.Unlock()
	return s.client, s.generation, s.hasClient
}

func (s *reconnectingStream[C]) installClient(client C) error {
	s.closeMu.Lock()
	if s.closed {
		s.closeMu.Unlock()
		return errListenerClosed
	}

	s.clientMu.Lock()
	oldClient := s.client
	hadOldClient := s.hasClient
	s.client = client
	s.hasClient = true
	s.generation++
	s.clientMu.Unlock()
	s.closeMu.Unlock()

	if hadOldClient && s.closeSend != nil {
		return s.closeSend(oldClient)
	}

	return nil
}

func (s *reconnectingStream[C]) connectAndReplay(ctx context.Context) error {
	s.reconnectMu.Lock()
	defer s.reconnectMu.Unlock()

	if s.isClosed() {
		return errListenerClosed
	}

	client, err := s.constructor(ctx)
	if err != nil {
		return err
	}

	if s.replay != nil {
		if err := s.replay(ctx, client); err != nil {
			if s.closeSend != nil {
				_ = s.closeSend(client)
			}
			return err
		}
	}

	if err := s.installClient(client); err != nil {
		if s.closeSend != nil {
			_ = s.closeSend(client)
		}
		return err
	}

	return nil
}

func (s *reconnectingStream[C]) retryConnectSync(
	ctx context.Context,
	logAttempt func(error, int),
	exhaustedFormat string,
) error {
	if s.isClosed() {
		return errListenerClosed
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	_, err, _ := s.reconnectSyncGroup.Do("reconnect", func() (interface{}, error) {
		return nil, retryStreamConnectSync(ctx, s.isClosed, s.connectAndReplay, logAttempt, exhaustedFormat)
	})
	return err
}

func (s *reconnectingStream[C]) retryConnectBackground(
	ctx context.Context,
	logAttempt func(error, int),
	noProgressFormat string,
) error {
	if s.isClosed() {
		return errListenerClosed
	}

	_, err, _ := s.reconnectBackgroundGroup.Do("reconnect", func() (interface{}, error) {
		return nil, retryStreamConnectBackground(ctx, s.isClosed, s.connectAndReplay, logAttempt, noProgressFormat)
	})
	return err
}

func (s *reconnectingStream[C]) retrySend(
	ctx context.Context,
	send func(C) error,
	logSendFailure func(error, int),
	logReconnectFailure func(error),
	logReconnectAttempt func(error, int),
	exhaustedFormat string,
) error {
	for attempt := 0; attempt < retry.StreamSyncMaxAttempts; attempt++ {
		client, generation, ok := s.getClientSnapshot()

		if !ok {
			return fmt.Errorf("client is not connected")
		}

		err := send(client)
		if err == nil {
			return nil
		}

		logSendFailure(err, attempt+1)

		if _, genAfter, _ := s.getClientSnapshot(); genAfter != generation {
			continue
		}

		if retryErr := s.retryConnectSync(ctx, logReconnectAttempt, "could not reconnect stream after %d retries"); retryErr != nil {
			logReconnectFailure(retryErr)
		}

		if sleepErr := retry.SleepStreamBackoff(ctx, attempt); sleepErr != nil {
			return sleepErr
		}
	}

	return fmt.Errorf(exhaustedFormat, retry.StreamSyncMaxAttempts)
}

func (s *reconnectingStream[C]) closeStream() error {
	s.clientMu.Lock()
	client := s.client
	ok := s.hasClient
	s.clientMu.Unlock()

	if !ok || s.closeSend == nil {
		return nil
	}

	return s.closeSend(client)
}

func (s *reconnectingStream[C]) Close() error {
	s.closeMu.Lock()
	s.closed = true
	s.closeMu.Unlock()

	if s.lifecycleCancel != nil {
		s.lifecycleCancel()
	}

	return s.closeStream()
}

func retryStreamConnectSync(
	ctx context.Context,
	isClosed func() bool,
	connect func(context.Context) error,
	logAttempt func(error, int),
	exhaustedFormat string,
) error {
	for attempt := 0; attempt < retry.StreamSyncMaxAttempts; attempt++ {
		if attempt > 0 {
			if err := retry.SleepStreamBackoff(ctx, attempt-1); err != nil {
				return err
			}
		}

		if isClosed() {
			return errListenerClosed
		}

		if err := ctx.Err(); err != nil {
			return err
		}

		err := connect(ctx)
		if err == nil {
			return nil
		}

		if retry.ClassifyStreamError(ctx, err) == retry.StreamDecisionStop {
			return err
		}

		logAttempt(err, attempt+1)
	}

	return fmt.Errorf(exhaustedFormat, retry.StreamSyncMaxAttempts)
}

func retryStreamConnectBackground(
	ctx context.Context,
	isClosed func() bool,
	connect func(context.Context) error,
	logAttempt func(error, int),
	noProgressFormat string,
) error {
	attempt := 0
	consecutiveNoProgress := 0

	for {
		if attempt > 0 {
			if err := retry.SleepStreamBackoff(ctx, attempt-1); err != nil {
				return err
			}
		}

		if isClosed() {
			return errListenerClosed
		}

		if err := ctx.Err(); err != nil {
			return err
		}

		err := connect(ctx)
		if err == nil {
			return nil
		}

		switch retry.ClassifyStreamError(ctx, err) {
		case retry.StreamDecisionStop:
			return err
		case retry.StreamDecisionNoProgress:
			consecutiveNoProgress++
			if consecutiveNoProgress >= maxConsecutiveStreamNoProgress {
				return fmt.Errorf(noProgressFormat, consecutiveNoProgress, err)
			}

			return err
		}

		consecutiveNoProgress = 0
		logAttempt(err, attempt+1)
		attempt++
	}
}
