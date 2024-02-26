package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hatchet-dev/hatchet/pkg/client"
)

type HatchetContext interface {
	context.Context

	SetContext(ctx context.Context)

	GetContext() context.Context

	StepOutput(step string, target interface{}) error

	TriggeredByEvent() bool

	WorkflowInput(target interface{}) error
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
	action   *client.Action
	stepData *StepRunData
}

func newHatchetContext(ctx context.Context, action *client.Action) (HatchetContext, error) {
	c := &hatchetContext{
		Context: ctx,
		action:  action,
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

func (h *hatchetContext) populateStepDataForGroupKeyRun() error {
	if h.stepData != nil {
		return nil
	}

	inputData := map[string]interface{}{}

	err := json.Unmarshal(h.action.ActionPayload, &inputData)

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

	err := json.Unmarshal(h.action.ActionPayload, h.stepData)

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
