package v1

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

type TaskInput struct {
	Input map[string]interface{} `json:"input"`

	TriggerData map[string][]map[string]interface{} `json:"trigger_data"`
}

type CompletedData struct {
	StepReadableId string `json:"step_readable_id"`

	Output []byte `json:"output"`
}

type FailedData struct {
	StepReadableId string `json:"step_readable_id"`

	Error string `json:"error"`
}

func (s *sharedRepository) parseTriggerData(triggerData []byte) (*sqlcv1.V1MatchConditionAction, map[string][]map[string]interface{}, error) {
	var triggerDataMap map[string]map[string][]map[string]interface{}

	if len(triggerData) > 0 {
		err := json.Unmarshal(triggerData, &triggerDataMap)

		if err != nil {
			s.l.Error().Err(err).Msg("failed to unmarshal trigger data")
		}
	}

	for k, v := range triggerDataMap {
		switch k {
		case "CREATE":
			create := sqlcv1.V1MatchConditionActionCREATE
			return &create, v, nil
		case "CANCEL":
			cancel := sqlcv1.V1MatchConditionActionCANCEL
			return &cancel, v, nil
		case "SKIP":
			skip := sqlcv1.V1MatchConditionActionSKIP
			return &skip, v, nil
		default:
			s.l.Error().Str("action", k).Msg("unknown action")
		}
	}

	return nil, nil, fmt.Errorf("unknown action")
}

func (s *sharedRepository) newTaskInput(inputBytes []byte, triggerDataMap map[string][]map[string]interface{}) *TaskInput {
	var input map[string]interface{}

	if len(inputBytes) > 0 {
		err := json.Unmarshal(inputBytes, &input)

		if err != nil {
			s.l.Error().Err(err).Msg("failed to unmarshal input bytes")
		}
	}

	return &TaskInput{
		Input:       input,
		TriggerData: triggerDataMap,
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
	stepRunErrors := make(map[string]string)

	for k, v := range t.TriggerData {
		if strings.HasPrefix(k, "task.completed") {
			for _, data := range v {
				dataBytes, err := json.Marshal(data)

				if err != nil {
					s.l.Error().Err(err).Msg("failed to marshal trigger data")
					continue
				}

				var completedData CompletedData

				err = json.Unmarshal(dataBytes, &completedData)

				if err != nil {
					s.l.Error().Err(err).Msg("failed to unmarshal completed data")
					continue
				}

				outputMap := make(map[string]interface{})

				err = json.Unmarshal(completedData.Output, &outputMap)

				if err != nil {
					s.l.Error().Err(err).Msg("failed to unmarshal output")
				}

				parents[completedData.StepReadableId] = outputMap
			}
		} else if strings.HasPrefix(k, "task.failed") {
			for _, data := range v {
				dataBytes, err := json.Marshal(data)

				if err != nil {
					s.l.Error().Err(err).Msg("failed to marshal trigger data")
					continue
				}

				var failedData FailedData

				err = json.Unmarshal(dataBytes, &failedData)

				if err != nil {
					s.l.Error().Err(err).Msg("failed to unmarshal failed data")
					continue
				}

				stepRunErrors[failedData.StepReadableId] = failedData.Error
			}
		}
	}

	return &V1StepRunData{
		Input:         t.Input,
		TriggeredBy:   "manual",
		Parents:       parents,
		StepRunErrors: stepRunErrors,
	}
}

type V1StepRunData struct {
	Input       map[string]interface{}            `json:"input"`
	TriggeredBy string                            `json:"triggered_by"`
	Parents     map[string]map[string]interface{} `json:"parents"`

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
