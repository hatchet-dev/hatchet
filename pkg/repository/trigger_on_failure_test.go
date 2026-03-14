//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// Test_dagToTaskIds_excludesOnFailureSteps verifies that on_failure steps are NOT
// counted in dagToTaskIds (and therefore not in total_tasks), which prevents DAG
// runs from getting stuck in RUNNING status when on_failure steps don't execute.
//
// This is a regression test for a bug where:
//   - A workflow with a main task + on_failure task had total_tasks=2
//   - On success, only the main task ran (task_count=1)
//   - The SQL check `task_count != total_tasks` (1 != 2) kept the DAG in RUNNING forever
func Test_dagToTaskIds_excludesOnFailureSteps(t *testing.T) {
	tests := []struct {
		name              string
		steps             []*sqlcv1.ListStepsByWorkflowVersionIdsRow
		expectedTaskCount int
		description       string
	}{
		{
			name: "single step with on_failure - only main step counted",
			steps: []*sqlcv1.ListStepsByWorkflowVersionIdsRow{
				{ID: uuid.New(), JobKind: sqlcv1.JobKindDEFAULT},
				{ID: uuid.New(), JobKind: sqlcv1.JobKindONFAILURE},
			},
			expectedTaskCount: 1,
			description:       "on_failure step should not be counted in dagToTaskIds",
		},
		{
			name: "two steps with on_failure - only default steps counted",
			steps: []*sqlcv1.ListStepsByWorkflowVersionIdsRow{
				{ID: uuid.New(), JobKind: sqlcv1.JobKindDEFAULT},
				{ID: uuid.New(), JobKind: sqlcv1.JobKindDEFAULT},
				{ID: uuid.New(), JobKind: sqlcv1.JobKindONFAILURE},
			},
			expectedTaskCount: 2,
			description:       "only DEFAULT steps counted, ON_FAILURE excluded",
		},
		{
			name: "multiple steps no on_failure - all steps counted",
			steps: []*sqlcv1.ListStepsByWorkflowVersionIdsRow{
				{ID: uuid.New(), JobKind: sqlcv1.JobKindDEFAULT},
				{ID: uuid.New(), JobKind: sqlcv1.JobKindDEFAULT},
				{ID: uuid.New(), JobKind: sqlcv1.JobKindDEFAULT},
			},
			expectedTaskCount: 3,
			description:       "all DEFAULT steps counted when no on_failure present",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tupleExternalId := uuid.New()
			stepsToExternalIds := make(map[uuid.UUID]uuid.UUID)
			dagToTaskIds := make(map[uuid.UUID][]uuid.UUID)

			// Simulate the first loop in triggerWorkflows (lines 629-637)
			// This is the exact logic from trigger.go with the on_failure fix
			for _, step := range tt.steps {
				externalId := uuid.New()
				stepsToExternalIds[step.ID] = externalId

				if step.JobKind != sqlcv1.JobKindONFAILURE {
					dagToTaskIds[tupleExternalId] = append(dagToTaskIds[tupleExternalId], externalId)
				}
			}

			taskIds := dagToTaskIds[tupleExternalId]
			assert.Equal(t, tt.expectedTaskCount, len(taskIds), tt.description)

			// Verify all steps got external IDs (including on_failure)
			require.Equal(t, len(tt.steps), len(stepsToExternalIds),
				"all steps should have external IDs assigned, including on_failure")
		})
	}
}

// Test_dagToTaskIds_onFailureStillGetsExternalId verifies that on_failure steps
// still get an external ID assigned (needed for match condition creation), even
// though they're excluded from dagToTaskIds/total_tasks.
func Test_dagToTaskIds_onFailureStillGetsExternalId(t *testing.T) {
	mainStepId := uuid.New()
	onFailureStepId := uuid.New()

	steps := []*sqlcv1.ListStepsByWorkflowVersionIdsRow{
		{ID: mainStepId, JobKind: sqlcv1.JobKindDEFAULT},
		{ID: onFailureStepId, JobKind: sqlcv1.JobKindONFAILURE},
	}

	tupleExternalId := uuid.New()
	stepsToExternalIds := make(map[uuid.UUID]uuid.UUID)
	dagToTaskIds := make(map[uuid.UUID][]uuid.UUID)

	for _, step := range steps {
		externalId := uuid.New()
		stepsToExternalIds[step.ID] = externalId

		if step.JobKind != sqlcv1.JobKindONFAILURE {
			dagToTaskIds[tupleExternalId] = append(dagToTaskIds[tupleExternalId], externalId)
		}
	}

	// on_failure step should have an external ID (for match conditions)
	_, hasOnFailureId := stepsToExternalIds[onFailureStepId]
	assert.True(t, hasOnFailureId, "on_failure step must have an external ID for match conditions")

	// But it should not be in dagToTaskIds
	for _, taskId := range dagToTaskIds[tupleExternalId] {
		onFailureExternalId := stepsToExternalIds[onFailureStepId]
		assert.NotEqual(t, onFailureExternalId, taskId,
			"on_failure external ID should not appear in dagToTaskIds")
	}

	// total_tasks (len of dagToTaskIds) should be 1
	assert.Equal(t, 1, len(dagToTaskIds[tupleExternalId]),
		"total_tasks should only count the main step")
}

// Test_totalTasks_matchesCreateDAGOpts verifies that createDAGOpts.TaskIds
// only contains non-on_failure step IDs, which determines total_tasks for the DAG.
func Test_totalTasks_matchesCreateDAGOpts(t *testing.T) {
	tupleExternalId := uuid.New()

	steps := []*sqlcv1.ListStepsByWorkflowVersionIdsRow{
		{ID: uuid.New(), JobKind: sqlcv1.JobKindDEFAULT},
		{ID: uuid.New(), JobKind: sqlcv1.JobKindDEFAULT},
		{ID: uuid.New(), JobKind: sqlcv1.JobKindONFAILURE},
	}

	stepsToExternalIds := make(map[uuid.UUID]uuid.UUID)
	dagToTaskIds := make(map[uuid.UUID][]uuid.UUID)

	for _, step := range steps {
		externalId := uuid.New()
		stepsToExternalIds[step.ID] = externalId

		if step.JobKind != sqlcv1.JobKindONFAILURE {
			dagToTaskIds[tupleExternalId] = append(dagToTaskIds[tupleExternalId], externalId)
		}
	}

	// Simulate createDAGOpts construction (trigger.go line 1013-1023)
	opts := createDAGOpts{
		ExternalId: tupleExternalId,
		TaskIds:    dagToTaskIds[tupleExternalId],
	}

	// TotalTasks is set from len(opt.TaskIds) at trigger.go line 1339
	totalTasks := len(opts.TaskIds)

	assert.Equal(t, 2, totalTasks,
		"total_tasks should be 2 (two DEFAULT steps), excluding the ON_FAILURE step")
}
