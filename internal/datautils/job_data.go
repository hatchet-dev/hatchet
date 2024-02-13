package datautils

import (
	"fmt"

	"github.com/steebchen/prisma-client-go/runtime/types"
)

type TriggeredBy string

const (
	TriggeredByEvent    TriggeredBy = "event"
	TriggeredByCron     TriggeredBy = "cron"
	TriggeredBySchedule TriggeredBy = "schedule"
	TriggeredByManual   TriggeredBy = "manual"
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

	// custom-set user data for the step
	UserData map[string]interface{} `json:"user_data"`
}

type StepData map[string]interface{}

func NewJobRunLookupDataFromInputBytes(input []byte, triggeredBy TriggeredBy) (JobRunLookupData, error) {
	inputMap, err := jsonBytesToMap(input)

	if err != nil {
		return JobRunLookupData{}, fmt.Errorf("failed to convert input to map: %w", err)
	}

	return NewJobRunLookupData(inputMap, triggeredBy), nil
}

func NewJobRunLookupData(input map[string]interface{}, triggeredBy TriggeredBy) JobRunLookupData {
	return JobRunLookupData{
		Input:       input,
		TriggeredBy: triggeredBy,
		Steps:       map[string]StepData{},
	}
}

func GetJobRunLookupData(data *types.JSON) (JobRunLookupData, error) {
	if data == nil {
		return JobRunLookupData{}, nil
	}

	currData := JobRunLookupData{}
	err := FromJSONType(data, &currData)

	if err != nil {
		return JobRunLookupData{}, fmt.Errorf("failed to convert data to map: %w", err)
	}

	return currData, nil
}

// func AddStepOutput(data *types.JSON, stepReadableId string, stepOutput []byte) ([]byte, error) {
// 	if data == nil {
// 		data = &types.JSON{}
// 	}

// 	unquoted, err := strconv.Unquote(string(stepOutput))

// 	if err == nil {
// 		stepOutput = []byte(unquoted)
// 	}

// 	outputMap, err := jsonBytesToMap(stepOutput)

// 	if err != nil {
// 		return nil, fmt.Errorf("failed to convert step output to map: %w", err)
// 	}

// 	currData := JobRunLookupData{}
// 	err = FromJSONType(data, &currData)

// 	if err != nil {
// 		return nil, fmt.Errorf("failed to convert data to map: %w", err)
// 	}

// 	if currData.Steps == nil {
// 		currData.Steps = map[string]StepData{}
// 	}

// 	currData.Steps[stepReadableId] = outputMap

// 	jsonBytes, err := json.Marshal(currData)

// 	if err != nil {
// 		return nil, fmt.Errorf("failed to marshal data: %w", err)
// 	}

// 	return jsonBytes, nil
// }
