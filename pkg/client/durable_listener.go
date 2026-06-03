// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

type DurableEvent *contracts.DurableEvent

type DurableEventHandler func(e DurableEvent) error

type DurableEventsListener struct {
	constructor func(context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error)

	client     contracts.V1Dispatcher_ListenForDurableEventClient
	clientMu   sync.Mutex
	generation uint64

	reconnectGroup singleflight.Group
	listenMu       sync.Mutex
	listening      bool
	closed         bool

	l *zerolog.Logger

	// map of workflow run ids to a list of handlers
	handlers sync.Map
}

type listenTuple struct {
	taskId    string
	signalKey string
}

func (r *subscribeClientImpl) getDurableEventsListener(
	ctx context.Context,
) (*DurableEventsListener, error) {
	r.durableEventsListenerMu.Lock()
	defer r.durableEventsListenerMu.Unlock()

	if r.durableEventsListener != nil {
		return r.durableEventsListener, nil
	}

	constructor := func(ctx context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
		return r.clientv1.ListenForDurableEvent(r.ctx.newContext(ctx), grpc_retry.Disable())
	}

	w := &DurableEventsListener{
		constructor: constructor,
		l:           r.l,
	}

	err := w.retryListen(ctx)

	if err != nil {
		return nil, err
	}

	r.durableEventsListener = w
	if !w.startListening() {
		if closeErr := w.Close(); closeErr != nil {
			r.l.Error().Ctx(ctx).Err(closeErr).Msg("failed to close durable events listener")
		}

		r.durableEventsListener = nil
		return nil, errListenerClosed
	}

	go func() {
		defer func() {
			err := w.Close()

			if err != nil {
				r.l.Error().Ctx(ctx).Err(err).Msg("failed to close durable events listener")
			}

			r.durableEventsListenerMu.Lock()
			if r.durableEventsListener == w {
				r.durableEventsListener = nil
			}
			r.durableEventsListenerMu.Unlock()
		}()
		defer w.stopListening()

		err := w.listen(ctx)

		if err != nil {
			r.l.Error().Ctx(ctx).Err(err).Msg("failed to listen for durable events")
		}
	}()

	return w, nil
}

