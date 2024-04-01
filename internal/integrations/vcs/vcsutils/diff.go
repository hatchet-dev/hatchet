package vcsutils

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/steebchen/prisma-client-go/runtime/types"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
)

// GetStepRunOverrideDiffs returns a map of the override keys to the override values which have changed
// between the first step run and the latest step run.
func GetStepRunOverrideDiffs(repo repository.StepRunAPIRepository, stepRun *db.StepRunModel) (diffs map[string]string, original map[string]string, err error) {
	// get the first step run archived result, there will be at least one
	var archivedResult inputtable

	archivedResult, err = repo.GetFirstArchivedStepRunResult(stepRun.TenantID, stepRun.ID)

	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			archivedResult = stepRun
		} else {
			return nil, nil, fmt.Errorf("could not get first archived step run result: %w", err)
		}
	}

	firstInput, err := getStepRunInput(archivedResult)

	if err != nil {
		return nil, nil, fmt.Errorf("could not get input from archived result: %w", err)
	}

	secondInput, err := getStepRunInput(stepRun)

	if err != nil {
		return nil, nil, fmt.Errorf("could not get input from step run: %w", err)
	}

	// compare the data
	originalValues := map[string]string{}
	diff := map[string]string{}

	for key, value := range firstInput.Overrides {
		if secondValue, ok := secondInput.Overrides[key]; ok {
			if value != secondValue {
				newValue := formatNewValue(secondValue)

				if newValue != "" {
					diff[key] = newValue
				}
			}
		}

		originalValue := formatNewValue(value)

		if originalValue != "" {
			originalValues[key] = originalValue
		}
	}

	return diff, originalValues, nil
}

func formatNewValue(val interface{}) string {
	switch v := val.(type) {
	case string:
		return strconv.Quote(v)
	case float64, bool:
		return fmt.Sprintf("%v", v)
	case nil:
		return "null"
	default:
		return ""
	}
}

type inputtable interface {
	Input() (value types.JSON, ok bool)
}

func getStepRunInput(in inputtable) (*datautils.StepRunData, error) {
	input, ok := in.Input()

	if !ok {
		return nil, fmt.Errorf("could not get input from inputtable")
	}

	data := &datautils.StepRunData{}

	if err := json.Unmarshal(input, data); err != nil || data == nil {
		return nil, fmt.Errorf("could not unmarshal input: %w", err)
	}

	return data, nil
}
