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
	"google.golang.org/grpc"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	sharedcontracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/client/retry"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

const maxConsecutiveStreamNoProgress = 10

var errListenerClosed = errors.New("listener is closed")

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
	constructor func(context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error)

	client     dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient
	stream     *reconnectingStream[dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient]
	streamOnce sync.Once

	listenMu  sync.Mutex
	listening bool

	l *zerolog.Logger

	// map of workflow run ids to a list of handlers
	handlers sync.Map
}

func (w *WorkflowRunsListener) streamCore() *reconnectingStream[dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient] {
	w.streamOnce.Do(func() {
		w.stream = newReconnectingStream(
			w.constructor,
			func(client dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient) error {
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

func (w *WorkflowRunsListener) lifecycleContext() context.Context {
	return w.streamCore().lifecycleContext()
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

	err := w.retrySubscribeSync(ctx)

	if err != nil {
		return nil, err
	}

	r.workflowRunListener = w
	if !w.startListening() {
		if closeErr := w.closeStream(); closeErr != nil {
			r.l.Error().Err(closeErr).Msg("failed to close workflow run events listener")
		}

		r.workflowRunListener = nil
		return nil, errListenerClosed
	}

	go func() {
		defer func() {
			r.workflowRunListenerMu.Lock()
			if r.workflowRunListener == w {
				r.workflowRunListener = nil
			}
			r.workflowRunListenerMu.Unlock()
		}()
		defer w.stopListening()

		err := w.listen(w.lifecycleContext())

		if err != nil {
			r.l.Error().Err(err).Msg("failed to listen for workflow run events")
		}
	}()

	return w, nil
}

// getClientSnapshot returns the current client and its generation without holding the lock.
func (w *WorkflowRunsListener) getClientSnapshot() (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, uint64) {
	client, generation, _ := w.streamCore().getClientSnapshot()
	return client, generation
}

func (w *WorkflowRunsListener) isListening() bool {
	w.listenMu.Lock()
	defer w.listenMu.Unlock()
	return w.listening
}

func (w *WorkflowRunsListener) isClosed() bool {
	return w.streamCore().isClosed()
}

func (w *WorkflowRunsListener) startListening() bool {
	w.listenMu.Lock()
	defer w.listenMu.Unlock()

	if w.listening || w.isClosed() {
		return false
	}

	w.listening = true
	return true
}

func (w *WorkflowRunsListener) stopListening() {
	w.listenMu.Lock()
	w.listening = false
	w.listenMu.Unlock()
}

func (w *WorkflowRunsListener) ensureListening(ctx context.Context) error {
	if w.isClosed() {
		return errListenerClosed
	}

	if w.isListening() {
		return nil
	}

	if err := w.retrySubscribeSync(ctx); err != nil {
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
			w.l.Error().Err(err).Msg("failed to listen for workflow run events")
		}
	}()

	return nil
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

func (w *WorkflowRunsListener) doRetrySubscribeSync(ctx context.Context) error {
	return w.retrySubscribeSync(ctx)
}

func (w *WorkflowRunsListener) doRetrySubscribeBackground(ctx context.Context) error {
	return w.retrySubscribeBackground(ctx)
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

	handlers, _ := l.handlers.LoadOrStore(workflowRunId, &threadSafeHandlers{
		handlers: map[string]WorkflowRunEventHandler{},
	})

	h := handlers.(*threadSafeHandlers)

	h.mu.Lock()
	h.handlers[sessionId] = handler
	l.handlers.Store(workflowRunId, h)
	h.mu.Unlock()

	if err := l.retrySend(workflowRunId); err != nil {
		if !l.isListening() {
			if listenErr := l.ensureListening(l.lifecycleContext()); listenErr != nil {
				return listenErr
			}

			return l.retrySend(workflowRunId)
		}

		return err
	}

	return l.ensureListening(l.lifecycleContext())
}

func (l *WorkflowRunsListener) RemoveWorkflowRun(
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
	if !l.startListening() {
		return nil
	}
	defer l.stopListening()

	return l.listen(ctx)
}

func (l *WorkflowRunsListener) listen(ctx context.Context) error {
	consecutiveNoProgress := 0
	reconnectAttempt := 0

	client, generation, ok := l.streamCore().getClientSnapshot()
	if !ok {
		return fmt.Errorf("client is not connected")
	}
	defer func() {
		if closeErr := client.CloseSend(); closeErr != nil {
			l.l.Warn().Err(closeErr).Msg("failed to close workflow run stream after listen exit")
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

			retryErr := l.retrySubscribeBackground(l.lifecycleContext())

			if retryErr != nil {
				if errors.Is(retryErr, errListenerClosed) {
					return nil
				}

				retryDecision := retry.ClassifyStreamError(ctx, retryErr)
				if streamDecisionStopsReconnect(retryDecision) {
					return fmt.Errorf("failed to resubscribe: %w", retryErr)
				}

				l.l.Error().Err(retryErr).Msgf("failed to resubscribe (consecutive no-progress: %d/%d)", consecutiveNoProgress, maxConsecutiveStreamNoProgress)

				if consecutiveNoProgress >= maxConsecutiveStreamNoProgress {
					return fmt.Errorf("failed to resubscribe after %d consecutive errors: %w", consecutiveNoProgress, retryErr)
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

		if err := l.handleWorkflowRun(event); err != nil {
			return err
		}
	}
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
	consecutiveNoProgress := 0
	reconnectAttempt := 0

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		stream, err := r.client.SubscribeToWorkflowEvents(r.ctx.newContext(ctx), &dispatchercontracts.SubscribeToWorkflowEventsRequest{
			AdditionalMetaKey:   &key,
			AdditionalMetaValue: &value,
		}, grpc_retry.Disable())

		if err != nil {
			decision := retry.ClassifyStreamError(ctx, err)
			if decision == retry.StreamDecisionStop {
				return err
			}

			if decision == retry.StreamDecisionNoProgress {
				consecutiveNoProgress++
				if consecutiveNoProgress >= maxConsecutiveStreamNoProgress {
					return fmt.Errorf("stream made no progress after %d consecutive errors: %w", consecutiveNoProgress, err)
				}

				return err
			}

			consecutiveNoProgress++

			if reconnectAttempt > 0 {
				if sleepErr := retry.SleepStreamBackoff(ctx, reconnectAttempt-1); sleepErr != nil {
					return sleepErr
				}
			}

			reconnectAttempt++
			continue
		}

		for {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			event, recvErr := stream.Recv()

			if recvErr != nil {
				decision := classifyStreamRecvError(ctx, recvErr, streamEOFRetries)
				if decision == retry.StreamDecisionStop {
					if errors.Is(recvErr, io.EOF) && ctx.Err() != nil {
						return ctx.Err()
					}

					return recvErr
				}

				if decision == retry.StreamDecisionNoProgress {
					consecutiveNoProgress++
					if consecutiveNoProgress >= maxConsecutiveStreamNoProgress {
						return fmt.Errorf("stream made no progress: %w", recvErr)
					}

					return recvErr
				}

				consecutiveNoProgress++

				if reconnectAttempt > 0 {
					if sleepErr := retry.SleepStreamBackoff(ctx, reconnectAttempt-1); sleepErr != nil {
						return sleepErr
					}
				}

				reconnectAttempt++
				break
			}

			consecutiveNoProgress = 0
			reconnectAttempt = 0

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
