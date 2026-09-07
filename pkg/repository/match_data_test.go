//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestMatchDataParentOutputs(t *testing.T) {
	completedOutput, err := json.Marshal(&TaskOutputEvent{
		EventType: sqlcv1.V1TaskEventTypeCOMPLETED,
		Output:    []byte(`{"result": "parent output", "count": 3}`),
	})
	require.NoError(t, err)

	failedOutput, err := json.Marshal(&TaskOutputEvent{
		EventType:    sqlcv1.V1TaskEventTypeFAILED,
		ErrorMessage: "something went wrong",
	})
	require.NoError(t, err)

	aggregatedData, err := json.Marshal(map[string]map[string][]interface{}{
		"QUEUE": {
			"step1": {json.RawMessage(completedOutput)},
			"step2": {json.RawMessage(failedOutput)},
		},
	})
	require.NoError(t, err)

	matchData, err := NewMatchData(aggregatedData)
	require.NoError(t, err)

	outputs := matchData.ParentOutputs()

	// only completed parents should appear, keyed by readable id with their output data
	assert.Equal(t, map[string]map[string]interface{}{
		"step1": {
			"result": "parent output",
			"count":  float64(3),
		},
	}, outputs)
}

func TestMatchDataParentOutputsNil(t *testing.T) {
	var matchData *MatchData

	assert.Equal(t, map[string]map[string]interface{}{}, matchData.ParentOutputs())
}
