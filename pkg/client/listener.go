// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	sharedcontracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/client/retry"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

// ResourceType represents the type of resource
type ResourceType int32

const (
	ResourceType_RESOURCE_TYPE_UNKNOWN      ResourceType = 0
	ResourceType_RESOURCE_TYPE_STEP_RUN     ResourceType = 1
	ResourceType_RESOURCE_TYPE_WORKFLOW_RUN ResourceType = 2
)

// ResourceEventType represents the type of event
type ResourceEventType int32

const (
	ResourceEventType_RESOURCE_EVENT_TYPE_UNKNOWN   ResourceEventType = 0
	ResourceEventType_RESOURCE_EVENT_TYPE_STARTED   ResourceEventType = 1
	ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED ResourceEventType = 2
	ResourceEventType_RESOURCE_EVENT_TYPE_FAILED    ResourceEventType = 3
	ResourceEventType_RESOURCE_EVENT_TYPE_CANCELLED ResourceEventType = 4
	ResourceEventType_RESOURCE_EVENT_TYPE_TIMED_OUT ResourceEventType = 5
	ResourceEventType_RESOURCE_EVENT_TYPE_STREAM    ResourceEventType = 6
)

// WorkflowRunEventType represents the type of workflow run event
type WorkflowRunEventType int32

const (
	WorkflowRunEventType_WORKFLOW_RUN_EVENT_TYPE_FINISHED WorkflowRunEventType = 0
)

// workflowEvent is the internal representation of a workflow event
type workflowEvent struct {
	EventTimestamp *time.Time
	StepRetries    *int32
	RetryCount     *int32
	EventIndex     *int64
	WorkflowRunId  string
	ResourceId     string
	EventPayload   string
	ResourceType   ResourceType
	EventType      ResourceEventType
	Hangup         bool
}

// StepRunResult represents the result of a step run
type StepRunResult struct {
	Error          *string
	Output         *string
	StepRunId      string
	StepReadableId string
	JobRunId       string
}

// workflowRunEvent is the internal representation of a workflow run event
type workflowRunEvent struct {
	EventTimestamp *time.Time
	WorkflowRunId  string
	Results        []*StepRunResult
	EventType      WorkflowRunEventType
}

type WorkflowEvent *workflowEvent

type WorkflowRunEvent *workflowRunEvent

type StreamEvent struct {
	Message []byte
}

type RunHandler func(event WorkflowEvent) error
type StreamHandler func(event StreamEvent) error
type WorkflowRunEventHandler func(event WorkflowRunEvent) error

type WorkflowRunsListener struct {
	base streamListenerBase[dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient]

	constructor func(context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error)

	client dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient

	l *zerolog.Logger

	// map of workflow run ids to a list of handlers
	handlers sync.Map
}

func (w *WorkflowRunsListener) streamCore() *reconnectingStream[dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient] {
	return w.base.streamCore(streamCoreConfig[dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient]{
		constructor: w.constructor,
		closeSend: func(client dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient) error {
			return client.CloseSend()
		},
		replay: w.replayHandlers,
		initialClient: func() (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, bool) {
			return w.client, w.client != nil
		},
	})
}

func (w *WorkflowRunsListener) lifecycleContext() context.Context {
	return w.streamCore().lifecycleContext()
}

func (r *subscribeClientImpl) getWorkflowRunsListener(
	ctx context.Context,
) (*WorkflowRunsListener, error) {
	r.workflowRunListenerMu.Lock()
	defer r.workflowRunListenerMu.Unlock()

	constructor := func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		return r.client.SubscribeToWorkflowRuns(r.ctx.newContext(ctx), grpc_retry.Disable())
	}

	return startStreamListener(ctx, streamListenerStartConfig[WorkflowRunsListener]{
		existing: r.workflowRunListener,
		build: func() *WorkflowRunsListener {
			return &WorkflowRunsListener{
				constructor: constructor,
				l:           r.l,
			}
		},
		connect: func(w *WorkflowRunsListener, ctx context.Context) error {
			return w.retrySubscribeSync(ctx)
		},
		start: func(w *WorkflowRunsListener) bool {
			return w.startListening()
		},
		closeStream: func(w *WorkflowRunsListener) error {
			return w.closeStream()
		},
		stop: func(w *WorkflowRunsListener) {
			w.stopListening()
		},
		listen: func(w *WorkflowRunsListener, ctx context.Context) error {
			return w.listen(ctx)
		},
		lifecycleContext: func(w *WorkflowRunsListener) context.Context {
			return w.lifecycleContext()
		},
		store: func(w *WorkflowRunsListener) {
			r.workflowRunListener = w
		},
		clear: func(w *WorkflowRunsListener) {
			r.workflowRunListenerMu.Lock()
			if r.workflowRunListener == w {
				r.workflowRunListener = nil
			}
			r.workflowRunListenerMu.Unlock()
		},
		l:              r.l,
		closeErrorMsg:  "failed to close workflow run events listener",
		listenErrorMsg: "failed to listen for workflow run events",
	})
}

