package repository

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type RunningStatusFilter string

const (
	RunningStatusFilterAll      RunningStatusFilter = "ALL"
	RunningStatusFilterOnWorker RunningStatusFilter = "ON_WORKER"
	RunningStatusFilterEvicted  RunningStatusFilter = "EVICTED"
)

// ValidateReadableStatuses rejects unknown statuses, since ReadableStatusesForFilter silently drops them.
func ValidateReadableStatuses(statuses []string) error {
	for _, status := range statuses {
		switch sqlcv1.V1ReadableStatusOlap(status) {
		case sqlcv1.V1ReadableStatusOlapQUEUED,
			sqlcv1.V1ReadableStatusOlapRUNNING,
			sqlcv1.V1ReadableStatusOlapFAILED,
			sqlcv1.V1ReadableStatusOlapCOMPLETED,
			sqlcv1.V1ReadableStatusOlapCANCELLED,
			sqlcv1.V1ReadableStatusOlapEVICTED:
		default:
			return fmt.Errorf("invalid status: %q", status)
		}
	}

	return nil
}

// ReadableStatusesForFilter maps public run statuses to OLAP statuses, expanding RUNNING into the
// separately stored RUNNING and EVICTED statuses. Empty statuses means all statuses; unknown ones are dropped.
func ReadableStatusesForFilter(statuses []string, runningFilter RunningStatusFilter) []sqlcv1.V1ReadableStatusOlap {
	if len(statuses) == 0 {
		statuses = []string{
			string(sqlcv1.V1ReadableStatusOlapQUEUED),
			string(sqlcv1.V1ReadableStatusOlapRUNNING),
			string(sqlcv1.V1ReadableStatusOlapFAILED),
			string(sqlcv1.V1ReadableStatusOlapCOMPLETED),
			string(sqlcv1.V1ReadableStatusOlapCANCELLED),
		}
	}

	normalized := make([]sqlcv1.V1ReadableStatusOlap, 0, len(statuses)+1)
	seen := make(map[sqlcv1.V1ReadableStatusOlap]struct{}, len(statuses)+1)

	add := func(status sqlcv1.V1ReadableStatusOlap) {
		if _, exists := seen[status]; exists {
			return
		}

		seen[status] = struct{}{}
		normalized = append(normalized, status)
	}

	for _, status := range statuses {
		switch s := sqlcv1.V1ReadableStatusOlap(status); s {
		case sqlcv1.V1ReadableStatusOlapRUNNING:
			if runningFilter != RunningStatusFilterEvicted {
				add(sqlcv1.V1ReadableStatusOlapRUNNING)
			}

			if runningFilter != RunningStatusFilterOnWorker {
				add(sqlcv1.V1ReadableStatusOlapEVICTED)
			}
		case sqlcv1.V1ReadableStatusOlapQUEUED,
			sqlcv1.V1ReadableStatusOlapFAILED,
			sqlcv1.V1ReadableStatusOlapCOMPLETED,
			sqlcv1.V1ReadableStatusOlapCANCELLED,
			sqlcv1.V1ReadableStatusOlapEVICTED:
			add(s)
		}
	}

	return normalized
}
