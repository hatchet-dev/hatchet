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
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/config/database"
	repo "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/scheduling/v1/concurrency"
)

// sharedConcurrencyTestSetup holds a tenant with one tenant-scoped strategy and workflows
// whose steps reference it by name.
type sharedConcurrencyTestSetup struct {
	tenantId  uuid.UUID
	strategy  *sqlcv1.V1TenantConcurrency
	repo      repo.ConcurrencyRepository
	workflows []*workflowWithStep
}

type workflowWithStep struct {
	workflowId        uuid.UUID
	workflowVersionId uuid.UUID
	stepId            uuid.UUID
}

// strategyDescriptor builds the in-memory strategy descriptor for a tenant-scoped
// strategy, as the scheduler's lease listing would.
func (s *sharedConcurrencyTestSetup) strategyDescriptor() *sqlcv1.V1StepConcurrency {
	return repo.TenantConcurrencyDescriptor(s.strategy)
}

func createWorkflowWithConcurrency(
	t *testing.T,
	ctx context.Context,
	conf *database.Layer,
	tenantId uuid.UUID,
	name string,
	entries []repo.CreateConcurrencyOpts,
) *workflowWithStep {
	t.Helper()

	desc := "test workflow"
	wfVersion, err := conf.V1.Workflows().PutWorkflowVersion(ctx, tenantId, &repo.CreateWorkflowVersionOpts{
		Name:        name,
		Description: &desc,
		Tasks: []repo.CreateStepOpts{
			{
				ReadableId:  "my-task",
				Action:      "test:run",
				Concurrency: entries,
			},
		},
	})
	require.NoError(t, err)

	steps, err := conf.V1.Workflows().ListStepsByWorkflowVersionId(ctx, tenantId, wfVersion.WorkflowVersion.ID)
	require.NoError(t, err)
	require.Len(t, steps, 1)

	return &workflowWithStep{
		workflowId:        wfVersion.WorkflowVersion.WorkflowId,
		workflowVersionId: wfVersion.WorkflowVersion.ID,
		stepId:            steps[0].ID,
	}
}

// setupSharedConcurrencyTest creates a tenant, registers a tenant-scoped strategy, and
// creates numWorkflows workflows whose single step references it by name.
func setupSharedConcurrencyTest(
	t *testing.T,
	ctx context.Context,
	conf *database.Layer,
	name string,
	strategyType string,
	maxRuns int32,
	numWorkflows int,
) *sharedConcurrencyTestSetup {
	t.Helper()

	tenantId := uuid.New()
	_, err := conf.V1.Tenant().CreateTenant(ctx, &repo.CreateTenantOpts{
		ID:   &tenantId,
		Name: name,
		Slug: fmt.Sprintf("%s-%s", name, tenantId.String()),
	})
	require.NoError(t, err)

	// the definition is a tenant-scoped Concurrency entry riding on a workflow put (the
	// only registration path); every declaring task carries the full definition
	def := repo.CreateConcurrencyOpts{
		Name:           "shared-limit",
		IsTenantScoped: true,
		Expression:     "input.my_id",
		MaxRuns:        &maxRuns,
		LimitStrategy:  &strategyType,
	}

	createWorkflowWithConcurrency(t, ctx, conf, tenantId, fmt.Sprintf("%s-registrar", name), []repo.CreateConcurrencyOpts{def})

	queries := sqlcv1.New()
	strats, err := queries.GetTenantConcurrencyStrategiesByNames(ctx, conf.Pool, sqlcv1.GetTenantConcurrencyStrategiesByNamesParams{
		Tenantid: tenantId,
		Names:    []string{"shared-limit"},
	})
	require.NoError(t, err)
	require.Len(t, strats, 1)

	strat := strats[0]
	require.Equal(t, maxRuns, strat.MaxConcurrency, "registration must map max_runs onto the strategy")

	s := &sharedConcurrencyTestSetup{
		tenantId: tenantId,
		strategy: strat,
		repo:     conf.V1.Scheduler().Concurrency(),
	}

	for i := 0; i < numWorkflows; i++ {
		s.workflows = append(s.workflows, createWorkflowWithConcurrency(
			t, ctx, conf, tenantId, fmt.Sprintf("%s-wf-%d", name, i),
			[]repo.CreateConcurrencyOpts{def},
		))
	}

	return s
}