// getClientSnapshot returns the current client and its generation without holding the lock.
func (w *WorkflowRunsListener) getClientSnapshot() (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, uint64) {
	client, generation, _ := w.streamCore().getClientSnapshot()
	return client, generation
}

func (w *WorkflowRunsListener) isListening() bool {
	return w.base.isListening()
}

func (w *WorkflowRunsListener) isClosed() bool {
	return w.streamCore().isClosed()
}

func (w *WorkflowRunsListener) startListening() bool {
	return w.base.startListening(w.isClosed)
}

func (w *WorkflowRunsListener) stopListening() {
	w.base.stopListening()
}

func (w *WorkflowRunsListener) ensureListening(ctx context.Context) error {
	return w.base.ensureListening(ctx, streamListenerRunConfig{
		isClosed:         w.isClosed,
		retryConnectSync: w.retrySubscribeSync,
		lifecycleContext: w.lifecycleContext,
		listen:           w.listen,
		l:                w.l,
		listenErrorMsg:   "failed to listen for workflow run events",
	})
}

// retrySubscribeSync coalesces concurrent bounded reconnection attempts via singleflight.
func (w *WorkflowRunsListener) retrySubscribeSync(ctx context.Context) error {
	return w.streamCore().retryConnectSync(
		ctx,
		func(err error, attempt int) {
			w.l.Error().Err(err).Msgf("could not resubscribe to the listener (attempt %d/%d)", attempt, retry.StreamSyncMaxAttempts)
		},
		"could not subscribe to the worker after %d retries",
	)
}

// retrySubscribeBackground coalesces concurrent unbounded reconnection attempts via singleflight.
func (w *WorkflowRunsListener) retrySubscribeBackground(ctx context.Context) error {
	return w.streamCore().retryConnectBackground(
		ctx,
		func(err error, attempt int) {
			w.l.Error().Err(err).Msgf("could not resubscribe to the listener (background attempt %d)", attempt)
		},
		"could not resubscribe after %d consecutive no-progress errors: %w",
	)
}

