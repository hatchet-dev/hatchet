package v1

import (
	"encoding/json"
	"testing"

	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func TestNewMatchData_ValidActions(t *testing.T) {
	tests := []struct {
		name           string
		actionKey      string
		expectedAction sqlcv1.V1MatchConditionAction
	}{
		{
			name:           "CREATE action",
			actionKey:      "CREATE",
			expectedAction: sqlcv1.V1MatchConditionActionCREATE,
		},
		{
			name:           "QUEUE action",
			actionKey:      "QUEUE",
			expectedAction: sqlcv1.V1MatchConditionActionQUEUE,
		},
		{
			name:           "CANCEL action",
			actionKey:      "CANCEL",
			expectedAction: sqlcv1.V1MatchConditionActionCANCEL,
		},
		{
			name:           "SKIP action",
			actionKey:      "SKIP",
			expectedAction: sqlcv1.V1MatchConditionActionSKIP,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := map[string]map[string][]interface{}{
				tt.actionKey: {
					"test_key": []interface{}{"test_value"},
				},
			}

			dataBytes, err := json.Marshal(data)
			if err != nil {
				t.Fatalf("Failed to marshal test data: %v", err)
			}

			matchData, err := NewMatchData(dataBytes)
			if err != nil {
				t.Fatalf("NewMatchData failed: %v", err)
			}

			if matchData.Action() != tt.expectedAction {
				t.Errorf("Expected action %v, got %v", tt.expectedAction, matchData.Action())
			}
		})
	}
}

func TestNewMatchData_InvalidAction(t *testing.T) {
	// Test that invalid action keys return an error
	invalidData := map[string]map[string][]interface{}{
		"INVALID_ACTION": {
			"test_key": []interface{}{"test_value"},
		},
	}

	dataBytes, err := json.Marshal(invalidData)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	_, err = NewMatchData(dataBytes)
	if err == nil {
		t.Fatal("Expected error for invalid action, but got nil")
	}

	expectedError := "invalid match condition action: INVALID_ACTION"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestNewMatchData_CreateMatchHandling(t *testing.T) {
	// Test that CREATE_MATCH is handled properly and doesn't cause an error
	data := map[string]map[string][]interface{}{
		"CREATE_MATCH": {
			"existing_key": []interface{}{"existing_value"},
		},
		"QUEUE": {
			"trigger_key": []interface{}{"trigger_value"},
		},
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Failed to marshal test data: %v", err)
	}

	matchData, err := NewMatchData(dataBytes)
	if err != nil {
		t.Fatalf("NewMatchData failed: %v", err)
	}

	if matchData.Action() != sqlcv1.V1MatchConditionActionQUEUE {
		t.Errorf("Expected action QUEUE, got %v", matchData.Action())
	}

	// Verify that CREATE_MATCH data was merged into dataKeys
	dataKeys := matchData.DataKeys()
	found := false
	for _, key := range dataKeys {
		if key == "existing_key" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected CREATE_MATCH data to be merged into dataKeys")
	}
}

func TestNewMatchData_EmptyData(t *testing.T) {
	// Test that empty data returns an error
	_, err := NewMatchData([]byte{})
	if err == nil {
		t.Fatal("Expected error for empty data, but got nil")
	}

	expectedError := "no match condition aggregated data"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}