// createSharedConcurrencyTasks inserts numTasks queued tasks for the given workflow, all
// consuming the tenant-scoped strategy under a single concurrency key.
func createSharedConcurrencyTasks(
	t *testing.T,
	ctx context.Context,
	conf *database.Layer,
	s *sharedConcurrencyTestSetup,
	wf *workflowWithStep,
	numTasks int,
) {
	t.Helper()

	queries := sqlcv1.New()

	taskParams := newCreateTasksParams(numTasks)
	for i := 0; i < numTasks; i++ {
		taskParams.Tenantids[i] = s.tenantId
		taskParams.Queues[i] = "default"
		taskParams.Actionids[i] = "test:run"
		taskParams.Stepids[i] = wf.stepId
		taskParams.Stepreadableids[i] = "my-task"
		taskParams.Workflowids[i] = wf.workflowId
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
		taskParams.WorkflowVersionIds[i] = wf.workflowVersionId
		taskParams.WorkflowRunIds[i] = uuid.New()
		taskParams.IsDagOrchestrators[i] = false
	}

	tasks, err := queries.CreateTasks(ctx, conf.Pool, taskParams)
	require.NoError(t, err)
	require.Len(t, tasks, numTasks)
}

// TestConcurrency_SharedStrategy_CrossWorkflow proves the core property: tasks from two
// different workflows referencing the same tenant-scoped strategy resolve to the same
// strategy id and share its limit in the in-memory index.
func TestConcurrency_SharedStrategy_CrossWorkflow(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		s := setupSharedConcurrencyTest(t, ctx, conf, "shared-xwf-test", "GROUP_ROUND_ROBIN", 1, 2)

		queries := sqlcv1.New()

		// both steps' strategy rows resolve to the tenant strategy's id and definition
		rows, err := queries.ListConcurrencyStrategiesByStepId(ctx, conf.Pool, sqlcv1.ListConcurrencyStrategiesByStepIdParams{
			Tenantid: s.tenantId,
			Stepids:  []uuid.UUID{s.workflows[0].stepId, s.workflows[1].stepId},
		})
		require.NoError(t, err)
		require.Len(t, rows, 2)
		for _, row := range rows {
			require.True(t, row.TenantStrategyID.Valid)
			require.Equal(t, s.strategy.ID, row.TenantStrategyID.Int64, "both steps should reference the tenant strategy")
			require.Equal(t, s.strategy.Expression, row.Expression, "the definition copy is kept in sync")
		}
		// referencing rows are excluded from the step-strategy lease listing; the tenant
		// strategy is listed separately and is what gets a concurrency manager
		stepStrategies, err := queries.ListActiveConcurrencyStrategies(ctx, conf.Pool, s.tenantId)
		require.NoError(t, err)
		require.Empty(t, stepStrategies)

		tenantStrategies, err := queries.ListActiveTenantConcurrencyStrategies(ctx, conf.Pool, s.tenantId)
		require.NoError(t, err)
		require.Len(t, tenantStrategies, 1)
		require.Equal(t, s.strategy.ID, tenantStrategies[0].ID)

		// drive the in-memory index: build against the empty table first, then insert tasks
		// from BOTH workflows and confirm only maxRuns=1 is queued to run in total.
		l := zerolog.Nop()
		outbox := newTestOutbox(t, conf)
		cs := concurrency.NewConcurrencyStrategy(ctx, s.repo, s.strategyDescriptor(), outbox, &l)

		_, err = cs.Run(ctx)
		require.NoError(t, err)

		createSharedConcurrencyTasks(t, ctx, conf, s, s.workflows[0], 2)
		createSharedConcurrencyTasks(t, ctx, conf, s, s.workflows[1], 2)

		res, err := cs.Run(ctx)
		require.NoError(t, err)
		require.Len(t, res.Queued, 1, "4 tasks across 2 workflows against shared max=1 should queue exactly 1")
		require.Empty(t, res.Cancelled, "GROUP_ROUND_ROBIN should not cancel excess tasks")

		return nil
	})
}

