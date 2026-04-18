package durabletasks

import (
	"encoding/json"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func (t *DurableTasksService) V1DurableTaskEventLogList(ctx echo.Context, request gen.V1DurableTaskEventLogListRequestObject) (gen.V1DurableTaskEventLogListResponseObject, error) {
	taskInterface := ctx.Get("durable-task")

	if taskInterface == nil {
		return gen.V1DurableTaskEventLogList404JSONResponse{
			Errors: []gen.APIError{{Description: "durable task not found"}},
		}, nil
	}

	task, ok := taskInterface.(*sqlcv1.V1TasksOlap)
	if !ok {
		return nil, echo.NewHTTPError(500, "durable task type assertion failed")
	}

	entries, err := t.config.V1.OLAP().ListDurableEventLog(
		ctx.Request().Context(),
		task.TenantID,
		task.ID,
		task.InsertedAt,
	)
	if err != nil {
		return nil, err
	}

	return gen.V1DurableTaskEventLogList200JSONResponse(toDurableEventLogEntries(entries)), nil
}

func toDurableEventLogEntries(entries []*sqlcv1.V1DurableEventLogEntry) []gen.V1DurableEventLogEntry {
	result := make([]gen.V1DurableEventLogEntry, 0, len(entries))

	for _, e := range entries {
		var insertedAt time.Time
		if e.InsertedAt.Valid {
			insertedAt = e.InsertedAt.Time
		}

		entry := gen.V1DurableEventLogEntry{
			NodeId:      e.NodeID,
			BranchId:    e.BranchID,
			Kind:        gen.V1DurableEventLogKind(e.Kind),
			IsSatisfied: e.IsSatisfied,
			InsertedAt:  insertedAt,
		}

		if e.UserMessage.Valid {
			entry.UserMessage = &e.UserMessage.String
		}

		if e.SatisfiedAt.Valid {
			entry.SatisfiedAt = &e.SatisfiedAt.Time
		}

		if len(e.WaitData) > 0 {
			entry.WaitData = toGenWaitData(e.WaitData)
		}

		result = append(result, entry)
	}

	return result
}

func toGenWaitData(raw []byte) *gen.V1WaitData {
	var wd repository.WaitData
	if err := json.Unmarshal(raw, &wd); err != nil {
		return nil
	}

	orGroups := make([]gen.V1DurableWaitOrGroup, 0, len(wd.OrGroups))
	for _, g := range wd.OrGroups {
		conditions := make([]gen.V1DurableWaitCondition, 0, len(g.Conditions))
		for _, c := range g.Conditions {
			cond := gen.V1DurableWaitCondition{
				Kind:            gen.V1DurableWaitConditionKind(c.Kind),
				SleepDurationMs: c.SleepDurationMs,
				EventKey:        c.EventKey,
				WorkflowName:    c.WorkflowName,
			}
			conditions = append(conditions, cond)
		}
		orGroups = append(orGroups, gen.V1DurableWaitOrGroup{Conditions: conditions})
	}

	return &gen.V1WaitData{OrGroups: orGroups}
}
