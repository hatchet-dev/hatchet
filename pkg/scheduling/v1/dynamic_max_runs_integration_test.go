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

// dynamicMaxRunsTestSetup holds a tenant with one workflow whose single step carries an
// inline strategy with a max_runs_expression.
type dynamicMaxRunsTestSetup struct {
	tenantId uuid.UUID
	strategy *sqlcv1.V1StepConcurrency
	repo     repo.ConcurrencyRepository
	wf       *workflowWithStep
}

func setupDynamicMaxRunsTest(
	t *testing.T,
	ctx context.Context,
	conf *database.Layer,
	name string,
	strategyType string,
) *dynamicMaxRunsTestSetup {
	t.Helper()

	tenantId := uuid.New()
	_, err := conf.V1.Tenant().CreateTenant(ctx, &repo.CreateTenantOpts{
		ID:   &tenantId,
		Name: name,
		Slug: fmt.Sprintf("%s-%s", name, tenantId.String()),
	})
	require.NoError(t, err)

	maxRuns := int32(1)
	maxRunsExpr := "input.limit"

	wf := createWorkflowWithConcurrency(t, ctx, conf, tenantId, name, []repo.CreateConcurrencyOpts{
		{
			Expression:        "input.group",
			MaxRuns:           &maxRuns,
			LimitStrategy:     &strategyType,
			MaxRunsExpression: &maxRunsExpr,
		},
	})

	queries := sqlcv1.New()
	rows, err := queries.ListConcurrencyStrategiesByStepId(ctx, conf.Pool, sqlcv1.ListConcurrencyStrategiesByStepIdParams{
		Tenantid: tenantId,
		Stepids:  []uuid.UUID{wf.stepId},
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.True(t, rows[0].MaxRunsExpression.Valid, "registration must persist max_runs_expression on the strategy row")
	require.Equal(t, maxRunsExpr, rows[0].MaxRunsExpression.String)

	return &dynamicMaxRunsTestSetup{
		tenantId: tenantId,
		strategy: rows[0],
		repo:     conf.V1.Scheduler().Concurrency(),
		wf:       wf,
	}
}

// createDynamicMaxRunsTasks inserts queued tasks whose per-task evaluated max-runs values
// and group keys are given per task, exercising the real insert trigger, slot columns, and
// outbox payloads.
func createDynamicMaxRunsTasks(
	t *testing.T,
	ctx context.Context,
	conf *database.Layer,
	s *dynamicMaxRunsTestSetup,
	keys []string,
	maxRuns []pgtype.Int4,
) []*sqlcv1.V1Task {
	t.Helper()

	require.Equal(t, len(keys), len(maxRuns))

	queries := sqlcv1.New()

	n := len(keys)
	taskParams := newCreateTasksParams(n)
	taskParams.ConcurrencyMaxRuns = make([][]pgtype.Int4, n)
	for i := 0; i < n; i++ {
		taskParams.Tenantids[i] = s.tenantId
		taskParams.Queues[i] = "default"
		taskParams.Actionids[i] = "test:run"
		taskParams.Stepids[i] = s.wf.stepId
		taskParams.Stepreadableids[i] = "my-task"
		taskParams.Workflowids[i] = s.wf.workflowId
		taskParams.Scheduletimeouts[i] = "5m"
		taskParams.Priorities[i] = 1
		taskParams.Stickies[i] = string(sqlcv1.V1StickyStrategyNONE)
		taskParams.Externalids[i] = uuid.New()
		taskParams.Displaynames[i] = fmt.Sprintf("task-%d", i)
		taskParams.Additionalmetadatas[i] = []byte(`{}`)
		taskParams.InitialStates[i] = string(sqlcv1.V1TaskInitialStateQUEUED)
		taskParams.Concurrencyparentstrategyids[i] = []pgtype.Int8{{}}
		taskParams.ConcurrencyStrategyIds[i] = []int64{s.strategy.ID}
		taskParams.ConcurrencyKeys[i] = []string{keys[i]}
		taskParams.ConcurrencyMaxRuns[i] = []pgtype.Int4{maxRuns[i]}
		taskParams.WorkflowVersionIds[i] = s.wf.workflowVersionId
		taskParams.WorkflowRunIds[i] = uuid.New()
		taskParams.IsDagOrchestrators[i] = false
	}

	tasks, err := queries.CreateTasks(ctx, conf.Pool, taskParams)
	require.NoError(t, err)
	require.Len(t, tasks, n)

	return tasks
}

// TestConcurrency_DynamicMaxRuns_PerGroupLimits proves the core property end to end
// through the real insert trigger, slot columns, and WAL payloads: two groups on the same
// strategy run at different limits carried by their slots' evaluated values, overriding
// the static max of 1.
func TestConcurrency_DynamicMaxRuns_PerGroupLimits(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		s := setupDynamicMaxRunsTest(t, ctx, conf, "dyn-per-group", "GROUP_ROUND_ROBIN")

		l := zerolog.Nop()
		outbox := newTestOutbox(t, conf)
		cs := concurrency.NewConcurrencyStrategy(ctx, s.repo, s.strategy, outbox, &l)

		_, err := cs.Run(ctx)
		require.NoError(t, err)

		val := func(v int32) pgtype.Int4 { return pgtype.Int4{Int32: v, Valid: true} }

		createDynamicMaxRunsTasks(t, ctx, conf, s,
			[]string{"premium", "premium", "premium", "free", "free"},
			[]pgtype.Int4{val(3), val(3), val(3), val(1), val(1)},
		)

		res, err := cs.Run(ctx)
		require.NoError(t, err)
		require.Len(t, res.Queued, 4, "premium (limit 3) fills 3 and free (limit 1) fills 1; the static max of 1 must not apply")
		require.Empty(t, res.Cancelled)

		return nil
	})
}

