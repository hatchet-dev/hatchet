package v1

import (
	"encoding/json"
)

type TaskInput struct {
	Input map[string]interface{} `json:"input"`

	TriggerData *MatchData `json:"trigger_datas"`

	FilterPayload map[string]interface{} `json:"filter_payload"`
}

func (s *sharedRepository) DesiredWorkerId(t *TaskInput) *string {
	if t.TriggerData != nil {
		for _, stepReadableId := range t.TriggerData.DataKeys() {
			data := t.TriggerData.DataValueAsTaskOutputEvent(stepReadableId)

			return data.WorkerId
		}
	}

	return nil
}

func (s *sharedRepository) newTaskInputFromExistingBytes(inputBytes []byte) *TaskInput {
	i := &TaskInput{}

	err := json.Unmarshal(inputBytes, i)

	if err != nil {
		s.l.Error().Err(err).Msg("failed to unmarshal input bytes")
	}

	return i
}

func (s *sharedRepository) newTaskInput(inputBytes []byte, triggerData *MatchData, filterPayload []byte) *TaskInput {
	var input map[string]interface{}

	if len(inputBytes) > 0 {
		err := json.Unmarshal(inputBytes, &input)

		if err != nil {
			s.l.Error().Err(err).Msg("failed to unmarshal input bytes")
		}
	}

	var filterPayloadMap map[string]interface{}
	if len(filterPayload) > 0 {
		err := json.Unmarshal(filterPayload, &filterPayloadMap)
		if err != nil {
			s.l.Error().Err(err).Msg("failed to unmarshal event filter payload bytes")
		}
	}

	return &TaskInput{
		Input:         input,
		TriggerData:   triggerData,
		FilterPayload: filterPayloadMap,
	}
}

func (t *TaskInput) Bytes() []byte {
	if t == nil {
		return nil
	}

	out, err := json.Marshal(t)

	if err != nil {
		return nil
	}

	return out
}

func (s *sharedRepository) ToV1StepRunData(t *TaskInput) *V1StepRunData {
	if t == nil {
		return nil
	}

	parents := make(map[string]map[string]interface{})
	triggers := make(map[string]map[string]interface{})
	stepRunErrors := make(map[string]string)

	if t.TriggerData != nil {
		for _, stepReadableId := range t.TriggerData.DataKeys() {
			data := t.TriggerData.DataValueAsTaskOutputEvent(stepReadableId)

			switch {
			case data.IsCompleted():
				dataMap := make(map[string]interface{})

				err := json.Unmarshal(data.Output, &dataMap)

				if err != nil {
					s.l.Warn().Err(err).Msg("failed to unmarshal output")
				}

				parents[stepReadableId] = dataMap
			case data.IsFailed():
				stepRunErrors[stepReadableId] = data.ErrorMessage
			}
		}

		for _, key := range t.TriggerData.TriggerDataKeys() {
			dataMap := t.TriggerData.TriggerDataValue(key)

			triggers[key] = dataMap
		}
	}

	triggers["filter_payload"] = t.FilterPayload

	return &V1StepRunData{
		Input:         t.Input,
		TriggeredBy:   "manual",
		Parents:       parents,
		Triggers:      triggers,
		StepRunErrors: stepRunErrors,
	}
}

type V1StepRunData struct {
	Input       map[string]interface{}            `json:"input"`
	TriggeredBy string                            `json:"triggered_by"`
	Parents     map[string]map[string]interface{} `json:"parents"`

	Triggers map[string]map[string]interface{} `json:"triggers"`

	// custom-set user data for the step
	UserData map[string]interface{} `json:"user_data"`

	// overrides set from the playground
	Overrides map[string]interface{} `json:"overrides"`

	// errors in upstream steps (only used in on-failure step)
	StepRunErrors map[string]string `json:"step_run_errors,omitempty"`
}

func (v1 *V1StepRunData) Bytes() []byte {
	out, err := json.Marshal(v1)

	if err != nil {
		return []byte("{}")
	}

	return out
}
