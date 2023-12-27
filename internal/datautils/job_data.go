package datautils

import (
	"fmt"
	"strconv"

	"github.com/steebchen/prisma-client-go/runtime/types"
)

type JobRunLookupData struct {
	Input map[string]interface{}    `json:"input"`
	Steps map[string]stepLookupData `json:"steps,omitempty"`
}

type stepLookupData map[string]interface{}

func NewJobRunLookupDataFromInputBytes(input []byte) (JobRunLookupData, error) {
	inputMap, err := jsonBytesToMap(input)

	if err != nil {
		return JobRunLookupData{}, fmt.Errorf("failed to convert input to map: %w", err)
	}

	return NewJobRunLookupData(inputMap, input), nil
}

func NewJobRunLookupData(input map[string]interface{}, rawInput []byte) JobRunLookupData {
	input["json"] = string(rawInput)

	return JobRunLookupData{
		Input: input,
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

func AddStepOutput(data *types.JSON, stepReadableId string, stepOutput []byte) (*types.JSON, error) {
	if data == nil {
		data = &types.JSON{}
	}

	unquoted, err := strconv.Unquote(string(stepOutput))

	if err == nil {
		stepOutput = []byte(unquoted)
	}

	outputMap, err := jsonBytesToMap(stepOutput)

	if err != nil {
		return nil, fmt.Errorf("failed to convert step output to map: %w", err)
	}

	// add a "json" accessor to the output
	outputMap["json"] = unquoted

	currData := JobRunLookupData{}
	err = FromJSONType(data, &currData)

	if err != nil {
		return nil, fmt.Errorf("failed to convert data to map: %w", err)
	}

	if currData.Steps == nil {
		currData.Steps = map[string]stepLookupData{}
	}

	currData.Steps[stepReadableId] = outputMap

	return ToJSONType(currData)
}