// TestConcurrency_SharedStrategy_MixedWithInline verifies a step can hold both an inline
// (workflow-scoped) strategy and a tenant-scoped strategy, and both resolve for the step
// with distinct ids.
func TestConcurrency_SharedStrategy_MixedWithInline(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		s := setupSharedConcurrencyTest(t, ctx, conf, "shared-mixed-test", "GROUP_ROUND_ROBIN", 1, 0)

		maxRuns := int32(2)
		shMax := int32(1)
		strategyType := "GROUP_ROUND_ROBIN"
		inline := repo.CreateConcurrencyOpts{
			MaxRuns:       &maxRuns,
			LimitStrategy: &strategyType,
			Expression:    "input.other_id",
		}
		ref := repo.CreateConcurrencyOpts{
			Name:           "shared-limit",
			IsTenantScoped: true,
			Expression:     "input.my_id",
			MaxRuns:        &shMax,
			LimitStrategy:  &strategyType,
		}

		// declared order is the chain order: inline first here, tenant entry first below
		wfInlineFirst := createWorkflowWithConcurrency(t, ctx, conf, s.tenantId, "shared-mixed-wf", []repo.CreateConcurrencyOpts{inline, ref})
		wfRefFirst := createWorkflowWithConcurrency(t, ctx, conf, s.tenantId, "shared-mixed-wf-flipped", []repo.CreateConcurrencyOpts{ref, inline})

		queries := sqlcv1.New()

		assertOrder := func(stepId uuid.UUID, tenantFirst bool) {
			rows, err := queries.ListConcurrencyStrategiesByStepId(ctx, conf.Pool, sqlcv1.ListConcurrencyStrategiesByStepIdParams{
				Tenantid: s.tenantId,
				Stepids:  []uuid.UUID{stepId},
			})
			require.NoError(t, err)
			require.Len(t, rows, 2, "step should hold both the inline strategy and the tenant reference")

			// creation order (ascending row id) encodes the declared order
			sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })

			first, second := rows[0], rows[1]
			if tenantFirst {
				first, second = rows[1], rows[0]
			}

			require.False(t, first.TenantStrategyID.Valid, "inline entry position mismatch")
			require.True(t, second.TenantStrategyID.Valid, "tenant entry position mismatch")
			require.Equal(t, s.strategy.ID, second.TenantStrategyID.Int64)
		}

		assertOrder(wfInlineFirst.stepId, false)
		assertOrder(wfRefFirst.stepId, true)

		// the two inline strategies are listed as step strategies, the tenant strategy separately
		stepStrategies, err := queries.ListActiveConcurrencyStrategies(ctx, conf.Pool, s.tenantId)
		require.NoError(t, err)
		require.Len(t, stepStrategies, 2)

		tenantStrategies, err := queries.ListActiveTenantConcurrencyStrategies(ctx, conf.Pool, s.tenantId)
		require.NoError(t, err)
		require.Len(t, tenantStrategies, 1)

		return nil
	})
}

