package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type DispatcherClient interface {
	GetActionListener(ctx context.Context, req *GetActionListenerRequest) (WorkerActionListener, error)

	SendStepActionEvent(ctx context.Context, in *ActionEvent) (*ActionEventResponse, error)

	SendGroupKeyActionEvent(ctx context.Context, in *ActionEvent) (*ActionEventResponse, error)

	ReleaseSlot(ctx context.Context, stepRunId string) error

	RefreshTimeout(ctx context.Context, stepRunId string, incrementTimeoutBy string) error
}

const (
	DefaultActionListenerRetryInterval = 5 * time.Second
	DefaultActionListenerRetryCount    = 5
)

// TODO: add validator to client side
type GetActionListenerRequest struct {
	WorkerName string
	Services   []string
	Actions    []string
	MaxRuns    *int
}

// ActionPayload unmarshals the action payload into the target. It also validates the resulting target.
type ActionPayload func(target interface{}) error

type ActionType string

const (
	ActionTypeStartStepRun     ActionType = "START_STEP_RUN"
	ActionTypeCancelStepRun    ActionType = "CANCEL_STEP_RUN"
	ActionTypeStartGetGroupKey ActionType = "START_GET_GROUP_KEY"
)

type Action struct {
	// the worker id
	WorkerId string

	// the tenant id
	TenantId string

	// the workflow run id
	WorkflowRunId string

	// the get group key run id
	GetGroupKeyRunId string

	// the job id
	JobId string

	// the job name
	JobName string

	// the job run id
	JobRunId string

	// the step id
	StepId string

	// the step name
	StepName string

	// the step run id
	StepRunId string

	// the action id
	ActionId string

	// the action payload
	ActionPayload []byte

	// the action type
	ActionType ActionType

	// the count of the retry attempt
	RetryCount int32
}

type WorkerActionListener interface {
	Actions(ctx context.Context) (<-chan *Action, error)

	Unregister() error
}

type ActionEventType string

const (
	ActionEventTypeUnknown   ActionEventType = "STEP_EVENT_TYPE_UNKNOWN"
	ActionEventTypeStarted   ActionEventType = "STEP_EVENT_TYPE_STARTED"
	ActionEventTypeCompleted ActionEventType = "STEP_EVENT_TYPE_COMPLETED"
	ActionEventTypeFailed    ActionEventType = "STEP_EVENT_TYPE_FAILED"
)

type ActionEvent struct {
	*Action

	// the event timestamp
	EventTimestamp *time.Time

	// the step event type
	EventType ActionEventType

	// The event payload. This must be JSON-compatible as it gets marshalled to a JSON string.
	EventPayload interface{}
}

type ActionEventResponse struct {
	// the tenant id
	TenantId string

	// the id of the worker
	WorkerId string
}

type dispatcherClientImpl struct {
	client dispatchercontracts.DispatcherClient

	tenantId string

	l *zerolog.Logger

	v validator.Validator

	ctx *contextLoader
}

func newDispatcher(conn *grpc.ClientConn, opts *sharedClientOpts) DispatcherClient {
	return &dispatcherClientImpl{
		client:   dispatchercontracts.NewDispatcherClient(conn),
		tenantId: opts.tenantId,
		l:        opts.l,
		v:        opts.v,
		ctx:      opts.ctxLoader,
	}
}

type ListenerStrategy string

const (
	ListenerStrategyV1 ListenerStrategy = "v1"
	ListenerStrategyV2 ListenerStrategy = "v2"
)

type actionListenerImpl struct {
	client dispatchercontracts.DispatcherClient

	tenantId string

	listenClient dispatchercontracts.Dispatcher_ListenClient

	workerId string

	l *zerolog.Logger

	v validator.Validator

	ctx *contextLoader

	listenerStrategy ListenerStrategy
}

func (d *dispatcherClientImpl) newActionListener(ctx context.Context, req *GetActionListenerRequest) (*actionListenerImpl, error) {
	// validate the request
	if err := d.v.Validate(req); err != nil {
		return nil, err
	}

	registerReq := &dispatchercontracts.WorkerRegisterRequest{
		WorkerName: req.WorkerName,
		Actions:    req.Actions,
		Services:   req.Services,
	}

	if req.MaxRuns != nil {
		mr := int32(*req.MaxRuns)
		registerReq.MaxRuns = &mr
	}

	// register the worker
	resp, err := d.client.Register(d.ctx.newContext(ctx), registerReq)

	if err != nil {
		return nil, fmt.Errorf("could not register the worker: %w", err)
	}

	d.l.Debug().Msgf("Registered worker with id: %s", resp.WorkerId)

	// subscribe to the worker
	listener, err := d.client.ListenV2(d.ctx.newContext(ctx), &dispatchercontracts.WorkerListenRequest{
		WorkerId: resp.WorkerId,
	})

	if err != nil {
		return nil, fmt.Errorf("could not subscribe to the worker: %w", err)
	}

	return &actionListenerImpl{
		client:           d.client,
		listenClient:     listener,
		workerId:         resp.WorkerId,
		l:                d.l,
		v:                d.v,
		tenantId:         d.tenantId,
		ctx:              d.ctx,
		listenerStrategy: ListenerStrategyV2,
	}, nil
}

