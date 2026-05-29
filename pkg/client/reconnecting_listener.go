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

type bidiStream[Req any, Ev any] interface {
	Send(Req) error
	Recv() (Ev, error)
	CloseSend() error
}

type reconnectingListener[K comparable, Req any, Ev any, S bidiStream[Req, Ev]] struct {
	client         S
	reconnectGroup singleflight.Group
	constructor    func(context.Context) (S, error)
	l              *zerolog.Logger
	requestForKey  func(K) Req
	keyForEvent    func(Ev) K
	handlers       sync.Map
	generation     uint64
	clientMu       sync.Mutex
}

type handlerSet[Ev any] struct {
	handlers map[string]func(Ev) error
	mu       sync.RWMutex
}

func (l *reconnectingListener[K, Req, Ev, S]) getClientSnapshot() (S, uint64) {
	l.clientMu.Lock()
	defer l.clientMu.Unlock()
	return l.client, l.generation
}

func (l *reconnectingListener[K, Req, Ev, S]) retrySubscribe(ctx context.Context) error {
	_, err, _ := l.reconnectGroup.Do("reconnect", func() (interface{}, error) {
		return nil, l.doRetrySubscribe(ctx)
	})

	return err
}

func (l *reconnectingListener[K, Req, Ev, S]) doRetrySubscribe(ctx context.Context) error {
	l.clientMu.Lock()
	defer l.clientMu.Unlock()

	retries := 0

	for retries < DefaultActionListenerRetryCount {
		if retries > 0 {
			time.Sleep(DefaultActionListenerRetryInterval)
		}

		client, err := l.constructor(ctx)

		if err != nil {
			retries++
			l.l.Error().Err(err).Msg("could not resubscribe to the listener")
			continue
		}

		l.client = client

		var rangeErr error

		l.handlers.Range(func(key, value interface{}) bool {
			k := key.(K)

			err := l.client.Send(l.requestForKey(k))
			if err != nil {
				l.l.Error().Err(err).Msg("could not resubscribe to the worker")
				rangeErr = err
				return false
			}

			return true
		})

		if rangeErr != nil {
			retries++
			continue
		}

		l.generation++
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
		client, genBefore := l.getClientSnapshot()

		if isNil(client) {
			return fmt.Errorf("client is not connected")
		}

		err := client.Send(l.requestForKey(key))

		if err == nil {
			return nil
		}

		l.l.Warn().Err(err).Msgf("failed to send listener subscription, attempt %d/%d", i+1, DefaultActionListenerRetryCount)

		if _, genAfter := l.getClientSnapshot(); genAfter != genBefore {
			continue
		}

		if retryErr := l.retrySubscribe(context.Background()); retryErr != nil {
			l.l.Error().Err(retryErr).Msg("failed to resubscribe after send failure")
		}

		time.Sleep(DefaultActionListenerRetryInterval)
	}

	return fmt.Errorf("could not send to the worker after %d retries", DefaultActionListenerRetryCount)
}

func (l *reconnectingListener[K, Req, Ev, S]) Listen(ctx context.Context) error {
	consecutiveErrors := 0
	maxConsecutiveErrors := 10

	client, _ := l.getClientSnapshot()

	if isNil(client) {
		return fmt.Errorf("client is not connected")
	}

	for {
		event, err := client.Recv()

		if err != nil {
			if errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled {
				return nil
			}

			consecutiveErrors++

			if status.Code(err) == codes.Unavailable {
				l.l.Warn().Err(err).Msg("dispatcher is unavailable, retrying subscribe after 1 second")
				time.Sleep(1 * time.Second)
			}

			retryErr := l.retrySubscribe(ctx)

			if retryErr != nil {
				l.l.Error().Err(retryErr).Msgf("failed to resubscribe (consecutive errors: %d/%d)", consecutiveErrors, maxConsecutiveErrors)

				if consecutiveErrors >= maxConsecutiveErrors {
					return fmt.Errorf("failed to resubscribe after %d consecutive errors: %w", consecutiveErrors, retryErr)
				}

				time.Sleep(DefaultActionListenerRetryInterval)
				continue
			}

			client, _ = l.getClientSnapshot()
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
	client := l.client
	if isNil(client) {
		l.clientMu.Unlock()
		return nil
	}

	var zero S
	l.client = zero
	l.clientMu.Unlock()

	return client.CloseSend()
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
