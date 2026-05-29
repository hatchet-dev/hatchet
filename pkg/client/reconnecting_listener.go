// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var errListenerClosed = errors.New("listener is closed")

type bidiStream[Req any, Ev any] interface {
	Send(Req) error
	Recv() (Ev, error)
	CloseSend() error
}

type reconnectingListenerRetryPolicy struct {
	maxConsecutiveReconnectErrors int
	unavailableDelay              time.Duration
	disableUnavailableDelay       bool
	subscribeRetryCount           int
}

type reconnectingListener[K comparable, Req any, Ev any, S bidiStream[Req, Ev]] struct {
	client         S
	clientCancel   context.CancelFunc
	reconnectGroup singleflight.Group
	constructor    func(context.Context) (S, error)
	l              *zerolog.Logger
	requestForKey  func(K) Req
	keyForEvent    func(Ev) K
	handlers       sync.Map
	retryPolicy    reconnectingListenerRetryPolicy
	closed         bool
	generation     uint64
	clientMu       sync.Mutex
}

type handlerSet[Ev any] struct {
	handlers map[string]func(Ev) error
	mu       sync.RWMutex
}

func (l *reconnectingListener[K, Req, Ev, S]) maxConsecutiveReconnectErrors() int {
	if l.retryPolicy.maxConsecutiveReconnectErrors > 0 {
		return l.retryPolicy.maxConsecutiveReconnectErrors
	}

	return 10
}

func (l *reconnectingListener[K, Req, Ev, S]) unavailableDelay() time.Duration {
	if l.retryPolicy.disableUnavailableDelay {
		return 0
	}

	if l.retryPolicy.unavailableDelay > 0 {
		return l.retryPolicy.unavailableDelay
	}

	return 1 * time.Second
}

func (l *reconnectingListener[K, Req, Ev, S]) subscribeRetryCount() int {
	if l.retryPolicy.subscribeRetryCount > 0 {
		return l.retryPolicy.subscribeRetryCount
	}

	return DefaultActionListenerRetryCount
}

func (l *reconnectingListener[K, Req, Ev, S]) getClientSnapshot() (S, uint64, bool) {
	l.clientMu.Lock()
	defer l.clientMu.Unlock()
	return l.client, l.generation, l.closed
}

func closeStream[Req any, Ev any, S bidiStream[Req, Ev]](client S, cancel context.CancelFunc) error {
	if isNil(client) {
		if cancel != nil {
			cancel()
		}

		return nil
	}

	err := client.CloseSend()
	if cancel != nil {
		cancel()
	}

	return err
}

func (l *reconnectingListener[K, Req, Ev, S]) retrySubscribe(ctx context.Context) error {
	_, err, _ := l.reconnectGroup.Do("reconnect", func() (interface{}, error) {
		return nil, l.doRetrySubscribe(ctx)
	})

	return err
}

func (l *reconnectingListener[K, Req, Ev, S]) doRetrySubscribe(ctx context.Context) error {
	retries := 0
	retryCount := l.subscribeRetryCount()

	for retries < retryCount {
		l.clientMu.Lock()
		if l.closed {
			l.clientMu.Unlock()
			return errListenerClosed
		}
		l.clientMu.Unlock()

		if retries > 0 {
			time.Sleep(DefaultActionListenerRetryInterval)

			l.clientMu.Lock()
			if l.closed {
				l.clientMu.Unlock()
				return errListenerClosed
			}
			l.clientMu.Unlock()
		}

		streamCtx, cancel := context.WithCancel(ctx)
		client, err := l.constructor(streamCtx)

		if err != nil {
			cancel()
			retries++
			l.l.Error().Err(err).Msg("could not resubscribe to the listener")
			continue
		}

		var rangeErr error

		l.handlers.Range(func(key, value interface{}) bool {
			k := key.(K)

			err := client.Send(l.requestForKey(k))
			if err != nil {
				l.l.Error().Err(err).Msg("could not resubscribe to the worker")
				rangeErr = err
				return false
			}

			return true
		})

		if rangeErr != nil {
			if err := closeStream[Req, Ev, S](client, cancel); err != nil {
				l.l.Warn().Err(err).Msg("failed to close listener stream after resubscribe failure")
			}

			retries++
			continue
		}

		l.clientMu.Lock()
		if l.closed {
			l.clientMu.Unlock()
			if err := closeStream[Req, Ev, S](client, cancel); err != nil {
				l.l.Warn().Err(err).Msg("failed to close listener stream after close")
			}

			return errListenerClosed
		}

		oldClient := l.client
		oldCancel := l.clientCancel
		l.client = client
		l.clientCancel = cancel
		l.generation++
		l.clientMu.Unlock()

		if err := closeStream[Req, Ev, S](oldClient, oldCancel); err != nil {
			l.l.Warn().Err(err).Msg("failed to close replaced listener stream")
		}

		return nil
	}

	return fmt.Errorf("could not subscribe to the worker after %d retries", retries)
}

