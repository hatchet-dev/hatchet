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
// strategy and the given static rate limit keys, but inserts no tasks. Use createConcurrencyTasks to add tasks afterwards.
func setupStepConcurrencyStrategy(
	t *testing.T,
	ctx context.Context,
	conf *database.Layer,
	name string,
	strategyType string,
	maxRuns int32,
	rateLimitKeys ...string,
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
				RateLimits: stepRateLimits(t, ctx, conf, tenantId, rateLimitKeys),
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
		taskParams.IsDagOrchestrators[i] = false
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
	rateLimitKeys ...string,
) *concurrencyTestSetup {
	t.Helper()

	s := setupStepConcurrencyStrategy(t, ctx, conf, name, strategyType, maxRuns, rateLimitKeys...)
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
		IsDagOrchestrators:           make([]bool, n),
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
			taskParams.IsDagOrchestrators[i] = false
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
//
// SetTenants starts the tenant's lease manager, whose initial lease acquisition runs asynchronously
// and holds the concurrency lease mutex for several DB round trips. The on-demand path only
// try-locks that mutex and silently gives up if it is held, so a NotifyConcurrency that races the
// initial acquisition is dropped and the strategy is not picked up until the periodic poll. The
// test therefore waits for the initial acquisition to finish before creating the task: a second,
// warm strategy on the same tenant is leased by that initial acquisition, and its manager's first
// result on the shared results channel proves the acquisition has handed over its leases. Only then
// is the cold task created and notified, so the on-demand path is the only thing that can schedule
// it before the periodic poll.
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

		// A second, warm strategy on the same tenant. It stays active, so the pool's initial lease
		// acquisition leases it and starts a manager for it; that manager's results are the signal
		// that the initial acquisition has completed (see below). It never receives a task.
		warmDesc := "warm workflow for cold-start scheduling"
		_, err = r.Workflows().PutWorkflowVersion(ctx, tenantId, &repo.CreateWorkflowVersionOpts{
			Name:        "concurrency-cold-start-warm",
			Description: &warmDesc,
			Tasks: []repo.CreateStepOpts{
				{
					ReadableId: "warm-task",
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
		require.Len(t, activeAfterDeactivate, 1, "only the warm strategy must be active so no manager is leased for the cold one at pool start")
		require.NotEqual(t, strat.ID, activeAfterDeactivate[0].ID, "the cold strategy must be inactive")

		// Start the scheduling pool. Its initial lease acquisition runs now and finds only the warm
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

		// Wait for the initial lease acquisition to complete before the cold task arrives. The only
		// manager that can exist at this point is the warm strategy's, and it is created once the
		// initial acquisition hands over its leases (it releases the concurrency lease mutex right
		// after), so its first result proves the on-demand path below cannot collide with the
		// initial acquisition and that the cold strategy was not leased by it. The manager runs once
		// on creation, so this arrives after a single strategy run rather than a polling interval.
		select {
		case <-ctx.Done():
			t.Fatalf("context cancelled while waiting for the initial lease acquisition: %v", ctx.Err())
		case res := <-resultsChan:
			require.False(t, res.RunConcurrencyResult.FailedAdvisoryLock)
			require.Empty(t, res.RunConcurrencyResult.Queued, "the warm strategy has no tasks")
		}

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
		taskParams.IsDagOrchestrators[0] = false

		tasks, err := queries.CreateTasks(ctx, conf.Pool, taskParams)
		require.NoError(t, err)
		require.Len(t, tasks, 1)

		// This is exactly what the task-creation message does in production.
		start := time.Now()
		schedulingPool.NotifyConcurrency(ctx, tenantId, []int64{strat.ID})

		// The fix must schedule the waiting task without waiting for the lease-acquisition poll,
		// which LeaseManager.loopForLeases runs every 5s from pool start. The notify above happens
		// well under a second after pool start (right after the initial acquisition), so the poll
		// cannot rescue the task before roughly 4s from now; 3s stays clearly below that while
		// leaving headroom for the on-demand path's DB round trips on a loaded CI database (the
		// path completes in about 20ms locally and stayed under 1.2s with every core saturated).
		const coldStartDeadline = 3 * time.Second
		deadline := time.NewTimer(coldStartDeadline)
		defer deadline.Stop()

		// The warm manager's first result proves the initial acquisition handed over its leases,
		// but the acquisition releases the lease mutex only after that send, so a notification
		// landing in that gap still finds the mutex held and is dropped. Notifying again while
		// waiting closes the gap without changing what is measured: every notification takes the
		// same on-demand path, and none of them can be answered by the periodic poll before the
		// deadline.
		renotify := time.NewTicker(500 * time.Millisecond)
		defer renotify.Stop()

		for {
			select {
			case <-ctx.Done():
				t.Fatalf("context cancelled while waiting for cold strategy to be scheduled: %v", ctx.Err())
			case <-renotify.C:
				schedulingPool.NotifyConcurrency(ctx, tenantId, []int64{strat.ID})
			case <-deadline.C:
				t.Fatalf("cold concurrency strategy was not scheduled within %s; "+
					"the on-demand manager was not created (fell back to the periodic lease poll)", coldStartDeadline)
			case res := <-resultsChan:
				// the warm manager keeps polling and emits empty results; skip those
				require.False(t, res.RunConcurrencyResult.FailedAdvisoryLock)
				if len(res.RunConcurrencyResult.Queued) > 0 {
					require.Len(t, res.RunConcurrencyResult.Queued, 1, "the single waiting task should be queued")
					require.Equal(t, tasks[0].ID, res.RunConcurrencyResult.Queued[0].Id, "the cold task should be the one queued")
					t.Logf("cold strategy scheduled in %s", time.Since(start))
					return nil
				}
			}
		}
	})
}

// TestConcurrency_RateLimitedFilledSlotTimesOutAndUnblocks is the filled-slot
// half of the deadlock: GROUP_ROUND_ROBIN max=1 fills task A, A is then parked
// in v1_rate_limited_queue_items (slot stays is_filled), and task B waits.
// Once A's schedule_timeout_at has passed, A is not running, so the strategy
// must cancel A and let B fill.
func TestConcurrency_RateLimitedFilledSlotTimesOutAndUnblocks(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		s := setupStepConcurrencyTest(t, ctx, conf, "rl-filled-slot", "GROUP_ROUND_ROBIN", 1, 2)

		res, err := s.concurrencyRepo.RunConcurrencyStrategy(ctx, s.tenantId, s.strategy)
		require.NoError(t, err)
		require.Len(t, res.Queued, 1, "maxRuns=1 should fill exactly one slot")
		require.Len(t, res.Cancelled, 0)

		var filledCount int
		err = conf.Pool.QueryRow(ctx, `
			SELECT count(*) FROM v1_concurrency_slot
			WHERE workflow_id = $1 AND is_filled
		`, s.workflowId).Scan(&filledCount)
		require.NoError(t, err)
		require.Equal(t, 1, filledCount)

		tag, err := conf.Pool.Exec(ctx, `
			WITH moved AS (
				DELETE FROM v1_queue_item
				WHERE tenant_id = $1
				RETURNING *
			)
			INSERT INTO v1_rate_limited_queue_items (
				requeue_after, tenant_id, queue, task_id, task_inserted_at,
				external_id, action_id, step_id, workflow_id, workflow_run_id,
				schedule_timeout_at, step_timeout, priority, sticky,
				desired_worker_id, retry_count, desired_worker_label, batch_key
			)
			SELECT
				NOW() - INTERVAL '1 minute',
				tenant_id, queue, task_id, task_inserted_at,
				external_id, action_id, step_id, workflow_id, workflow_run_id,
				schedule_timeout_at, step_timeout, priority, sticky,
				desired_worker_id, retry_count, desired_worker_label, batch_key
			FROM moved
		`, s.tenantId)
		require.NoError(t, err)
		require.EqualValues(t, 1, tag.RowsAffected(), "filled task must have had a queue item to park")

		_, err = conf.Pool.Exec(ctx, `
			UPDATE v1_concurrency_slot
			SET schedule_timeout_at = NOW() - INTERVAL '1 minute'
			WHERE workflow_id = $1 AND is_filled
		`, s.workflowId)
		require.NoError(t, err)

		res2, err := s.concurrencyRepo.RunConcurrencyStrategy(ctx, s.tenantId, s.strategy)
		require.NoError(t, err)
		require.Len(t, res2.Cancelled, 1, "filled rate-limited slot past schedule timeout must be cancelled")
		require.Equal(t, repo.CancelledReasonSchedulingTimedOut, res2.Cancelled[0].CancelledReason)

		queued := res2.Queued
		if len(queued) == 0 {
			res3, err := s.concurrencyRepo.RunConcurrencyStrategy(ctx, s.tenantId, s.strategy)
			require.NoError(t, err)
			queued = res3.Queued
		}
		require.Len(t, queued, 1, "waiting task must fill the slot after the rate-limited holder times out")

		return nil
	})
}