func (w *WorkflowRunsListener) replayHandlers(ctx context.Context, client dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient) error {
	var rangeErr error

	w.handlers.Range(func(key, value interface{}) bool {
		workflowRunId := key.(string)

		err := client.Send(&dispatchercontracts.SubscribeToWorkflowRunsRequest{
			WorkflowRunId: workflowRunId,
		})

		if err != nil {
			w.l.Error().Err(err).Msgf("could not subscribe to the worker for workflow run id %s", workflowRunId)
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

type threadSafeHandlers struct {
	// map of session ids to handlers
	handlers map[string]WorkflowRunEventHandler
	mu       sync.RWMutex
}

func (l *WorkflowRunsListener) AddWorkflowRun(
	workflowRunId, sessionId string,
	handler WorkflowRunEventHandler,
) error {
	if l.isClosed() {
		return errListenerClosed
	}

	if !l.isListening() {
		if err := l.ensureListening(l.lifecycleContext()); err != nil {
			return err
		}
	}

	l.storeWorkflowRunHandler(workflowRunId, sessionId, handler)

	if err := l.retrySend(workflowRunId); err != nil {
		l.removeWorkflowRunHandler(workflowRunId, sessionId)

		if !l.isListening() {
			if listenErr := l.ensureListening(l.lifecycleContext()); listenErr != nil {
				return listenErr
			}

			if retryErr := l.retrySend(workflowRunId); retryErr != nil {
				return retryErr
			}

			l.storeWorkflowRunHandler(workflowRunId, sessionId, handler)
			return nil
		}

		return err
	}

	if err := l.ensureListening(l.lifecycleContext()); err != nil {
		l.removeWorkflowRunHandler(workflowRunId, sessionId)
		return err
	}

	return nil
}

func (l *WorkflowRunsListener) storeWorkflowRunHandler(
	workflowRunId, sessionId string,
	handler WorkflowRunEventHandler,
) {
	handlers, _ := l.handlers.LoadOrStore(workflowRunId, &threadSafeHandlers{
		handlers: map[string]WorkflowRunEventHandler{},
	})

	h := handlers.(*threadSafeHandlers)

	h.mu.Lock()
	h.handlers[sessionId] = handler
	l.handlers.Store(workflowRunId, h)
	h.mu.Unlock()
}

func (l *WorkflowRunsListener) removeWorkflowRunHandler(
	workflowRunId, sessionId string,
) {
	handlers, ok := l.handlers.Load(workflowRunId)
	if !ok {
		return
	}

	h := handlers.(*threadSafeHandlers)

	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.handlers, sessionId)

	if len(h.handlers) == 0 {
		l.handlers.Delete(workflowRunId)
	}
}

func (l *WorkflowRunsListener) RemoveWorkflowRun(
	workflowRunId, sessionId string,
) {
	l.removeWorkflowRunHandler(workflowRunId, sessionId)
}

func (l *WorkflowRunsListener) retrySend(workflowRunId string) error {
	return l.streamCore().retrySend(
		l.lifecycleContext(),
		func(client dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient) error {
			return client.Send(&dispatchercontracts.SubscribeToWorkflowRunsRequest{
				WorkflowRunId: workflowRunId,
			})
		},
		func(err error, attempt int) {
			l.l.Warn().Err(err).Msgf("failed to send workflow run subscription, attempt %d/%d", attempt, retry.StreamSyncMaxAttempts)
		},
		func(err error) {
			l.l.Error().Err(err).Msg("failed to resubscribe after send failure")
		},
		func(err error, attempt int) {
			l.l.Error().Err(err).Msgf("could not resubscribe to the listener (attempt %d/%d)", attempt, retry.StreamSyncMaxAttempts)
		},
		"could not send to the worker after %d retries",
	)
}

func (l *WorkflowRunsListener) Listen(ctx context.Context) error {
	return l.base.Listen(ctx, streamListenerRunConfig{
		isClosed:         l.isClosed,
		retryConnectSync: l.retrySubscribeSync,
		lifecycleContext: l.lifecycleContext,
		listen:           l.listen,
		l:                l.l,
		listenErrorMsg:   "failed to listen for workflow run events",
	})
}

func (l *WorkflowRunsListener) listen(ctx context.Context) error {
	return listenReconnectingStream(
		ctx,
		l.streamCore(),
		streamListenConfig[dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, *dispatchercontracts.WorkflowRunEvent]{
			recv: func(client dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient) (*dispatchercontracts.WorkflowRunEvent, error) {
				return client.Recv()
			},
			handle:               l.handleWorkflowRun,
			shouldReconnectOnEOF: l.shouldReconnectOnEOF,
			reconnectContext:     l.lifecycleContext(),
			labels: streamListenLabels{
				streamName:    "workflow run listener",
				reconnectVerb: "resubscribe",
			},
			l: l.l,
		},
	)
}

func (l *WorkflowRunsListener) Close() error {
	return l.streamCore().Close()
}

func (l *WorkflowRunsListener) closeStream() error {
	return l.streamCore().closeStream()
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
			workflowRunEvent, err := workflowRunEventToDeprecatedWorkflowRunEvent(event)
			if err != nil {
				return err
			}

			return handlerCp(workflowRunEvent)
		})
	}

	h.mu.RUnlock()

	err := eg.Wait()

	return err
}

func (l *WorkflowRunsListener) hasHandlers() bool {
	hasHandlers := false

	l.handlers.Range(func(key, value interface{}) bool {
		h := value.(*threadSafeHandlers)

		h.mu.RLock()
		hasHandlers = len(h.handlers) > 0
		h.mu.RUnlock()

		return !hasHandlers
	})

	return hasHandlers
}