// TestConcurrency_DynamicMaxRuns_ReplayCannotRegressLimit exercises the footgun path
// through the real v1_task_update_function slot re-insert: replaying an older task after a
// newer task lowered the group limit re-creates a slot carrying the old task's timestamp,
// which must not restore the old (higher) limit.
func TestConcurrency_DynamicMaxRuns_ReplayCannotRegressLimit(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		s := setupDynamicMaxRunsTest(t, ctx, conf, "dyn-replay", "GROUP_ROUND_ROBIN")

		l := zerolog.Nop()
		outbox := newTestOutbox(t, conf)
		cs := concurrency.NewConcurrencyStrategy(ctx, s.repo, s.strategy, outbox, &l)

		_, err := cs.Run(ctx)
		require.NoError(t, err)

		val := func(v int32) pgtype.Int4 { return pgtype.Int4{Int32: v, Valid: true} }

		// the old task sets limit 5; both it and a peer fill (5 slots available)
		oldTasks := createDynamicMaxRunsTasks(t, ctx, conf, s,
			[]string{"k", "k"},
			[]pgtype.Int4{val(5), val(5)},
		)

		res, err := cs.Run(ctx)
		require.NoError(t, err)
		require.Len(t, res.Queued, 2)

		// a newer task lowers the limit to 1; running work is grandfathered, the new
		// task itself stays queued
		time.Sleep(10 * time.Millisecond) // strictly newer inserted_at than the first batch
		createDynamicMaxRunsTasks(t, ctx, conf, s,
			[]string{"k"},
			[]pgtype.Int4{val(1)},
		)

		res, err = cs.Run(ctx)
		require.NoError(t, err)
		require.Empty(t, res.Queued, "the lowered limit must hold the new task while 2 grandfathered tasks run")

		// the old task completes (slot deleted) and is replayed with its stale value:
		// the update trigger re-creates its slot with the ORIGINAL task timestamp
		queries := sqlcv1.New()
		replayed := oldTasks[0]

		_, err = conf.Pool.Exec(ctx, "DELETE FROM v1_concurrency_slot WHERE task_id = $1 AND task_inserted_at = $2", replayed.ID, replayed.InsertedAt)
		require.NoError(t, err)

		replayParams := sqlcv1.ReplayTasksParams{
			Taskids:                    []int64{replayed.ID},
			Taskinsertedats:            []pgtype.Timestamptz{replayed.InsertedAt},
			Inputs:                     [][]byte{nil},
			InitialStates:              []string{string(sqlcv1.V1TaskInitialStateQUEUED)},
			InitialStateReasons:        []pgtype.Text{{}},
			Concurrencykeys:            [][]string{{"k"}},
			ConcurrencyMaxRuns:         [][]pgtype.Int4{{val(5)}},
			DesiredWorkerLabels:        [][]byte{nil},
			TriggeringEventExternalIds: []*uuid.UUID{nil},
			TriggeringEventKeys:        []pgtype.Text{{}},
			BatchKeys:                  []string{""},
		}
		replayedTasks, err := queries.ReplayTasks(ctx, conf.Pool, replayParams)
		require.NoError(t, err)
		require.Len(t, replayedTasks, 1)

		// the replayed slot's INSERT carries maxRuns 5 with the old task timestamp: the
		// guard must keep the limit at 1, so with 1 grandfathered task still running,
		// nothing new fills. Without the guard the limit would jump back to 5 and the
		// replayed slot (plus the held task) would fill immediately.
		res, err = cs.Run(ctx)
		require.NoError(t, err)
		require.Empty(t, res.Queued, "a replayed older task must not restore its stale higher limit")

		return nil
	})
}

