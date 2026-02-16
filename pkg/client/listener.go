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
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	sharedcontracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
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
	constructor func(context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error)

	client     dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient
	clientMu   sync.Mutex
	generation uint64

	reconnectGroup singleflight.Group

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

// getClientSnapshot returns the current client and its generation without holding the lock.
func (w *WorkflowRunsListener) getClientSnapshot() (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, uint64) {
	w.clientMu.Lock()
	defer w.clientMu.Unlock()
	return w.client, w.generation
}

// retrySubscribe coalesces concurrent reconnection attempts via singleflight.
// Multiple goroutines calling this concurrently will share a single reconnection attempt.
func (w *WorkflowRunsListener) retrySubscribe(ctx context.Context) error {
	_, err, _ := w.reconnectGroup.Do("reconnect", func() (interface{}, error) {
		return nil, w.doRetrySubscribe(ctx)
	})
	return err
}

func (w *WorkflowRunsListener) doRetrySubscribe(ctx context.Context) error {
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
				w.l.Error().Err(err).Msgf("could not subscribe to the worker for workflow run id %s", workflowRunId)
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

	return fmt.Errorf("could not subscribe to the worker after %d retries", retries)
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
	handlers, _ := l.handlers.LoadOrStore(workflowRunId, &threadSafeHandlers{
		handlers: map[string]WorkflowRunEventHandler{},
	})

	h := handlers.(*threadSafeHandlers)

	h.mu.Lock()
	h.handlers[sessionId] = handler
	l.handlers.Store(workflowRunId, h)
	h.mu.Unlock()

	err := l.retrySend(workflowRunId)

	if err != nil {
		return err
	}

	return nil
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
	for i := 0; i < DefaultActionListenerRetryCount; i++ {
		client, genBefore := l.getClientSnapshot()

		if client == nil {
			return fmt.Errorf("client is not connected")
		}

		err := client.Send(&dispatchercontracts.SubscribeToWorkflowRunsRequest{
			WorkflowRunId: workflowRunId,
		})

		if err == nil {
			return nil
		}

		l.l.Warn().Err(err).Msgf("failed to send workflow run subscription, attempt %d/%d", i+1, DefaultActionListenerRetryCount)

		// Check if someone else (e.g. Listen) already reconnected while we were sending.
		// If so, skip the reconnect and retry the send on the new client immediately.
		if _, genAfter := l.getClientSnapshot(); genAfter != genBefore {
			continue
		}

		if retryErr := l.retrySubscribe(context.Background()); retryErr != nil {
			l.l.Error().Err(retryErr).Msg("failed to resubscribe after send failure")
		}

		time.Sleep(DefaultActionListenerRetryInterval)
	}

	return fmt.Errorf("could not send to the worker after %d retries", DefaultActionListenerRetryCount)
}

func (l *WorkflowRunsListener) Listen(ctx context.Context) error {
	consecutiveErrors := 0
	maxConsecutiveErrors := 10

	// Take a snapshot of the client so we never hold the lock during a blocking Recv.
	client, _ := l.getClientSnapshot()

	for {
		event, err := client.Recv()

		if err != nil {
			if errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled {
				return nil
			}

			consecutiveErrors++

			if status.Code(err) == codes.Unavailable {
				l.l.Warn().Err(err).Msg("dispatcher is unavailable, retrying subscribe after 1 second")
				time.Sleep(1 * time.Second)
			}

			retryErr := l.retrySubscribe(ctx)

			if retryErr != nil {
				l.l.Error().Err(retryErr).Msgf("failed to resubscribe (consecutive errors: %d/%d)", consecutiveErrors, maxConsecutiveErrors)

				if consecutiveErrors >= maxConsecutiveErrors {
					return fmt.Errorf("failed to resubscribe after %d consecutive errors: %w", consecutiveErrors, retryErr)
				}

				time.Sleep(DefaultActionListenerRetryInterval)
				continue
			}

			client, _ = l.getClientSnapshot()
			consecutiveErrors = 0
			continue
		}

		consecutiveErrors = 0

		if err := l.handleWorkflowRun(event); err != nil {
			return err
		}
	}
}

func (l *WorkflowRunsListener) Close() error {
	l.clientMu.Lock()
	client := l.client
	l.clientMu.Unlock()

	if client == nil {
		return nil
	}

	return client.CloseSend()
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
	client   dispatchercontracts.DispatcherClient
	clientv1 sharedcontracts.DispatcherClient

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
		clientv1: sharedcontracts.NewDispatcherClient(conn),
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
