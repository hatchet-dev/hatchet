package client

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/validator"
)

type WorkflowEvent *dispatchercontracts.WorkflowEvent
type WorkflowRunEvent *dispatchercontracts.WorkflowRunEvent

type StreamEvent struct {
	Message []byte
}

type RunHandler func(event WorkflowEvent) error
type StreamHandler func(event StreamEvent) error
type WorkflowRunEventHandler func(event WorkflowRunEvent) error

type WorkflowRunsListener struct {
	client dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient

	// map of workflow run ids to a list of handlers
	handlers sync.Map
}

func newWorkflowRunsListener(
	client dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient,
) *WorkflowRunsListener {
	return &WorkflowRunsListener{
		client: client,
	}
}

type threadSafeHandlers struct {
	handlers []WorkflowRunEventHandler
	mu       sync.RWMutex
}

func (l *WorkflowRunsListener) AddWorkflowRun(
	workflowRunId string,
	handler WorkflowRunEventHandler,
) error {
	handlers, _ := l.handlers.LoadOrStore(workflowRunId, &threadSafeHandlers{
		handlers: []WorkflowRunEventHandler{},
	})

	h := handlers.(*threadSafeHandlers)

	h.mu.Lock()
	h.handlers = append(h.handlers, handler)
	l.handlers.Store(workflowRunId, h)
	h.mu.Unlock()

	err := l.client.Send(&dispatchercontracts.SubscribeToWorkflowRunsRequest{
		WorkflowRunId: workflowRunId,
	})

	if err != nil {
		return err
	}

	return nil
}

func (l *WorkflowRunsListener) handleWorkflowRun(event *dispatchercontracts.WorkflowRunEvent) error {
	// find all handlers for this workflow run
	handlers, ok := l.handlers.Load(event.WorkflowRunId)

	if !ok {
		return nil
	}

	eg := errgroup.Group{}

	h := handlers.(*threadSafeHandlers)

	h.mu.RLock()

	for _, handler := range h.handlers {
		handlerCp := handler

		eg.Go(func() error {
			return handlerCp(event)
		})
	}

	h.mu.RUnlock()

	err := eg.Wait()

	return err
}

type SubscribeClient interface {
	On(ctx context.Context, workflowRunId string, handler RunHandler) error

	Stream(ctx context.Context, workflowRunId string, handler StreamHandler) error

	SubscribeToWorkflowRunEvents(ctx context.Context) (*WorkflowRunsListener, error)
}

type ClientEventListener interface {
	OnWorkflowEvent(ctx context.Context, event *WorkflowEvent) error
}

type subscribeClientImpl struct {
	client dispatchercontracts.DispatcherClient

	l *zerolog.Logger

	v validator.Validator

	ctx *contextLoader
}

func newSubscribe(conn *grpc.ClientConn, opts *sharedClientOpts) SubscribeClient {
	return &subscribeClientImpl{
		client: dispatchercontracts.NewDispatcherClient(conn),
		l:      opts.l,
		v:      opts.v,
		ctx:    opts.ctxLoader,
	}
}

func (r *subscribeClientImpl) On(ctx context.Context, workflowRunId string, handler RunHandler) error {
	stream, err := r.client.SubscribeToWorkflowEvents(r.ctx.newContext(ctx), &dispatchercontracts.SubscribeToWorkflowEventsRequest{
		WorkflowRunId: workflowRunId,
	})

	if err != nil {
		return err
	}

	for {
		event, err := stream.Recv()

		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

			return err
		}

		if event.EventType == dispatchercontracts.ResourceEventType_RESOURCE_EVENT_TYPE_STREAM {
			continue
		}

		if err := handler(event); err != nil {
			return err
		}
	}
}

func (r *subscribeClientImpl) Stream(ctx context.Context, workflowRunId string, handler StreamHandler) error {
	stream, err := r.client.SubscribeToWorkflowEvents(r.ctx.newContext(ctx), &dispatchercontracts.SubscribeToWorkflowEventsRequest{
		WorkflowRunId: workflowRunId,
	})

	if err != nil {
		return err
	}

	for {
		event, err := stream.Recv()

		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

			return err
		}

		if event.EventType != dispatchercontracts.ResourceEventType_RESOURCE_EVENT_TYPE_STREAM {
			continue
		}

		if err := handler(StreamEvent{
			Message: []byte(event.EventPayload),
		}); err != nil {
			return err
		}
	}
}

func (r *subscribeClientImpl) SubscribeToWorkflowRunEvents(ctx context.Context) (*WorkflowRunsListener, error) {
	client, err := r.client.SubscribeToWorkflowRuns(r.ctx.newContext(ctx))

	if err != nil {
		return nil, err
	}

	l := newWorkflowRunsListener(client)

	go func() {
		for {
			event, err := client.Recv()

			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}

				r.l.Error().Err(err).Msg("failed to receive workflow run event")
				return
			}

			go func() {
				err := l.handleWorkflowRun(event)

				if err != nil {
					r.l.Error().Err(err).Msg("failed to handle workflow run event")
				}
			}()
		}
	}()

	return l, nil
}
