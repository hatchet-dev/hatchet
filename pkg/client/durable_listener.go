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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

type DurableEvent *contracts.DurableEvent

type DurableEventHandler func(e DurableEvent) error

type DurableEventsListener struct {
	constructor func(context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error)

	client   contracts.V1Dispatcher_ListenForDurableEventClient
	clientMu sync.RWMutex

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
		defer func() {
			err := w.Close()

			if err != nil {
				r.l.Error().Err(err).Msg("failed to close durable events listener")
			}

			r.durableEventsListenerMu.Lock()
			r.durableEventsListener = nil
			r.durableEventsListenerMu.Unlock()
		}()

		err := w.Listen(ctx)

		if err != nil {
			r.l.Error().Err(err).Msg("failed to listen for durable events")
		}
	}()

	return w, nil
}

func (w *DurableEventsListener) retryListen(ctx context.Context) error {
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
			w.l.Error().Err(err).Msgf("could not resubscribe to the durable event listener")
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
				w.l.Error().Err(err).Msgf("could not listen for durable events on the worker")
				rangeErr = err
				return false
			}

			return true
		})

		if rangeErr != nil {
			continue
		}

		return nil
	}

	return fmt.Errorf("could not listen for durable events on the worker after %d retries", retries)
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

	return nil
}

func (l *DurableEventsListener) retrySend(t listenTuple) error {
	l.clientMu.RLock()
	defer l.clientMu.RUnlock()

	if l.client == nil {
		return fmt.Errorf("client is not connected")
	}

	for i := 0; i < DefaultActionListenerRetryCount; i++ {
		err := l.client.Send(&contracts.ListenForDurableEventRequest{
			TaskId:    t.taskId,
			SignalKey: t.signalKey,
		})

		if err == nil {
			return nil
		}

		time.Sleep(DefaultActionListenerRetryInterval)
	}

	return fmt.Errorf("could not send to the worker after %d retries", DefaultActionListenerRetryCount)
}

func (l *DurableEventsListener) Listen(ctx context.Context) error {
	for {
		l.clientMu.RLock()
		event, err := l.client.Recv()
		l.clientMu.RUnlock()

		if err != nil {
			if errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled {
				return nil
			}

			retryErr := l.retryListen(ctx)

			if retryErr != nil {
				return retryErr
			}

			continue
		}

		if err := l.handleEvent(event); err != nil {
			return err
		}
	}
}

func (l *DurableEventsListener) Close() error {
	return l.client.CloseSend()
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