// TestConcurrency_SharedStrategy_UpsertInPlace verifies re-registering an existing name
// updates the same tenant strategy row (same id) with the new definition.
func TestConcurrency_SharedStrategy_UpsertInPlace(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		s := setupSharedConcurrencyTest(t, ctx, conf, "shared-upsert-test", "GROUP_ROUND_ROBIN", 1, 1)

		// re-declaring the name on another workflow put updates the strategy in place
		newMax := int32(5)
		newStrategy := "CANCEL_IN_PROGRESS"
		createWorkflowWithConcurrency(t, ctx, conf, s.tenantId, "shared-upsert-redeclare", []repo.CreateConcurrencyOpts{
			{
				Name:           "shared-limit",
				IsTenantScoped: true,
				Expression:     "input.other_id",
				MaxRuns:        &newMax,
				LimitStrategy:  &newStrategy,
			},
		})

		queries := sqlcv1.New()

		updatedStrats, err := queries.GetTenantConcurrencyStrategiesByNames(ctx, conf.Pool, sqlcv1.GetTenantConcurrencyStrategiesByNamesParams{
			Tenantid: s.tenantId,
			Names:    []string{"shared-limit"},
		})
		require.NoError(t, err)
		require.Len(t, updatedStrats, 1)

		updated := updatedStrats[0]
		require.Equal(t, s.strategy.ID, updated.ID, "upsert must update the same strategy row")
		require.Equal(t, int32(5), updated.MaxConcurrency)
		require.Equal(t, "input.other_id", updated.Expression)
		require.Equal(t, sqlcv1.V1ConcurrencyStrategyCANCELINPROGRESS, updated.Strategy)
		require.True(t, updated.IsActive)

		// the update trigger propagates the new definition to referencing rows, so
		// per-step reads pick it up without a join
		rows, err := queries.ListConcurrencyStrategiesByStepId(ctx, conf.Pool, sqlcv1.ListConcurrencyStrategiesByStepIdParams{
			Tenantid: s.tenantId,
			Stepids:  []uuid.UUID{s.workflows[0].stepId},
		})
		require.NoError(t, err)
		require.Len(t, rows, 1)
		require.Equal(t, int32(5), rows[0].MaxConcurrency, "the sync trigger must propagate the new definition to referencing rows")
		require.Equal(t, "input.other_id", rows[0].Expression)

		return nil
	})
}

// TestConcurrency_SharedStrategy_Reactivation is the regression test for the slot-insert
// trigger's tenant branch: inserting a slot carrying a tenant strategy's id must flip the
// tenant strategy back to active.
func TestConcurrency_SharedStrategy_Reactivation(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		s := setupSharedConcurrencyTest(t, ctx, conf, "shared-reactivate-test", "GROUP_ROUND_ROBIN", 1, 1)

		_, err := conf.Pool.Exec(ctx, "UPDATE v1_tenant_concurrency SET is_active = FALSE, last_active_at = NOW() - INTERVAL '2 hours' WHERE id = $1", s.strategy.ID)
		require.NoError(t, err)

		// inserting a task creates a concurrency slot, whose insert trigger reactivates the strategy
		createSharedConcurrencyTasks(t, ctx, conf, s, s.workflows[0], 1)

		var isActive bool
		err = conf.Pool.QueryRow(ctx, "SELECT is_active FROM v1_tenant_concurrency WHERE id = $1", s.strategy.ID).Scan(&isActive)
		require.NoError(t, err)
		require.True(t, isActive, "slot insert must reactivate the tenant strategy")

		return nil
	})
}

// TestConcurrency_SharedStrategy_StaleSweep verifies the 25h stale sweep deactivates a
// tenant strategy with no slots, removing it from the lease listing.
func TestConcurrency_SharedStrategy_StaleSweep(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		s := setupSharedConcurrencyTest(t, ctx, conf, "shared-stale-test", "GROUP_ROUND_ROBIN", 1, 0)

		queries := sqlcv1.New()

		_, err := conf.Pool.Exec(ctx, "UPDATE v1_tenant_concurrency SET last_active_at = NOW() - INTERVAL '26 hours' WHERE id = $1", s.strategy.ID)
		require.NoError(t, err)

		err = s.repo.DeactivateStaleStepConcurrency(ctx, s.tenantId)
		require.NoError(t, err)

		tenantStrategies, err := queries.ListActiveTenantConcurrencyStrategies(ctx, conf.Pool, s.tenantId)
		require.NoError(t, err)
		require.Empty(t, tenantStrategies, "stale tenant strategy should be deactivated")

		return nil
	})
}

