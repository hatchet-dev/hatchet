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
		}

		return &MatchData{
			action:   action,
			dataKeys: v,
		}, nil
	}

	return nil, fmt.Errorf("no match condition aggregated data")
}
