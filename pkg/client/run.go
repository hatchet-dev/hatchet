package client

import (
	"context"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/validator"
)

type RunHandler func(event *StepRunEvent) error

type RunClient interface {
	On(ctx context.Context, workflowRunId string, handler RunHandler) error
}

type StepRunEventType string

const (
	StepRunEventTypeStarted   StepRunEventType = "STEP_RUN_EVENT_TYPE_STARTED"
	StepRunEventTypeCompleted StepRunEventType = "STEP_RUN_EVENT_TYPE_COMPLETED"
	StepRunEventTypeFailed    StepRunEventType = "STEP_RUN_EVENT_TYPE_FAILED"
	StepRunEventTypeCancelled StepRunEventType = "STEP_RUN_EVENT_TYPE_CANCELLED"
	StepRunEventTypeTimedOut  StepRunEventType = "STEP_RUN_EVENT_TYPE_TIMED_OUT"
)

type StepRunEvent struct {
	Type StepRunEventType

	Payload []byte
}

type ClientEventListener interface {
	OnStepRunEvent(ctx context.Context, event *StepRunEvent) error
	// OnWorkflowRunEvent(ctx context.Context, event *WorkflowRunEvent) error
}

type runClientImpl struct {
	client dispatchercontracts.DispatcherClient

	l *zerolog.Logger

	v validator.Validator

	ctx *contextLoader
}

func newRun(conn *grpc.ClientConn, opts *sharedClientOpts) RunClient {
	return &runClientImpl{
		client: dispatchercontracts.NewDispatcherClient(conn),
		l:      opts.l,
		v:      opts.v,
		ctx:    opts.ctxLoader,
	}
}

func (r *runClientImpl) On(ctx context.Context, workflowRunId string, handler RunHandler) error {
	stream, err := r.client.SubscribeToWorkflowEvents(r.ctx.newContext(ctx), &dispatchercontracts.SubscribeToWorkflowEventsRequest{
		WorkflowRunId: workflowRunId,
	})

	if err != nil {
		return err
	}

	for {
		event, err := stream.Recv()

		if err != nil {
			return err
		}

		var eventType StepRunEventType

		switch event.EventType {
		case dispatchercontracts.ResourceEventType_RESOURCE_EVENT_TYPE_STARTED:
			eventType = StepRunEventTypeStarted
		case dispatchercontracts.ResourceEventType_RESOURCE_EVENT_TYPE_COMPLETED:
			eventType = StepRunEventTypeCompleted
		case dispatchercontracts.ResourceEventType_RESOURCE_EVENT_TYPE_FAILED:
			eventType = StepRunEventTypeFailed
		case dispatchercontracts.ResourceEventType_RESOURCE_EVENT_TYPE_CANCELLED:
			eventType = StepRunEventTypeCancelled
		case dispatchercontracts.ResourceEventType_RESOURCE_EVENT_TYPE_TIMED_OUT:
			eventType = StepRunEventTypeTimedOut
		}

		if err := handler(&StepRunEvent{
			Type:    eventType,
			Payload: []byte(event.EventPayload),
		}); err != nil {
			return err
		}
	}
}
