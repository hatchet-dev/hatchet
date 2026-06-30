// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"context"
	"errors"
	"fmt"
	"sync"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/client/retry"
)

type DurableEvent *contracts.DurableEvent

type DurableEventHandler func(e DurableEvent) error

type DurableEventsListener struct {
	constructor func(context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error)

	client     contracts.V1Dispatcher_ListenForDurableEventClient
	stream     *reconnectingStream[contracts.V1Dispatcher_ListenForDurableEventClient]
	streamOnce sync.Once

	listenMu  sync.Mutex
	listening bool

	l *zerolog.Logger

	// map of workflow run ids to a list of handlers
	handlers sync.Map
}

type listenTuple struct {
	taskId    string
	signalKey string
}

func (w *DurableEventsListener) streamCore() *reconnectingStream[contracts.V1Dispatcher_ListenForDurableEventClient] {
	w.streamOnce.Do(func() {
		w.stream = newReconnectingStream(
			w.constructor,
			func(client contracts.V1Dispatcher_ListenForDurableEventClient) error {
				return client.CloseSend()
			},
			w.replayHandlers,
		)

		if w.client != nil {
			w.stream.setInitialClient(w.client)
		}
	})

	return w.stream
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

	err := w.retryListenSync(ctx)

	if err != nil {
		return nil, err
	}

	r.durableEventsListener = w
	if !w.startListening() {
		if closeErr := w.closeStream(); closeErr != nil {
			r.l.Error().Ctx(ctx).Err(closeErr).Msg("failed to close durable events listener")
		}

		r.durableEventsListener = nil
		return nil, errListenerClosed
	}

	go func() {
		defer func() {
			r.durableEventsListenerMu.Lock()
			if r.durableEventsListener == w {
				r.durableEventsListener = nil
			}
			r.durableEventsListenerMu.Unlock()
		}()
		defer w.stopListening()

		err := w.listen(w.lifecycleContext())

		if err != nil {
			r.l.Error().Ctx(ctx).Err(err).Msg("failed to listen for durable events")
		}
	}()

	return w, nil
}

func (w *DurableEventsListener) lifecycleContext() context.Context {
	return w.streamCore().lifecycleContext()
}

func (w *DurableEventsListener) retryListenSync(ctx context.Context) error {
	return w.streamCore().retryConnectSync(
		ctx,
		func(err error, attempt int) {
			w.l.Error().Ctx(ctx).Err(err).Msgf("could not resubscribe to the durable event listener (attempt %d/%d)", attempt, retry.StreamSyncMaxAttempts)
		},
		"could not listen for durable events on the worker after %d retries",
	)
}

func (w *DurableEventsListener) retryListenBackground(ctx context.Context) error {
	return w.streamCore().retryConnectBackground(
		ctx,
		func(err error, attempt int) {
			w.l.Error().Ctx(ctx).Err(err).Msgf("could not resubscribe to the durable event listener (background attempt %d)", attempt)
		},
		"could not resubscribe after %d consecutive no-progress errors: %w",
	)
}

func (w *DurableEventsListener) doRetryListenBackground(ctx context.Context) error {
	return w.retryListenBackground(ctx)
}

func (w *DurableEventsListener) replayHandlers(ctx context.Context, client contracts.V1Dispatcher_ListenForDurableEventClient) error {
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
		return rangeErr
	}

	return nil
}

func (w *DurableEventsListener) getClientSnapshot() (contracts.V1Dispatcher_ListenForDurableEventClient, uint64) {
	client, generation, _ := w.streamCore().getClientSnapshot()
	return client, generation
}

func (w *DurableEventsListener) isListening() bool {
	w.listenMu.Lock()
	defer w.listenMu.Unlock()
	return w.listening
}

func (w *DurableEventsListener) isClosed() bool {
	return w.streamCore().isClosed()
}