// TestConcurrency_SharedStrategy_ActiveCheck verifies the workflow-version-based active
// check for tenant strategies: one stays active while a latest workflow version references
// it, and retires once nothing does (and no slots remain).
func TestConcurrency_SharedStrategy_ActiveCheck(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		queries := sqlcv1.New()

		// referenced by a latest workflow version: stays active
		s := setupSharedConcurrencyTest(t, ctx, conf, "shared-active-test", "GROUP_ROUND_ROBIN", 1, 1)

		err := s.repo.CheckAndDeactivateTenantConcurrency(ctx, s.tenantId, s.strategy.ID)
		require.NoError(t, err)

		strat, err := queries.GetTenantConcurrencyStrategyById(ctx, conf.Pool, sqlcv1.GetTenantConcurrencyStrategyByIdParams{
			Tenantid: s.tenantId,
			ID:       s.strategy.ID,
		})
		require.NoError(t, err)
		require.True(t, strat.IsActive, "a strategy referenced by a latest workflow version must stay active")

		// once every referencing workflow's latest version drops the entry (and no slots
		// remain), the strategy retires
		s2 := setupSharedConcurrencyTest(t, ctx, conf, "shared-inactive-test", "GROUP_ROUND_ROBIN", 1, 0)

		createWorkflowWithConcurrency(t, ctx, conf, s2.tenantId, "shared-inactive-test-registrar", nil)

		// while another session holds the strategy's advisory lock (as the slot-flush
		// path does mid-batch), the pass must skip the strategy rather than block on it
		lockConn, err := conf.Pool.Acquire(ctx)
		require.NoError(t, err)
		lockTx, err := lockConn.Begin(ctx)
		require.NoError(t, err)
		_, err = lockTx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", s2.strategy.ID)
		require.NoError(t, err)

		err = s2.repo.CheckAndDeactivateTenantConcurrency(ctx, s2.tenantId, s2.strategy.ID)
		require.NoError(t, err)

		strat2, err := queries.GetTenantConcurrencyStrategyById(ctx, conf.Pool, sqlcv1.GetTenantConcurrencyStrategyByIdParams{
			Tenantid: s2.tenantId,
			ID:       s2.strategy.ID,
		})
		require.NoError(t, err)
		require.True(t, strat2.IsActive, "the pass must skip (not block on) a strategy whose advisory lock is held")

		require.NoError(t, lockTx.Rollback(ctx))
		lockConn.Release()

		// with the lock free, the same pass retires it
		err = s2.repo.CheckAndDeactivateTenantConcurrency(ctx, s2.tenantId, s2.strategy.ID)
		require.NoError(t, err)

		strat2, err = queries.GetTenantConcurrencyStrategyById(ctx, conf.Pool, sqlcv1.GetTenantConcurrencyStrategyByIdParams{
			Tenantid: s2.tenantId,
			ID:       s2.strategy.ID,
		})
		require.NoError(t, err)
		require.False(t, strat2.IsActive, "a strategy referenced only by superseded versions with no slots must retire")

		return nil
	})
}

