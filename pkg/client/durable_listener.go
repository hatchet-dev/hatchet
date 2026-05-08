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

	listenMu  sync.Mutex
	listening bool
	started   bool
	closed    bool

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

	go func() {
		err := w.Listen(ctx)

		if err != nil {
			r.l.Error().Ctx(ctx).Err(err).Msg("failed to listen for durable events")
		}

		if ctx.Err() == nil {
			return
		}

		err = w.Close()

		if err != nil {
			r.l.Error().Ctx(ctx).Err(err).Msg("failed to close durable events listener")
		}

		r.durableEventsListenerMu.Lock()
		if r.durableEventsListener == w {
			r.durableEventsListener = nil
		}
		r.durableEventsListenerMu.Unlock()
	}()

	return w, nil
}

func (w *DurableEventsListener) retryListen(ctx context.Context) error {
	_, err, _ := w.reconnectGroup.Do("reconnect", func() (interface{}, error) {
		return nil, w.doRetryListen(ctx)
	})
	return err
}

func (w *DurableEventsListener) doRetryListen(ctx context.Context) error {
	w.clientMu.Lock()
	defer w.clientMu.Unlock()

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

		w.client = client

		// listen for all the same workflow runs
		var rangeErr error

		w.handlers.Range(func(key, value interface{}) bool {
			t := key.(listenTuple)

			err := w.client.Send(&contracts.ListenForDurableEventRequest{
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

func (w *DurableEventsListener) listeningState() (bool, bool) {
	w.listenMu.Lock()
	defer w.listenMu.Unlock()
	return w.listening, w.started
}

func (w *DurableEventsListener) tryStartListening() (bool, error) {
	w.listenMu.Lock()
	defer w.listenMu.Unlock()

	if w.closed {
		return false, fmt.Errorf("durable events listener is closed")
	}

	if w.listening {
		return false, nil
	}

	w.listening = true
	w.started = true
	return true, nil
}

func (w *DurableEventsListener) finishListening() {
	w.listenMu.Lock()
	w.listening = false
	w.listenMu.Unlock()
}

func (w *DurableEventsListener) startListening(ctx context.Context) error {
	started, err := w.tryStartListening()
	if err != nil || !started {
		return err
	}

	go func() {
		defer w.finishListening()

		if err := w.listen(ctx); err != nil {
			w.l.Error().Ctx(ctx).Err(err).Msg("failed to listen for durable events")
		}
	}()

	return nil
}

func (w *DurableEventsListener) ensureListening(ctx context.Context, genBefore uint64) error {
	listening, started := w.listeningState()
	if listening || !started {
		return nil
	}

	_, genAfter := w.getClientSnapshot()
	if genAfter == genBefore {
		if err := w.retryListen(ctx); err != nil {
			return err
		}
	}

	return w.startListening(ctx)
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
	_, genBefore := l.getClientSnapshot()

	t := listenTuple{
		taskId:    taskId,
		signalKey: signalKey,
	}
	handlers, _ := l.handlers.LoadOrStore(t, &threadSafeDurableEventHandlers{
		handlers: []DurableEventHandler{},
	})

	h := handlers.(*threadSafeDurableEventHandlers)

	h.mu.Lock()
	h.handlers = append(h.handlers, handler)
	l.handlers.Store(t, h)
	h.mu.Unlock()

	err := l.retrySend(t)

	if err != nil {
		return err
	}

	return l.ensureListening(context.Background(), genBefore)
}

func (l *DurableEventsListener) retrySend(t listenTuple) error {
	for i := range DefaultActionListenerRetryCount {
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
			l.l.Error().Err(retryErr).Msg("failed to resubscribe after durable event send failure")
			time.Sleep(DefaultActionListenerRetryInterval)
		}
	}

	return fmt.Errorf("could not send to the worker after %d retries", DefaultActionListenerRetryCount)
}

func (l *DurableEventsListener) Listen(ctx context.Context) error {
	started, err := l.tryStartListening()
	if err != nil || !started {
		return err
	}

	defer l.finishListening()

	return l.listen(ctx)
}

func (l *DurableEventsListener) listen(ctx context.Context) error {
	client, _ := l.getClientSnapshot()

	for {
		event, err := client.Recv()

		if err != nil {
			if errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled {
				return nil
			}

			retryErr := l.retryListen(ctx)

			if retryErr != nil {
				return retryErr
			}

			client, _ = l.getClientSnapshot()
			continue
		}

		if err := l.handleEvent(event); err != nil {
			return err
		}
	}
}

func (l *DurableEventsListener) Close() error {
	l.listenMu.Lock()
	l.closed = true
	l.listening = false
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
	// find all handlers for this workflow run
	handlers, ok := l.handlers.Load(listenTuple{
		taskId:    e.TaskId,
		signalKey: e.SignalKey,
	})

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

	return err
}
