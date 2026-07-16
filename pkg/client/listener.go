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

// WorkflowRunsListener streams workflow run completion events and dispatches
// them to per-run session handlers.
type WorkflowRunsListener struct {
	stream *reconnectingStream[dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient]
	reg    handlerRegistry[string, WorkflowRunEvent]
	gate   listenGate
	l      *zerolog.Logger
}

func newWorkflowRunsListener(
	l *zerolog.Logger,
	constructor func(context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error),
) *WorkflowRunsListener {
	w := &WorkflowRunsListener{
		reg: newHandlerRegistry[string, WorkflowRunEvent](),
		l:   l,
	}
	w.stream = newReconnectingStream(
		l,
		"workflow run listener",
		constructor,
		func(client dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient) error {
			return client.CloseSend()
		},
		w.replayHandlers,
	)
	return w
}

func (r *subscribeClientImpl) getWorkflowRunsListener(
	ctx context.Context,
) (*WorkflowRunsListener, error) {
	r.workflowRunListenerMu.Lock()
	defer r.workflowRunListenerMu.Unlock()

	if r.workflowRunListener != nil {
		return r.workflowRunListener, nil
	}

	l := newWorkflowRunsListener(r.l, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		return r.client.SubscribeToWorkflowRuns(r.ctx.newContext(ctx), grpc_retry.Disable())
	})

	if err := l.stream.connectSync(ctx); err != nil {
		return nil, err
	}

	onExit := func() {
		r.workflowRunListenerMu.Lock()
		if r.workflowRunListener == l {
			r.workflowRunListener = nil
		}
		r.workflowRunListenerMu.Unlock()
	}

	if err := l.startBackground(onExit); err != nil {
		_ = l.stream.closeStream()
		return nil, err
	}

	r.workflowRunListener = l
	return l, nil
}

func (l *WorkflowRunsListener) AddWorkflowRun(
	workflowRunId, sessionId string,
	handler WorkflowRunEventHandler,
) error {
	return l.addWorkflowRun(workflowRunId, sessionId, handler, nil)
}

func (l *WorkflowRunsListener) addWorkflowRun(
	workflowRunId, sessionId string,
	handler WorkflowRunEventHandler,
	onError func(error),
) error {
	if l.stream.isClosed() {
		return errListenerClosed
	}

	lifecycle := l.stream.lifecycleContext()
	if err := l.ensureListening(lifecycle); err != nil {
		return err
	}

	remove := l.reg.store(workflowRunId, sessionId, handler, onError)
	if err := l.retrySend(workflowRunId); err != nil {
		if err2 := l.ensureListening(lifecycle); err2 != nil {
			remove()
			return err2
		}
		if err2 := l.retrySend(workflowRunId); err2 != nil {
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

func (l *WorkflowRunsListener) RemoveWorkflowRun(workflowRunId, sessionId string) {
	l.reg.removeSession(workflowRunId, sessionId)
}

func (l *WorkflowRunsListener) retrySend(workflowRunId string) error {
	return l.stream.retrySend(l.stream.lifecycleContext(),
		func(c dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient) error {
			return c.Send(&dispatchercontracts.SubscribeToWorkflowRunsRequest{WorkflowRunId: workflowRunId})
		})
}

func (l *WorkflowRunsListener) replayHandlers(ctx context.Context, client dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient) error {
	for _, runId := range l.reg.keys() {
		if err := client.Send(&dispatchercontracts.SubscribeToWorkflowRunsRequest{WorkflowRunId: runId}); err != nil {
			return err
		}
	}
	return nil
}

func (l *WorkflowRunsListener) dispatch(event *dispatchercontracts.WorkflowRunEvent) error {
	regs := l.reg.snapshot(event.WorkflowRunId)
	if len(regs) == 0 {
		return nil
	}

	ev, err := workflowRunEventToDeprecatedWorkflowRunEvent(event)
	if err != nil {
		return err
	}

	eg := errgroup.Group{}
	for _, r := range regs {
		r := r
		eg.Go(func() error {
			if err := r.handle(ev); err != nil {
				l.l.Error().Err(err).Str("workflow_run_id", event.WorkflowRunId).
					Msg("workflow run handler failed")
				return err
			}
			return nil
		})
	}
	_ = eg.Wait()
	return nil
}

func (l *WorkflowRunsListener) shouldReconnectOnEOF(ctx context.Context) bool {
	return ctx.Err() == nil && !l.stream.isClosed() && l.reg.hasAny()
}

func (l *WorkflowRunsListener) runLoop(ctx context.Context) error {
	err := listenStream(ctx, l.stream,
		func(c dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient) (*dispatchercontracts.WorkflowRunEvent, error) {
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

func (l *WorkflowRunsListener) ensureListening(ctx context.Context) error {
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

func (l *WorkflowRunsListener) startBackground(onExit func()) error {
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

func (l *WorkflowRunsListener) listen(ctx context.Context) error {
	if !l.gate.tryStart(l.stream.isClosed()) {
		return nil
	}
	defer l.gate.stop()
	return l.runLoop(ctx)
}

func (l *WorkflowRunsListener) Listen(ctx context.Context) error {
	return l.listen(ctx)
}

func (l *WorkflowRunsListener) Close() error {
	return l.stream.Close()
}

func (l *WorkflowRunsListener) isListening() bool {
	return l.gate.active()
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

func (r *subscribeClientImpl) newMetadataStream(ctx context.Context, key, value string) *reconnectingStream[dispatchercontracts.Dispatcher_SubscribeToWorkflowEventsClient] {
	return newReconnectingStreamWithLifecycle(
		ctx,
		r.l,
		"metadata stream listener",
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
}

func metadataStreamClassifier() streamClassifier {
	base := newStreamClassifier(func(ctx context.Context) bool { return ctx.Err() == nil })
	return func(ctx context.Context, err error) streamVerdict {
		if v := base(ctx, err); v != verdictNoProgress {
			return v
		}
		return verdictStopError
	}
}

func runMetadataStream(
	ctx context.Context,
	stream *reconnectingStream[dispatchercontracts.Dispatcher_SubscribeToWorkflowEventsClient],
	handler StreamHandler,
) error {
	return listenStream(
		ctx,
		stream,
		func(client dispatchercontracts.Dispatcher_SubscribeToWorkflowEventsClient) (*dispatchercontracts.WorkflowEvent, error) {
			return client.Recv()
		},
		func(event *dispatchercontracts.WorkflowEvent) error {
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
		metadataStreamClassifier(),
	)
}

func (r *subscribeClientImpl) StreamByAdditionalMetadata(ctx context.Context, key string, value string, handler StreamHandler) error {
	stream := r.newMetadataStream(ctx, key, value)
	defer func() { _ = stream.Close() }()

	return runMetadataStream(ctx, stream, handler)
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