// listConcurrencyTestTasks returns the tenant's tasks oldest first.
func listConcurrencyTestTasks(t *testing.T, ctx context.Context, conf *database.Layer, tenantId uuid.UUID) []*sqlcv1.V1Task {
	t.Helper()

	rows, err := conf.Pool.Query(ctx, `SELECT id, inserted_at, retry_count FROM v1_task WHERE tenant_id = $1 ORDER BY id`, tenantId)
	require.NoError(t, err)
	defer rows.Close()

	tasks := make([]*sqlcv1.V1Task, 0)

	for rows.Next() {
		task := &sqlcv1.V1Task{TenantID: tenantId}
		require.NoError(t, rows.Scan(&task.ID, &task.InsertedAt, &task.RetryCount))
		tasks = append(tasks, task)
	}

	require.NoError(t, rows.Err())

	return tasks
}

// stepRateLimits creates the keys' rate limits and has a step consume one unit of each.
func stepRateLimits(t *testing.T, ctx context.Context, conf *database.Layer, tenantId uuid.UUID, keys []string) []repo.CreateWorkflowStepRateLimitOpts {
	t.Helper()

	opts := make([]repo.CreateWorkflowStepRateLimitOpts, 0, len(keys))
	units := 1

	for _, key := range keys {
		_, err := conf.V1.RateLimit().UpsertRateLimit(ctx, tenantId, key, &repo.UpsertRateLimitOpts{Limit: 10})
		require.NoError(t, err)

		opts = append(opts, repo.CreateWorkflowStepRateLimitOpts{Key: key, Units: &units})
	}

	return opts
}

