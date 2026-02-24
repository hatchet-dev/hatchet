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

// concurrencyTestSetup holds the common state for concurrency integration tests.
type concurrencyTestSetup struct {
	tenantId        uuid.UUID
	strategy        *sqlcv1.V1StepConcurrency
	concurrencyRepo repo.ConcurrencyRepository
}

// setupStepConcurrencyTest creates a tenant, workflow with a single step-level concurrency
// strategy, and inserts numTasks tasks. Returns the setup needed to call RunConcurrencyStrategy.
func setupStepConcurrencyTest(
	t *testing.T,
	ctx context.Context,
	conf *database.Layer,
	name string,
	strategyType string,
	maxRuns int32,
	numTasks int,
) *concurrencyTestSetup {
	t.Helper()

	r := conf.V1
	queries := sqlcv1.New()

	tenantId := uuid.New()
	_, err := r.Tenant().CreateTenant(ctx, &repo.CreateTenantOpts{
		ID:   &tenantId,
		Name: name,
		Slug: fmt.Sprintf("%s-%s", name, tenantId.String()),
	})
	require.NoError(t, err)

	desc := "test workflow"
	wfVersion, err := r.Workflows().PutWorkflowVersion(ctx, tenantId, &repo.CreateWorkflowVersionOpts{
		Name:        name,
		Description: &desc,
		Tasks: []repo.CreateStepOpts{
			{
				ReadableId: "my-task",
				Action:     "test:run",
				Concurrency: []repo.CreateConcurrencyOpts{
					{
						MaxRuns:       &maxRuns,
						LimitStrategy: &strategyType,
						Expression:    "input.my_id",
					},
				},
			},
		},
	})
	require.NoError(t, err)

	workflowId := wfVersion.WorkflowVersion.WorkflowId
	workflowVersionId := wfVersion.WorkflowVersion.ID

	strategies, err := queries.ListActiveConcurrencyStrategies(ctx, conf.Pool, tenantId)
	require.NoError(t, err)
	require.Len(t, strategies, 1)

	strat := strategies[0]
	require.Equal(t, sqlcv1.V1ConcurrencyStrategy(strategyType), strat.Strategy)
	require.Equal(t, maxRuns, strat.MaxConcurrency)
	require.False(t, strat.ParentStrategyID.Valid, "step-level concurrency should have no parent")

	taskParams := newCreateTasksParams(numTasks)
	for i := 0; i < numTasks; i++ {
		taskParams.Tenantids[i] = tenantId
		taskParams.Queues[i] = "default"
		taskParams.Actionids[i] = "test:run"
		taskParams.Stepids[i] = strat.StepID
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
		taskParams.Concurrencyparentstrategyids[i] = []pgtype.Int8{{}}
		taskParams.ConcurrencyStrategyIds[i] = []int64{strat.ID}
		taskParams.ConcurrencyKeys[i] = []string{"test-key"}
		taskParams.WorkflowVersionIds[i] = workflowVersionId
		taskParams.WorkflowRunIds[i] = uuid.New()
	}

	tasks, err := queries.CreateTasks(ctx, conf.Pool, taskParams)
	require.NoError(t, err)
	require.Len(t, tasks, numTasks)

	return &concurrencyTestSetup{
		tenantId:        tenantId,
		strategy:        strat,
		concurrencyRepo: r.Scheduler().Concurrency(),
	}
}

// newCreateTasksParams allocates a CreateTasksParams with all slices pre-sized to n.
func newCreateTasksParams(n int) sqlcv1.CreateTasksParams {
	return sqlcv1.CreateTasksParams{
		Tenantids:                    make([]uuid.UUID, n),
		Queues:                       make([]string, n),
		Actionids:                    make([]string, n),
		Stepids:                      make([]uuid.UUID, n),
		Stepreadableids:              make([]string, n),
		Workflowids:                  make([]uuid.UUID, n),
		Scheduletimeouts:             make([]string, n),
		Steptimeouts:                 make([]string, n),
		Priorities:                   make([]int32, n),
		Stickies:                     make([]string, n),
		Desiredworkerids:             make([]*uuid.UUID, n),
		Externalids:                  make([]uuid.UUID, n),
		Displaynames:                 make([]string, n),
		Inputs:                       make([][]byte, n),
		Retrycounts:                  make([]int32, n),
		Additionalmetadatas:          make([][]byte, n),
		InitialStates:                make([]string, n),
		InitialStateReasons:          make([]pgtype.Text, n),
		Dagids:                       make([]pgtype.Int8, n),
		Daginsertedats:               make([]pgtype.Timestamptz, n),
		Concurrencyparentstrategyids: make([][]pgtype.Int8, n),
		ConcurrencyStrategyIds:       make([][]int64, n),
		ConcurrencyKeys:              make([][]string, n),
		ParentTaskExternalIds:        make([]*uuid.UUID, n),
		ParentTaskIds:                make([]pgtype.Int8, n),
		ParentTaskInsertedAts:        make([]pgtype.Timestamptz, n),
		ChildIndex:                   make([]pgtype.Int8, n),
		ChildKey:                     make([]pgtype.Text, n),
		StepIndex:                    make([]int64, n),
		RetryBackoffFactor:           make([]pgtype.Float8, n),
		RetryMaxBackoff:              make([]pgtype.Int4, n),
		WorkflowVersionIds:           make([]uuid.UUID, n),
		WorkflowRunIds:               make([]uuid.UUID, n),
	}
}

// --- Standalone strategy tests (no chaining) ---