// TestConcurrency_DynamicMaxRuns_ChainedSlotCarriesValue covers the chain hand-off in
// v1_concurrency_slot_update_function: when the first slot in a chain fills, the trigger
// peels next_max_runs onto the second strategy's slot, whose WAL payload must carry the
// dynamic value for the SECOND strategy (a misaligned peel would silently give the second
// strategy the wrong limit).
func TestConcurrency_DynamicMaxRuns_ChainedSlotCarriesValue(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		tenantId := uuid.New()
		_, err := conf.V1.Tenant().CreateTenant(ctx, &repo.CreateTenantOpts{
			ID:   &tenantId,
			Name: "dyn-chain",
			Slug: fmt.Sprintf("dyn-chain-%s", tenantId.String()),
		})
		require.NoError(t, err)

		maxRuns := int32(1)
		strategyType := "GROUP_ROUND_ROBIN"
		maxRunsExpr := "input.limit"

		// chain order: static entry first, dynamic entry second
		wf := createWorkflowWithConcurrency(t, ctx, conf, tenantId, "dyn-chain-wf", []repo.CreateConcurrencyOpts{
			{Expression: "input.first", MaxRuns: &maxRuns, LimitStrategy: &strategyType},
			{Expression: "input.tier", MaxRuns: &maxRuns, LimitStrategy: &strategyType, MaxRunsExpression: &maxRunsExpr},
		})

		queries := sqlcv1.New()
		rows, err := queries.ListConcurrencyStrategiesByStepId(ctx, conf.Pool, sqlcv1.ListConcurrencyStrategiesByStepIdParams{
			Tenantid: tenantId,
			Stepids:  []uuid.UUID{wf.stepId},
		})
		require.NoError(t, err)
		require.Len(t, rows, 2)
		if rows[0].ID > rows[1].ID {
			rows[0], rows[1] = rows[1], rows[0]
		}
		staticStrat, dynStrat := rows[0], rows[1]
		require.False(t, staticStrat.MaxRunsExpression.Valid)
		require.True(t, dynStrat.MaxRunsExpression.Valid)

		l := zerolog.Nop()
		outbox := newTestOutbox(t, conf)
		cs1 := concurrency.NewConcurrencyStrategy(ctx, conf.V1.Scheduler().Concurrency(), staticStrat, outbox, &l)
		cs2 := concurrency.NewConcurrencyStrategy(ctx, conf.V1.Scheduler().Concurrency(), dynStrat, outbox, &l)

		_, err = cs1.Run(ctx)
		require.NoError(t, err)
		_, err = cs2.Run(ctx)
		require.NoError(t, err)

		// 3 tasks: distinct keys on the static strategy (all advance) and one shared
		// group on the dynamic strategy with an evaluated limit of 3. The dynamic value
		// rides in position 2 of the arrays; position 1 is NULL (static).
		val := func(v int32) pgtype.Int4 { return pgtype.Int4{Int32: v, Valid: true} }
		n := 3
		taskParams := newCreateTasksParams(n)
		taskParams.ConcurrencyMaxRuns = make([][]pgtype.Int4, n)
		for i := 0; i < n; i++ {
			taskParams.Tenantids[i] = tenantId
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
			taskParams.Concurrencyparentstrategyids[i] = []pgtype.Int8{{}, {}}
			taskParams.ConcurrencyStrategyIds[i] = []int64{staticStrat.ID, dynStrat.ID}
			taskParams.ConcurrencyKeys[i] = []string{fmt.Sprintf("first-%d", i), "premium"}
			taskParams.ConcurrencyMaxRuns[i] = []pgtype.Int4{{}, val(3)}
			taskParams.WorkflowVersionIds[i] = wf.workflowVersionId
			taskParams.WorkflowRunIds[i] = uuid.New()
			taskParams.IsDagOrchestrators[i] = false
		}
		tasks, err := queries.CreateTasks(ctx, conf.Pool, taskParams)
		require.NoError(t, err)
		require.Len(t, tasks, n)

		// first strategy fills all 3 (distinct keys); each fill makes the slot-update
		// trigger create the chained slot for the dynamic strategy
		res, err := cs1.Run(ctx)
		require.NoError(t, err)
		require.Empty(t, res.Queued, "head slots have next strategies; filling them advances the chain rather than queueing")

		// the chained slots must carry max_runs 3, so the dynamic strategy fills all 3
		// despite its static max of 1
		res, err = cs2.Run(ctx)
		require.NoError(t, err)
		require.Len(t, res.Queued, 3, "the chained slots must carry the dynamic value peeled from next_max_runs")

		return nil
	})
}

