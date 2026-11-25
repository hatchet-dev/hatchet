//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/enums"
)

func TestReadableStatusesForFilter(t *testing.T) {
	tests := []struct {
		name          string
		statuses      []string
		runningFilter RunningStatusFilter
		expected      []enums.V1ReadableStatusOlap
	}{
		{
			name:          "empty includes evicted",
			runningFilter: RunningStatusFilterAll,
			expected: []enums.V1ReadableStatusOlap{
				enums.V1ReadableStatusOlapQUEUED,
				enums.V1ReadableStatusOlapRUNNING,
				enums.V1ReadableStatusOlapEVICTED,
				enums.V1ReadableStatusOlapFAILED,
				enums.V1ReadableStatusOlapCOMPLETED,
				enums.V1ReadableStatusOlapCANCELLED,
			},
		},
		{
			name:          "running expands to evicted",
			statuses:      []string{"QUEUED", "RUNNING"},
			runningFilter: RunningStatusFilterAll,
			expected: []enums.V1ReadableStatusOlap{
				enums.V1ReadableStatusOlapQUEUED,
				enums.V1ReadableStatusOlapRUNNING,
				enums.V1ReadableStatusOlapEVICTED,
			},
		},
		{
			name:          "explicit evicted is deduplicated",
			statuses:      []string{"RUNNING", "EVICTED"},
			runningFilter: RunningStatusFilterAll,
			expected: []enums.V1ReadableStatusOlap{
				enums.V1ReadableStatusOlapRUNNING,
				enums.V1ReadableStatusOlapEVICTED,
			},
		},
		{
			name:          "on worker excludes evicted",
			statuses:      []string{"RUNNING"},
			runningFilter: RunningStatusFilterOnWorker,
			expected:      []enums.V1ReadableStatusOlap{enums.V1ReadableStatusOlapRUNNING},
		},
		{
			name:          "evicted excludes on worker",
			statuses:      []string{"RUNNING"},
			runningFilter: RunningStatusFilterEvicted,
			expected:      []enums.V1ReadableStatusOlap{enums.V1ReadableStatusOlapEVICTED},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := ReadableStatusesForFilter(tt.statuses, tt.runningFilter)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestReadableStatusesForFilterRejectsUnknown(t *testing.T) {
	for _, statuses := range [][]string{{"running"}, {"FAILED", "BOGUS"}} {
		_, err := ReadableStatusesForFilter(statuses, RunningStatusFilterAll)
		assert.Error(t, err, statuses)
	}
}
