package transformers

import (
	"math"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository"
)

func ToV1ConcurrencyStrategy(s *repository.ConcurrencyStrategySummary) gen.V1ConcurrencyStrategy {
	res := gen.V1ConcurrencyStrategy{
		Id:                s.ID,
		Scope:             gen.V1ConcurrencyScope(s.Scope),
		Name:              s.Name,
		WorkflowId:        s.WorkflowID,
		WorkflowVersionId: s.WorkflowVersionID,
		Expression:        s.Expression,
		LimitStrategy:     gen.V1ConcurrencyLimitStrategy(s.Strategy),
		MaxRuns:           s.MaxRuns,
		MaxRunsExpression: s.MaxRunsExpression,
		KeyCount:          s.KeyCount,
		QueuedKeyCount:    s.QueuedKeyCount,
		RunningCount:      s.RunningCount,
		QueuedCount:       s.QueuedCount,
		OldestQueuedAt:    s.OldestQueuedAt,
	}

	if s.WorkflowID != nil {
		res.WorkflowName = &s.WorkflowName
	}

	if s.WorkflowVersion != "" {
		res.WorkflowVersion = &s.WorkflowVersion
	}

	if s.TaskName != "" {
		res.TaskName = &s.TaskName
	}

	if s.OldestQueuedKey != "" {
		res.OldestQueuedKey = &s.OldestQueuedKey
	}

	if s.Scope == repository.ConcurrencyScopeShared {
		usedBy := make([]gen.V1ConcurrencyStrategyUsage, len(s.UsedBy))

		for i, u := range s.UsedBy {
			usedBy[i] = gen.V1ConcurrencyStrategyUsage{
				WorkflowId:   u.WorkflowID,
				WorkflowName: u.WorkflowName,
				TaskName:     u.TaskName,
			}
		}

		res.UsedBy = &usedBy
	}

	return res
}

func ToV1ConcurrencyStrategyList(strategies []*repository.ConcurrencyStrategySummary) gen.V1ConcurrencyStrategyList {
	rows := make([]gen.V1ConcurrencyStrategy, len(strategies))

	for i, s := range strategies {
		rows[i] = ToV1ConcurrencyStrategy(s)
	}

	return gen.V1ConcurrencyStrategyList{
		Rows: rows,
	}
}

func ToV1ConcurrencyKeyList(keys []*repository.ConcurrencyKey, total, limit, offset int64) gen.V1ConcurrencyKeyList {
	rows := make([]gen.V1ConcurrencyKey, len(keys))

	for i, k := range keys {
		rows[i] = gen.V1ConcurrencyKey{
			Key:            k.Key,
			MaxRuns:        k.MaxRuns,
			RunningCount:   k.RunningCount,
			QueuedCount:    k.QueuedCount,
			OldestQueuedAt: k.OldestQueuedAt,
		}
	}

	currentPage := offset / limit
	nextPage := currentPage + 1
	numPages := int64(math.Ceil(float64(total) / float64(limit)))

	return gen.V1ConcurrencyKeyList{
		Rows:  rows,
		Total: total,
		Pagination: &gen.PaginationResponse{
			CurrentPage: &currentPage,
			NextPage:    &nextPage,
			NumPages:    &numPages,
		},
	}
}
