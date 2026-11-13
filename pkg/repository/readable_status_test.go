//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestReadableStatusesForFilter(t *testing.T) {
	tests := []struct {
		name          string
		statuses      []string
		runningFilter RunningStatusFilter
		expected      []sqlcv1.V1ReadableStatusOlap
	}{
		{
			name:          "empty includes evicted",
			runningFilter: RunningStatusFilterAll,
			expected: []sqlcv1.V1ReadableStatusOlap{
				sqlcv1.V1ReadableStatusOlapQUEUED,
				sqlcv1.V1ReadableStatusOlapRUNNING,
				sqlcv1.V1ReadableStatusOlapEVICTED,
				sqlcv1.V1ReadableStatusOlapFAILED,
				sqlcv1.V1ReadableStatusOlapCOMPLETED,
				sqlcv1.V1ReadableStatusOlapCANCELLED,
			},
		},
		{
			name:          "running expands to evicted",
			statuses:      []string{"QUEUED", "RUNNING"},
			runningFilter: RunningStatusFilterAll,
			expected: []sqlcv1.V1ReadableStatusOlap{
				sqlcv1.V1ReadableStatusOlapQUEUED,
				sqlcv1.V1ReadableStatusOlapRUNNING,
				sqlcv1.V1ReadableStatusOlapEVICTED,
			},
		},
		{
			name:          "explicit evicted is deduplicated",
			statuses:      []string{"RUNNING", "EVICTED"},
			runningFilter: RunningStatusFilterAll,
			expected: []sqlcv1.V1ReadableStatusOlap{
				sqlcv1.V1ReadableStatusOlapRUNNING,
				sqlcv1.V1ReadableStatusOlapEVICTED,
			},
		},
		{
			name:          "on worker excludes evicted",
			statuses:      []string{"RUNNING"},
			runningFilter: RunningStatusFilterOnWorker,
			expected:      []sqlcv1.V1ReadableStatusOlap{sqlcv1.V1ReadableStatusOlapRUNNING},
		},
		{
			name:          "evicted excludes on worker",
			statuses:      []string{"RUNNING"},
			runningFilter: RunningStatusFilterEvicted,
			expected:      []sqlcv1.V1ReadableStatusOlap{sqlcv1.V1ReadableStatusOlapEVICTED},
		},
		{
			name:          "unknown is dropped",
			statuses:      []string{"FAILED", "BOGUS"},
			runningFilter: RunningStatusFilterAll,
			expected:      []sqlcv1.V1ReadableStatusOlap{sqlcv1.V1ReadableStatusOlapFAILED},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ReadableStatusesForFilter(tt.statuses, tt.runningFilter))
		})
	}
}

func TestValidateReadableStatuses(t *testing.T) {
	assert.NoError(t, ValidateReadableStatuses(nil))
	assert.NoError(t, ValidateReadableStatuses([]string{"QUEUED", "RUNNING", "FAILED", "COMPLETED", "CANCELLED", "EVICTED"}))
	assert.Error(t, ValidateReadableStatuses([]string{"running"}))
	assert.Error(t, ValidateReadableStatuses([]string{"FAILED", "BOGUS"}))
}