func TestConcurrency_GroupRoundRobin(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		// GROUP_ROUND_ROBIN with maxRuns=2 and 5 tasks (same key):
		// Fills 2 oldest slots per key. Remaining 3 stay unfilled (not cancelled).
		s := setupStepConcurrencyTest(t, ctx, conf, "grr-test", "GROUP_ROUND_ROBIN", 2, 5)

		res, err := s.concurrencyRepo.RunConcurrencyStrategy(ctx, s.tenantId, s.strategy)
		require.NoError(t, err)
		require.Len(t, res.Queued, 2, "GROUP_ROUND_ROBIN should queue maxRuns tasks")
		require.Len(t, res.Cancelled, 0, "GROUP_ROUND_ROBIN should not cancel tasks")

		return nil
	})
}

func TestConcurrency_CancelNewest(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		// CANCEL_NEWEST with maxRuns=2 and 5 tasks (same key):
		// Fills 2 oldest slots, cancels 3 newest.
		s := setupStepConcurrencyTest(t, ctx, conf, "cn-test", "CANCEL_NEWEST", 2, 5)

		res, err := s.concurrencyRepo.RunConcurrencyStrategy(ctx, s.tenantId, s.strategy)
		require.NoError(t, err)
		require.Len(t, res.Queued, 2, "CANCEL_NEWEST should queue maxRuns oldest tasks")
		require.Len(t, res.Cancelled, 3, "CANCEL_NEWEST should cancel excess newest tasks")

		return nil
	})
}

func TestConcurrency_CancelInProgress(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		// CANCEL_IN_PROGRESS with maxRuns=2 and 5 tasks (same key):
		// Fills 2 newest slots, cancels 3 oldest.
		s := setupStepConcurrencyTest(t, ctx, conf, "cip-test", "CANCEL_IN_PROGRESS", 2, 5)

		res, err := s.concurrencyRepo.RunConcurrencyStrategy(ctx, s.tenantId, s.strategy)
		require.NoError(t, err)
		require.Len(t, res.Queued, 2, "CANCEL_IN_PROGRESS should queue maxRuns newest tasks")
		require.Len(t, res.Cancelled, 3, "CANCEL_IN_PROGRESS should cancel excess oldest tasks")

		return nil
	})
}

// --- Chained strategy regression test ---

// TestConcurrency_ChainedStrategiesDoNotContaminate verifies that when two concurrency
// strategies are chained (CANCEL_NEWEST → GROUP_ROUND_ROBIN), re-running the first
// strategy does not accidentally fill the second strategy's slots.
func TestConcurrency_ChainedStrategiesDoNotContaminate(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		requireSchedulerSchema(t, ctx, conf)

		r := conf.V1
		queries := sqlcv1.New()

		tenantId := uuid.New()
		_, err := r.Tenant().CreateTenant(ctx, &repo.CreateTenantOpts{
			ID:   &tenantId,
			Name: "concurrency-chain-test",
			Slug: fmt.Sprintf("concurrency-chain-test-%s", tenantId.String()),
		})
		require.NoError(t, err)

		// Create a workflow with two chained workflow-level concurrency strategies:
		//   Gate 1: CANCEL_NEWEST maxRuns=3
		//   Gate 2: GROUP_ROUND_ROBIN maxRuns=1
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

		strategies, err := queries.ListActiveConcurrencyStrategies(ctx, conf.Pool, tenantId)
		require.NoError(t, err)
		require.Len(t, strategies, 2, "expected 2 step concurrency strategies")

		sort.Slice(strategies, func(i, j int) bool {
			return strategies[i].ID < strategies[j].ID
		})

		strat1 := strategies[0] // CANCEL_NEWEST
		strat2 := strategies[1] // GROUP_ROUND_ROBIN

		require.Equal(t, sqlcv1.V1ConcurrencyStrategyCANCELNEWEST, strat1.Strategy)
		require.Equal(t, sqlcv1.V1ConcurrencyStrategyGROUPROUNDROBIN, strat2.Strategy)

		stepId := strat1.StepID

		// Insert 5 tasks with both concurrency strategies
		numTasks := 5
		taskParams := newCreateTasksParams(numTasks)
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

		// Run strategy 1 (CANCEL_NEWEST, maxRuns=3):
		//   3 oldest advance to next strategy, 2 newest cancelled.
		res1, err := concurrencyRepo.RunConcurrencyStrategy(ctx, tenantId, strat1)
		require.NoError(t, err)
		require.Len(t, res1.NextConcurrencyStrategies, 3, "expected 3 tasks to advance to next strategy")
		require.Len(t, res1.Cancelled, 2, "expected 2 tasks cancelled by CANCEL_NEWEST")
		require.Len(t, res1.Queued, 0, "no tasks should be queued directly")

		// Run strategy 1 AGAIN — the key regression check.
		// Before the fix, this filled strategy 2's unfilled slots and queued them directly.
		res1Again, err := concurrencyRepo.RunConcurrencyStrategy(ctx, tenantId, strat1)
		require.NoError(t, err)
		require.Len(t, res1Again.Queued, 0, "re-running strategy 1 must not queue tasks from strategy 2's slots")
		require.Len(t, res1Again.Cancelled, 0, "re-running strategy 1 must not cancel anything")
		require.Len(t, res1Again.NextConcurrencyStrategies, 0, "re-running strategy 1 must not advance anything")

		// Run strategy 2 (GROUP_ROUND_ROBIN, maxRuns=1):
		//   Only 1 task should be queued per concurrency key.
		res2, err := concurrencyRepo.RunConcurrencyStrategy(ctx, tenantId, strat2)
		require.NoError(t, err)
		require.Len(t, res2.Queued, 1, "GROUP_ROUND_ROBIN with maxRuns=1 should queue exactly 1 task")

		return nil
	})
}
