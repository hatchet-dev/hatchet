package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/validator"
)

type DispatcherClient interface {
	GetActionListener(ctx context.Context, req *GetActionListenerRequest) (WorkerActionListener, error)

	SendActionEvent(ctx context.Context, in *ActionEvent) (*ActionEventResponse, error)
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
}

// ActionPayload unmarshals the action payload into the target. It also validates the resulting target.
type ActionPayload func(target interface{}) error

type ActionType string

const (
	ActionTypeStartStepRun  ActionType = "START_STEP_RUN"
	ActionTypeCancelStepRun ActionType = "CANCEL_STEP_RUN"
)

type Action struct {
	// the worker id
	WorkerId string

	// the tenant id
	TenantId string

	// the job id
	JobId string

	// the job name
	JobName string

	// the job run id
	JobRunId string

	// the step id
	StepId string

	// the step run id
	StepRunId string

	// the action id
	ActionId string

	// the action payload
	ActionPayload []byte

	// the action type
	ActionType ActionType
}

type WorkerActionListener interface {
	Actions(ctx context.Context, errCh chan<- error) (<-chan *Action, error)

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

type actionListenerImpl struct {
	client dispatchercontracts.DispatcherClient

	tenantId string

	listenClient dispatchercontracts.Dispatcher_ListenClient

	workerId string

	l *zerolog.Logger

	v validator.Validator

	ctx *contextLoader
}

func (d *dispatcherClientImpl) newActionListener(ctx context.Context, req *GetActionListenerRequest) (*actionListenerImpl, error) {
	// validate the request
	if err := d.v.Validate(req); err != nil {
		return nil, err
	}

	// register the worker
	resp, err := d.client.Register(d.ctx.newContext(ctx), &dispatchercontracts.WorkerRegisterRequest{
		WorkerName: req.WorkerName,
		Actions:    req.Actions,
		Services:   req.Services,
	})

	if err != nil {
		return nil, fmt.Errorf("could not register the worker: %w", err)
	}

	d.l.Debug().Msgf("Registered worker with id: %s", resp.WorkerId)

	// subscribe to the worker
	listener, err := d.client.Listen(d.ctx.newContext(ctx), &dispatchercontracts.WorkerListenRequest{
		WorkerId: resp.WorkerId,
	})

	if err != nil {
		return nil, fmt.Errorf("could not subscribe to the worker: %w", err)
	}

	return &actionListenerImpl{
		client:       d.client,
		listenClient: listener,
		workerId:     resp.WorkerId,
		l:            d.l,
		v:            d.v,
		tenantId:     d.tenantId,
		ctx:          d.ctx,
	}, nil
}

func (a *actionListenerImpl) Actions(ctx context.Context, errCh chan<- error) (<-chan *Action, error) {
	ch := make(chan *Action)

	a.l.Debug().Msgf("Starting to listen for actions")

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
						errCh <- fmt.Errorf("failed to close send: %w", err)
					}

					return
				}

				statusErr, isStatusErr := status.FromError(err)

				// latter case handles errors like `rpc error: code = Unavailable desc = error reading from server: EOF`
				// which apparently is not an EOF error
				if errors.Is(err, io.EOF) || (isStatusErr && statusErr.Code() == codes.Unavailable) {
					err = a.retrySubscribe(ctx)

					if err != nil {
						a.l.Error().Msgf("Failed to subscribe: %v", err)
						errCh <- fmt.Errorf("failed to subscribe: %w", err)
						return
					}

					continue
				}

				a.l.Error().Msgf("Failed to receive message: %v", err)
				errCh <- fmt.Errorf("failed to receive message: %w", err)
				return
			}

			var actionType ActionType

			switch assignedAction.ActionType {
			case dispatchercontracts.ActionType_START_STEP_RUN:
				actionType = ActionTypeStartStepRun
			case dispatchercontracts.ActionType_CANCEL_STEP_RUN:
				actionType = ActionTypeCancelStepRun
			default:
				a.l.Error().Msgf("Unknown action type: %s", assignedAction.ActionType)
				continue
			}

			a.l.Debug().Msgf("Received action type: %s", actionType)

			unquoted, err := strconv.Unquote(assignedAction.ActionPayload)

			if err != nil {
				a.l.Err(err).Msgf("Error unquoting payload for action: %s", assignedAction.ActionType)
				continue
			}

			ch <- &Action{
				TenantId:      assignedAction.TenantId,
				WorkerId:      a.workerId,
				JobId:         assignedAction.JobId,
				JobName:       assignedAction.JobName,
				JobRunId:      assignedAction.JobRunId,
				StepId:        assignedAction.StepId,
				StepRunId:     assignedAction.StepRunId,
				ActionId:      assignedAction.ActionId,
				ActionType:    actionType,
				ActionPayload: []byte(unquoted),
			}
		}
	}()

	return ch, nil
}

func (a *actionListenerImpl) retrySubscribe(ctx context.Context) error {
	retries := 0

	for retries < DefaultActionListenerRetryCount {
		time.Sleep(DefaultActionListenerRetryInterval)

		listenClient, err := a.client.Listen(a.ctx.newContext(ctx), &dispatchercontracts.WorkerListenRequest{
			WorkerId: a.workerId,
		})

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

func (d *dispatcherClientImpl) SendActionEvent(ctx context.Context, in *ActionEvent) (*ActionEventResponse, error) {
	// validate the request
	if err := d.v.Validate(in); err != nil {
		return nil, err
	}

	payloadBytes, err := json.Marshal(in.EventPayload)

	if err != nil {
		return nil, err
	}

	var actionEventType dispatchercontracts.ActionEventType

	switch in.EventType {
	case ActionEventTypeStarted:
		actionEventType = dispatchercontracts.ActionEventType_STEP_EVENT_TYPE_STARTED
	case ActionEventTypeCompleted:
		actionEventType = dispatchercontracts.ActionEventType_STEP_EVENT_TYPE_COMPLETED
	case ActionEventTypeFailed:
		actionEventType = dispatchercontracts.ActionEventType_STEP_EVENT_TYPE_FAILED
	default:
		actionEventType = dispatchercontracts.ActionEventType_STEP_EVENT_TYPE_UNKNOWN
	}

	resp, err := d.client.SendActionEvent(d.ctx.newContext(ctx), &dispatchercontracts.ActionEvent{
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
