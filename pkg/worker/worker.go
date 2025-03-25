package worker

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/compute"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/integrations"
	"github.com/hatchet-dev/hatchet/pkg/logger"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

type actionFunc func(args ...any) []any

// Action is an individual action that can be run by the worker.
type Action interface {
	// Name returns the name of the action
	Name() string

	// Run runs the action
	Run(args ...any) []any

	MethodFn() any

	ConcurrencyFn() GetWorkflowConcurrencyGroupFn

	// Service returns the service that the action belongs to
	Service() string

	Compute() *compute.Compute
}

type actionImpl struct {
	name                 string
	run                  actionFunc
	runConcurrencyAction GetWorkflowConcurrencyGroupFn
	method               any
	service              string

	compute *compute.Compute
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

func (j *actionImpl) ConcurrencyFn() GetWorkflowConcurrencyGroupFn {
	return j.runConcurrencyAction
}

func (j *actionImpl) Service() string {
	return j.service
}

func (j *actionImpl) Compute() *compute.Compute {
	return j.compute
}

type ActionRegistry map[string]Action

type Worker struct {
	client client.Client

	name string

	actions ActionRegistry

	registered_workflows map[string]bool

	l *zerolog.Logger

	cancelMap sync.Map

	cancelConcurrencyMap sync.Map

	services sync.Map

	alerter errors.Alerter

	middlewares *middlewares

	maxRuns *int

	initActionNames []string

	labels map[string]interface{}

	id *string
}

type WorkerOpt func(*WorkerOpts)

type WorkerOpts struct {
	client client.Client
	name   string
	l      *zerolog.Logger

	integrations []integrations.Integration
	alerter      errors.Alerter
	maxRuns      *int

	actions []string

	labels map[string]interface{}
}

func defaultWorkerOpts() *WorkerOpts {

	return &WorkerOpts{
		name:         getHostName(),
		integrations: []integrations.Integration{},
		alerter:      errors.NoOpAlerter{},
	}
}

func WithInternalData(actions []string) WorkerOpt {
	return func(opts *WorkerOpts) {
		opts.actions = actions
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

func WithMaxRuns(maxRuns int) WorkerOpt {
	return func(opts *WorkerOpts) {
		opts.maxRuns = &maxRuns
	}
}

func WithLabels(labels map[string]interface{}) WorkerOpt {
	return func(opts *WorkerOpts) {
		opts.labels = labels
	}
}

func WithLogger(l *zerolog.Logger) WorkerOpt {
	return func(opts *WorkerOpts) {
		if opts.l != nil {
			opts.l.Warn().Msg("WithLogger called multiple times or after WithLogLevel, ignoring")
			return
		}

		opts.l = l
	}
}

func WithLogLevel(lvl string) WorkerOpt {
	return func(opts *WorkerOpts) {
		var l zerolog.Logger

		if opts.l == nil {
			l = logger.NewDefaultLogger("worker")
		} else {
			l = *opts.l
		}

		lvl, err := zerolog.ParseLevel(lvl)

		if err == nil {
			l = l.Level(lvl)
		}

		opts.l = &l
	}
}

// NewWorker creates a new worker instance
func NewWorker(fs ...WorkerOpt) (*Worker, error) {
	opts := defaultWorkerOpts()

	for _, f := range fs {
		f(opts)
	}

	mws := newMiddlewares()

	if opts.labels != nil {
		for _, value := range opts.labels {
			if reflect.TypeOf(value).Kind() != reflect.String && reflect.TypeOf(value).Kind() != reflect.Int {
				return nil, fmt.Errorf("invalid label value: %v", value)
			}
		}
	}

	if opts.l == nil {
		l := logger.NewDefaultLogger("worker")
		opts.l = &l
	}

	w := &Worker{
		client:               opts.client,
		name:                 opts.name,
		l:                    opts.l,
		actions:              ActionRegistry{},
		alerter:              opts.alerter,
		middlewares:          mws,
		maxRuns:              opts.maxRuns,
		initActionNames:      opts.actions,
		labels:               opts.labels,
		registered_workflows: map[string]bool{},
	}

	mws.add(w.panicMiddleware)

	// FIXME: Remove integrations
	// register all integrations
	for _, integration := range opts.integrations {
		actions := integration.Actions()
		integrationId := integration.GetId()

		for _, integrationAction := range actions {
			action := fmt.Sprintf("%s:%s", integrationId, integrationAction)

			err := w.registerAction(integrationId, action, integration.ActionHandler(integrationAction), nil)

			if err != nil {
				return nil, fmt.Errorf("could not register integration action %s: %w", action, err)
			}
		}
	}

	return w, nil
}

func (w *Worker) Use(mws ...MiddlewareFunc) {
	w.middlewares.add(mws...)
}

func (w *Worker) NewService(name string) *Service {
	namespace := w.client.Namespace()
	namespaced := namespace + name

	svc := &Service{
		Name:   namespaced,
		worker: w,
		mws:    newMiddlewares(),
	}

	w.services.Store(namespaced, svc)

	return svc
}

func (w *Worker) RegisterWorkflow(workflow workflowConverter) error {
	wf, ok := workflow.(*WorkflowJob)
	if ok && wf.On == nil {
		return fmt.Errorf("workflow must have an trigger defined via the `On` field")
	}

	w.registered_workflows[wf.Name] = true

	return w.On(workflow.ToWorkflowTrigger(), workflow)
}

func (w *Worker) RegisterWorkflowV1(workflow *contracts.CreateWorkflowVersionRequest) error {
	namespace := w.client.Namespace()
	namespaced := namespace + workflow.Name

	w.registered_workflows[namespaced] = true

	return w.client.Admin().PutWorkflowV1(workflow)
}

// Deprecated: Use RegisterWorkflow instead
func (w *Worker) On(t triggerConverter, workflow workflowConverter) error {
	svcName := workflow.ToWorkflow("", "").Name
	svcName = strings.ToLower(svcName)

	// get the default service
	svc, ok := w.services.Load(svcName)

	if !ok {
		return w.NewService(svcName).On(t, workflow)
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

	return w.registerAction(action.Service, action.Verb, method, nil)
}

func (w *Worker) registerAction(service, verb string, method any, compute *compute.Compute) error {

	actionId := fmt.Sprintf("%s:%s", service, verb)

	// if the service is "concurrency", then this is a special action
	if service == "concurrency" {
		w.actions[actionId] = &actionImpl{
			name:                 actionId,
			runConcurrencyAction: method.(GetWorkflowConcurrencyGroupFn),
			method:               method,
			service:              service,
			compute:              compute,
		}

		return nil
	}

	actionFunc, err := getFnFromMethod(method)

	if err != nil {
		fmt.Println("err", err)
		return fmt.Errorf("could not get function from method: %w", err)
	}

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
		compute: compute,
	}

	return nil
}

// Start starts the worker in non-blocking fashion, returning a cleanup function and an error if the
// worker could not be started.
func (w *Worker) Start() (func() error, error) {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		err := w.startBlocking(ctx)

		if err != nil {
			// NOTE: this matches the behavior of the old worker, until we change the signature of Start
			panic(err)
		}
	}()

	cleanup := func() error {
		cancel()

		w.l.Debug().Msgf("worker %s is stopping...", w.name)

		return nil
	}

	return cleanup, nil
}

// Run starts the worker in blocking fashion, returning an error if the worker could not be started
// or if the worker stopped due to a networking issue.
func (w *Worker) Run(ctx context.Context) error {
	return w.startBlocking(ctx)
}

func (w *Worker) Logger() *zerolog.Logger {
	return w.l
}

func (w *Worker) ID() *string {
	return w.id
}

func (w *Worker) startBlocking(ctx context.Context) error {
	actionNames := []string{}

	fmt.Println("actions", w.actions)

	// Create a dictionary of services by splitting actions "service:action"
	services := make(map[string][]Action)
	for _, action := range w.actions {
		parts := strings.Split(action.Name(), ":")
		if len(parts) == 2 {
			serviceName := parts[0]
			services[serviceName] = append(services[serviceName], action)
		}
	}

	// Create a new service for each service name
	for serviceName := range services {
		// Create a new service for this group of actions
		// FIXME none of this service stuff makes sense...
		w.NewService(serviceName)
	}

	for _, action := range w.actions {

		if w.client.RunnableActions() != nil {
			if !slices.Contains(w.client.RunnableActions(), action.Name()) {
				continue
			}
		}

		actionNames = append(actionNames, action.Name())
	}

	w.l.Debug().Msgf("worker %s is listening for actions: %v", w.name, actionNames)

	_ = NewManagedCompute(&w.actions, w.client, 1)

	listener, id, err := w.client.Dispatcher().GetActionListener(ctx, &client.GetActionListenerRequest{
		WorkerName: w.name,
		Actions:    actionNames,
		MaxRuns:    w.maxRuns,
		Labels:     w.labels,
	})

	w.id = id

	if err != nil {
		return fmt.Errorf("could not get action listener: %w", err)
	}

	defer func() {
		err := listener.Unregister()

		if err != nil {
			w.l.Error().Err(err).Msg("could not unregister worker")
		}
	}()

	listenerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	actionCh, errCh, err := listener.Actions(listenerCtx)

	if err != nil {
		return fmt.Errorf("could not get action channel: %w", err)
	}

	go func() {
		for {
			select {
			case action, ok := <-actionCh:
				if !ok {
					w.l.Debug().Msgf("worker %s received action channel closed, stopping", w.name)
					return
				}

				go func(action *client.Action) {
					err := w.executeAction(context.Background(), action)

					if err != nil {
						w.l.Error().Err(err).Msgf("could not execute action: %s", action.ActionId)
					}

					w.l.Debug().Msgf("action %s completed", action.ActionId)
				}(action)
			case <-ctx.Done():
				w.l.Debug().Msgf("worker %s received context done, stopping", w.name)
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		w.l.Debug().Msgf("worker %s received context done, stopping", w.name)
		return nil
	case err := <-errCh:
		w.l.Error().Err(err).Msg("error from listener")
		return err
	}
}

func (w *Worker) executeAction(ctx context.Context, assignedAction *client.Action) error {
	switch assignedAction.ActionType {
	case client.ActionTypeStartStepRun:
		return w.startStepRun(ctx, assignedAction)
	case client.ActionTypeCancelStepRun:
		return w.cancelStepRun(ctx, assignedAction)
	case client.ActionTypeStartGetGroupKey:
		return w.startGetGroupKey(ctx, assignedAction)
	default:
		return fmt.Errorf("unknown action type: %s", assignedAction.ActionType)
	}
}

func (w *Worker) startStepRun(ctx context.Context, assignedAction *client.Action) error {
	// send a message that the step run started
	_, err := w.client.Dispatcher().SendStepActionEvent(
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

	runContext, cancel := context.WithCancel(context.Background())

	w.cancelMap.Store(assignedAction.StepRunId, cancel)
	defer w.cancelMap.Delete(assignedAction.StepRunId)

	hCtx, err := newHatchetContext(runContext, assignedAction, w.client, w.l, w)

	if err != nil {
		return fmt.Errorf("could not create hatchet context: %w", err)
	}

	// get the action's service
	svcAny, ok := w.services.Load(action.Service())

	if !ok {
		return fmt.Errorf("could not load service %s", action.Service())
	}

	svc := svcAny.(*Service)

	// wrap the run with middleware. start by wrapping the global worker middleware, then
	// the service-specific middleware
	return w.middlewares.runAll(hCtx, func(ctx HatchetContext) error {
		return svc.mws.runAll(ctx, func(ctx HatchetContext) error {
			defer cancel()

			args := []any{ctx}

			if arg != nil {
				args = append(args, arg)
			}

			runResults := action.Run(args...)

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
				return w.sendFailureEvent(ctx, err)
			}

			// send a message that the step run completed
			finishedEvent, err := w.getActionFinishedEvent(assignedAction, result)

			if err != nil {
				return fmt.Errorf("could not create finished event: %w", err)
			}

			_, err = w.client.Dispatcher().SendStepActionEvent(
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

func (w *Worker) startGetGroupKey(ctx context.Context, assignedAction *client.Action) error {
	// send a message that the step run started
	_, err := w.client.Dispatcher().SendGroupKeyActionEvent(
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

	// action should be concurrency action
	if action.ConcurrencyFn() == nil {
		return fmt.Errorf("action %s is not a concurrency action", action.Name())
	}

	runContext, cancel := context.WithCancel(context.Background())

	w.cancelConcurrencyMap.Store(assignedAction.WorkflowRunId, cancel)

	hCtx, err := newHatchetContext(runContext, assignedAction, w.client, w.l, w)

	if err != nil {
		return fmt.Errorf("could not create hatchet context: %w", err)
	}

	concurrencyKey, err := action.ConcurrencyFn()(hCtx)

	if err != nil {
		failureEvent := w.getActionEvent(assignedAction, client.ActionEventTypeFailed)

		w.alerter.SendAlert(context.Background(), err, map[string]interface{}{
			"actionId":      assignedAction.ActionId,
			"workerId":      assignedAction.WorkerId,
			"workflowRunId": assignedAction.WorkflowRunId,
			"jobName":       assignedAction.JobName,
			"actionType":    assignedAction.ActionType,
		})

		failureEvent.EventPayload = err.Error()

		_, err := w.client.Dispatcher().SendGroupKeyActionEvent(
			ctx,
			failureEvent,
		)

		if err != nil {
			return fmt.Errorf("could not send action event: %w", err)
		}

		return err
	}

	// send a message that the step run completed
	finishedEvent, err := w.getGroupKeyActionFinishedEvent(assignedAction, concurrencyKey)

	if err != nil {
		return fmt.Errorf("could not create finished event: %w", err)
	}

	_, err = w.client.Dispatcher().SendGroupKeyActionEvent(
		ctx,
		finishedEvent,
	)

	if err != nil {
		return fmt.Errorf("could not send action event: %w", err)
	}

	return nil
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

	event.EventPayload = output

	return event, nil
}

func (w *Worker) getGroupKeyActionFinishedEvent(action *client.Action, output string) (*client.ActionEvent, error) {
	event := w.getActionEvent(action, client.ActionEventTypeCompleted)

	event.EventPayload = output

	return event, nil
}

func (w *Worker) sendFailureEvent(ctx HatchetContext, err error) error {
	assignedAction := ctx.action()

	failureEvent := w.getActionEvent(assignedAction, client.ActionEventTypeFailed)

	w.alerter.SendAlert(context.Background(), err, map[string]interface{}{
		"actionId":      assignedAction.ActionId,
		"workerId":      assignedAction.WorkerId,
		"workflowRunId": assignedAction.WorkflowRunId,
		"stepRunId":     assignedAction.StepRunId,
		"jobName":       assignedAction.JobName,
		"actionType":    assignedAction.ActionType,
	})

	failureEvent.EventPayload = err.Error()

	innerCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = w.client.Dispatcher().SendStepActionEvent(
		innerCtx,
		failureEvent,
	)

	if err != nil {
		return fmt.Errorf("could not send action event: %w", err)
	}

	return err
}

func getHostName() string {
	hostName, err := os.Hostname()
	if err != nil {
		hostName = "Unknown"
	}
	return hostName
}