// pauseRateLimitKey does what the admin service and the task controller do for a pause request.
func pauseRateLimitKey(t *testing.T, ctx context.Context, conf *database.Layer, tenantId uuid.UUID, key string, until time.Time) *sqlcv1.V1RateLimitPause {
	t.Helper()

	pause, err := conf.V1.RateLimit().PauseRateLimit(ctx, tenantId, key, until)
	require.NoError(t, err)
	require.NoError(t, conf.V1.RateLimit().HoldRateLimitPausedQueuedTasks(ctx, tenantId))

	return pause
}

func waitForPauseToEnd(pause *sqlcv1.V1RateLimitPause) {
	time.Sleep(time.Until(pause.PausedUntil.Time) + 100*time.Millisecond)
}

func countRateLimitHeldItems(t *testing.T, ctx context.Context, conf *database.Layer, task *sqlcv1.V1Task) int {
	t.Helper()

	var count int
	err := conf.Pool.QueryRow(ctx, `
		SELECT count(*) FROM v1_paused_workflow_queue_item
		WHERE task_id = $1 AND task_inserted_at = $2 AND rate_limit_paused_until IS NOT NULL
	`, task.ID, task.InsertedAt).Scan(&count)
	require.NoError(t, err)

	return count
}

func countTaskConcurrencySlots(t *testing.T, ctx context.Context, conf *database.Layer, task *sqlcv1.V1Task) int {
	t.Helper()

	var count int
	err := conf.Pool.QueryRow(ctx, `
		SELECT count(*) FROM v1_concurrency_slot WHERE task_id = $1 AND task_inserted_at = $2
	`, task.ID, task.InsertedAt).Scan(&count)
	require.NoError(t, err)

	return count
}

