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

type WorkflowRunsListener struct {
	*reconnectingListener[string, *dispatchercontracts.SubscribeToWorkflowRunsRequest, *dispatchercontracts.WorkflowRunEvent, dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient]
}

func newWorkflowRunsListener(
	constructor func(context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error),
	l *zerolog.Logger,
) *WorkflowRunsListener {
	return &WorkflowRunsListener{
		reconnectingListener: &reconnectingListener[string, *dispatchercontracts.SubscribeToWorkflowRunsRequest, *dispatchercontracts.WorkflowRunEvent, dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient]{
			constructor: constructor,
			l:           l,
			requestForKey: func(workflowRunId string) *dispatchercontracts.SubscribeToWorkflowRunsRequest {
				return &dispatchercontracts.SubscribeToWorkflowRunsRequest{
					WorkflowRunId: workflowRunId,
				}
			},
			keyForEvent: func(event *dispatchercontracts.WorkflowRunEvent) string {
				return event.WorkflowRunId
			},
		},
	}
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

	w := newWorkflowRunsListener(constructor, r.l)

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

func (l *WorkflowRunsListener) AddWorkflowRun(
	workflowRunId, sessionId string,
	handler WorkflowRunEventHandler,
) error {
	return l.addHandler(workflowRunId, sessionId, func(event *dispatchercontracts.WorkflowRunEvent) error {
		workflowRunEvent, err := workflowRunEventToDeprecatedWorkflowRunEvent(event)
		if err != nil {
			return err
		}

		return handler(workflowRunEvent)
	})
}

func (l *WorkflowRunsListener) RemoveWorkflowRun(
	workflowRunId, sessionId string,
) {
	l.removeHandler(workflowRunId, sessionId)
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
	client                  dispatchercontracts.DispatcherClient
	clientv1                sharedcontracts.V1DispatcherClient
	v                       validator.Validator
	l                       *zerolog.Logger
	ctx                     *contextLoader
	workflowRunListener     *WorkflowRunsListener
	durableEventsListener   *DurableEventsListener
	workflowRunListenerMu   sync.Mutex
	durableEventsListenerMu sync.Mutex
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
