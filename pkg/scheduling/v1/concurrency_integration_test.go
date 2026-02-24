//go:build integration

package v1_test

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/config/database"
	repo "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestConcurrency_ChainedStrategiesDoNotContaminate(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		requireSchedulerSchema(t, ctx, conf)

		r := conf.V1

		// 1. Create a tenant
		tenantId := uuid.New()
		_, err := r.Tenant().CreateTenant(ctx, &repo.CreateTenantOpts{
			ID:   &tenantId,
			Name: "concurrency-chain-test",
			Slug: fmt.Sprintf("concurrency-chain-test-%s", tenantId.String()),
		})
		require.NoError(t, err)

		// 2. Create a workflow with two chained workflow-level concurrency strategies:
		//    Gate 1: CANCEL_NEWEST maxRuns=3
		//    Gate 2: GROUP_ROUND_ROBIN maxRuns=1
		desc := "test workflow for chained concurrency"
		cancelNewest := "CANCEL_NEWEST"
		groupRR := "GROUP_ROUND_ROBIN"
		var maxRunsGate1 int32 = 3
		var maxRunsGate2 int32 = 1

		wfVersion, err := r.Workflows().PutWorkflowVersion(ctx, tenantId, &repo.CreateWorkflowVersionOpts{
			Name:        "chained-concurrency-test",
			Description: &desc,
			Tasks: []repo.CreateStepOpts{
				{
					ReadableId: "my-task",
					Action:     "test:run",
				},
			},
			Concurrency: []repo.CreateConcurrencyOpts{
				{
					MaxRuns:       &maxRunsGate1,
					LimitStrategy: &cancelNewest,
					Expression:    "input.my_id",
				},
				{
					MaxRuns:       &maxRunsGate2,
					LimitStrategy: &groupRR,
					Expression:    "input.my_id",
				},
			},
		})
		require.NoError(t, err)

		workflowId := wfVersion.WorkflowVersion.WorkflowId
		workflowVersionId := wfVersion.WorkflowVersion.ID

		// 3. Get the step concurrency strategies created by the trigger
		queries := sqlcv1.New()
		strategies, err := queries.ListActiveConcurrencyStrategies(ctx, conf.Pool, tenantId)
		require.NoError(t, err)
		require.Len(t, strategies, 2, "expected 2 step concurrency strategies")

		// Sort by ID (lower ID = first strategy created = CANCEL_NEWEST)
		sort.Slice(strategies, func(i, j int) bool {
			return strategies[i].ID < strategies[j].ID
		})

		strat1 := strategies[0] // CANCEL_NEWEST
		strat2 := strategies[1] // GROUP_ROUND_ROBIN

		require.Equal(t, sqlcv1.V1ConcurrencyStrategyCANCELNEWEST, strat1.Strategy)
		require.Equal(t, sqlcv1.V1ConcurrencyStrategyGROUPROUNDROBIN, strat2.Strategy)
		require.Equal(t, int32(3), strat1.MaxConcurrency)
		require.Equal(t, int32(1), strat2.MaxConcurrency)

		stepId := strat1.StepID

		// 4. Insert 5 tasks with both concurrency strategies
		numTasks := 5
		taskParams := sqlcv1.CreateTasksParams{
			Tenantids:                    make([]uuid.UUID, numTasks),
			Queues:                       make([]string, numTasks),
			Actionids:                    make([]string, numTasks),
			Stepids:                      make([]uuid.UUID, numTasks),
			Stepreadableids:              make([]string, numTasks),
			Workflowids:                  make([]uuid.UUID, numTasks),
			Scheduletimeouts:             make([]string, numTasks),
			Steptimeouts:                 make([]string, numTasks),
			Priorities:                   make([]int32, numTasks),
			Stickies:                     make([]string, numTasks),
			Desiredworkerids:             make([]*uuid.UUID, numTasks),
			Externalids:                  make([]uuid.UUID, numTasks),
			Displaynames:                 make([]string, numTasks),
			Inputs:                       make([][]byte, numTasks),
			Retrycounts:                  make([]int32, numTasks),
			Additionalmetadatas:          make([][]byte, numTasks),
			InitialStates:                make([]string, numTasks),
			InitialStateReasons:          make([]pgtype.Text, numTasks),
			Dagids:                       make([]pgtype.Int8, numTasks),
			Daginsertedats:               make([]pgtype.Timestamptz, numTasks),
			Concurrencyparentstrategyids: make([][]pgtype.Int8, numTasks),
			ConcurrencyStrategyIds:       make([][]int64, numTasks),
			ConcurrencyKeys:              make([][]string, numTasks),
			ParentTaskExternalIds:        make([]*uuid.UUID, numTasks),
			ParentTaskIds:                make([]pgtype.Int8, numTasks),
			ParentTaskInsertedAts:        make([]pgtype.Timestamptz, numTasks),
			ChildIndex:                   make([]pgtype.Int8, numTasks),
			ChildKey:                     make([]pgtype.Text, numTasks),
			StepIndex:                    make([]int64, numTasks),
			RetryBackoffFactor:           make([]pgtype.Float8, numTasks),
			RetryMaxBackoff:              make([]pgtype.Int4, numTasks),
			WorkflowVersionIds:           make([]uuid.UUID, numTasks),
			WorkflowRunIds:               make([]uuid.UUID, numTasks),
		}

		for i := 0; i < numTasks; i++ {
			taskParams.Tenantids[i] = tenantId
			taskParams.Queues[i] = "default"
			taskParams.Actionids[i] = "test:run"
			taskParams.Stepids[i] = stepId
			taskParams.Stepreadableids[i] = "my-task"
			taskParams.Workflowids[i] = workflowId
			taskParams.Scheduletimeouts[i] = "5m"
			taskParams.Priorities[i] = 1
			taskParams.Stickies[i] = string(sqlcv1.V1StickyStrategyNONE)
			taskParams.Externalids[i] = uuid.New()
			taskParams.Displaynames[i] = fmt.Sprintf("task-%d", i)
			taskParams.Inputs[i] = []byte(`{"my_id": "test-key"}`)
			taskParams.Additionalmetadatas[i] = []byte(`{}`)
			taskParams.InitialStates[i] = string(sqlcv1.V1TaskInitialStateQUEUED)
			taskParams.Concurrencyparentstrategyids[i] = []pgtype.Int8{strat1.ParentStrategyID, strat2.ParentStrategyID}
			taskParams.ConcurrencyStrategyIds[i] = []int64{strat1.ID, strat2.ID}
			taskParams.ConcurrencyKeys[i] = []string{"test-key", "test-key"}
			taskParams.WorkflowVersionIds[i] = workflowVersionId
			taskParams.WorkflowRunIds[i] = uuid.New()
		}

		tasks, err := queries.CreateTasks(ctx, conf.Pool, taskParams)
		require.NoError(t, err)
		require.Len(t, tasks, numTasks)

		concurrencyRepo := r.Scheduler().Concurrency()

		// 5. Run strategy 1 (CANCEL_NEWEST, maxRuns=3):
		//    - 3 oldest tasks should advance to next strategy (slots filled, triggers slot creation for strategy 2)
		//    - 2 newest tasks should be cancelled
		res1, err := concurrencyRepo.RunConcurrencyStrategy(ctx, tenantId, strat1)
		require.NoError(t, err)
		require.Len(t, res1.NextConcurrencyStrategies, 3, "expected 3 tasks to advance to next strategy")
		require.Len(t, res1.Cancelled, 2, "expected 2 tasks cancelled by CANCEL_NEWEST")
		require.Len(t, res1.Queued, 0, "no tasks should be queued directly")

		// 6. Run strategy 1 AGAIN — this is the key regression check.
		//    Before the fix, this would fill strategy 2's unfilled slots (same task_id, key, is_filled=FALSE)
		//    and queue them directly, bypassing strategy 2's round-robin logic.
		res1Again, err := concurrencyRepo.RunConcurrencyStrategy(ctx, tenantId, strat1)
		require.NoError(t, err)
		require.Len(t, res1Again.Queued, 0, "re-running strategy 1 must not queue tasks from strategy 2's slots")
		require.Len(t, res1Again.Cancelled, 0, "re-running strategy 1 must not cancel anything")
		require.Len(t, res1Again.NextConcurrencyStrategies, 0, "re-running strategy 1 must not advance anything")

		// 7. Run strategy 2 (GROUP_ROUND_ROBIN, maxRuns=1):
		//    Only 1 task should be queued per concurrency key.
		res2, err := concurrencyRepo.RunConcurrencyStrategy(ctx, tenantId, strat2)
		require.NoError(t, err)
		require.Len(t, res2.Queued, 1, "GROUP_ROUND_ROBIN with maxRuns=1 should queue exactly 1 task")

		return nil
	})
}