// TestConcurrency_PausedRateLimitReleasesFilledSlot: GROUP_ROUND_ROBIN max=1 fills task A, whose rate limit
// key is then found paused by the scheduler. A must give up its slot so task B, which shares the concurrency
// key but not the rate limit key, can run, and A must get a slot again once the pause ends.
func TestConcurrency_PausedRateLimitReleasesFilledSlot(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		s := setupSharedConcurrencyTest(t, ctx, conf, "rl-paused-slot", "GROUP_ROUND_ROBIN", 1, 0)
		def := sharedLimitDef("GROUP_ROUND_ROBIN", 1)

		wfA := createWorkflowWithConcurrency(t, ctx, conf, s.tenantId, "rl-paused-slot-a", []repo.CreateConcurrencyOpts{def}, "upstream-api")
		wfB := createWorkflowWithConcurrency(t, ctx, conf, s.tenantId, "rl-paused-slot-b", []repo.CreateConcurrencyOpts{def})

		createSharedConcurrencyTasks(t, ctx, conf, s, wfA, 1)
		createSharedConcurrencyTasks(t, ctx, conf, s, wfB, 1)

		tasks := listConcurrencyTestTasks(t, ctx, conf, s.tenantId)
		require.Len(t, tasks, 2)

		taskA, taskB := tasks[0], tasks[1]
		strategy := s.strategyDescriptor()

		res, err := s.repo.RunConcurrencyStrategy(ctx, s.tenantId, strategy)
		require.NoError(t, err)
		require.Len(t, res.Queued, 1)
		require.Equal(t, taskA.ID, res.Queued[0].Id, "the oldest task fills the only slot")

		// not held here, so only the scheduler's check can hold A
		pause, err := conf.V1.RateLimit().PauseRateLimit(ctx, s.tenantId, "upstream-api", time.Now().Add(2*time.Second))
		require.NoError(t, err)

		qi := &sqlcv1.V1QueueItem{TenantID: s.tenantId}
		err = conf.Pool.QueryRow(ctx, `
			SELECT id, task_id, task_inserted_at, retry_count FROM v1_queue_item WHERE task_id = $1
		`, taskA.ID).Scan(&qi.ID, &qi.TaskID, &qi.TaskInsertedAt, &qi.RetryCount)
		require.NoError(t, err)

		queueRepo := conf.V1.Scheduler().QueueFactory().NewQueue(s.tenantId, "default")
		defer queueRepo.Cleanup()

		pausedUntil := pause.PausedUntil.Time

		_, _, err = queueRepo.MarkQueueItemsProcessed(ctx, &repo.AssignResults{
			RateLimitedToMove: []*repo.RateLimitResult{{
				V1QueueItem:    qi,
				ExceededKey:    "upstream-api",
				NextRefillAt:   &pausedUntil,
				PausedUntil:    &pausedUntil,
				TaskId:         qi.TaskID,
				TaskInsertedAt: qi.TaskInsertedAt,
				RetryCount:     qi.RetryCount,
			}},
		})
		require.NoError(t, err)

		require.Equal(t, 1, countRateLimitHeldItems(t, ctx, conf, taskA))
		require.Zero(t, countTaskConcurrencySlots(t, ctx, conf, taskA), "a held task must not keep its slot")

		res, err = s.repo.RunConcurrencyStrategy(ctx, s.tenantId, strategy)
		require.NoError(t, err)
		require.Len(t, res.Queued, 1)
		require.Equal(t, taskB.ID, res.Queued[0].Id, "the slot A gave up goes to B")

		waitForPauseToEnd(pause)

		_, strategyIds, err := conf.V1.Workflows().RequeueRateLimitPausedQueueItems(ctx, s.tenantId)
		require.NoError(t, err)
		require.Equal(t, []int64{s.strategy.ID}, strategyIds)

		require.Zero(t, countRateLimitHeldItems(t, ctx, conf, taskA))
		require.Equal(t, 1, countTaskConcurrencySlots(t, ctx, conf, taskA), "A waits for a slot again once the pause ends")

		return nil
	})
}

