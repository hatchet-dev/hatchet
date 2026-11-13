package repository

import (
	"fmt"
	"slices"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type RunningStatusFilter string

const (
	RunningStatusFilterAll      RunningStatusFilter = "ALL"
	RunningStatusFilterOnWorker RunningStatusFilter = "ON_WORKER"
	RunningStatusFilterEvicted  RunningStatusFilter = "EVICTED"
)

var defaultReadableStatuses = []sqlcv1.V1ReadableStatusOlap{
	sqlcv1.V1ReadableStatusOlapQUEUED,
	sqlcv1.V1ReadableStatusOlapRUNNING,
	sqlcv1.V1ReadableStatusOlapFAILED,
	sqlcv1.V1ReadableStatusOlapCOMPLETED,
	sqlcv1.V1ReadableStatusOlapCANCELLED,
}

// expand splits public RUNNING into the separately stored RUNNING and EVICTED statuses.
func (f RunningStatusFilter) expand(status sqlcv1.V1ReadableStatusOlap) []sqlcv1.V1ReadableStatusOlap {
	if status != sqlcv1.V1ReadableStatusOlapRUNNING {
		return []sqlcv1.V1ReadableStatusOlap{status}
	}

	switch f {
	case RunningStatusFilterOnWorker:
		return []sqlcv1.V1ReadableStatusOlap{sqlcv1.V1ReadableStatusOlapRUNNING}
	case RunningStatusFilterEvicted:
		return []sqlcv1.V1ReadableStatusOlap{sqlcv1.V1ReadableStatusOlapEVICTED}
	case RunningStatusFilterAll:
	}

	return []sqlcv1.V1ReadableStatusOlap{sqlcv1.V1ReadableStatusOlapRUNNING, sqlcv1.V1ReadableStatusOlapEVICTED}
}

// ReadableStatusesForFilter rejects unknown statuses so an invalid-only filter can't fall back to all statuses.
func ReadableStatusesForFilter[S ~string](statuses []S, runningFilter RunningStatusFilter) ([]sqlcv1.V1ReadableStatusOlap, error) {
	parsed := make([]sqlcv1.V1ReadableStatusOlap, 0, len(statuses))

	for _, s := range statuses {
		status := sqlcv1.V1ReadableStatusOlap(s)
		if !status.Valid() {
			return nil, fmt.Errorf("invalid status: %q", s)
		}

		parsed = append(parsed, status)
	}

	if len(parsed) == 0 {
		parsed = defaultReadableStatuses
	}

	result := make([]sqlcv1.V1ReadableStatusOlap, 0, len(parsed)+1)

	for _, status := range parsed {
		for _, expanded := range runningFilter.expand(status) {
			if !slices.Contains(result, expanded) {
				result = append(result, expanded)
			}
		}
	}

	return result, nil
}
