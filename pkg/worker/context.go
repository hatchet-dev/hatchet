package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/client"
)

type HatchetContext interface {
	context.Context

	SetContext(ctx context.Context)

	GetContext() context.Context

	StepOutput(step string, target interface{}) error

	TriggeredByEvent() bool

	WorkflowInput(target interface{}) error

	StepName() string

	StepRunId() string

	WorkflowRunId() string

	Log(message string)

	StreamEvent(message []byte)

	SpawnWorkflow(workflowName string, input any, opts *SpawnWorkflowOpts) (*ChildWorkflow, error)

	client() client.Client

	action() *client.Action

	index() int
	inc()
}

// TODO: move this into proto definitions
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
	Input       map[string]interface{} `json:"input"`
	TriggeredBy TriggeredBy            `json:"triggered_by"`
	Parents     map[string]StepData    `json:"parents"`
}

type StepData map[string]interface{}

type hatchetContext struct {
	context.Context
	a        *client.Action
	stepData *StepRunData
	c        client.Client
	l        *zerolog.Logger

	i          int
	indexMu    sync.Mutex
	listener   *client.WorkflowRunsListener
	listenerMu sync.Mutex
}

func newHatchetContext(
	ctx context.Context,
	action *client.Action,
	client client.Client,
	l *zerolog.Logger,
) (HatchetContext, error) {
	c := &hatchetContext{
		Context: ctx,
		a:       action,
		c:       client,
		l:       l,
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

func (h *hatchetContext) TriggeredByEvent() bool {
	return h.stepData.TriggeredBy == TriggeredByEvent
}

func (h *hatchetContext) WorkflowInput(target interface{}) error {
	return toTarget(h.stepData.Input, target)
}

func (h *hatchetContext) StepName() string {
	return h.a.StepName
}

func (h *hatchetContext) StepRunId() string {
	return h.a.StepRunId
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

func (h *hatchetContext) StreamEvent(message []byte) {
	err := h.c.Event().PutStreamEvent(h, h.a.StepRunId, message)

	if err != nil {
		h.l.Err(err).Msg("could not put stream event")
	}
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
	Key *string
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

func (h *hatchetContext) SpawnWorkflow(workflowName string, input any, opts *SpawnWorkflowOpts) (*ChildWorkflow, error) {
	listener, err := h.saveOrLoadListener()

	if err != nil {
		return nil, err
	}

	workflowRunId, err := h.client().Admin().RunChildWorkflow(
		workflowName,
		input,
		&client.ChildWorkflowOpts{
			ParentId:        h.WorkflowRunId(),
			ParentStepRunId: h.StepRunId(),
			ChildIndex:      h.index(),
			ChildKey:        opts.Key,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to spawn workflow: %w", err)
	}

	// increment the index
	h.inc()

	return &ChildWorkflow{
		workflowRunId: workflowRunId,
		l:             h.l,
		listener:      listener,
	}, nil
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