func (l *reconnectingListener[K, Req, Ev, S]) addHandler(key K, handlerId string, handler func(Ev) error) error {
	handlers, _ := l.handlers.LoadOrStore(key, &handlerSet[Ev]{
		handlers: map[string]func(Ev) error{},
	})

	h := handlers.(*handlerSet[Ev])

	h.mu.Lock()
	h.handlers[handlerId] = handler
	l.handlers.Store(key, h)
	h.mu.Unlock()

	return l.retrySend(key)
}

func (l *reconnectingListener[K, Req, Ev, S]) removeHandler(key K, handlerId string) {
	handlers, ok := l.handlers.Load(key)

	if !ok {
		return
	}

	h := handlers.(*handlerSet[Ev])

	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.handlers, handlerId)

	if len(h.handlers) == 0 {
		l.handlers.Delete(key)
	}
}

func (l *reconnectingListener[K, Req, Ev, S]) retrySend(key K) error {
	for i := 0; i < DefaultActionListenerRetryCount; i++ {
		client, genBefore, closed := l.getClientSnapshot()

		if closed {
			return errListenerClosed
		}

		if isNil(client) {
			return fmt.Errorf("client is not connected")
		}

		err := client.Send(l.requestForKey(key))

		if err == nil {
			return nil
		}

		l.l.Warn().Err(err).Msgf("failed to send listener subscription, attempt %d/%d", i+1, DefaultActionListenerRetryCount)

		if _, genAfter, closed := l.getClientSnapshot(); closed {
			return errListenerClosed
		} else if genAfter != genBefore {
			continue
		}

		if retryErr := l.retrySubscribe(context.Background()); retryErr != nil {
			if errors.Is(retryErr, errListenerClosed) {
				return retryErr
			}

			l.l.Error().Err(retryErr).Msg("failed to resubscribe after send failure")
			time.Sleep(DefaultActionListenerRetryInterval)
		}
	}

	return fmt.Errorf("could not send to the worker after %d retries", DefaultActionListenerRetryCount)
}

func (l *reconnectingListener[K, Req, Ev, S]) Listen(ctx context.Context) error {
	consecutiveErrors := 0
	maxConsecutiveErrors := l.maxConsecutiveReconnectErrors()

	client, generation, closed := l.getClientSnapshot()

	if closed {
		return nil
	}

	if isNil(client) {
		return fmt.Errorf("client is not connected")
	}

	for {
		event, err := client.Recv()

		if err != nil {
			nextClient, nextGeneration, closed := l.getClientSnapshot()
			if closed {
				return nil
			}

			if nextGeneration != generation {
				if isNil(nextClient) {
					return fmt.Errorf("client is not connected")
				}

				client = nextClient
				generation = nextGeneration
				consecutiveErrors = 0
				continue
			}

			if errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled {
				return nil
			}

			consecutiveErrors++

			if delay := l.unavailableDelay(); delay > 0 && status.Code(err) == codes.Unavailable {
				l.l.Warn().Err(err).Msg("dispatcher is unavailable, retrying subscribe after 1 second")
				time.Sleep(delay)
			}

			retryErr := l.retrySubscribe(ctx)

			if retryErr != nil {
				if errors.Is(retryErr, errListenerClosed) {
					return nil
				}

				l.l.Error().Err(retryErr).Msgf("failed to resubscribe (consecutive errors: %d/%d)", consecutiveErrors, maxConsecutiveErrors)

				if consecutiveErrors >= maxConsecutiveErrors {
					return fmt.Errorf("failed to resubscribe after %d consecutive errors: %w", consecutiveErrors, retryErr)
				}

				time.Sleep(DefaultActionListenerRetryInterval)
				continue
			}

			client, generation, closed = l.getClientSnapshot()
			if closed {
				return nil
			}

			if isNil(client) {
				return fmt.Errorf("client is not connected")
			}

			consecutiveErrors = 0
			continue
		}

		consecutiveErrors = 0

		if err := l.handleEvent(event); err != nil {
			return err
		}
	}
}

func (l *reconnectingListener[K, Req, Ev, S]) Close() error {
	l.clientMu.Lock()
	if l.closed {
		l.clientMu.Unlock()
		return nil
	}

	l.closed = true
	client := l.client
	cancel := l.clientCancel

	var zero S
	l.client = zero
	l.clientCancel = nil
	l.generation++
	l.clientMu.Unlock()

	return closeStream[Req, Ev, S](client, cancel)
}

func (l *reconnectingListener[K, Req, Ev, S]) handleEvent(event Ev) error {
	handlers, ok := l.handlers.Load(l.keyForEvent(event))

	if !ok {
		return nil
	}

	h := handlers.(*handlerSet[Ev])

	h.mu.RLock()
	handlerCopies := make([]func(Ev) error, 0, len(h.handlers))

	for _, handler := range h.handlers {
		handlerCopies = append(handlerCopies, handler)
	}

	h.mu.RUnlock()

	eg := errgroup.Group{}

	for _, handler := range handlerCopies {
		handlerCp := handler

		eg.Go(func() error {
			return handlerCp(event)
		})
	}

	return eg.Wait()
}

func isNil(value interface{}) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)

	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