// TestConcurrency_PauseRateLimitHoldsWaitingSlots: pausing a key holds the tasks already waiting on it before
// the strategy can fill their slots, and a pause extended after they were held keeps holding them.
func TestConcurrency_PauseRateLimitHoldsWaitingSlots(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		s := setupStepConcurrencyTest(t, ctx, conf, "rl-paused-waiting", "GROUP_ROUND_ROBIN", 1, 2, "upstream-api")
		tasks := listConcurrencyTestTasks(t, ctx, conf, s.tenantId)
		require.Len(t, tasks, 2)

		pause := pauseRateLimitKey(t, ctx, conf, s.tenantId, "upstream-api", time.Now().Add(time.Second))

		require.Equal(t, 1, countRateLimitHeldItems(t, ctx, conf, tasks[0]))
		require.Equal(t, 1, countRateLimitHeldItems(t, ctx, conf, tasks[1]))
		require.Zero(t, countTaskConcurrencySlots(t, ctx, conf, tasks[0]))

		res, err := s.concurrencyRepo.RunConcurrencyStrategy(ctx, s.tenantId, s.strategy)
		require.NoError(t, err)
		require.Empty(t, res.Queued, "no task behind the paused key may fill the slot")

		// extended without holding again, so the held items come due while the pause is still active
		_, err = conf.V1.RateLimit().PauseRateLimit(ctx, s.tenantId, "upstream-api", time.Now().Add(time.Hour))
		require.NoError(t, err)

		waitForPauseToEnd(pause)

		queueNames, strategyIds, err := conf.V1.Workflows().RequeueRateLimitPausedQueueItems(ctx, s.tenantId)
		require.NoError(t, err)
		require.Empty(t, queueNames)
		require.Empty(t, strategyIds)

		var stillHeld int
		err = conf.Pool.QueryRow(ctx, `
			SELECT count(*) FROM v1_paused_workflow_queue_item WHERE tenant_id = $1 AND rate_limit_paused_until > NOW()
		`, s.tenantId).Scan(&stillHeld)
		require.NoError(t, err)
		require.Equal(t, 2, stillHeld, "held items move to the extended deadline")

		return nil
	})
}

// TestConcurrency_PausedRateLimitHoldsNewRuns: a run triggered while its static rate limit key is paused is
// held when it's created, so it never gets a concurrency slot to fill.
func TestConcurrency_PausedRateLimitHoldsNewRuns(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		name := "rl-paused-new-run"
		s := setupStepConcurrencyStrategy(t, ctx, conf, name, "GROUP_ROUND_ROBIN", 1, "upstream-api")

		pauseRateLimitKey(t, ctx, conf, s.tenantId, "upstream-api", time.Now().Add(time.Hour))

		tasks, _, _, _, err := conf.V1.Triggers().TriggerFromWorkflowNames(ctx, s.tenantId, []*repo.WorkflowNameTriggerOpts{
			{
				TriggerTaskData: &repo.TriggerTaskData{
					WorkflowName: name,
					Data:         []byte(`{"my_id": "a"}`),
				},
				ExternalId: uuid.New(),
			},
		})
		require.NoError(t, err)
		require.Len(t, tasks, 1)

		task := tasks[0].V1Task

		require.Equal(t, 1, countRateLimitHeldItems(t, ctx, conf, task))
		require.Zero(t, countTaskConcurrencySlots(t, ctx, conf, task), "a run behind a paused key must not get a slot")

		return nil
	})
}

