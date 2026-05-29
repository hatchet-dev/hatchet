// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"context"
	"strconv"
	"sync/atomic"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"github.com/rs/zerolog"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

type DurableEvent *contracts.DurableEvent

type DurableEventHandler func(e DurableEvent) error

type DurableEventsListener struct {
	*reconnectingListener[listenTuple, *contracts.ListenForDurableEventRequest, *contracts.DurableEvent, contracts.V1Dispatcher_ListenForDurableEventClient]
	nextID atomic.Uint64
}

type listenTuple struct {
	taskId    string
	signalKey string
}

func newDurableEventsListener(
	constructor func(context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error),
	l *zerolog.Logger,
) *DurableEventsListener {
	return &DurableEventsListener{
		reconnectingListener: &reconnectingListener[listenTuple, *contracts.ListenForDurableEventRequest, *contracts.DurableEvent, contracts.V1Dispatcher_ListenForDurableEventClient]{
			constructor: constructor,
			l:           l,
			retryPolicy: reconnectingListenerRetryPolicy{
				maxConsecutiveReconnectErrors: 1,
				disableUnavailableDelay:       true,
			},
			requestForKey: func(t listenTuple) *contracts.ListenForDurableEventRequest {
				return &contracts.ListenForDurableEventRequest{
					TaskId:    t.taskId,
					SignalKey: t.signalKey,
				}
			},
			keyForEvent: func(event *contracts.DurableEvent) listenTuple {
				return listenTuple{
					taskId:    event.TaskId,
					signalKey: event.SignalKey,
				}
			},
		},
	}
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

	w := newDurableEventsListener(constructor, r.l)

	err := w.retrySubscribe(ctx)

	if err != nil {
		return nil, err
	}

	r.durableEventsListener = w

	go func() {
		defer func() {
			err := w.Close()

			if err != nil {
				r.l.Error().Ctx(ctx).Err(err).Msg("failed to close durable events listener")
			}

			r.durableEventsListenerMu.Lock()
			r.durableEventsListener = nil
			r.durableEventsListenerMu.Unlock()
		}()

		err := w.Listen(ctx)

		if err != nil {
			r.l.Error().Ctx(ctx).Err(err).Msg("failed to listen for durable events")
		}
	}()

	return w, nil
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

	handlerId := strconv.FormatUint(l.nextID.Add(1), 10)
	return l.addHandler(t, handlerId, func(event *contracts.DurableEvent) error {
		return handler(event)
	})
}