// TestConcurrency_DynamicMaxRuns_TenantScopedPropagation covers the definition-sync path
// for tenant-scoped strategies: max_runs_expression is copied onto referencing rows at
// registration and kept in sync by the update trigger on later upserts.
func TestConcurrency_DynamicMaxRuns_TenantScopedPropagation(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		tenantId := uuid.New()
		_, err := conf.V1.Tenant().CreateTenant(ctx, &repo.CreateTenantOpts{
			ID:   &tenantId,
			Name: "dyn-tenant-sync",
			Slug: fmt.Sprintf("dyn-tenant-sync-%s", tenantId.String()),
		})
		require.NoError(t, err)

		maxRuns := int32(1)
		strategyType := "GROUP_ROUND_ROBIN"
		exprV1 := "input.limit"

		def := repo.CreateConcurrencyOpts{
			Name:              "dyn-shared",
			TenantScoped:      true,
			Expression:        "input.group",
			MaxRuns:           &maxRuns,
			LimitStrategy:     &strategyType,
			MaxRunsExpression: &exprV1,
		}

		wf := createWorkflowWithConcurrency(t, ctx, conf, tenantId, "dyn-tenant-sync-wf", []repo.CreateConcurrencyOpts{def})

		queries := sqlcv1.New()

		rows, err := queries.ListConcurrencyStrategiesByStepId(ctx, conf.Pool, sqlcv1.ListConcurrencyStrategiesByStepIdParams{
			Tenantid: tenantId,
			Stepids:  []uuid.UUID{wf.stepId},
		})
		require.NoError(t, err)
		require.Len(t, rows, 1)
		require.Equal(t, exprV1, rows[0].MaxRunsExpression.String, "the referencing row must carry the tenant strategy's max_runs_expression")

		// re-registering with a different expression updates the tenant strategy in
		// place; the sync trigger must propagate it to referencing rows
		exprV2 := "input.limit * 2"
		def.MaxRunsExpression = &exprV2
		createWorkflowWithConcurrency(t, ctx, conf, tenantId, "dyn-tenant-sync-wf-2", []repo.CreateConcurrencyOpts{def})

		rows, err = queries.ListConcurrencyStrategiesByStepId(ctx, conf.Pool, sqlcv1.ListConcurrencyStrategiesByStepIdParams{
			Tenantid: tenantId,
			Stepids:  []uuid.UUID{wf.stepId},
		})
		require.NoError(t, err)
		require.Len(t, rows, 1)
		require.Equal(t, exprV2, rows[0].MaxRunsExpression.String, "the sync trigger must propagate max_runs_expression to referencing rows")

		// the tenant descriptor carries the expression, so strategyDiffers-driven manager
		// rebuilds see the change
		strats, err := queries.GetTenantConcurrencyStrategiesByNames(ctx, conf.Pool, sqlcv1.GetTenantConcurrencyStrategiesByNamesParams{
			Tenantid: tenantId,
			Names:    []string{"dyn-shared"},
		})
		require.NoError(t, err)
		require.Len(t, strats, 1)
		require.Equal(t, exprV2, repo.TenantConcurrencyDescriptor(strats[0]).MaxRunsExpression.String)

		return nil
	})
}

