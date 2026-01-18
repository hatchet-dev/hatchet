package msgqueue

import (
	"strings"
	"testing"
)

// TestJSONConvertPayload is used for testing JSONConvert
type TestJSONConvertPayload struct {
	TaskId     int64  `json:"task_id"`
	ExternalId string `json:"external_id"`
}

// TestJSONConvert_BatchWithInvalidPayload_ReturnsError is a regression test for BH-CO-005.
// Previously, JSONConvert would silently return nil when any payload failed to unmarshal,
// causing the entire batch to be dropped without any error indication.
// This test verifies that an error is now returned when any payload is invalid.
func TestJSONConvert_BatchWithInvalidPayload_ReturnsError(t *testing.T) {
	tests := []struct {
		name           string
		payloads       [][]byte
		expectError    bool
		expectedCount  int // only checked if expectError is false
	}{
		{
			name: "all valid payloads should succeed",
			payloads: [][]byte{
				[]byte(`{"task_id": 1, "external_id": "abc"}`),
				[]byte(`{"task_id": 2, "external_id": "def"}`),
				[]byte(`{"task_id": 3, "external_id": "ghi"}`),
			},
			expectError:   false,
			expectedCount: 3,
		},
		{
			name: "single invalid payload in batch should return error",
			payloads: [][]byte{
				[]byte(`{"task_id": 1, "external_id": "abc"}`),
				[]byte(`{invalid json`), // This one is invalid
				[]byte(`{"task_id": 3, "external_id": "ghi"}`),
			},
			expectError: true,
		},
		{
			name: "truncated JSON payload should return error",
			payloads: [][]byte{
				[]byte(`{"task_id": 1, "external_id": "abc"}`),
				[]byte(`{"task_id": 2, "external_id":`), // truncated
			},
			expectError: true,
		},
		{
			name: "empty payload in batch should return error",
			payloads: [][]byte{
				[]byte(`{"task_id": 1, "external_id": "abc"}`),
				[]byte(``), // empty
			},
			expectError: true,
		},
		{
			name: "first payload invalid should return error",
			payloads: [][]byte{
				[]byte(`not json at all`),
				[]byte(`{"task_id": 2, "external_id": "def"}`),
			},
			expectError: true,
		},
		{
			name: "last payload invalid should return error",
			payloads: [][]byte{
				[]byte(`{"task_id": 1, "external_id": "abc"}`),
				[]byte(`{"task_id": 2, "external_id": "def"}`),
				[]byte(`}`), // invalid
			},
			expectError: true,
		},
		{
			name:          "empty batch should succeed with empty result",
			payloads:      [][]byte{},
			expectError:   false,
			expectedCount: 0,
		},
		{
			name: "type mismatch should return error",
			payloads: [][]byte{
				[]byte(`{"task_id": "not_a_number", "external_id": "abc"}`), // task_id should be int64
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := JSONConvert[TestJSONConvertPayload](tt.payloads)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil; result had %d items", len(result))
				}
				if result != nil {
					t.Errorf("expected nil result when error occurs, got %d items", len(result))
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result == nil && tt.expectedCount > 0 {
					t.Errorf("expected %d items but got nil result", tt.expectedCount)
				}
				if result != nil && len(result) != tt.expectedCount {
					t.Errorf("expected %d items but got %d", tt.expectedCount, len(result))
				}
			}
		})
	}
}

// TestJSONConvert_ErrorMessageContainsPayloadIndex verifies the error message
// includes which payload failed, making debugging easier.
func TestJSONConvert_ErrorMessageContainsPayloadIndex(t *testing.T) {
	payloads := [][]byte{
		[]byte(`{"task_id": 1, "external_id": "abc"}`),
		[]byte(`{"task_id": 2, "external_id": "def"}`),
		[]byte(`{invalid`), // This is payload 3 of 3
	}

	_, err := JSONConvert[TestJSONConvertPayload](payloads)

	if err == nil {
		t.Fatal("expected error but got nil")
	}

	errMsg := err.Error()
	// Should mention it's payload 3 of 3
	if !strings.Contains(errMsg, "3 of 3") {
		t.Errorf("error message should indicate payload position; got: %s", errMsg)
	}
}

// TestJSONConvert_ValidPayloadsReturnCorrectData verifies that valid payloads
// are correctly unmarshaled.
func TestJSONConvert_ValidPayloadsReturnCorrectData(t *testing.T) {
	payloads := [][]byte{
		[]byte(`{"task_id": 123, "external_id": "abc-123"}`),
		[]byte(`{"task_id": 456, "external_id": "def-456"}`),
	}

	result, err := JSONConvert[TestJSONConvertPayload](payloads)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result))
	}

	if result[0].TaskId != 123 || result[0].ExternalId != "abc-123" {
		t.Errorf("first payload mismatch: got %+v", result[0])
	}

	if result[1].TaskId != 456 || result[1].ExternalId != "def-456" {
		t.Errorf("second payload mismatch: got %+v", result[1])
	}
}
