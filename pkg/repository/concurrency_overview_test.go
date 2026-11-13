//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCombineStrategySummaries(t *testing.T) {
	workflowId := uuid.New()
	oldVersion, newVersion := uuid.New(), uuid.New()
	older, newer := time.Now().Add(-time.Hour), time.Now()

	task := func(id int64, version uuid.UUID, taskName, expression string, draining bool) *ConcurrencyStrategySummary {
		return &ConcurrencyStrategySummary{
			StrategyIDs:       []int64{id},
			definitionID:      id,
			Scope:             ConcurrencyScopeTask,
			Name:              taskName,
			WorkflowID:        &workflowId,
			WorkflowVersionID: &version,
			TaskName:          taskName,
			Expression:        expression,
			IsDraining:        draining,
			KeyCount:          1,
			RunningCount:      2,
			QueuedCount:       3,
		}
	}

	drainingVersion := task(10, oldVersion, "process", "input.old", true)
	drainingVersion.OldestQueuedAt = &older
	drainingVersion.OldestQueuedKey = "old-key"

	latest := task(20, newVersion, "process", "input.userId", false)
	latest.OldestQueuedAt = &newer
	latest.OldestQueuedKey = "new-key"

	secondOnTask := task(21, newVersion, "process", "input.tenantId", false)
	otherTask := task(22, newVersion, "notify", "input.userId", false)
	shared := &ConcurrencyStrategySummary{StrategyIDs: []int64{5}, definitionID: 5, Scope: ConcurrencyScopeShared, Name: "per-customer"}

	res := combineStrategySummaries([]*ConcurrencyStrategySummary{shared, drainingVersion, latest, secondOnTask, otherTask})

	require.Len(t, res, 3)
	require.Equal(t, ConcurrencyStrategyRef{SharedName: "per-customer"}, res[0].Ref())
	require.Equal(t, ConcurrencyStrategyRef{WorkflowID: workflowId, TaskName: "notify"}, res[2].Ref())

	process := res[1]
	require.Equal(t, ConcurrencyStrategyRef{WorkflowID: workflowId, TaskName: "process"}, process.Ref())
	require.ElementsMatch(t, []int64{10, 20, 21}, process.StrategyIDs)
	require.False(t, process.IsDraining)

	// the newest strategy on a latest version provides the definition
	require.Equal(t, "input.tenantId", process.Expression)
	require.Equal(t, newVersion, *process.WorkflowVersionID)

	require.EqualValues(t, 3, process.KeyCount)
	require.EqualValues(t, 6, process.RunningCount)
	require.EqualValues(t, 9, process.QueuedCount)
	require.Equal(t, older, *process.OldestQueuedAt)
	require.Equal(t, "old-key", process.OldestQueuedKey)
}
