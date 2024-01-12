package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/logger"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/integrations"
)

type actionFunc func(args ...any) []any

// Action is an individual action that can be run by the worker.
type Action interface {
	// Name returns the name of the action
	Name() string

	// Run runs the action
	Run(args ...any) []any

	MethodFn() any

	// Service returns the service that the action belongs to
	Service() string
}

type actionImpl struct {
	name    string
	run     actionFunc
	method  any
	service string
}

func (j *actionImpl) Name() string {
	return j.name
}

func (j *actionImpl) Run(args ...interface{}) []interface{} {
	return j.run(args...)
}

func (j *actionImpl) MethodFn() any {
	return j.method
}

func (j *actionImpl) Service() string {
	return j.service
}

type Worker struct {
	client client.Client

	name string

	actions map[string]Action

	l *zerolog.Logger

	cancelMap sync.Map

	services sync.Map

	alerter errors.Alerter

	middlewares *middlewares
}

type WorkerOpt func(*WorkerOpts)

type WorkerOpts struct {
	client client.Client
	name   string
	l      *zerolog.Logger

	integrations []integrations.Integration
	alerter      errors.Alerter
}

func defaultWorkerOpts() *WorkerOpts {
	logger := logger.NewDefaultLogger("worker")

	return &WorkerOpts{
		name:         getHostName(),
		l:            &logger,
		integrations: []integrations.Integration{},
		alerter:      errors.NoOpAlerter{},
	}
}

func WithName(name string) WorkerOpt {
	return func(opts *WorkerOpts) {
		opts.name = name
	}
}

func WithClient(client client.Client) WorkerOpt {
	return func(opts *WorkerOpts) {
		opts.client = client
	}
}

func WithIntegration(integration integrations.Integration) WorkerOpt {
	return func(opts *WorkerOpts) {
		opts.integrations = append(opts.integrations, integration)
	}
}

func WithErrorAlerter(alerter errors.Alerter) WorkerOpt {
	return func(opts *WorkerOpts) {
		opts.alerter = alerter
	}
}

// NewWorker creates a new worker instance
func NewWorker(fs ...WorkerOpt) (*Worker, error) {
	opts := defaultWorkerOpts()

	for _, f := range fs {
		f(opts)
	}

	mws := newMiddlewares()

	mws.add(panicMiddleware)

	w := &Worker{
		client:      opts.client,
		name:        opts.name,
		l:           opts.l,
		actions:     map[string]Action{},
		alerter:     opts.alerter,
		middlewares: mws,
	}

	// register all integrations
	for _, integration := range opts.integrations {
		actions := integration.Actions()
		integrationId := integration.GetId()

		for _, integrationAction := range actions {
			action := fmt.Sprintf("%s:%s", integrationId, integrationAction)

			err := w.registerAction(integrationId, action, integration.ActionHandler(integrationAction))

			if err != nil {
				return nil, fmt.Errorf("could not register integration action %s: %w", action, err)
			}
		}
	}

	w.NewService("default")

	return w, nil
}

func (w *Worker) Use(mws ...MiddlewareFunc) {
	w.middlewares.add(mws...)
}

func (w *Worker) NewService(name string) *Service {
	svc := &Service{
		Name:   name,
		worker: w,
		mws:    newMiddlewares(),
	}

	w.services.Store(name, svc)

	return svc
}

func (w *Worker) On(t triggerConverter, workflow workflowConverter) error {
	// get the default service
	svc, ok := w.services.Load("default")

	if !ok {
		return fmt.Errorf("could not load default service")
	}

	return svc.(*Service).On(t, workflow)
}

// RegisterAction can be used to register a single action which can be reused across multiple workflows.
//
// An action should be of the format <service>:<verb>, for example slack:create-channel.
//
// The method must match the following signatures:
// - func(ctx context.Context) error
// - func(ctx context.Context, input *Input) error
// - func(ctx context.Context, input *Input) (*Output, error)
// - func(ctx context.Context) (*Output, error)
func (w *Worker) RegisterAction(actionId string, method any) error {
	// parse the action
	action, err := types.ParseActionID(actionId)

	if err != nil {
		return fmt.Errorf("could not parse action id: %w", err)
	}

	return w.registerAction(action.Service, action.Verb, method)
}

func (w *Worker) registerAction(service, verb string, method any) error {
	actionFunc, err := getFnFromMethod(method)

	if err != nil {
		return fmt.Errorf("could not get function from method: %w", err)
	}

	actionId := fmt.Sprintf("%s:%s", service, verb)

	// if action has already been registered, ensure that the method is the same
	if currMethod, ok := w.actions[actionId]; ok {
		if reflect.ValueOf(currMethod.MethodFn()).Pointer() != reflect.ValueOf(method).Pointer() {
			return fmt.Errorf("action %s is already registered with function %s", actionId, getFnName(currMethod.MethodFn()))
		}
	}

	w.actions[actionId] = &actionImpl{
		name:    actionId,
		run:     actionFunc,
		method:  method,
		service: service,
	}

	return nil
}

