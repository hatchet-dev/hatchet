//go:build integration

package v1_test

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	v1 "github.com/hatchet-dev/hatchet/pkg/scheduling/v1"
	"github.com/hatchet-dev/pgoutbox"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/config/database"
	repo "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/scheduling/v1/concurrency"
)

// concurrencyTestSetup holds the common state for concurrency integration tests.
type concurrencyTestSetup struct {
	tenantId          uuid.UUID
	strategy          *sqlcv1.V1StepConcurrency
	concurrencyRepo   repo.ConcurrencyRepository
	workflowId        uuid.UUID
	workflowVersionId uuid.UUID
}

// setupStepConcurrencyStrategy creates a tenant and a workflow with a single step-level concurrency
// strategy, but inserts no tasks. Use createConcurrencyTasks to add tasks afterwards.
func setupStepConcurrencyStrategy(
	t *testing.T,
	ctx context.Context,
	conf *database.Layer,
	name string,
	strategyType string,
	maxRuns int32,
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

	strategies, err := queries.ListActiveConcurrencyStrategies(ctx, conf.Pool, tenantId)
	require.NoError(t, err)
	require.Len(t, strategies, 1)

	strat := strategies[0]
	require.Equal(t, sqlcv1.V1ConcurrencyStrategy(strategyType), strat.Strategy)
	require.Equal(t, maxRuns, strat.MaxConcurrency)
	require.False(t, strat.ParentStrategyID.Valid, "step-level concurrency should have no parent")

	return &concurrencyTestSetup{
		tenantId:          tenantId,
		strategy:          strat,
		concurrencyRepo:   r.Scheduler().Concurrency(),
		workflowId:        wfVersion.WorkflowVersion.WorkflowId,
		workflowVersionId: wfVersion.WorkflowVersion.ID,
	}
}

// createConcurrencyTasks inserts numTasks queued tasks that share a single concurrency key against
// the setup's strategy. The v1_concurrency_slot insert trigger emits the WAL the in-memory index
// consumes.
func createConcurrencyTasks(t *testing.T, ctx context.Context, conf *database.Layer, s *concurrencyTestSetup, numTasks int) {
	t.Helper()

	queries := sqlcv1.New()

	taskParams := newCreateTasksParams(numTasks)
	for i := 0; i < numTasks; i++ {
		taskParams.Tenantids[i] = s.tenantId
		taskParams.Queues[i] = "default"
		taskParams.Actionids[i] = "test:run"
		taskParams.Stepids[i] = s.strategy.StepID
		taskParams.Stepreadableids[i] = "my-task"
		taskParams.Workflowids[i] = s.workflowId
		taskParams.Scheduletimeouts[i] = "5m"
		taskParams.Priorities[i] = 1
		taskParams.Stickies[i] = string(sqlcv1.V1StickyStrategyNONE)
		taskParams.Externalids[i] = uuid.New()
		taskParams.Displaynames[i] = fmt.Sprintf("task-%d", i)
		taskParams.Additionalmetadatas[i] = []byte(`{}`)
		taskParams.InitialStates[i] = string(sqlcv1.V1TaskInitialStateQUEUED)
		taskParams.Concurrencyparentstrategyids[i] = []pgtype.Int8{{}}
		taskParams.ConcurrencyStrategyIds[i] = []int64{s.strategy.ID}
		taskParams.ConcurrencyKeys[i] = []string{"test-key"}
		taskParams.WorkflowVersionIds[i] = s.workflowVersionId
		taskParams.WorkflowRunIds[i] = uuid.New()
	}

	tasks, err := queries.CreateTasks(ctx, conf.Pool, taskParams)
	require.NoError(t, err)
	require.Len(t, tasks, numTasks)
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

	s := setupStepConcurrencyStrategy(t, ctx, conf, name, strategyType, maxRuns)
	createConcurrencyTasks(t, ctx, conf, s, numTasks)
	return s
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

// TestConcurrency_CancelInProgress_InMemory exercises the new outbox-backed in-memory index for
// CANCEL_IN_PROGRESS: it keeps the newest maxRuns slots (highest priority, then latest) running and
// cancels the older ones with CONCURRENCY_LIMIT. With equal-priority tasks this comes down to the
// newest maxRuns running and the older remainder cancelled. (Recency direction is pinned by the unit
// tests in cancel_in_progress_test.go; this asserts the end-to-end counts.)
func TestConcurrency_CancelInProgress_InMemory(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		// create the strategy first, with no tasks yet
		s := setupStepConcurrencyStrategy(t, ctx, conf, "cip-inmem-test", "CANCEL_IN_PROGRESS", 2)

		l := zerolog.Nop()
		outbox := newTestOutbox(t, conf)
		cs := concurrency.NewConcurrencyStrategy(ctx, s.concurrencyRepo, s.strategy, outbox, &l)

		// The first Run blocks until the async index build finishes (against the currently-empty slot
		// table) and drains any pending WAL. Running it before inserting tasks guarantees the build
		// completes first, so the subsequent INSERT WAL messages are processed fresh rather than
		// double-counted against an index that already hydrated them.
		_, err := cs.Run(ctx)
		require.NoError(t, err)

		// now insert 5 tasks sharing one concurrency key; the insert trigger emits INSERT WAL.
		createConcurrencyTasks(t, ctx, conf, s, 5)

		// drain the WAL through the in-memory CANCEL_IN_PROGRESS decide step.
		res, err := cs.Run(ctx)
		require.NoError(t, err)
		require.Len(t, res.Queued, 2, "CANCEL_IN_PROGRESS in-memory should queue maxRuns tasks")
		require.Len(t, res.Cancelled, 3, "CANCEL_IN_PROGRESS in-memory should cancel the excess tasks")
		for _, c := range res.Cancelled {
			require.Equal(t, repo.CancelledReasonConcurrencyLimit, c.CancelledReason,
				"excess tasks should be cancelled with CONCURRENCY_LIMIT")
		}

		return nil
	})
}