// TestConcurrency_DynamicMaxRuns_RegistrationRejections covers the two registration-time
// gates: an expression that can never evaluate to an integer is rejected by validation,
// and workflow-level entries on the old parent/child DAG path are rejected because that
// machinery never sees per-slot max-runs values.
func TestConcurrency_DynamicMaxRuns_RegistrationRejections(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		requireSchedulerSchema(t, ctx, conf)

		tenantId := uuid.New()
		_, err := conf.V1.Tenant().CreateTenant(ctx, &repo.CreateTenantOpts{
			ID:   &tenantId,
			Name: "dyn-reject",
			Slug: fmt.Sprintf("dyn-reject-%s", tenantId.String()),
		})
		require.NoError(t, err)

		maxRuns := int32(1)
		badExpr := "'a string'"
		desc := "test workflow"

		_, err = conf.V1.Workflows().PutWorkflowVersion(ctx, tenantId, &repo.CreateWorkflowVersionOpts{
			Name:        "dyn-reject-bad-type",
			Description: &desc,
			Tasks: []repo.CreateStepOpts{
				{
					ReadableId: "my-task",
					Action:     "test:run",
					Concurrency: []repo.CreateConcurrencyOpts{
						{Expression: "input.group", MaxRuns: &maxRuns, MaxRunsExpression: &badExpr},
					},
				},
			},
		})
		require.Error(t, err, "a max_runs_expression that statically evaluates to a string must be rejected")

		goodExpr := "input.limit"
		_, err = conf.V1.Workflows().PutWorkflowVersion(ctx, tenantId, &repo.CreateWorkflowVersionOpts{
			Name:        "dyn-reject-workflow-level",
			Description: &desc,
			Concurrency: []repo.CreateConcurrencyOpts{
				{Expression: "input.group", MaxRuns: &maxRuns, MaxRunsExpression: &goodExpr},
			},
			Tasks: []repo.CreateStepOpts{
				{ReadableId: "task-a", Action: "test:a"},
				{ReadableId: "task-b", Action: "test:b", Parents: []string{"task-a"}},
			},
		})
		require.Error(t, err, "workflow-level max_runs_expression on the old DAG path must be rejected")
		require.ErrorContains(t, err, "max_runs_expression")

		return nil
	})
}
