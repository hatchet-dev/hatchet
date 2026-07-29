// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"context"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

type DurableEvent *contracts.DurableEvent

type DurableEventHandler func(e DurableEvent) error

type listenTuple struct {
	taskId    string
	signalKey string
}

// DurableEventsListener streams durable task signals and dispatches them to
// registered one-shot handlers.
//
// NOTE: flow methods mirror WorkflowRunsListener in listener.go; keep changes
// in sync when adjusting add/ensure/dispatch behavior.
type DurableEventsListener struct {
	stream *reconnectingStream[contracts.V1Dispatcher_ListenForDurableEventClient]
	l      *zerolog.Logger
	reg    handlerRegistry[listenTuple, DurableEvent]
	gate   listenGate
}

func newDurableEventsListener(
	l *zerolog.Logger,
	constructor func(context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error),
) *DurableEventsListener {
	d := &DurableEventsListener{
		reg: newHandlerRegistry[listenTuple, DurableEvent](),
		l:   l,
	}
	d.stream = newReconnectingStream(
		l,
		"durable event listener",
		constructor,
		func(client contracts.V1Dispatcher_ListenForDurableEventClient) error {
			return client.CloseSend()
		},
		d.replayHandlers,
	)
	return d
}

func (r *subscribeClientImpl) getDurableEventsListener(
	ctx context.Context,
) (*DurableEventsListener, error) {
	r.durableEventsListenerMu.Lock()
	defer r.durableEventsListenerMu.Unlock()

	if r.durableEventsListener != nil {
		return r.durableEventsListener, nil
	}

	l := newDurableEventsListener(r.l, func(ctx context.Context) (contracts.V1Dispatcher_ListenForDurableEventClient, error) {
		return r.clientv1.ListenForDurableEvent(r.ctx.newContext(ctx), grpc_retry.Disable())
	})

	if err := l.stream.connectSync(ctx); err != nil {
		return nil, err
	}

	onExit := func() {
		r.durableEventsListenerMu.Lock()
		if r.durableEventsListener == l {
			r.durableEventsListener = nil
		}
		r.durableEventsListenerMu.Unlock()
	}

	if err := l.startBackground(onExit); err != nil {
		_ = l.stream.closeStream()
		return nil, err
	}

	r.durableEventsListener = l
	return l, nil
}

func (l *DurableEventsListener) AddSignal(
	taskId string,
	signalKey string,
	handler DurableEventHandler,
) error {
	return l.add(listenTuple{taskId: taskId, signalKey: signalKey}, "", handler, nil)
}

func (l *DurableEventsListener) add(
	key listenTuple,
	session string,
	handler DurableEventHandler,
	onError func(error),
) error {
	if l.stream.isClosed() {
		return errListenerClosed
	}

	lifecycle := l.stream.lifecycleContext()
	if err := l.ensureListening(lifecycle); err != nil {
		return err
	}

	remove := l.reg.store(key, session, handler, onError)
	if err := l.retrySend(key); err != nil {
		if err2 := l.ensureListening(lifecycle); err2 != nil {
			remove()
			return err2
		}
		if err2 := l.retrySend(key); err2 != nil {
			remove()
			return err2
		}
	}

	if err := l.ensureListening(lifecycle); err != nil {
		remove()
		return err
	}

	return nil
}

func (l *DurableEventsListener) retrySend(t listenTuple) error {
	return l.stream.retrySend(l.stream.lifecycleContext(),
		func(c contracts.V1Dispatcher_ListenForDurableEventClient) error {
			return c.Send(&contracts.ListenForDurableEventRequest{
				TaskId:    t.taskId,
				SignalKey: t.signalKey,
			})
		})
}

func (l *DurableEventsListener) replayHandlers(ctx context.Context, client contracts.V1Dispatcher_ListenForDurableEventClient) error {
	for _, key := range l.reg.keys() {
		if err := client.Send(&contracts.ListenForDurableEventRequest{
			TaskId:    key.taskId,
			SignalKey: key.signalKey,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (l *DurableEventsListener) dispatch(event *contracts.DurableEvent) error {
	key := listenTuple{taskId: event.TaskId, signalKey: event.SignalKey}
	regs := l.reg.snapshot(key)
	if len(regs) == 0 {
		return nil
	}

	eg := errgroup.Group{}
	for _, r := range regs {
		r := r
		eg.Go(func() error {
			if err := r.handle(event); err != nil {
				l.l.Error().Err(err).
					Str("task_id", event.TaskId).
					Str("signal_key", event.SignalKey).
					Msg("durable event handler failed")
				return err
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil
	}
	l.reg.removeRegistrations(key, regs)
	return nil
}

func (l *DurableEventsListener) shouldReconnectOnEOF(ctx context.Context) bool {
	return ctx.Err() == nil && !l.stream.isClosed() && l.reg.hasAny()
}

func (l *DurableEventsListener) runLoop(ctx context.Context) error {
	err := listenStream(ctx, l.stream,
		func(c contracts.V1Dispatcher_ListenForDurableEventClient) (*contracts.DurableEvent, error) {
			return c.Recv()
		},
		l.dispatch,
		newStreamClassifier(l.shouldReconnectOnEOF),
	)
	if err != nil {
		n := l.reg.failAll(err)
		l.l.Error().Err(err).Str("stream", l.stream.name).Int("handlers", n).
			Msg("stream listener terminated; failing registered handlers")
	}
	return err
}

func (l *DurableEventsListener) ensureListening(ctx context.Context) error {
	if l.stream.isClosed() {
		return errListenerClosed
	}
	if l.gate.active() {
		return nil
	}
	if err := l.stream.connectSync(ctx); err != nil {
		return err
	}
	if !l.gate.tryStart(l.stream.isClosed()) {
		if l.stream.isClosed() {
			return errListenerClosed
		}
		return nil
	}
	go func() {
		defer l.gate.stop()
		_ = l.runLoop(l.stream.lifecycleContext())
	}()
	return nil
}

func (l *DurableEventsListener) startBackground(onExit func()) error {
	if !l.gate.tryStart(l.stream.isClosed()) {
		if l.stream.isClosed() {
			return errListenerClosed
		}
		return nil
	}
	go func() {
		defer onExit()
		defer l.gate.stop()
		_ = l.runLoop(l.stream.lifecycleContext())
	}()
	return nil
}

func (l *DurableEventsListener) listen(ctx context.Context) error {
	if !l.gate.tryStart(l.stream.isClosed()) {
		return nil
	}
	defer l.gate.stop()
	return l.runLoop(ctx)
}

func (l *DurableEventsListener) Listen(ctx context.Context) error {
	return l.listen(ctx)
}

func (l *DurableEventsListener) Close() error {
	return l.stream.Close()
}

func (l *DurableEventsListener) isListening() bool {
	return l.gate.active()
}
