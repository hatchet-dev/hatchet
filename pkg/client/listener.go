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
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	sharedcontracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/validator"
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
	constructor func(context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error)

	client   dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient
	clientMu sync.RWMutex

	l *zerolog.Logger

	// map of workflow run ids to a list of handlers
	handlers sync.Map
}

func (r *subscribeClientImpl) getWorkflowRunsListener(
	ctx context.Context,
) (*WorkflowRunsListener, error) {
	r.workflowRunListenerMu.Lock()
	defer r.workflowRunListenerMu.Unlock()

	if r.workflowRunListener != nil {
		return r.workflowRunListener, nil
	}

	constructor := func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		return r.client.SubscribeToWorkflowRuns(r.ctx.newContext(ctx), grpc_retry.Disable())
	}

	w := &WorkflowRunsListener{
		constructor: constructor,
		l:           r.l,
	}

	err := w.retrySubscribe(ctx)

	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	r.workflowRunListener = w

	go func() {
		defer func() {
			err := w.Close()

			if err != nil {
				r.l.Error().Err(err).Msg("failed to close workflow run events listener")
			}

			r.workflowRunListenerMu.Lock()
			r.workflowRunListener = nil
			r.workflowRunListenerMu.Unlock()
		}()

		err := w.Listen(ctx)

		if err != nil {
			r.l.Error().Err(err).Msg("failed to listen for workflow run events")
		}
	}()

	return w, nil
}

func (w *WorkflowRunsListener) retrySubscribe(ctx context.Context) error {
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
			w.l.Error().Err(err).Msgf("could not resubscribe to the listener")
			continue
		}

		w.client = client

		// listen for all the same workflow runs
		var rangeErr error

		w.handlers.Range(func(key, value interface{}) bool {
			workflowRunId := key.(string)

			err := w.client.Send(&dispatchercontracts.SubscribeToWorkflowRunsRequest{
				WorkflowRunId: workflowRunId,
			})

			if err != nil {
				w.l.Error().Err(err).Msgf("could not subscribe to the worker")
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

	return fmt.Errorf("could not subscribe to the worker after %d retries", retries)
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

	err := l.retrySend(workflowRunId)

	if err != nil {
		return err
	}

	return nil
}

func (l *WorkflowRunsListener) retrySend(workflowRunId string) error {
	l.clientMu.RLock()
	defer l.clientMu.RUnlock()

	if l.client == nil {
		return fmt.Errorf("client is not connected")
	}

	for i := 0; i < DefaultActionListenerRetryCount; i++ {
		err := l.client.Send(&dispatchercontracts.SubscribeToWorkflowRunsRequest{
			WorkflowRunId: workflowRunId,
		})

		if err == nil {
			return nil
		}

		time.Sleep(DefaultActionListenerRetryInterval)
	}

	return fmt.Errorf("could not send to the worker after %d retries", DefaultActionListenerRetryCount)
}

func (l *WorkflowRunsListener) Listen(ctx context.Context) error {
	for {
		l.clientMu.RLock()
		event, err := l.client.Recv()
		l.clientMu.RUnlock()

		if err != nil {
			if errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled {
				return nil
			}

			retryErr := l.retrySubscribe(ctx)

			if retryErr != nil {
				return retryErr
			}

			continue
		}

		if err := l.handleWorkflowRun(event); err != nil {
			return err
		}
	}
}

func (l *WorkflowRunsListener) Close() error {
	return l.client.CloseSend()
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

	StreamByAdditionalMetadata(ctx context.Context, key string, value string, handler StreamHandler) error

	SubscribeToWorkflowRunEvents(ctx context.Context) (*WorkflowRunsListener, error)

	ListenForDurableEvents(ctx context.Context) (*DurableEventsListener, error)
}

type ClientEventListener interface {
	OnWorkflowEvent(ctx context.Context, event *WorkflowEvent) error
}

type subscribeClientImpl struct {
	client dispatchercontracts.DispatcherClient

	clientv1 sharedcontracts.V1DispatcherClient

	l *zerolog.Logger

	v validator.Validator

	ctx *contextLoader

	workflowRunListenerMu sync.Mutex
	workflowRunListener   *WorkflowRunsListener

	durableEventsListenerMu sync.Mutex
	durableEventsListener   *DurableEventsListener
}

func newSubscribe(conn *grpc.ClientConn, opts *sharedClientOpts) SubscribeClient {
	return &subscribeClientImpl{
		client:   dispatchercontracts.NewDispatcherClient(conn),
		clientv1: sharedcontracts.NewV1DispatcherClient(conn),
		l:        opts.l,
		v:        opts.v,
		ctx:      opts.ctxLoader,
	}
}

func (r *subscribeClientImpl) On(ctx context.Context, workflowRunId string, handler RunHandler) error {
	stream, err := r.client.SubscribeToWorkflowEvents(r.ctx.newContext(ctx), &dispatchercontracts.SubscribeToWorkflowEventsRequest{
		WorkflowRunId: &workflowRunId,
	}, grpc_retry.Disable())

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
		WorkflowRunId: &workflowRunId,
	}, grpc_retry.Disable())

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

func (r *subscribeClientImpl) StreamByAdditionalMetadata(ctx context.Context, key string, value string, handler StreamHandler) error {
	stream, err := r.client.SubscribeToWorkflowEvents(r.ctx.newContext(ctx), &dispatchercontracts.SubscribeToWorkflowEventsRequest{
		AdditionalMetaKey:   &key,
		AdditionalMetaValue: &value,
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
	return r.getWorkflowRunsListener(context.Background())
}

func (r *subscribeClientImpl) ListenForDurableEvents(ctx context.Context) (*DurableEventsListener, error) {
	return r.getDurableEventsListener(context.Background())
}