// TestConcurrency_SharedStrategy_WorkflowLevel verifies workflow-level tenant-scoped
// concurrency: with the DAG operator entitlement it lands on the orchestrator task, and
// without it (the old parent/child path) it is rejected loudly.
func TestConcurrency_SharedStrategy_WorkflowLevel(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		queries := sqlcv1.New()

		newTenant := func(name string, dagOperator bool) uuid.UUID {
			tenantId := uuid.New()
			_, err := conf.V1.Tenant().CreateTenant(ctx, &repo.CreateTenantOpts{
				ID:   &tenantId,
				Name: name,
				Slug: fmt.Sprintf("%s-%s", name, tenantId.String()),
			})
			require.NoError(t, err)

			err = conf.V1.TenantEntitlement().SetEntitlements(ctx, tenantId, repo.TenantEntitlements{DAGOperator: dagOperator})
			require.NoError(t, err)

			return tenantId
		}

		desc := "test workflow"
		maxRuns := int32(1)
		strategyType := "GROUP_ROUND_ROBIN"
		multiTask := []repo.CreateStepOpts{
			{ReadableId: "task-a", Action: "test:run"},
			{ReadableId: "task-b", Action: "test:run", Parents: []string{"task-a"}},
		}
		workflowConcurrency := []repo.CreateConcurrencyOpts{
			{
				Name:           "wf-shared-limit",
				IsTenantScoped: true,
				Expression:     "input.my_id",
				MaxRuns:        &maxRuns,
				LimitStrategy:  &strategyType,
			},
		}

		// new path: the entry becomes concurrency on the DAG orchestrator task
		dagTenant := newTenant("shared-wf-dag-test", true)

		wfVersion, err := conf.V1.Workflows().PutWorkflowVersion(ctx, dagTenant, &repo.CreateWorkflowVersionOpts{
			Name:        "shared-wf-dag",
			Description: &desc,
			Concurrency: workflowConcurrency,
			Tasks:       multiTask,
		})
		require.NoError(t, err)

		steps, err := conf.V1.Workflows().ListStepsByWorkflowVersionId(ctx, dagTenant, wfVersion.WorkflowVersion.ID)
		require.NoError(t, err)

		var orchestratorStepId uuid.UUID
		for _, step := range steps {
			if step.IsDagOrchestrator {
				orchestratorStepId = step.ID
			}
		}
		require.NotEqual(t, uuid.Nil, orchestratorStepId, "the workflow should have an orchestrator step")

		rows, err := queries.ListConcurrencyStrategiesByStepId(ctx, conf.Pool, sqlcv1.ListConcurrencyStrategiesByStepIdParams{
			Tenantid: dagTenant,
			Stepids:  []uuid.UUID{orchestratorStepId},
		})
		require.NoError(t, err)
		require.Len(t, rows, 1, "the tenant-scoped entry should land on the orchestrator step")
		require.True(t, rows[0].TenantStrategyID.Valid)

		// old path: rejected loudly
		oldTenant := newTenant("shared-wf-old-test", false)

		_, err = conf.V1.Workflows().PutWorkflowVersion(ctx, oldTenant, &repo.CreateWorkflowVersionOpts{
			Name:        "shared-wf-old",
			Description: &desc,
			Concurrency: workflowConcurrency,
			Tasks:       multiTask,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "not supported at the workflow level")

		return nil
	})
}

