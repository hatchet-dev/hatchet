package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/v1/task"
)

type HatchetWorkerContext interface {
	context.Context

	SetContext(ctx context.Context)

	GetContext() context.Context

	ID() string

	GetLabels() map[string]interface{}

	UpsertLabels(labels map[string]interface{}) error
}

type HatchetContext interface {
	context.Context

	SetContext(ctx context.Context)

	GetContext() context.Context

	Worker() HatchetWorkerContext

	StepOutput(step string, target interface{}) error

	StepRunErrors() map[string]string

	TriggeredByEvent() bool

	WorkflowInput(target interface{}) error

	UserData(target interface{}) error

	AdditionalMetadata() map[string]string

	StepName() string

	StepRunId() string

	StepId() string

	WorkflowRunId() string

	Log(message string)

	StreamEvent(message []byte)

	SpawnWorkflow(workflowName string, input any, opts *SpawnWorkflowOpts) (*client.Workflow, error)

	SpawnWorkflows(childWorkflows []*SpawnWorkflowsOpts) ([]*client.Workflow, error)

	ReleaseSlot() error

	RefreshTimeout(incrementTimeoutBy string) error

	RetryCount() int

	ParentOutput(parent *task.TaskDeclaration[any], target interface{}) error

	client() client.Client

	action() *client.Action

	index() int
	inc()
}

type TriggeredBy string

const (
	TriggeredByEvent    TriggeredBy = "event"
	TriggeredByCron     TriggeredBy = "cron"
	TriggeredBySchedule TriggeredBy = "schedule"
)

type JobRunLookupData struct {
	Input       map[string]interface{} `json:"input"`
	TriggeredBy TriggeredBy            `json:"triggered_by"`
	Steps       map[string]StepData    `json:"steps,omitempty"`
}

type StepRunData struct {
	Input              map[string]interface{} `json:"input"`
	TriggeredBy        TriggeredBy            `json:"triggered_by"`
	Parents            map[string]StepData    `json:"parents"`
	AdditionalMetadata map[string]string      `json:"additional_metadata"`
	UserData           map[string]interface{} `json:"user_data"`
	StepRunErrors      map[string]string      `json:"step_run_errors,omitempty"`
}

type StepData map[string]interface{}

type hatchetContext struct {
	context.Context

	w *hatchetWorkerContext

	a        *client.Action
	stepData *StepRunData
	c        client.Client
	l        *zerolog.Logger

	i          int
	indexMu    sync.Mutex
	listener   *client.WorkflowRunsListener
	listenerMu sync.Mutex
}

type hatchetWorkerContext struct {
	context.Context
	id     *string
	worker *Worker
}