// TestConcurrency_PauseDynamicRateLimitKeyHoldsMatchingRuns: a dynamic key is only known once a run renders it, so
// pausing one rendered key holds the runs that rendered it and leaves the step's runs with other keys waiting.
func TestConcurrency_PauseDynamicRateLimitKeyHoldsMatchingRuns(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		name := "rl-paused-dynamic"
		tenantId := uuid.New()
		_, err := conf.V1.Tenant().CreateTenant(ctx, &repo.CreateTenantOpts{
			ID:   &tenantId,
			Name: name,
			Slug: fmt.Sprintf("%s-%s", name, tenantId.String()),
		})
		require.NoError(t, err)

		desc := "test workflow"
		maxRuns := int32(1)
		strategyType := "GROUP_ROUND_ROBIN"
		keyExpr := `"region-" + input.region`
		units := 1

		_, err = conf.V1.Workflows().PutWorkflowVersion(ctx, tenantId, &repo.CreateWorkflowVersionOpts{
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
					RateLimits: []repo.CreateWorkflowStepRateLimitOpts{{Key: "region", KeyExpr: &keyExpr, Units: &units}},
				},
			},
		})
		require.NoError(t, err)

		triggerRun := func(region string) *sqlcv1.V1Task {
			tasks, _, _, _, err := conf.V1.Triggers().TriggerFromWorkflowNames(ctx, tenantId, []*repo.WorkflowNameTriggerOpts{
				{
					TriggerTaskData: &repo.TriggerTaskData{
						WorkflowName: name,
						Data:         []byte(fmt.Sprintf(`{"my_id": "a", "region": %q}`, region)),
					},
					ExternalId: uuid.New(),
				},
			})
			require.NoError(t, err)
			require.Len(t, tasks, 1)

			return tasks[0].V1Task
		}

		us := triggerRun("us")
		eu := triggerRun("eu")

		pauseRateLimitKey(t, ctx, conf, tenantId, "region-us", time.Now().Add(time.Hour))

		require.Equal(t, 1, countRateLimitHeldItems(t, ctx, conf, us))
		require.Zero(t, countTaskConcurrencySlots(t, ctx, conf, us), "a run that rendered the paused key must give up its slot")

		require.Zero(t, countRateLimitHeldItems(t, ctx, conf, eu))
		require.Equal(t, 1, countTaskConcurrencySlots(t, ctx, conf, eu), "a run that rendered another key keeps waiting")

		return nil
	})
}

func pauseConcurrencyTestWorkflow(t *testing.T, ctx context.Context, conf *database.Layer, s *concurrencyTestSetup) {
	t.Helper()

	_, err := conf.V1.Workflows().PauseWorkflow(ctx, s.workflowId, repo.PauseWorkflowOpts{
		CronRunQueueBehavior:      "QUEUE",
		ScheduledRunQueueBehavior: "QUEUE",
		QueueTTL:                  "1h",
	})
	require.NoError(t, err)

	require.NoError(t, conf.V1.Workflows().MovePausedWorkflowQueueItems(ctx, s.tenantId, []uuid.UUID{s.workflowId}))
}

// unpauseConcurrencyTestWorkflow does what the task controller does when it receives the unpause message.
func unpauseConcurrencyTestWorkflow(t *testing.T, ctx context.Context, conf *database.Layer, s *concurrencyTestSetup) {
	t.Helper()

	_, err := conf.V1.Workflows().UnpauseWorkflow(ctx, s.workflowId)
	require.NoError(t, err)

	_, _, err = conf.V1.Workflows().RequeuePausedWorkflowQueueItems(ctx, s.tenantId, []uuid.UUID{s.workflowId})
	require.NoError(t, err)
}

func countPausedQueueItems(t *testing.T, ctx context.Context, conf *database.Layer, task *sqlcv1.V1Task) int {
	t.Helper()

	var count int
	err := conf.Pool.QueryRow(ctx, `
		SELECT count(*) FROM v1_paused_workflow_queue_item WHERE task_id = $1 AND task_inserted_at = $2
	`, task.ID, task.InsertedAt).Scan(&count)
	require.NoError(t, err)

	return count
}

