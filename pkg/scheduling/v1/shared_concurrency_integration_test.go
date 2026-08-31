//go:build integration

package v1_test

import (
	"context"
	"fmt"
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
// strategy: the scheduler represents both kinds of strategies as V1StepConcurrency, with
// zero-uuid workflow columns marking tenant scope.
func (s *sharedConcurrencyTestSetup) strategyDescriptor() *sqlcv1.V1StepConcurrency {
	return &sqlcv1.V1StepConcurrency{
		ID:             s.strategy.ID,
		TenantID:       s.tenantId,
		IsActive:       s.strategy.IsActive,
		LastActiveAt:   s.strategy.LastActiveAt,
		Strategy:       s.strategy.Strategy,
		Expression:     s.strategy.Expression,
		MaxConcurrency: s.strategy.MaxConcurrency,
	}
}

func createWorkflowWithSharedRef(
	t *testing.T,
	ctx context.Context,
	conf *database.Layer,
	tenantId uuid.UUID,
	name string,
	defs []repo.CreateSharedConcurrencyOpts,
	sharedNames []string,
	inline []repo.CreateConcurrencyOpts,
) *workflowWithStep {
	t.Helper()

	desc := "test workflow"
	wfVersion, err := conf.V1.Workflows().PutWorkflowVersion(ctx, tenantId, &repo.CreateWorkflowVersionOpts{
		Name:              name,
		Description:       &desc,
		SharedConcurrency: defs,
		Tasks: []repo.CreateStepOpts{
			{
				ReadableId:        "my-task",
				Action:            "test:run",
				Concurrency:       inline,
				SharedConcurrency: sharedNames,
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

	// the definition rides on a workflow put (the only registration path); the registrar
	// workflow's step does not reference the strategy, so no refs or slots exist yet
	def := repo.CreateSharedConcurrencyOpts{
		Name:          "shared-limit",
		Expression:    "input.my_id",
		MaxRuns:       &maxRuns,
		LimitStrategy: &strategyType,
	}

	createWorkflowWithSharedRef(t, ctx, conf, tenantId, fmt.Sprintf("%s-registrar", name), []repo.CreateSharedConcurrencyOpts{def}, nil, nil)

	queries := sqlcv1.New()
	strats, err := queries.GetTenantConcurrencyStrategiesByNames(ctx, conf.Pool, sqlcv1.GetTenantConcurrencyStrategiesByNamesParams{
		Tenantid: tenantId,
		Names:    []string{"shared-limit"},
	})
	require.NoError(t, err)
	require.Len(t, strats, 1)

	strat := strats[0]
	require.Equal(t, "shared-limit", strat.Name)
	require.Equal(t, maxRuns, strat.MaxConcurrency)

	s := &sharedConcurrencyTestSetup{
		tenantId: tenantId,
		strategy: strat,
		repo:     conf.V1.Scheduler().Concurrency(),
	}

	for i := 0; i < numWorkflows; i++ {
		s.workflows = append(s.workflows, createWorkflowWithSharedRef(
			t, ctx, conf, tenantId, fmt.Sprintf("%s-wf-%d", name, i), nil, []string{"shared-limit"}, nil,
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
			require.Equal(t, s.strategy.ID, row.ID, "both steps should resolve to the tenant strategy id")
			require.Equal(t, s.strategy.Expression, row.Expression)
		}
		require.NotEqual(t, rows[0].StepID, rows[1].StepID, "each row belongs to its own step")

		// the lease listing returns the tenant strategy (not the referencing rows)
		strategies, err := queries.ListActiveConcurrencyStrategies(ctx, conf.Pool, s.tenantId)
		require.NoError(t, err)
		require.Len(t, strategies, 1)
		require.Equal(t, s.strategy.ID, strategies[0].ID)
		require.Equal(t, uuid.Nil, strategies[0].WorkflowID, "tenant strategies carry zero-uuid workflow columns")

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
		strategyType := "GROUP_ROUND_ROBIN"
		wf := createWorkflowWithSharedRef(t, ctx, conf, s.tenantId, "shared-mixed-wf", nil, []string{"shared-limit"}, []repo.CreateConcurrencyOpts{
			{
				MaxRuns:       &maxRuns,
				LimitStrategy: &strategyType,
				Expression:    "input.other_id",
			},
		})

		queries := sqlcv1.New()

		rows, err := queries.ListConcurrencyStrategiesByStepId(ctx, conf.Pool, sqlcv1.ListConcurrencyStrategiesByStepIdParams{
			Tenantid: s.tenantId,
			Stepids:  []uuid.UUID{wf.stepId},
		})
		require.NoError(t, err)
		require.Len(t, rows, 2, "step should resolve both the inline and the tenant strategy")

		ids := map[int64]bool{}
		for _, row := range rows {
			ids[row.ID] = true
			require.Equal(t, wf.stepId, row.StepID)
		}
		require.True(t, ids[s.strategy.ID], "tenant strategy should resolve for the step")
		require.Len(t, ids, 2, "inline and tenant strategies must have distinct ids")

		// both strategies are schedulable for the tenant
		strategies, err := queries.ListActiveConcurrencyStrategies(ctx, conf.Pool, s.tenantId)
		require.NoError(t, err)
		require.Len(t, strategies, 2)

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
		createWorkflowWithSharedRef(t, ctx, conf, s.tenantId, "shared-upsert-redeclare", []repo.CreateSharedConcurrencyOpts{
			{
				Name:          "shared-limit",
				Expression:    "input.other_id",
				MaxRuns:       &newMax,
				LimitStrategy: &newStrategy,
			},
		}, nil, nil)

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

		// the step's strategies resolve the NEW definition through the reference, even
		// though the referencing row still carries its point-in-time copy
		rows, err := queries.ListConcurrencyStrategiesByStepId(ctx, conf.Pool, sqlcv1.ListConcurrencyStrategiesByStepIdParams{
			Tenantid: s.tenantId,
			Stepids:  []uuid.UUID{s.workflows[0].stepId},
		})
		require.NoError(t, err)
		require.Len(t, rows, 1)
		require.Equal(t, int32(5), rows[0].MaxConcurrency)
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

		strategies, err := queries.ListActiveConcurrencyStrategies(ctx, conf.Pool, s.tenantId)
		require.NoError(t, err)
		require.Empty(t, strategies, "stale tenant strategy should be deactivated")

		return nil
	})
}

// TestConcurrency_SharedStrategy_UnknownNameErrors verifies workflow registration fails
// loudly when a step references a tenant strategy that was never registered.
func TestConcurrency_SharedStrategy_UnknownNameErrors(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		tenantId := uuid.New()
		_, err := conf.V1.Tenant().CreateTenant(ctx, &repo.CreateTenantOpts{
			ID:   &tenantId,
			Name: "shared-unknown-test",
			Slug: fmt.Sprintf("shared-unknown-test-%s", tenantId.String()),
		})
		require.NoError(t, err)

		desc := "test workflow"
		_, err = conf.V1.Workflows().PutWorkflowVersion(ctx, tenantId, &repo.CreateWorkflowVersionOpts{
			Name:        "shared-unknown-wf",
			Description: &desc,
			Tasks: []repo.CreateStepOpts{
				{
					ReadableId:        "my-task",
					Action:            "test:run",
					SharedConcurrency: []string{"never-registered"},
				},
			},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "never-registered")

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
				SharedConcurrency: []repo.CreateSharedConcurrencyOpts{
					{
						Name:          "putwf-shared-limit",
						Expression:    "input.my_id",
						MaxRuns:       &defMaxRuns,
						LimitStrategy: &strategyType,
					},
				},
				Tasks: []repo.CreateStepOpts{
					{
						ReadableId:        "my-task",
						Action:            "test:run",
						SharedConcurrency: []string{"putwf-shared-limit"},
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

		var name string
		err = conf.Pool.QueryRow(ctx, "SELECT name FROM v1_tenant_concurrency WHERE id = $1", rows[0].ID).Scan(&name)
		require.NoError(t, err)
		require.Equal(t, "putwf-shared-limit", name, "the resolved strategy id must belong to the tenant strategy")

		strategyId := rows[0].ID

		// a second workflow re-declaring the strategy with a different limit updates it in
		// place (same id), and its own step resolves the same strategy
		wfB := putWorkflow("shared-putwf-b", 3)

		rows, err = queries.ListConcurrencyStrategiesByStepId(ctx, conf.Pool, sqlcv1.ListConcurrencyStrategiesByStepIdParams{
			Tenantid: tenantId,
			Stepids:  []uuid.UUID{wfB.stepId},
		})
		require.NoError(t, err)
		require.Len(t, rows, 1)
		require.Equal(t, strategyId, rows[0].ID, "re-declaring the name must update the same strategy row")
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