func (w *DurableEventsListener) retryListen(ctx context.Context) error {
	if w.isClosed() {
		return errListenerClosed
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	_, err, _ := w.reconnectGroup.Do("reconnect", func() (interface{}, error) {
		return nil, w.doRetryListen(context.Background())
	})
	return err
}

func (w *DurableEventsListener) doRetryListen(ctx context.Context) error {
	w.clientMu.Lock()
	defer w.clientMu.Unlock()

	if w.isClosed() {
		return errListenerClosed
	}

	retries := 0

	for retries < DefaultActionListenerRetryCount {
		if retries > 0 {
			time.Sleep(DefaultActionListenerRetryInterval)
		}

		client, err := w.constructor(ctx)

		if err != nil {
			retries++
			w.l.Error().Ctx(ctx).Err(err).Msgf("could not resubscribe to the durable event listener")
			continue
		}

		var rangeErr error

		w.handlers.Range(func(key, value interface{}) bool {
			t := key.(listenTuple)

			err := client.Send(&contracts.ListenForDurableEventRequest{
				TaskId:    t.taskId,
				SignalKey: t.signalKey,
			})

			if err != nil {
				w.l.Error().Ctx(ctx).Err(err).Msgf("could not listen for durable events on the worker")
				rangeErr = err
				return false
			}

			return true
		})

		if rangeErr != nil {
			retries++
			continue
		}

		if w.isClosed() {
			return errListenerClosed
		}

		w.client = client
		w.generation++
		return nil
	}

	return fmt.Errorf("could not listen for durable events on the worker after %d retries", retries)
}

func (w *DurableEventsListener) getClientSnapshot() (contracts.V1Dispatcher_ListenForDurableEventClient, uint64) {
	w.clientMu.Lock()
	defer w.clientMu.Unlock()
	return w.client, w.generation
}

func (w *DurableEventsListener) isListening() bool {
	w.listenMu.Lock()
	defer w.listenMu.Unlock()
	return w.listening
}

func (w *DurableEventsListener) isClosed() bool {
	w.listenMu.Lock()
	defer w.listenMu.Unlock()
	return w.closed
}

func (w *DurableEventsListener) startListening() bool {
	w.listenMu.Lock()
	defer w.listenMu.Unlock()

	if w.listening || w.closed {
		return false
	}

	w.listening = true
	return true
}

func (w *DurableEventsListener) stopListening() {
	w.listenMu.Lock()
	w.listening = false
	w.listenMu.Unlock()
}

func (w *DurableEventsListener) ensureListening(ctx context.Context) error {
	if w.isClosed() {
		return errListenerClosed
	}

	if w.isListening() {
		return nil
	}

	if err := w.retryListen(ctx); err != nil {
		return err
	}

	if !w.startListening() {
		if w.isClosed() {
			return errListenerClosed
		}

		return nil
	}

	go func() {
		defer w.stopListening()

		if err := w.listen(ctx); err != nil {
			w.l.Error().Ctx(ctx).Err(err).Msg("failed to listen for durable events")
		}
	}()

	return nil
}

type threadSafeDurableEventHandlers struct {
	handlers []DurableEventHandler
	mu       sync.RWMutex
}

func (l *DurableEventsListener) AddSignal(
	taskId string,
	signalKey string,
	handler DurableEventHandler,
) error {
	if l.isClosed() {
		return errListenerClosed
	}

	t := listenTuple{
		taskId:    taskId,
		signalKey: signalKey,
	}

	if !l.isListening() {
		if err := l.ensureListening(context.Background()); err != nil {
			return err
		}
	}

	handlers, _ := l.handlers.LoadOrStore(t, &threadSafeDurableEventHandlers{
		handlers: []DurableEventHandler{},
	})

	h := handlers.(*threadSafeDurableEventHandlers)

	h.mu.Lock()
	h.handlers = append(h.handlers, handler)
	l.handlers.Store(t, h)
	h.mu.Unlock()

	if err := l.retrySend(t); err != nil {
		if !l.isListening() {
			if listenErr := l.ensureListening(context.Background()); listenErr != nil {
				return listenErr
			}

			return l.retrySend(t)
		}

		return err
	}

	return l.ensureListening(context.Background())
}

func (l *DurableEventsListener) retrySend(t listenTuple) error {
	for i := 0; i < DefaultActionListenerRetryCount; i++ {
		client, genBefore := l.getClientSnapshot()

		if client == nil {
			return fmt.Errorf("client is not connected")
		}

		err := client.Send(&contracts.ListenForDurableEventRequest{
			TaskId:    t.taskId,
			SignalKey: t.signalKey,
		})

		if err == nil {
			return nil
		}

		l.l.Warn().Err(err).Msgf("failed to send durable event subscription, attempt %d/%d", i+1, DefaultActionListenerRetryCount)

		if _, genAfter := l.getClientSnapshot(); genAfter != genBefore {
			continue
		}

		if retryErr := l.retryListen(context.Background()); retryErr != nil {
			l.l.Error().Err(retryErr).Msg("failed to relisten after send failure")
		}

		time.Sleep(DefaultActionListenerRetryInterval)
	}

	return fmt.Errorf("could not send to the worker after %d retries", DefaultActionListenerRetryCount)
}

func (l *DurableEventsListener) Listen(ctx context.Context) error {
	if !l.startListening() {
		return nil
	}
	defer l.stopListening()

	return l.listen(ctx)
}

func (l *DurableEventsListener) listen(ctx context.Context) error {
	consecutiveErrors := 0
	maxConsecutiveErrors := 10

	client, generation := l.getClientSnapshot()
	if client == nil {
		return fmt.Errorf("client is not connected")
	}

	for {
		event, err := client.Recv()

		if err != nil {
			switch {
			case errors.Is(err, io.EOF):
				if !l.shouldReconnectOnEOF(ctx) {
					return nil
				}
			case status.Code(err) == codes.Canceled:
				return nil
			case status.Code(err) == codes.Unavailable:
				l.l.Warn().Err(err).Msg("dispatcher is unavailable, retrying durable event listener after 1 second")
				time.Sleep(1 * time.Second)
			}

			consecutiveErrors++

			if _, genAfter := l.getClientSnapshot(); genAfter != generation {
				client, generation = l.getClientSnapshot()
				consecutiveErrors = 0
				continue
			}

			retryErr := l.retryListen(ctx)

			if retryErr != nil {
				if errors.Is(retryErr, errListenerClosed) {
					return nil
				}

				l.l.Error().Err(retryErr).Msgf("failed to relisten (consecutive errors: %d/%d)", consecutiveErrors, maxConsecutiveErrors)

				if consecutiveErrors >= maxConsecutiveErrors {
					return fmt.Errorf("failed to relisten after %d consecutive errors: %w", consecutiveErrors, retryErr)
				}

				time.Sleep(DefaultActionListenerRetryInterval)
				continue
			}

			client, generation = l.getClientSnapshot()
			consecutiveErrors = 0
			continue
		}

		consecutiveErrors = 0

		if err := l.handleEvent(event); err != nil {
			return err
		}
	}
}

func (l *DurableEventsListener) Close() error {
	l.listenMu.Lock()
	l.closed = true
	l.listenMu.Unlock()

	l.clientMu.Lock()
	client := l.client
	l.clientMu.Unlock()

	if client == nil {
		return nil
	}

	return client.CloseSend()
}

func (l *DurableEventsListener) handleEvent(e *contracts.DurableEvent) error {
	t := listenTuple{
		taskId:    e.TaskId,
		signalKey: e.SignalKey,
	}

	// find all handlers for this workflow run
	handlers, ok := l.handlers.Load(t)

	if !ok {
		return nil
	}

	eg := errgroup.Group{}

	h := handlers.(*threadSafeDurableEventHandlers)

	h.mu.RLock()

	for _, handler := range h.handlers {
		handlerCp := handler

		eg.Go(func() error {
			return handlerCp(e)
		})
	}

	h.mu.RUnlock()

	err := eg.Wait()

	if err == nil {
		l.handlers.Delete(t)
	}

	return err
}

func (l *DurableEventsListener) hasHandlers() bool {
	hasHandlers := false

	l.handlers.Range(func(key, value interface{}) bool {
		h := value.(*threadSafeDurableEventHandlers)

		h.mu.RLock()
		hasHandlers = len(h.handlers) > 0
		h.mu.RUnlock()

		return !hasHandlers
	})

	return hasHandlers
}

func (l *DurableEventsListener) shouldReconnectOnEOF(ctx context.Context) bool {
	if ctx.Err() != nil || l.isClosed() {
		return false
	}

	return l.hasHandlers()
}