// TestConcurrency_WorkflowUnpausedWhileRateLimitPaused: a task held by both a paused workflow and a paused rate
// limit key stays held when the workflow unpauses, and is released once the key's pause ends.
func TestConcurrency_WorkflowUnpausedWhileRateLimitPaused(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		s := setupStepConcurrencyTest(t, ctx, conf, "rl-wf-unpause-first", "GROUP_ROUND_ROBIN", 1, 1, "upstream-api")
		task := listConcurrencyTestTasks(t, ctx, conf, s.tenantId)[0]

		pause := pauseRateLimitKey(t, ctx, conf, s.tenantId, "upstream-api", time.Now().Add(2*time.Second))

		pauseConcurrencyTestWorkflow(t, ctx, conf, s)
		require.Equal(t, 1, countRateLimitHeldItems(t, ctx, conf, task))

		unpauseConcurrencyTestWorkflow(t, ctx, conf, s)
		require.Equal(t, 1, countRateLimitHeldItems(t, ctx, conf, task), "the key is still paused")
		require.Zero(t, countTaskConcurrencySlots(t, ctx, conf, task))

		stranded, err := conf.V1.Workflows().ListUnpausedWorkflowsWithPausedQueueItems(ctx, s.tenantId)
		require.NoError(t, err)
		require.Empty(t, stranded, "an item held by a key isn't stranded by the workflow unpause")

		waitForPauseToEnd(pause)

		_, strategyIds, err := conf.V1.Workflows().RequeueRateLimitPausedQueueItems(ctx, s.tenantId)
		require.NoError(t, err)
		require.Equal(t, []int64{s.strategy.ID}, strategyIds)

		require.Zero(t, countPausedQueueItems(t, ctx, conf, task))
		require.Equal(t, 1, countTaskConcurrencySlots(t, ctx, conf, task))

		return nil
	})
}

// TestConcurrency_RateLimitPauseEndsWhileWorkflowPaused: when a key's pause ends while the task's workflow is
// still paused, the task stays held by the workflow pause and is released when the workflow unpauses.
func TestConcurrency_RateLimitPauseEndsWhileWorkflowPaused(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		s := setupStepConcurrencyTest(t, ctx, conf, "rl-ends-wf-paused", "GROUP_ROUND_ROBIN", 1, 1, "upstream-api")
		task := listConcurrencyTestTasks(t, ctx, conf, s.tenantId)[0]

		pause := pauseRateLimitKey(t, ctx, conf, s.tenantId, "upstream-api", time.Now().Add(time.Second))

		pauseConcurrencyTestWorkflow(t, ctx, conf, s)

		waitForPauseToEnd(pause)

		queueNames, strategyIds, err := conf.V1.Workflows().RequeueRateLimitPausedQueueItems(ctx, s.tenantId)
		require.NoError(t, err)
		require.Empty(t, queueNames)
		require.Empty(t, strategyIds)

		require.Equal(t, 1, countPausedQueueItems(t, ctx, conf, task), "the workflow is still paused")
		require.Zero(t, countRateLimitHeldItems(t, ctx, conf, task), "the item is handed to the workflow pause")
		require.Zero(t, countTaskConcurrencySlots(t, ctx, conf, task))

		unpauseConcurrencyTestWorkflow(t, ctx, conf, s)

		require.Zero(t, countPausedQueueItems(t, ctx, conf, task))
		require.Equal(t, 1, countTaskConcurrencySlots(t, ctx, conf, task))

		return nil
	})
}

// TestConcurrency_RateLimitPausedWhileWorkflowPaused: a task the workflow pause already holds when its key is
// paused must stay held by the key once the workflow unpauses, rather than getting a slot back.
func TestConcurrency_RateLimitPausedWhileWorkflowPaused(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		s := setupStepConcurrencyTest(t, ctx, conf, "rl-paused-wf-paused", "GROUP_ROUND_ROBIN", 1, 1, "upstream-api")
		task := listConcurrencyTestTasks(t, ctx, conf, s.tenantId)[0]

		pauseConcurrencyTestWorkflow(t, ctx, conf, s)
		require.Equal(t, 1, countPausedQueueItems(t, ctx, conf, task))
		require.Zero(t, countRateLimitHeldItems(t, ctx, conf, task))

		pause := pauseRateLimitKey(t, ctx, conf, s.tenantId, "upstream-api", time.Now().Add(2*time.Second))

		unpauseConcurrencyTestWorkflow(t, ctx, conf, s)
		require.Equal(t, 1, countRateLimitHeldItems(t, ctx, conf, task), "the key is still paused")
		require.Zero(t, countTaskConcurrencySlots(t, ctx, conf, task))

		waitForPauseToEnd(pause)

		_, _, err := conf.V1.Workflows().RequeueRateLimitPausedQueueItems(ctx, s.tenantId)
		require.NoError(t, err)

		require.Zero(t, countPausedQueueItems(t, ctx, conf, task))
		require.Equal(t, 1, countTaskConcurrencySlots(t, ctx, conf, task))

		return nil
	})
}