func (a *actionListenerImpl) Actions(ctx context.Context) (<-chan *Action, error) {
	ch := make(chan *Action)

	a.l.Debug().Msgf("Starting to listen for actions")

	// update the worker with a last heartbeat time every 4 seconds as long as the worker is connected
	go func() {
		timer := time.NewTicker(100 * time.Millisecond)
		defer timer.Stop()

		// set last heartbeat to 5 seconds ago so that the first heartbeat is sent immediately
		lastHeartbeat := time.Now().Add(-5 * time.Second)

		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				if now := time.Now().UTC(); lastHeartbeat.Add(4 * time.Second).Before(now) {
					a.l.Debug().Msgf("updating worker %s heartbeat", a.workerId)

					_, err := a.client.Heartbeat(a.ctx.newContext(ctx), &dispatchercontracts.HeartbeatRequest{
						WorkerId:    a.workerId,
						HeartbeatAt: timestamppb.New(now),
					})

					if err != nil {
						a.l.Error().Err(err).Msgf("could not update worker %s heartbeat", a.workerId)

						// if the heartbeat method is unimplemented, don't continue to send heartbeats
						if status.Code(err) == codes.Unimplemented {
							return
						}
					}

					lastHeartbeat = time.Now().UTC()
				}
			}
		}
	}()

	go func() {
		for {
			assignedAction, err := a.listenClient.Recv()

			if err != nil {
				// if context is cancelled, unsubscribe and close the channel
				if ctx.Err() != nil {
					a.l.Debug().Msgf("Context cancelled, closing channel")

					defer close(ch)
					err := a.listenClient.CloseSend()

					if err != nil {
						a.l.Error().Msgf("Failed to close send: %v", err)
						panic(fmt.Errorf("failed to close send: %w", err))
					}

					return
				}

				// if this is an unimplemented error, default to v1
				if a.listenerStrategy == ListenerStrategyV2 && status.Code(err) == codes.Unimplemented {
					a.l.Debug().Msgf("Falling back to v1 listener strategy")
					a.listenerStrategy = ListenerStrategyV1
				}

				err = a.retrySubscribe(ctx)

				if err != nil {
					a.l.Error().Msgf("Failed to resubscribe: %v", err)
					panic(fmt.Errorf("failed to resubscribe: %w", err))
				}

				continue
			}

			var actionType ActionType

			switch assignedAction.ActionType {
			case dispatchercontracts.ActionType_START_STEP_RUN:
				actionType = ActionTypeStartStepRun
			case dispatchercontracts.ActionType_CANCEL_STEP_RUN:
				actionType = ActionTypeCancelStepRun
			case dispatchercontracts.ActionType_START_GET_GROUP_KEY:
				actionType = ActionTypeStartGetGroupKey
			default:
				a.l.Error().Msgf("Unknown action type: %s", assignedAction.ActionType)
				continue
			}

			a.l.Debug().Msgf("Received action type: %s", actionType)

			unquoted := assignedAction.ActionPayload

			ch <- &Action{
				TenantId:         assignedAction.TenantId,
				WorkflowRunId:    assignedAction.WorkflowRunId,
				GetGroupKeyRunId: assignedAction.GetGroupKeyRunId,
				WorkerId:         a.workerId,
				JobId:            assignedAction.JobId,
				JobName:          assignedAction.JobName,
				JobRunId:         assignedAction.JobRunId,
				StepId:           assignedAction.StepId,
				StepName:         assignedAction.StepName,
				StepRunId:        assignedAction.StepRunId,
				ActionId:         assignedAction.ActionId,
				ActionType:       actionType,
				ActionPayload:    []byte(unquoted),
				RetryCount:       assignedAction.RetryCount,
			}
		}
	}()

	return ch, nil
}

func (a *actionListenerImpl) retrySubscribe(ctx context.Context) error {
	retries := 0

	for retries < DefaultActionListenerRetryCount {
		time.Sleep(DefaultActionListenerRetryInterval)

		var err error
		var listenClient dispatchercontracts.Dispatcher_ListenClient

		if a.listenerStrategy == ListenerStrategyV1 {
			listenClient, err = a.client.Listen(a.ctx.newContext(ctx), &dispatchercontracts.WorkerListenRequest{
				WorkerId: a.workerId,
			})
		} else if a.listenerStrategy == ListenerStrategyV2 {
			listenClient, err = a.client.ListenV2(a.ctx.newContext(ctx), &dispatchercontracts.WorkerListenRequest{
				WorkerId: a.workerId,
			})
		}

		if err != nil {
			retries++
			a.l.Error().Err(err).Msgf("could not subscribe to the worker")
			continue
		}

		a.listenClient = listenClient

		return nil
	}

	return fmt.Errorf("could not subscribe to the worker after %d retries", retries)
}