func newHatchetContext(
	ctx context.Context,
	action *client.Action,
	client client.Client,
	l *zerolog.Logger,
	w *Worker,
) (HatchetContext, error) {
	c := &hatchetContext{
		Context: ctx,
		a:       action,
		c:       client,
		l:       l,
		w: &hatchetWorkerContext{
			Context: ctx,
			id:      w.id,
			worker:  w,
		},
	}

	if action.GetGroupKeyRunId != "" {
		err := c.populateStepDataForGroupKeyRun()

		if err != nil {
			return nil, err
		}
	} else {
		err := c.populateStepData()

		if err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (h *hatchetContext) client() client.Client {
	return h.c
}

func (h *hatchetContext) action() *client.Action {
	return h.a
}

func (h *hatchetContext) Worker() HatchetWorkerContext {
	return h.w
}

func (h *hatchetContext) SetContext(ctx context.Context) {
	h.Context = ctx
}

func (h *hatchetContext) GetContext() context.Context {
	return h.Context
}

func (h *hatchetContext) StepOutput(step string, target interface{}) error {
	if val, ok := h.stepData.Parents[step]; ok {
		return toTarget(val, target)
	}

	return fmt.Errorf("step %s not found in action payload", step)
}

func (h *hatchetContext) ParentOutput(parent *task.TaskDeclaration[any], target interface{}) error {
	stepName := parent.Name

	if val, ok := h.stepData.Parents[stepName]; ok {
		return toTarget(val, target)
	}

	return fmt.Errorf("parent %s not found in action payload", stepName)
}

func (h *hatchetContext) TriggeredByEvent() bool {
	return h.stepData.TriggeredBy == TriggeredByEvent
}

func (h *hatchetContext) WorkflowInput(target interface{}) error {
	return toTarget(h.stepData.Input, target)
}

func (h *hatchetContext) StepRunErrors() map[string]string {
	errors := h.stepData.StepRunErrors

	if len(errors) == 0 {
		h.l.Error().Msg("No step run errors found. `ctx.StepRunErrors` is intended to be run in an on-failure step, and will only work on engine versions more recent than v0.53.10")
	}

	return errors
}

func (h *hatchetContext) UserData(target interface{}) error {
	return toTarget(h.stepData.UserData, target)
}

func (h *hatchetContext) AdditionalMetadata() map[string]string {
	return h.stepData.AdditionalMetadata
}

func (h *hatchetContext) StepName() string {
	return h.a.StepName
}

func (h *hatchetContext) StepRunId() string {
	return h.a.StepRunId
}

func (h *hatchetContext) StepId() string {
	return h.a.StepId
}

func (h *hatchetContext) WorkflowRunId() string {
	return h.a.WorkflowRunId
}

func (h *hatchetContext) Log(message string) {
	err := h.c.Event().PutLog(h, h.a.StepRunId, message)

	if err != nil {
		h.l.Err(err).Msg("could not put log")
	}
}

func (h *hatchetContext) ReleaseSlot() error {
	err := h.c.Dispatcher().ReleaseSlot(h, h.a.StepRunId)

	if err != nil {
		return fmt.Errorf("failed to release slot: %w", err)
	}

	return nil
}

func (h *hatchetContext) RefreshTimeout(incrementTimeoutBy string) error {
	err := h.c.Dispatcher().RefreshTimeout(h, h.a.StepRunId, incrementTimeoutBy)

	if err != nil {
		return fmt.Errorf("failed to refresh timeout: %w", err)
	}

	return nil
}

func (h *hatchetContext) StreamEvent(message []byte) {
	err := h.c.Event().PutStreamEvent(h, h.a.StepRunId, message)

	if err != nil {
		h.l.Err(err).Msg("could not put stream event")
	}
}

func (h *hatchetContext) RetryCount() int {
	return int(h.a.RetryCount)
}

func (h *hatchetContext) index() int {
	return h.i
}

func (h *hatchetContext) inc() {
	h.indexMu.Lock()
	h.i++
	h.indexMu.Unlock()
}

type SpawnWorkflowOpts struct {
	Key                *string
	Sticky             *bool
	AdditionalMetadata *map[string]string
}

func (h *hatchetContext) saveOrLoadListener() (*client.WorkflowRunsListener, error) {
	h.listenerMu.Lock()
	defer h.listenerMu.Unlock()

	if h.listener != nil {
		return h.listener, nil
	}

	listener, err := h.client().Subscribe().SubscribeToWorkflowRunEvents(h)

	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to workflow run events: %w", err)
	}

	h.listener = listener

	return listener, nil
}

func (h *hatchetContext) SpawnWorkflow(workflowName string, input any, opts *SpawnWorkflowOpts) (*client.Workflow, error) {
	if opts == nil {
		opts = &SpawnWorkflowOpts{}
	}

	var desiredWorker *string

	if opts.Sticky != nil {
		if _, exists := h.w.worker.registered_workflows[workflowName]; !exists {
			return nil, fmt.Errorf("cannot run with sticky: workflow %s is not registered on this worker", workflowName)
		}

		desiredWorker = h.w.id
	}

	listener, err := h.saveOrLoadListener()

	if err != nil {
		return nil, err
	}

	if ns := h.client().Namespace(); ns != "" && !strings.HasPrefix(workflowName, ns) {
		workflowName = fmt.Sprintf("%s%s", ns, workflowName)
	}

	workflowRunId, err := h.client().Admin().RunChildWorkflow(
		workflowName,
		input,
		&client.ChildWorkflowOpts{
			ParentId:           h.WorkflowRunId(),
			ParentStepRunId:    h.StepRunId(),
			ChildIndex:         h.index(),
			ChildKey:           opts.Key,
			DesiredWorkerId:    desiredWorker,
			AdditionalMetadata: opts.AdditionalMetadata,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to spawn workflow: %w", err)
	}

	// increment the index
	h.inc()

	return client.NewWorkflow(workflowRunId, listener), nil
}

type SpawnWorkflowsOpts struct {
	WorkflowName       string
	Input              any
	Key                *string
	Sticky             *bool
	AdditionalMetadata *map[string]string
}

func (h *hatchetContext) SpawnWorkflows(childWorkflows []*SpawnWorkflowsOpts) ([]*client.Workflow, error) {

	triggerWorkflows := make([]*client.RunChildWorkflowsOpts, len(childWorkflows))
	listener, err := h.saveOrLoadListener()

	for i, c := range childWorkflows {

		var desiredWorker *string

		if c.Sticky != nil {
			if _, exists := h.w.worker.registered_workflows[c.WorkflowName]; !exists {
				return nil, fmt.Errorf("cannot run with sticky: workflow %s is not registered on this worker", c.WorkflowName)
			}

			desiredWorker = h.w.id
		}

		if err != nil {
			return nil, err
		}
		workflowName := c.WorkflowName

		if ns := h.client().Namespace(); ns != "" && !strings.HasPrefix(c.WorkflowName, ns) {
			workflowName = fmt.Sprintf("%s%s", ns, workflowName)
		}

		// increment the index
		h.inc()

		triggerWorkflows[i] = &client.RunChildWorkflowsOpts{
			WorkflowName: workflowName,
			Input:        c.Input,
			Opts: &client.ChildWorkflowOpts{
				ParentId:           h.WorkflowRunId(),
				ParentStepRunId:    h.StepRunId(),
				ChildIndex:         h.index(),
				ChildKey:           c.Key,
				DesiredWorkerId:    desiredWorker,
				AdditionalMetadata: c.AdditionalMetadata,
			},
		}
	}

	workflowRunIds, err := h.client().Admin().RunChildWorkflows(
		triggerWorkflows,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to spawn workflow: %w", err)
	}

	createdWorkflows := make([]*client.Workflow, len(workflowRunIds))

	for i, workflowRunId := range workflowRunIds {
		createdWorkflows[i] = client.NewWorkflow(workflowRunId, listener)
	}

	return createdWorkflows, nil
}

func (h *hatchetContext) ChildIndex() *int32 {
	return h.a.ChildIndex
}

func (h *hatchetContext) ChildKey() *string {
	return h.a.ChildKey
}

func (h *hatchetContext) ParentWorkflowRunId() *string {
	return h.a.ParentWorkflowRunId
}

func (h *hatchetContext) populateStepDataForGroupKeyRun() error {
	if h.stepData != nil {
		return nil
	}

	inputData := map[string]interface{}{}

	err := json.Unmarshal(h.a.ActionPayload, &inputData)

	if err != nil {
		return err
	}

	h.stepData = &StepRunData{
		Input: inputData,
	}

	return nil
}

func (h *hatchetContext) populateStepData() error {
	if h.stepData != nil {
		return nil
	}

	h.stepData = &StepRunData{}

	jsonBytes := h.a.ActionPayload

	if len(jsonBytes) == 0 {
		jsonBytes = []byte("{}")
	}

	err := json.Unmarshal(jsonBytes, h.stepData)

	if err != nil {
		return err
	}

	h.stepData.AdditionalMetadata = h.a.AdditionalMetadata

	return nil
}

func toTarget(data interface{}, target interface{}) error {
	dataBytes, err := json.Marshal(data)

	if err != nil {
		return err
	}

	err = json.Unmarshal(dataBytes, target)

	if err != nil {
		return err
	}

	return nil
}

func (wc *hatchetWorkerContext) SetContext(ctx context.Context) {
	wc.Context = ctx
}

func (wc *hatchetWorkerContext) GetContext() context.Context {
	return wc.Context
}

func (wc *hatchetWorkerContext) ID() string {
	if wc.id == nil {
		return ""
	}

	return *wc.id
}

func (wc *hatchetWorkerContext) GetLabels() map[string]interface{} {
	return wc.worker.labels
}

func (wc *hatchetWorkerContext) UpsertLabels(labels map[string]interface{}) error {

	if wc.id == nil {
		return fmt.Errorf("worker id is nil, cannot upsert labels (are on web worker?)")
	}

	err := wc.worker.client.Dispatcher().UpsertWorkerLabels(wc.Context, *wc.id, labels)

	if err != nil {
		return fmt.Errorf("failed to upsert labels: %w", err)
	}

	wc.worker.labels = labels
	return nil
}

// DurableHatchetContext extends HatchetContext with methods for durable tasks.
type DurableHatchetContext interface {
	HatchetContext

	// SleepFor pauses execution for the specified duration and returns after that time has elapsed.
	// Duration is "global" meaning it will wait in real time regardless of transient failures
	// like worker restarts.
	// Example: "10s" for 10 seconds, "1m" for 1 minute, etc.
	SleepFor(duration time.Duration) (interface{}, error)

	// WaitFor pauses execution until the specified conditions are met.
	// Conditions are "global" meaning they will wait in real time regardless of transient failures
	// like worker restarts.
	WaitFor(conditions map[string]interface{}) (interface{}, error)
}

// durableHatchetContext implements the DurableHatchetContext interface.
type durableHatchetContext struct {
	*hatchetContext
	waitKeyCounter int
}

// SleepFor implements the DurableHatchetContext.SleepFor method.
func (d *durableHatchetContext) SleepFor(duration time.Duration) (interface{}, error) {
	// Implement SleepFor functionality
	// Call appropriate client methods to register a durable event
	return d.WaitFor(map[string]interface{}{
		"sleepFor": duration,
	})
}

// WaitFor implements the DurableHatchetContext.WaitFor method.
func (d *durableHatchetContext) WaitFor(conditions map[string]interface{}) (interface{}, error) {
	// Increment wait key to ensure unique keys for multiple wait operations
	d.waitKeyCounter++
	// key := fmt.Sprintf("waitFor-%d", d.waitKeyCounter)

	// Convert conditions to appropriate format and register with the durable event system
	// This is a simplified implementation - in a real system, you'd need to convert
	// the conditions to the format expected by your durable event subsystem

	// Call client methods to register and wait for the event
	// For now, we'll return a placeholder implementation
	return nil, fmt.Errorf("WaitFor not fully implemented yet")
}

// NewDurableHatchetContext creates a DurableHatchetContext from a HatchetContext.
func NewDurableHatchetContext(ctx HatchetContext) DurableHatchetContext {
	// Try to cast directly if it's already a DurableHatchetContext
	if durableCtx, ok := ctx.(DurableHatchetContext); ok {
		return durableCtx
	}

	// If it's a hatchetContext, wrap it in a durableHatchetContext
	if hCtx, ok := ctx.(*hatchetContext); ok {
		return &durableHatchetContext{
			hatchetContext: hCtx,
			waitKeyCounter: 0,
		}
	}

	// Create a new wrapper if it's some other implementation
	return &durableHatchetContext{
		hatchetContext: &hatchetContext{
			Context: ctx,
			a:       ctx.action(),
			c:       ctx.client(),
			w:       ctx.Worker().(*hatchetWorkerContext),
		},
		waitKeyCounter: 0,
	}
}