// TestConcurrency_TriggerHoldsQueuedRunsOfPausedWorkflow: triggering a paused workflow holds the new run and any of
// the workflow's runs that were queued before the pause was applied to them.
func TestConcurrency_TriggerHoldsQueuedRunsOfPausedWorkflow(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		name := "wf-paused-trigger"
		s := setupStepConcurrencyTest(t, ctx, conf, name, "GROUP_ROUND_ROBIN", 1, 1)
		queuedBeforePause := listConcurrencyTestTasks(t, ctx, conf, s.tenantId)[0]

		// paused without the toggle message, so the earlier run is still waiting on its slot
		_, err := conf.V1.Workflows().PauseWorkflow(ctx, s.workflowId, repo.PauseWorkflowOpts{
			CronRunQueueBehavior:      "QUEUE",
			ScheduledRunQueueBehavior: "QUEUE",
			QueueTTL:                  "1h",
		})
		require.NoError(t, err)

		tasks, _, _, _, err := conf.V1.Triggers().TriggerFromWorkflowNames(ctx, s.tenantId, []*repo.WorkflowNameTriggerOpts{
			{
				TriggerTaskData: &repo.TriggerTaskData{
					WorkflowName: name,
					Data:         []byte(`{"my_id": "a"}`),
				},
				ExternalId: uuid.New(),
			},
		})
		require.NoError(t, err)
		require.Len(t, tasks, 1)

		for _, task := range []*sqlcv1.V1Task{queuedBeforePause, tasks[0].V1Task} {
			require.Equal(t, 1, countPausedQueueItems(t, ctx, conf, task))
			require.Zero(t, countRateLimitHeldItems(t, ctx, conf, task), "only the workflow holds it")
			require.Zero(t, countTaskConcurrencySlots(t, ctx, conf, task))
		}

		return nil
	})
}

// TestConcurrency_ReleaseDueHeldItemsContinuesPastStillHeldBatch: a full batch of due items that all stay held, here
// because their key's pause was extended, must not stop the release before the ready items behind it.
func TestConcurrency_ReleaseDueHeldItemsContinuesPastStillHeldBatch(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		s := setupSharedConcurrencyTest(t, ctx, conf, "rl-release-batches", "GROUP_ROUND_ROBIN", 1, 0)
		def := sharedLimitDef("GROUP_ROUND_ROBIN", 1)

		extendedWf := createWorkflowWithConcurrency(t, ctx, conf, s.tenantId, "rl-release-batches-extended", []repo.CreateConcurrencyOpts{def}, "extended")
		endedWf := createWorkflowWithConcurrency(t, ctx, conf, s.tenantId, "rl-release-batches-ended", []repo.CreateConcurrencyOpts{def}, "ended")

		createSharedConcurrencyTasks(t, ctx, conf, s, extendedWf, 1000)
		createSharedConcurrencyTasks(t, ctx, conf, s, endedWf, 1)

		tasks := listConcurrencyTestTasks(t, ctx, conf, s.tenantId)
		require.Len(t, tasks, 1001)
		ready := tasks[len(tasks)-1]

		until := time.Now().Add(time.Second)
		extended := pauseRateLimitKey(t, ctx, conf, s.tenantId, "extended", until)
		ended := pauseRateLimitKey(t, ctx, conf, s.tenantId, "ended", until)
		require.Equal(t, 1, countRateLimitHeldItems(t, ctx, conf, ready))

		// extended without holding again, so its items come due while it is still paused
		_, err := conf.V1.RateLimit().PauseRateLimit(ctx, s.tenantId, "extended", time.Now().Add(time.Hour))
		require.NoError(t, err)

		waitForPauseToEnd(extended)
		waitForPauseToEnd(ended)

		_, _, err = conf.V1.Workflows().RequeueRateLimitPausedQueueItems(ctx, s.tenantId)
		require.NoError(t, err)

		require.Zero(t, countPausedQueueItems(t, ctx, conf, ready), "the ready item behind a full still-held batch must be released")
		require.Equal(t, 1, countRateLimitHeldItems(t, ctx, conf, tasks[0]), "the extended key still holds its items")

		return nil
	})
}