func (a *actionListenerImpl) Unregister() error {
	_, err := a.client.Unsubscribe(
		a.ctx.newContext(context.Background()),
		&dispatchercontracts.WorkerUnsubscribeRequest{
			WorkerId: a.workerId,
		},
	)

	if err != nil {
		return err
	}

	return nil
}

func (d *dispatcherClientImpl) GetActionListener(ctx context.Context, req *GetActionListenerRequest) (WorkerActionListener, error) {
	return d.newActionListener(ctx, req)
}

func (d *dispatcherClientImpl) SendStepActionEvent(ctx context.Context, in *ActionEvent) (*ActionEventResponse, error) {
	// validate the request
	if err := d.v.Validate(in); err != nil {
		return nil, err
	}

	payloadBytes, err := json.Marshal(in.EventPayload)

	if err != nil {
		return nil, err
	}

	var actionEventType dispatchercontracts.StepActionEventType

	switch in.EventType {
	case ActionEventTypeStarted:
		actionEventType = dispatchercontracts.StepActionEventType_STEP_EVENT_TYPE_STARTED
	case ActionEventTypeCompleted:
		actionEventType = dispatchercontracts.StepActionEventType_STEP_EVENT_TYPE_COMPLETED
	case ActionEventTypeFailed:
		actionEventType = dispatchercontracts.StepActionEventType_STEP_EVENT_TYPE_FAILED
	default:
		actionEventType = dispatchercontracts.StepActionEventType_STEP_EVENT_TYPE_UNKNOWN
	}

	resp, err := d.client.SendStepActionEvent(d.ctx.newContext(ctx), &dispatchercontracts.StepActionEvent{
		WorkerId:       in.WorkerId,
		JobId:          in.JobId,
		JobRunId:       in.JobRunId,
		StepId:         in.StepId,
		StepRunId:      in.StepRunId,
		ActionId:       in.ActionId,
		EventTimestamp: timestamppb.New(*in.EventTimestamp),
		EventType:      actionEventType,
		EventPayload:   string(payloadBytes),
	})

	if err != nil {
		return nil, err
	}

	return &ActionEventResponse{
		TenantId: resp.TenantId,
		WorkerId: resp.WorkerId,
	}, nil
}

func (d *dispatcherClientImpl) SendGroupKeyActionEvent(ctx context.Context, in *ActionEvent) (*ActionEventResponse, error) {
	// validate the request
	if err := d.v.Validate(in); err != nil {
		return nil, err
	}

	payloadBytes, err := json.Marshal(in.EventPayload)

	if err != nil {
		return nil, err
	}

	var actionEventType dispatchercontracts.GroupKeyActionEventType

	switch in.EventType {
	case ActionEventTypeStarted:
		actionEventType = dispatchercontracts.GroupKeyActionEventType_GROUP_KEY_EVENT_TYPE_STARTED
	case ActionEventTypeCompleted:
		actionEventType = dispatchercontracts.GroupKeyActionEventType_GROUP_KEY_EVENT_TYPE_COMPLETED
	case ActionEventTypeFailed:
		actionEventType = dispatchercontracts.GroupKeyActionEventType_GROUP_KEY_EVENT_TYPE_FAILED
	default:
		actionEventType = dispatchercontracts.GroupKeyActionEventType_GROUP_KEY_EVENT_TYPE_UNKNOWN
	}

	resp, err := d.client.SendGroupKeyActionEvent(d.ctx.newContext(ctx), &dispatchercontracts.GroupKeyActionEvent{
		WorkerId:         in.WorkerId,
		WorkflowRunId:    in.WorkflowRunId,
		GetGroupKeyRunId: in.GetGroupKeyRunId,
		ActionId:         in.ActionId,
		EventTimestamp:   timestamppb.New(*in.EventTimestamp),
		EventType:        actionEventType,
		EventPayload:     string(payloadBytes),
	})

	if err != nil {
		return nil, err
	}

	return &ActionEventResponse{
		TenantId: resp.TenantId,
		WorkerId: resp.WorkerId,
	}, nil
}

func (a *dispatcherClientImpl) ReleaseSlot(ctx context.Context, stepRunId string) error {
	_, err := a.client.ReleaseSlot(a.ctx.newContext(ctx), &dispatchercontracts.ReleaseSlotRequest{
		StepRunId: stepRunId,
	})

	if err != nil {
		return err
	}

	return nil
}

func (a *dispatcherClientImpl) RefreshTimeout(ctx context.Context, stepRunId string, incrementTimeoutBy string) error {
	_, err := a.client.RefreshTimeout(a.ctx.newContext(ctx), &dispatchercontracts.RefreshTimeoutRequest{
		StepRunId:          stepRunId,
		IncrementTimeoutBy: incrementTimeoutBy,
	})

	if err != nil {
		return err
	}

	return nil
}