// Start starts the worker in blocking fashion
func (w *Worker) Start(ctx context.Context) error {
	actionNames := []string{}

	for _, action := range w.actions {
		actionNames = append(actionNames, action.Name())
	}

	listener, err := w.client.Dispatcher().GetActionListener(ctx, &client.GetActionListenerRequest{
		WorkerName: w.name,
		Actions:    actionNames,
	})

	if err != nil {
		return fmt.Errorf("could not get action listener: %w", err)
	}

	errCh := make(chan error)

	actionCh, err := listener.Actions(ctx, errCh)

	if err != nil {
		return fmt.Errorf("could not get action channel: %w", err)
	}

RunWorker:
	for {
		select {
		case err := <-errCh:
			w.l.Err(err).Msg("action listener error")
			break RunWorker
		case action := <-actionCh:
			go func(action *client.Action) {
				err := w.executeAction(context.Background(), action)

				if err != nil {
					w.l.Error().Err(err).Msgf("could not execute action: %s", action.ActionId)
				}

				w.l.Debug().Msgf("action %s completed", action.ActionId)
			}(action)
		case <-ctx.Done():
			w.l.Debug().Msgf("worker %s received context done, stopping", w.name)
			break RunWorker
		}
	}

	w.l.Debug().Msgf("worker %s stopped", w.name)

	err = listener.Unregister()

	if err != nil {
		return fmt.Errorf("could not unregister worker: %w", err)
	}

	return nil
}

func (w *Worker) executeAction(ctx context.Context, assignedAction *client.Action) error {
	if assignedAction.ActionType == client.ActionTypeStartStepRun {
		return w.startStepRun(ctx, assignedAction)
	} else if assignedAction.ActionType == client.ActionTypeCancelStepRun {
		return w.cancelStepRun(ctx, assignedAction)
	}

	return fmt.Errorf("unknown action type: %s", assignedAction.ActionType)
}

func (w *Worker) startStepRun(ctx context.Context, assignedAction *client.Action) error {
	// send a message that the step run started
	_, err := w.client.Dispatcher().SendActionEvent(
		ctx,
		w.getActionEvent(assignedAction, client.ActionEventTypeStarted),
	)

	if err != nil {
		return fmt.Errorf("could not send action event: %w", err)
	}

	action, ok := w.actions[assignedAction.ActionId]

	if !ok {
		return fmt.Errorf("job not found")
	}

	arg, err := decodeArgsToInterface(reflect.TypeOf(action.MethodFn()))

	if err != nil {
		return fmt.Errorf("could not decode args to interface: %w", err)
	}

	err = assignedAction.ActionPayload(arg)

	if err != nil {
		return fmt.Errorf("could not decode action payload: %w", err)
	}

	runContext, cancel := context.WithCancel(context.Background())

	w.cancelMap.Store(assignedAction.StepRunId, cancel)

	// get the action's service
	svcAny, ok := w.services.Load(action.Service())

	if !ok {
		return fmt.Errorf("could not load service %s", action.Service())
	}

	svc := svcAny.(*Service)

	// wrap the run with middleware. start by wrapping the global worker middleware, then
	// the service-specific middleware
	return w.middlewares.runAll(runContext, func(ctx context.Context) error {
		return svc.mws.runAll(ctx, func(ctx context.Context) error {
			runResults := action.Run(ctx, arg)

			// check whether run context was cancelled while action was running
			select {
			case <-ctx.Done():
				w.l.Debug().Msgf("step run %s was cancelled, returning", assignedAction.StepRunId)
				return nil
			default:
			}

			var result any

			if len(runResults) == 2 {
				result = runResults[0]
			}

			if runResults[len(runResults)-1] != nil {
				err = runResults[len(runResults)-1].(error)
			}

			if err != nil {
				failureEvent := w.getActionEvent(assignedAction, client.ActionEventTypeFailed)

				w.alerter.SendAlert(context.Background(), err, map[string]interface{}{
					"actionId":   assignedAction.ActionId,
					"workerId":   assignedAction.WorkerId,
					"stepRunId":  assignedAction.StepRunId,
					"jobName":    assignedAction.JobName,
					"actionType": assignedAction.ActionType,
				})

				failureEvent.EventPayload = err.Error()

				_, err := w.client.Dispatcher().SendActionEvent(
					ctx,
					failureEvent,
				)

				if err != nil {
					return fmt.Errorf("could not send action event: %w", err)
				}

				return err
			}

			// send a message that the step run completed
			finishedEvent, err := w.getActionFinishedEvent(assignedAction, result)

			if err != nil {
				return fmt.Errorf("could not create finished event: %w", err)
			}

			_, err = w.client.Dispatcher().SendActionEvent(
				ctx,
				finishedEvent,
			)

			if err != nil {
				return fmt.Errorf("could not send action event: %w", err)
			}

			return nil
		})
	})
}

func (w *Worker) cancelStepRun(ctx context.Context, assignedAction *client.Action) error {
	cancel, ok := w.cancelMap.Load(assignedAction.StepRunId)

	if !ok {
		return fmt.Errorf("could not find step run to cancel")
	}

	w.l.Debug().Msgf("cancelling step run %s", assignedAction.StepRunId)

	cancelFn := cancel.(context.CancelFunc)

	cancelFn()

	return nil
}

func (w *Worker) getActionEvent(action *client.Action, eventType client.ActionEventType) *client.ActionEvent {
	timestamp := time.Now().UTC()

	return &client.ActionEvent{
		Action:         action,
		EventTimestamp: &timestamp,
		EventType:      eventType,
	}
}

func (w *Worker) getActionFinishedEvent(action *client.Action, output any) (*client.ActionEvent, error) {
	event := w.getActionEvent(action, client.ActionEventTypeCompleted)

	outputBytes, err := json.Marshal(output)

	if err != nil {
		return nil, fmt.Errorf("could not marshal step output: %w", err)
	}

	event.EventPayload = string(outputBytes)

	return event, nil
}

func getHostName() string {
	hostName, err := os.Hostname()
	if err != nil {
		hostName = "Unknown"
	}
	return hostName
}