func (w *DurableEventsListener) startListening() bool {
	w.listenMu.Lock()
	defer w.listenMu.Unlock()

	if w.listening || w.isClosed() {
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

	if err := w.retryListenSync(ctx); err != nil {
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

		if err := w.listen(w.lifecycleContext()); err != nil {
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
		if err := l.ensureListening(l.lifecycleContext()); err != nil {
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
			if listenErr := l.ensureListening(l.lifecycleContext()); listenErr != nil {
				return listenErr
			}

			return l.retrySend(t)
		}

		return err
	}

	return l.ensureListening(l.lifecycleContext())
}

func (l *DurableEventsListener) retrySend(t listenTuple) error {
	return l.streamCore().retrySend(
		l.lifecycleContext(),
		func(client contracts.V1Dispatcher_ListenForDurableEventClient) error {
			return client.Send(&contracts.ListenForDurableEventRequest{
				TaskId:    t.taskId,
				SignalKey: t.signalKey,
			})
		},
		func(err error, attempt int) {
			l.l.Warn().Err(err).Msgf("failed to send durable event subscription, attempt %d/%d", attempt, retry.StreamSyncMaxAttempts)
		},
		func(err error) {
			l.l.Error().Err(err).Msg("failed to relisten after send failure")
		},
		func(err error, attempt int) {
			l.l.Error().Err(err).Msgf("could not resubscribe to the durable event listener (attempt %d/%d)", attempt, retry.StreamSyncMaxAttempts)
		},
		"could not send to the worker after %d retries",
	)
}

func (l *DurableEventsListener) Listen(ctx context.Context) error {
	if !l.startListening() {
		return nil
	}
	defer l.stopListening()

	return l.listen(ctx)
}

func (l *DurableEventsListener) listen(ctx context.Context) error {
	consecutiveNoProgress := 0
	reconnectAttempt := 0

	client, generation, ok := l.streamCore().getClientSnapshot()
	if !ok {
		return fmt.Errorf("client is not connected")
	}
	defer func() {
		if closeErr := client.CloseSend(); closeErr != nil {
			l.l.Warn().Err(closeErr).Msg("failed to close durable event stream after listen exit")
		}
	}()

	for {
		event, err := client.Recv()

		if err != nil {
			eofPolicy := streamEOFStops
			if l.shouldReconnectOnEOF(ctx) {
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
			default:
				consecutiveNoProgress++
			}

			if _, genAfter := l.getClientSnapshot(); genAfter != generation {
				client, generation = l.getClientSnapshot()
				consecutiveNoProgress = 0
				reconnectAttempt = 0
				continue
			}

			if reconnectAttempt > 0 {
				if sleepErr := retry.SleepStreamBackoff(ctx, reconnectAttempt-1); sleepErr != nil {
					return nil
				}
			}

			retryErr := l.retryListenBackground(l.lifecycleContext())

			if retryErr != nil {
				if errors.Is(retryErr, errListenerClosed) {
					return nil
				}

				retryDecision := retry.ClassifyStreamError(ctx, retryErr)
				if streamDecisionStopsReconnect(retryDecision) {
					return fmt.Errorf("failed to relisten: %w", retryErr)
				}

				l.l.Error().Err(retryErr).Msgf("failed to relisten (consecutive no-progress: %d/%d)", consecutiveNoProgress, maxConsecutiveStreamNoProgress)

				if consecutiveNoProgress >= maxConsecutiveStreamNoProgress {
					return fmt.Errorf("failed to relisten after %d consecutive errors: %w", consecutiveNoProgress, retryErr)
				}

				reconnectAttempt++
				continue
			}

			client, generation = l.getClientSnapshot()
			consecutiveNoProgress = 0
			reconnectAttempt = 0
			continue
		}

		consecutiveNoProgress = 0
		reconnectAttempt = 0

		if err := l.handleEvent(event); err != nil {
			return err
		}
	}
}

func (l *DurableEventsListener) Close() error {
	return l.streamCore().Close()
}

func (l *DurableEventsListener) closeStream() error {
	return l.streamCore().closeStream()
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
