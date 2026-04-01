package repository

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_cleanAdditionalMetadataTableTest(t *testing.T) {

	tests := []struct {
		name               string
		additionalMetadata []byte
		expected           map[string]interface{}
	}{

		{
			name:               "empty",
			additionalMetadata: []byte(""),
			expected:           map[string]interface{}{},
		},
		{
			name:               "null",
			additionalMetadata: []byte("null"),
			expected:           map[string]interface{}{},
		},
		{
			name:               "valid",
			additionalMetadata: []byte(`{"key":"value"}`),
			expected:           map[string]interface{}{"key": "value"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := cleanAdditionalMetadata(test.additionalMetadata)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func Test_ensureTraceparent_creates_when_absent(t *testing.T) {
	wfRunID := uuid.New()
	meta := []byte(`{"key":"value"}`)

	result := ensureTraceparent(meta, wfRunID)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(result, &m))

	tp, ok := m["traceparent"].(string)
	require.True(t, ok, "traceparent must be set")

	traceID, spanID, ok := parseW3CTraceparent(tp)
	require.True(t, ok)

	expectedTraceID := hex.EncodeToString(DeriveWorkflowRunTraceID(wfRunID))
	expectedSpanID := hex.EncodeToString(DeriveWorkflowRunSpanID(wfRunID))

	assert.Equal(t, expectedTraceID, traceID)
	assert.Equal(t, expectedSpanID, spanID)
	assert.Nil(t, m["hatchet__traceparent_parent_span_id"],
		"no parent span_id stored when SDK didn't inject one")
	assert.Equal(t, "value", m["key"], "existing metadata preserved")
}

func Test_ensureTraceparent_inherits_when_present(t *testing.T) {
	wfRunID := uuid.New()
	sdkTraceID := "4bf92f3577b34da6a3ce929d0e0e4736"
	sdkSpanID := "00f067aa0ba902b7"

	meta, _ := json.Marshal(map[string]string{
		"traceparent": "00-" + sdkTraceID + "-" + sdkSpanID + "-01",
	})

	result := ensureTraceparent(meta, wfRunID)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(result, &m))

	tp, ok := m["traceparent"].(string)
	require.True(t, ok)

	gotTraceID, gotSpanID, ok := parseW3CTraceparent(tp)
	require.True(t, ok)

	assert.Equal(t, sdkTraceID, gotTraceID,
		"trace_id from SDK is preserved")
	assert.Equal(t, hex.EncodeToString(DeriveWorkflowRunSpanID(wfRunID)), gotSpanID,
		"span_id is engine-derived, not SDK's original")
	assert.Equal(t, sdkSpanID, m["hatchet__traceparent_parent_span_id"],
		"SDK span_id stored for parent linkage")
}