// TestConcurrency_CancelNewest_InMemory exercises the new outbox-backed in-memory index for
// CANCEL_NEWEST: it fills maxRuns slots from the queued backlog (priority, then inserted_at, then
// taskId) and cancels the rest with CONCURRENCY_LIMIT, never preempting running work. With
// equal-priority tasks this comes down to maxRuns running and the remainder cancelled.
func TestConcurrency_CancelNewest_InMemory(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		// create the strategy first, with no tasks yet
		s := setupStepConcurrencyStrategy(t, ctx, conf, "cn-inmem-test", "CANCEL_NEWEST", 2)

		l := zerolog.Nop()
		outbox := newTestOutbox(t, conf)
		cs := concurrency.NewConcurrencyStrategy(ctx, s.concurrencyRepo, s.strategy, outbox, &l)

		// The first Run blocks until the async index build finishes (against the currently-empty slot
		// table) and drains any pending WAL. Running it before inserting tasks guarantees the build
		// completes first, so the subsequent INSERT WAL messages are processed fresh rather than
		// double-counted against an index that already hydrated them.
		_, err := cs.Run(ctx)
		require.NoError(t, err)

		// now insert 5 tasks sharing one concurrency key; the insert trigger emits INSERT WAL.
		createConcurrencyTasks(t, ctx, conf, s, 5)

		// drain the WAL through the in-memory CANCEL_NEWEST decide step.
		res, err := cs.Run(ctx)
		require.NoError(t, err)
		require.Len(t, res.Queued, 2, "CANCEL_NEWEST in-memory should queue maxRuns tasks")
		require.Len(t, res.Cancelled, 3, "CANCEL_NEWEST in-memory should cancel the excess tasks")
		for _, c := range res.Cancelled {
			require.Equal(t, repo.CancelledReasonConcurrencyLimit, c.CancelledReason,
				"excess tasks should be cancelled with CONCURRENCY_LIMIT")
		}

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

func TestConcurrency_MultipleStrategiesContention(t *testing.T) {
	// tests that multiple concurrency strategy ids that share a common parent id
	// will not fail advisory lock
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		requireSchedulerSchema(t, ctx, conf)

		r := conf.V1
		queries := sqlcv1.New()

		tenantId := uuid.New()
		tenant, err := r.Tenant().CreateTenant(ctx, &repo.CreateTenantOpts{
			ID:   &tenantId,
			Name: "concurrency-contention-test",
			Slug: fmt.Sprintf("concurrency-contention-test-%s", tenantId.String()),
		})
		require.NoError(t, err)

		l := zerolog.Nop()
		// the outbox table and v1_concurrency_slot triggers are provided by migrations; mirror
		// production by disabling pgoutbox's auto-migration.
		outbox, err := pgoutbox.NewOutbox(t.Context(), conf.Pool, pgoutbox.WithAutoMigrate(false))
		require.NoError(t, err)
		schedulingPool, cleanup, err := v1.NewSchedulingPool(
			r.Scheduler(),
			r.Tasks(),
			outbox,
			&l,
			100,
			20,
			5*time.Millisecond,
			6*time.Millisecond,
			50*time.Millisecond,
			100*time.Millisecond,
			5*time.Millisecond,
			false,
			1,
			true,
			nil,
		)
		require.NoError(t, err)
		defer func() { _ = cleanup() }()
		schedulingPool.SetTenants([]*sqlcv1.Tenant{tenant})
		resultsChan := schedulingPool.GetConcurrencyResultsCh()

		desc := "test workflow for concurrency contention"
		cancelNewest := "CANCEL_NEWEST"
		groupRR := "GROUP_ROUND_ROBIN"
		var maxRunsGate1 int32 = 3
		var maxRunsGate2 int32 = 1

		_, err = r.Workflows().PutWorkflowVersion(ctx, tenantId, &repo.CreateWorkflowVersionOpts{
			Name:        "concurrency-contention-test",
			Description: &desc,
			Tasks: []repo.CreateStepOpts{
				{
					ReadableId: "my-task",
					Action:     "test:run",
				},
				{
					ReadableId: "my-task-2",
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

		strategies, err := queries.ListActiveConcurrencyStrategies(ctx, conf.Pool, tenantId)
		require.NoError(t, err)
		for _, strat := range strategies {
			schedulingPool.NotifyNewConcurrencyStrategy(ctx, tenant.ID, strat.ID)
		}

		for i := 0; i < 10; i++ {
			select {
			case <-ctx.Done():
				t.Fatalf("context cancelled while waiting for concurrency result: %v", ctx.Err())
			case res := <-resultsChan:
				require.False(t, res.RunConcurrencyResult.FailedAdvisoryLock)
			}

		}
		return nil
	})
}

// TestConcurrency_ColdStrategyScheduledPromptly is a regression test for cold-start scheduling
// latency on concurrency-keyed tasks.
//
// A concurrency strategy that has been idle long enough to be deactivated (is_active=FALSE, no
// slots) has no ConcurrencyManager running in the scheduler. When the next task arrives, the
// scheduler is notified via NotifyConcurrency (the in-process effect of the CheckTenantQueue /
// NotifyTaskCreated message published on task creation). The scheduler must create a manager for
// that strategy and schedule the waiting task.
//
// Before the fix, NotifyConcurrency only woke *already-running* managers, so a cold strategy was
// not picked up until the periodic lease-acquisition poll (every 5s) happened to discover it -- the
// "first run is slow, then warm" symptom (observed as intermittent 5-11s queued->started spikes).
// After the fix, NotifyConcurrency acquires the lease on-demand and the new manager runs immediately.
//
// This test reproduces the cold path through the full SchedulingPool and asserts the task is queued
// well within the 5s lease-poll window. Run against the pre-fix commit and it waits ~5s and fails the
// deadline; against the fix it completes in milliseconds.
func TestConcurrency_ColdStrategyScheduledPromptly(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		r := conf.V1
		queries := sqlcv1.New()

		tenantId := uuid.New()
		tenant, err := r.Tenant().CreateTenant(ctx, &repo.CreateTenantOpts{
			ID:   &tenantId,
			Name: "concurrency-cold-start-test",
			Slug: fmt.Sprintf("concurrency-cold-start-test-%s", tenantId.String()),
		})
		require.NoError(t, err)

		desc := "test workflow for cold-start scheduling"
		groupRR := "GROUP_ROUND_ROBIN"
		var maxRuns int32 = 10

		wfVersion, err := r.Workflows().PutWorkflowVersion(ctx, tenantId, &repo.CreateWorkflowVersionOpts{
			Name:        "concurrency-cold-start-test",
			Description: &desc,
			Tasks: []repo.CreateStepOpts{
				{
					ReadableId: "my-task",
					Action:     "test:run",
					Concurrency: []repo.CreateConcurrencyOpts{
						{
							MaxRuns:       &maxRuns,
							LimitStrategy: &groupRR,
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

		concurrencyRepo := r.Scheduler().Concurrency()

		// The stale-deactivation sweep only considers strategies whose last_active_at is over 25
		// hours old (last_active_at is otherwise refreshed at most once per hour on slot inserts).
		// The strategy was just created with last_active_at=NOW(), so backdate it to simulate a
		// strategy that has genuinely been idle for the deactivation window.
		_, err = conf.Pool.Exec(ctx, `UPDATE v1_step_concurrency SET last_active_at = NOW() - INTERVAL '26 hours' WHERE id = $1`, strat.ID)
		require.NoError(t, err)

		// Make the strategy "cold": with no slots yet, the stale-deactivation sweep flips it to
		// is_active=FALSE. This mirrors a strategy that has been idle for the deactivation window.
		require.NoError(t, concurrencyRepo.DeactivateStaleStepConcurrency(ctx, tenantId))

		activeAfterDeactivate, err := queries.ListActiveConcurrencyStrategies(ctx, conf.Pool, tenantId)
		require.NoError(t, err)
		require.Len(t, activeAfterDeactivate, 0, "strategy must be inactive so no manager is leased at pool start")

		// Start the scheduling pool. Its initial lease acquisition runs now and finds no active
		// strategy, so no ConcurrencyManager exists for our strategy -- it is genuinely cold.
		l := zerolog.Nop()
		outbox, err := pgoutbox.NewOutbox(ctx, conf.Pool, pgoutbox.WithAutoMigrate(false))
		require.NoError(t, err)
		schedulingPool, cleanup, err := v1.NewSchedulingPool(
			r.Scheduler(),
			r.Tasks(),
			outbox,
			&l,
			100,
			20,
			10*time.Millisecond,
			20*time.Millisecond,
			50*time.Millisecond,
			100*time.Millisecond,
			5*time.Millisecond,
			false,
			1,
			true,
			nil,
		)
		require.NoError(t, err)
		defer func() { _ = cleanup() }()

		schedulingPool.SetTenants([]*sqlcv1.Tenant{tenant})
		resultsChan := schedulingPool.GetConcurrencyResultsCh()

		// A task arrives for the cold strategy. Creating the task inserts a concurrency slot, whose
		// insert trigger reactivates the strategy in the DB (is_active=TRUE) -- but the scheduler does
		// not know about it yet.
		taskParams := newCreateTasksParams(1)
		taskParams.Tenantids[0] = tenantId
		taskParams.Queues[0] = "default"
		taskParams.Actionids[0] = "test:run"
		taskParams.Stepids[0] = strat.StepID
		taskParams.Stepreadableids[0] = "my-task"
		taskParams.Workflowids[0] = workflowId
		taskParams.Scheduletimeouts[0] = "5m"
		taskParams.Priorities[0] = 1
		taskParams.Stickies[0] = string(sqlcv1.V1StickyStrategyNONE)
		taskParams.Externalids[0] = uuid.New()
		taskParams.Displaynames[0] = "cold-task"
		taskParams.Additionalmetadatas[0] = []byte(`{}`)
		taskParams.InitialStates[0] = string(sqlcv1.V1TaskInitialStateQUEUED)
		taskParams.Concurrencyparentstrategyids[0] = []pgtype.Int8{{}}
		taskParams.ConcurrencyStrategyIds[0] = []int64{strat.ID}
		taskParams.ConcurrencyKeys[0] = []string{"thread-1"}
		taskParams.WorkflowVersionIds[0] = workflowVersionId
		taskParams.WorkflowRunIds[0] = uuid.New()

		tasks, err := queries.CreateTasks(ctx, conf.Pool, taskParams)
		require.NoError(t, err)
		require.Len(t, tasks, 1)

		// This is exactly what the task-creation message does in production.
		start := time.Now()
		schedulingPool.NotifyConcurrency(ctx, tenantId, []int64{strat.ID})

		// The fix must schedule the waiting task without waiting for the 5s lease-acquisition poll.
		const coldStartDeadline = 2 * time.Second
		deadline := time.NewTimer(coldStartDeadline)
		defer deadline.Stop()

		for {
			select {
			case <-ctx.Done():
				t.Fatalf("context cancelled while waiting for cold strategy to be scheduled: %v", ctx.Err())
			case <-deadline.C:
				t.Fatalf("cold concurrency strategy was not scheduled within %s; "+
					"the on-demand manager was not created (fell back to the periodic lease poll)", coldStartDeadline)
			case res := <-resultsChan:
				require.False(t, res.RunConcurrencyResult.FailedAdvisoryLock)
				if len(res.RunConcurrencyResult.Queued) > 0 {
					require.Len(t, res.RunConcurrencyResult.Queued, 1, "the single waiting task should be queued")
					t.Logf("cold strategy scheduled in %s", time.Since(start))
					return nil
				}
			}
		}
	})
}
