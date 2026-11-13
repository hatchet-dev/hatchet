package repository

import (
	"fmt"
	"slices"

	"github.com/hatchet-dev/hatchet/pkg/enums"
)

type RunningStatusFilter string

const (
	RunningStatusFilterAll      RunningStatusFilter = "ALL"
	RunningStatusFilterOnWorker RunningStatusFilter = "ON_WORKER"
	RunningStatusFilterEvicted  RunningStatusFilter = "EVICTED"
)

var defaultReadableStatuses = []enums.V1ReadableStatusOlap{
	enums.V1ReadableStatusOlapQUEUED,
	enums.V1ReadableStatusOlapRUNNING,
	enums.V1ReadableStatusOlapFAILED,
	enums.V1ReadableStatusOlapCOMPLETED,
	enums.V1ReadableStatusOlapCANCELLED,
}

// expand splits public RUNNING into the separately stored RUNNING and EVICTED statuses.
func (f RunningStatusFilter) expand(status enums.V1ReadableStatusOlap) []enums.V1ReadableStatusOlap {
	if status != enums.V1ReadableStatusOlapRUNNING {
		return []enums.V1ReadableStatusOlap{status}
	}

	switch f {
	case RunningStatusFilterOnWorker:
		return []enums.V1ReadableStatusOlap{enums.V1ReadableStatusOlapRUNNING}
	case RunningStatusFilterEvicted:
		return []enums.V1ReadableStatusOlap{enums.V1ReadableStatusOlapEVICTED}
	case RunningStatusFilterAll:
	}

	return []enums.V1ReadableStatusOlap{enums.V1ReadableStatusOlapRUNNING, enums.V1ReadableStatusOlapEVICTED}
}

// ReadableStatusesForFilter rejects unknown statuses so an invalid-only filter can't fall back to all statuses.
func ReadableStatusesForFilter[S ~string](statuses []S, runningFilter RunningStatusFilter) ([]enums.V1ReadableStatusOlap, error) {
	parsed := make([]enums.V1ReadableStatusOlap, 0, len(statuses))

	for _, s := range statuses {
		status := enums.V1ReadableStatusOlap(s)
		if !status.Valid() {
			return nil, fmt.Errorf("invalid status: %q", s)
		}

		parsed = append(parsed, status)
	}

	if len(parsed) == 0 {
		parsed = defaultReadableStatuses
	}

	result := make([]enums.V1ReadableStatusOlap, 0, len(parsed)+1)

	for _, status := range parsed {
		for _, expanded := range runningFilter.expand(status) {
			if !slices.Contains(result, expanded) {
				result = append(result, expanded)
			}
		}
	}

	return result, nil
}
