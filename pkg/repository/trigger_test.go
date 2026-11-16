package repository

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func Test_meteredTaskCount(t *testing.T) {
	step := &sqlcv1.ListStepsByWorkflowVersionIdsRow{ActionId: "wf:step"}
	onFailure := &sqlcv1.ListStepsByWorkflowVersionIdsRow{ActionId: "wf:on-failure", JobKind: sqlcv1.JobKindONFAILURE}
	orchestrator := &sqlcv1.ListStepsByWorkflowVersionIdsRow{ActionId: "wf:orchestrator", IsDagOrchestrator: true}
	target := "wf:step"

	tests := []struct {
		name   string
		tuple  triggerTuple
		steps  []*sqlcv1.ListStepsByWorkflowVersionIdsRow
		expect int
	}{
		{"single task", triggerTuple{}, []*sqlcv1.ListStepsByWorkflowVersionIdsRow{step}, 1},
		{"dag without operator", triggerTuple{}, []*sqlcv1.ListStepsByWorkflowVersionIdsRow{step, step, onFailure}, 3},
		{"operator dag start", triggerTuple{}, []*sqlcv1.ListStepsByWorkflowVersionIdsRow{step, step, onFailure, orchestrator}, 3},
		{"operator dag step", triggerTuple{targetActionId: &target}, []*sqlcv1.ListStepsByWorkflowVersionIdsRow{step, step, onFailure, orchestrator}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, meteredTaskCount(tt.tuple, tt.steps))
		})
	}
}

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
	assert.Equal(t, sdkSpanID, gotSpanID,
		"span_id from SDK is preserved (not overwritten)")
	assert.Nil(t, m["hatchet__traceparent_parent_span_id"],
		"no separate parent span_id key stored")
}
