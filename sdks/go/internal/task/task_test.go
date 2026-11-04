//go:build !e2e && !load && !rampup && !integration

package task_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/worker/condition"
	"github.com/hatchet-dev/hatchet/sdks/go/internal/task"
)

func TestDurableTaskDeclaration_DumpPreservesConditions(t *testing.T) {
	waitCond := condition.SleepCondition(5 * time.Second)
	skipCond := condition.UserEventCondition("skip-event", "input.skip == true")
	cancelCond := condition.SleepCondition(30 * time.Second)

	decl := &task.DurableTaskDeclaration[any]{
		Name:     "durable-task-1",
		Parents:  []string{"parent-task"},
		WaitFor:  waitCond,
		SkipIf:   skipCond,
		CancelIf: cancelCond,
	}

	opts := decl.Dump("test-workflow", nil)

	require.NotNil(t, opts)
	assert.Equal(t, "durable-task-1", opts.ReadableId)
	assert.True(t, opts.IsDurable)
	assert.Equal(t, []string{"parent-task"}, opts.Parents)

	require.NotNil(t, opts.Conditions)
	assert.Len(t, opts.Conditions.SleepConditions, 2)
	assert.Len(t, opts.Conditions.UserEventConditions, 1)

	assert.Equal(t, contracts.Action_QUEUE, opts.Conditions.SleepConditions[0].Base.Action)
	assert.Equal(t, "5000ms", opts.Conditions.SleepConditions[0].SleepFor)

	assert.Equal(t, contracts.Action_CANCEL, opts.Conditions.SleepConditions[1].Base.Action)
	assert.Equal(t, "30000ms", opts.Conditions.SleepConditions[1].SleepFor)

	assert.Equal(t, contracts.Action_SKIP, opts.Conditions.UserEventConditions[0].Base.Action)
	assert.Equal(t, "skip-event", opts.Conditions.UserEventConditions[0].UserEventKey)
}

func TestTaskDeclaration_DumpPreservesConditions(t *testing.T) {
	waitCond := condition.SleepCondition(2 * time.Second)
	skipCond := condition.UserEventCondition("skip-event", "input.skip == true")

	decl := &task.TaskDeclaration[any]{
		Name:    "standard-task-1",
		Parents: []string{"parent-task"},
		WaitFor: waitCond,
		SkipIf:  skipCond,
	}

	opts := decl.Dump("test-workflow", nil)

	require.NotNil(t, opts)
	assert.Equal(t, "standard-task-1", opts.ReadableId)
	assert.False(t, opts.IsDurable)
	assert.Equal(t, []string{"parent-task"}, opts.Parents)

	require.NotNil(t, opts.Conditions)
	assert.Len(t, opts.Conditions.SleepConditions, 1)
	assert.Len(t, opts.Conditions.UserEventConditions, 1)

	assert.Equal(t, contracts.Action_QUEUE, opts.Conditions.SleepConditions[0].Base.Action)
	assert.Equal(t, "2000ms", opts.Conditions.SleepConditions[0].SleepFor)

	assert.Equal(t, contracts.Action_SKIP, opts.Conditions.UserEventConditions[0].Base.Action)
	assert.Equal(t, "skip-event", opts.Conditions.UserEventConditions[0].UserEventKey)
}