func (l *WorkflowRunsListener) shouldReconnectOnEOF(ctx context.Context) bool {
	if ctx.Err() != nil || l.isClosed() {
		return false
	}

	return l.hasHandlers()
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

		workflowEvent, err := workflowEventToDeprecatedWorkflowEvent(event)
		if err != nil {
			return err
		}

		if err := handler(workflowEvent); err != nil {
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
	stream := newReconnectingStream(
		func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowEventsClient, error) {
			return r.client.SubscribeToWorkflowEvents(r.ctx.newContext(ctx), &dispatchercontracts.SubscribeToWorkflowEventsRequest{
				AdditionalMetaKey:   &key,
				AdditionalMetaValue: &value,
			}, grpc_retry.Disable())
		},
		func(client dispatchercontracts.Dispatcher_SubscribeToWorkflowEventsClient) error {
			return client.CloseSend()
		},
		nil,
	)

	if err := stream.retryConnectBackground(
		ctx,
		func(err error, attempt int) {
			r.l.Error().Err(err).Msgf("could not resubscribe to metadata stream (background attempt %d)", attempt)
		},
		"could not resubscribe to metadata stream after %d consecutive no-progress errors: %w",
	); err != nil {
		return err
	}

	return listenReconnectingStream(
		ctx,
		stream,
		streamListenConfig[dispatchercontracts.Dispatcher_SubscribeToWorkflowEventsClient, *dispatchercontracts.WorkflowEvent]{
			recv: func(client dispatchercontracts.Dispatcher_SubscribeToWorkflowEventsClient) (*dispatchercontracts.WorkflowEvent, error) {
				return client.Recv()
			},
			handle: func(event *dispatchercontracts.WorkflowEvent) error {
				if event.EventType != dispatchercontracts.ResourceEventType_RESOURCE_EVENT_TYPE_STREAM {
					return ctx.Err()
				}

				if err := handler(StreamEvent{
					Message: []byte(event.EventPayload),
				}); err != nil {
					return err
				}

				return ctx.Err()
			},
			shouldReconnectOnEOF: func(ctx context.Context) bool {
				return ctx.Err() == nil
			},
			reconnectContext: ctx,
			noProgressPolicy: streamNoProgressStopsImmediately,
			labels: streamListenLabels{
				streamName:    "metadata stream listener",
				reconnectVerb: "resubscribe",
			},
			l: r.l,
		},
	)
}

func (r *subscribeClientImpl) SubscribeToWorkflowRunEvents(ctx context.Context) (*WorkflowRunsListener, error) {
	return r.getWorkflowRunsListener(ctx)
}

func (r *subscribeClientImpl) ListenForDurableEvents(ctx context.Context) (*DurableEventsListener, error) {
	return r.getDurableEventsListener(ctx)
}

func workflowEventToDeprecatedWorkflowEvent(event *dispatchercontracts.WorkflowEvent) (*workflowEvent, error) {
	result := &workflowEvent{
		WorkflowRunId: event.WorkflowRunId,
		ResourceType:  ResourceType(event.ResourceType),
		EventType:     ResourceEventType(event.EventType),
		ResourceId:    event.ResourceId,
		EventPayload:  event.EventPayload,
		Hangup:        event.Hangup,
		StepRetries:   event.TaskRetries,
		RetryCount:    event.RetryCount,
		EventIndex:    event.EventIndex,
	}

	if event.EventTimestamp != nil {
		t := event.EventTimestamp.AsTime()
		result.EventTimestamp = &t
	}

	return result, nil
}

func workflowRunEventToDeprecatedWorkflowRunEvent(event *dispatchercontracts.WorkflowRunEvent) (*workflowRunEvent, error) {
	result := &workflowRunEvent{
		WorkflowRunId: event.WorkflowRunId,
		EventType:     WorkflowRunEventType(event.EventType),
	}

	if event.EventTimestamp != nil {
		t := event.EventTimestamp.AsTime()
		result.EventTimestamp = &t
	}

	if event.Results != nil {
		result.Results = make([]*StepRunResult, len(event.Results))
		for i, r := range event.Results {
			result.Results[i] = &StepRunResult{
				StepRunId:      r.TaskRunExternalId,
				StepReadableId: r.TaskName,
				JobRunId:       r.JobRunId,
				Error:          r.Error,
				Output:         r.Output,
			}
		}
	}

	return result, nil
}