// TestConcurrency_SharedStrategy_OrderConflictRejected verifies deadlock prevention:
// registrations whose chains order the same tenant-scoped strategies inconsistently are
// rejected, both across workflows and within a single registration; re-registering the
// same workflow with a new order is allowed once nothing else constrains it.
func TestConcurrency_SharedStrategy_OrderConflictRejected(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		tenantId := uuid.New()
		_, err := conf.V1.Tenant().CreateTenant(ctx, &repo.CreateTenantOpts{
			ID:   &tenantId,
			Name: "shared-order-test",
			Slug: fmt.Sprintf("shared-order-test-%s", tenantId.String()),
		})
		require.NoError(t, err)

		maxRuns := int32(1)
		strategyType := "GROUP_ROUND_ROBIN"
		t1 := repo.CreateConcurrencyOpts{Name: "order-t1", IsTenantScoped: true, Expression: "input.my_id", MaxRuns: &maxRuns, LimitStrategy: &strategyType}
		t2 := repo.CreateConcurrencyOpts{Name: "order-t2", IsTenantScoped: true, Expression: "input.my_id", MaxRuns: &maxRuns, LimitStrategy: &strategyType}

		desc := "test workflow"

		putWorkflow := func(name string, entries []repo.CreateConcurrencyOpts) (uuid.UUID, error) {
			wfVersion, err := conf.V1.Workflows().PutWorkflowVersion(ctx, tenantId, &repo.CreateWorkflowVersionOpts{
				Name:        name,
				Description: &desc,
				Tasks: []repo.CreateStepOpts{
					{ReadableId: "my-task", Action: "test:run", Concurrency: entries},
				},
			})

			if err != nil {
				return uuid.Nil, err
			}

			return wfVersion.WorkflowVersion.ID, nil
		}

		// first workflow establishes t1 before t2
		_, err = putWorkflow("order-wf-1", []repo.CreateConcurrencyOpts{t1, t2})
		require.NoError(t, err)

		// a second workflow with the opposite order is rejected
		_, err = putWorkflow("order-wf-2", []repo.CreateConcurrencyOpts{t2, t1})
		require.Error(t, err)
		require.Contains(t, err.Error(), "ordered inconsistently")

		// the same order (with unrelated workflow-scoped entries interleaved) is fine
		inline := repo.CreateConcurrencyOpts{Expression: "input.other_id", MaxRuns: &maxRuns, LimitStrategy: &strategyType}
		wf3VersionId, err := putWorkflow("order-wf-3", []repo.CreateConcurrencyOpts{t1, inline, t2})
		require.NoError(t, err)

		// a conflict between two chains of a single registration is also rejected
		_, err = conf.V1.Workflows().PutWorkflowVersion(ctx, tenantId, &repo.CreateWorkflowVersionOpts{
			Name:        "order-wf-4",
			Description: &desc,
			Tasks: []repo.CreateStepOpts{
				{ReadableId: "task-a", Action: "test:run", Concurrency: []repo.CreateConcurrencyOpts{t1, t2}},
				{ReadableId: "task-b", Action: "test:run", Concurrency: []repo.CreateConcurrencyOpts{t2, t1}},
			},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "ordered inconsistently")

		// re-registering an existing workflow with a flipped order only conflicts with
		// OTHER workflows' latest versions, not its own previous version
		_, err = putWorkflow("order-wf-3", []repo.CreateConcurrencyOpts{inline, t1, t2})
		require.NoError(t, err)
		_, err = putWorkflow("order-wf-1", []repo.CreateConcurrencyOpts{t2, t1})
		require.Error(t, err, "wf-3 still orders t1 before t2")

		// a superseded version whose runs still hold concurrency slots keeps constraining
		// its own workflow: reordering before those runs drain would deadlock against them
		var t1Id int64
		err = conf.Pool.QueryRow(ctx, "SELECT id FROM v1_tenant_concurrency WHERE tenant_id = $1 AND name = 'order-t1'", tenantId).Scan(&t1Id)
		require.NoError(t, err)

		_, err = conf.Pool.Exec(ctx, `INSERT INTO v1_concurrency_slot
			(task_id, task_inserted_at, task_retry_count, external_id, tenant_id, workflow_id, workflow_version_id, workflow_run_id, strategy_id, priority, key, queue_to_notify, schedule_timeout_at)
			VALUES (1, NOW(), 0, gen_random_uuid(), $1, gen_random_uuid(), $2, gen_random_uuid(), $3, 1, 'k', 'q', NOW() + INTERVAL '5 minutes')`,
			tenantId, wf3VersionId, t1Id)
		require.NoError(t, err)

		// drop wf-1's ordering constraint (a single-entry chain imposes none) so the only
		// remaining constraint is the live slot on wf-3's superseded version
		_, err = putWorkflow("order-wf-1", []repo.CreateConcurrencyOpts{t1})
		require.NoError(t, err)

		_, err = putWorkflow("order-wf-3", []repo.CreateConcurrencyOpts{t2, inline, t1})
		require.Error(t, err, "a superseded version with live slots must still constrain ordering")

		// once the old runs drain, the reorder is allowed
		_, err = conf.Pool.Exec(ctx, "DELETE FROM v1_concurrency_slot WHERE tenant_id = $1 AND strategy_id = $2", tenantId, t1Id)
		require.NoError(t, err)

		_, err = putWorkflow("order-wf-3", []repo.CreateConcurrencyOpts{t2, inline, t1})
		require.NoError(t, err)

		return nil
	})
}

