package v1

import (
	"encoding/json"
	"fmt"

	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

// parses match aggregated data
type MatchData struct {
	action sqlcv1.V1MatchConditionAction

	// maps readable data keys to a list of data values
	dataKeys map[string][]interface{}

	triggerDataKeys map[string][]interface{}
}

func (m *MatchData) Action() sqlcv1.V1MatchConditionAction {
	return m.action
}

func (m *MatchData) DataKeys() []string {
	if len(m.dataKeys) == 0 {
		return []string{}
	}

	keys := make([]string, 0, len(m.dataKeys))

	for k := range m.dataKeys {
		keys = append(keys, k)
	}

	return keys
}

func (m *MatchData) TriggerDataKeys() []string {
	if len(m.triggerDataKeys) == 0 {
		return []string{}
	}

	keys := make([]string, 0, len(m.triggerDataKeys))

	for k := range m.triggerDataKeys {
		keys = append(keys, k)
	}

	return keys
}

func (m *MatchData) TriggerDataValue(key string) map[string]interface{} {
	values := m.triggerDataKeys[key]

	for _, v := range values {
		// convert the values to a byte array, then to a map
		vBytes, err := json.Marshal(v)

		if err != nil {
			continue
		}

		data := map[string]interface{}{}

		err = json.Unmarshal(vBytes, &data)

		if err != nil {
			continue
		}

		return data
	}

	return nil
}

// Helper function for internal events
func (m *MatchData) DataValueAsTaskOutputEvent(key string) *TaskOutputEvent {
	values := m.dataKeys[key]

	for _, v := range values {
		// convert the values to a byte array, then to a TaskOutputEvent
		vBytes, err := json.Marshal(v)

		if err != nil {
			continue
		}

		event := &TaskOutputEvent{}

		err = json.Unmarshal(vBytes, event)

		if err != nil {
			continue
		}

		return event
	}

	return nil
}

func NewMatchData(mcAggregatedData []byte) (*MatchData, error) {
	var triggerDataMap map[string]map[string][]interface{}

	if len(mcAggregatedData) > 0 {
		err := json.Unmarshal(mcAggregatedData, &triggerDataMap)

		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("no match condition aggregated data")
	}

	// Extract CREATE_MATCH data first - this contains existing data that should be merged
	// CREATE_MATCH is used to create additional match conditions dynamically, not to create tasks directly
	existingDataKeys := make(map[string][]interface{})
	if createMatchData, exists := triggerDataMap["CREATE_MATCH"]; exists {
		existingDataKeys = createMatchData
	}

	// Find the action and its associated data
	for k, v := range triggerDataMap {
		var action sqlcv1.V1MatchConditionAction

		switch k {
		case "CREATE":
			action = sqlcv1.V1MatchConditionActionCREATE
		case "QUEUE":
			action = sqlcv1.V1MatchConditionActionQUEUE
		case "CANCEL":
			action = sqlcv1.V1MatchConditionActionCANCEL
		case "SKIP":
			action = sqlcv1.V1MatchConditionActionSKIP
		case "CREATE_MATCH":
			// CREATE_MATCH is not an action that creates tasks, skip it
			continue
		default:
			return nil, fmt.Errorf("invalid match condition action: %s", k)
		}

		// If we have existing data from CREATE_MATCH, use it as dataKeys
		// and the current action's data as triggerDataKeys
		var dataKeys, triggerDataKeys map[string][]interface{}
		if len(existingDataKeys) > 0 {
			dataKeys = existingDataKeys
			triggerDataKeys = v
		} else {
			dataKeys = v
			triggerDataKeys = make(map[string][]interface{})
		}

		return &MatchData{
			action:          action,
			dataKeys:        dataKeys,
			triggerDataKeys: triggerDataKeys,
		}, nil
	}

	return nil, fmt.Errorf("no match condition aggregated data")
}