// TestConcurrency_SharedStrategy_RegisteredViaPutWorkflow verifies the folded registration
// path: strategy definitions carried on the workflow put are upserted in the same
// transaction, and a later put re-declaring the name updates the strategy in place.
func TestConcurrency_SharedStrategy_RegisteredViaPutWorkflow(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		tenantId := uuid.New()
		_, err := conf.V1.Tenant().CreateTenant(ctx, &repo.CreateTenantOpts{
			ID:   &tenantId,
			Name: "shared-putwf-test",
			Slug: fmt.Sprintf("shared-putwf-test-%s", tenantId.String()),
		})
		require.NoError(t, err)

		strategyType := "GROUP_ROUND_ROBIN"
		desc := "test workflow"

		putWorkflow := func(name string, defMaxRuns int32) *workflowWithStep {
			wfVersion, err := conf.V1.Workflows().PutWorkflowVersion(ctx, tenantId, &repo.CreateWorkflowVersionOpts{
				Name:        name,
				Description: &desc,
				Tasks: []repo.CreateStepOpts{
					{
						ReadableId: "my-task",
						Action:     "test:run",
						Concurrency: []repo.CreateConcurrencyOpts{
							{
								Name:           "putwf-shared-limit",
								IsTenantScoped: true,
								Expression:     "input.my_id",
								MaxRuns:        &defMaxRuns,
								LimitStrategy:  &strategyType,
							},
						},
					},
				},
			})
			require.NoError(t, err)

			steps, err := conf.V1.Workflows().ListStepsByWorkflowVersionId(ctx, tenantId, wfVersion.WorkflowVersion.ID)
			require.NoError(t, err)
			require.Len(t, steps, 1)

			return &workflowWithStep{
				workflowId:        wfVersion.WorkflowVersion.WorkflowId,
				workflowVersionId: wfVersion.WorkflowVersion.ID,
				stepId:            steps[0].ID,
			}
		}

		wfA := putWorkflow("shared-putwf-a", 1)

		queries := sqlcv1.New()

		// the strategy was upserted as part of the workflow put and resolves for the step
		rows, err := queries.ListConcurrencyStrategiesByStepId(ctx, conf.Pool, sqlcv1.ListConcurrencyStrategiesByStepIdParams{
			Tenantid: tenantId,
			Stepids:  []uuid.UUID{wfA.stepId},
		})
		require.NoError(t, err)
		require.Len(t, rows, 1)
		require.Equal(t, int32(1), rows[0].MaxConcurrency)
		require.True(t, rows[0].TenantStrategyID.Valid)

		var name string
		err = conf.Pool.QueryRow(ctx, "SELECT name FROM v1_tenant_concurrency WHERE id = $1", rows[0].TenantStrategyID.Int64).Scan(&name)
		require.NoError(t, err)
		require.Equal(t, "putwf-shared-limit", name, "the reference must point at the tenant strategy")

		strategyId := rows[0].TenantStrategyID.Int64

		// a second workflow re-declaring the strategy with a different limit updates it in
		// place (same id), and its own step resolves the same strategy
		wfB := putWorkflow("shared-putwf-b", 3)

		rows, err = queries.ListConcurrencyStrategiesByStepId(ctx, conf.Pool, sqlcv1.ListConcurrencyStrategiesByStepIdParams{
			Tenantid: tenantId,
			Stepids:  []uuid.UUID{wfB.stepId},
		})
		require.NoError(t, err)
		require.Len(t, rows, 1)
		require.True(t, rows[0].TenantStrategyID.Valid)
		require.Equal(t, strategyId, rows[0].TenantStrategyID.Int64, "re-declaring the name must update the same strategy row")
		require.Equal(t, int32(3), rows[0].MaxConcurrency)

		// the shared definition is tenant-level state, not workflow shape: re-putting an
		// unchanged workflow with a changed definition must reuse the workflow version
		// (checksum unchanged) while still updating the strategy in place
		wfARePut := putWorkflow("shared-putwf-a", 7)
		require.Equal(t, wfA.workflowVersionId, wfARePut.workflowVersionId, "definition changes must not mint a new workflow version")

		rows, err = queries.ListConcurrencyStrategiesByStepId(ctx, conf.Pool, sqlcv1.ListConcurrencyStrategiesByStepIdParams{
			Tenantid: tenantId,
			Stepids:  []uuid.UUID{wfA.stepId},
		})
		require.NoError(t, err)
		require.Len(t, rows, 1)
		require.Equal(t, int32(7), rows[0].MaxConcurrency, "the definition must still update in place on version reuse")

		return nil
	})
}
