//go:build integration

package v1_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/config/database"
	repo "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// countConcurrencySlots returns the number of filled (running) and unfilled (still-queued)
// v1_concurrency_slot rows for a strategy. The CANCEL_QUEUED_EXCEPT_* queries leave "spared" queued
// slots untouched (no result row), so counting the table directly is the only way to see how many
// survivors a pass actually kept.
func countConcurrencySlots(t *testing.T, ctx context.Context, conf *database.Layer, strategyID int64) (running, queued int) {
	t.Helper()

	err := conf.Pool.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE is_filled),
			count(*) FILTER (WHERE NOT is_filled)
		FROM v1_concurrency_slot
		WHERE strategy_id = $1
	`, strategyID).Scan(&running, &queued)
	require.NoError(t, err)

	return running, queued
}

// TestConcurrency_CancelQueuedExcept_StandaloneKeepsOnlyMaxRuns pins the core invariant of the
// CANCEL_QUEUED_EXCEPT_NEWEST / CANCEL_QUEUED_EXCEPT_OLDEST strategies on the no-parent
// (task-level) path, which runs RunCancelQueuedExceptNewest / RunCancelQueuedExceptOldest:
//
// after one scheduling pass over a backlog on a single key, the surviving footprint is exactly
// 2*maxRuns slots - maxRuns promoted to running, plus maxRuns still queued (kept for later
// promotion) - and everything else is cancelled.
func TestConcurrency_CancelQueuedExcept_StandaloneKeepsOnlyMaxRuns(t *testing.T) {
	const (
		maxRuns  = 2
		numTasks = 8 // 2 running + 2 queued survivors + 4 cancelled
	)

	cases := []struct {
		name     string
		strategy string
	}{
		{"except_newest", "CANCEL_QUEUED_EXCEPT_NEWEST"},
		{"except_oldest", "CANCEL_QUEUED_EXCEPT_OLDEST"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			runWithDatabase(t, func(conf *database.Layer) error {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				requireSchedulerSchema(t, ctx, conf)

				s := setupStepConcurrencyTest(t, ctx, conf, "cqe-standalone-"+tc.name, tc.strategy, maxRuns, numTasks)

				res, err := s.concurrencyRepo.RunConcurrencyStrategy(ctx, s.tenantId, s.strategy)
				require.NoError(t, err)

				require.Len(t, res.Queued, maxRuns,
					"%s must promote exactly maxRuns tasks to running", tc.strategy)
				require.Len(t, res.Cancelled, numTasks-2*maxRuns,
					"%s must cancel everything outside the maxRuns running + maxRuns queued bands", tc.strategy)

				running, queued := countConcurrencySlots(t, ctx, conf, s.strategy.ID)
				require.Equal(t, maxRuns, running, "%s: exactly maxRuns slots filled", tc.strategy)
				require.Equal(t, maxRuns, queued,
					"%s: exactly maxRuns slots kept queued (expected maxRuns, not 2*maxRuns and not 0)", tc.strategy)

				return nil
			})
		})
	}
}

// TestConcurrency_CancelQueuedExcept_WithParentConcurrencyKeepsOnlyMaxRuns is the same invariant on
// the parent/child path: a workflow with >1 task keeps workflow-level concurrency as a real parent
// strategy (mergeWorkflowConcurrencyOntoSingleTask only collapses single-task workflows), so
// scheduling runs RunParentCancelQueuedExcept* followed by RunChildCancelQueuedExcept*.
//
// This is the path where the two child queries diverge (EXCEPT_NEWEST spares rn > key_count-maxRuns,
// EXCEPT_OLDEST spares rn <= 2*maxRuns, ranked over a different population than the running band), so
// the assertion is deliberately identical for both strategies: exactly maxRuns running + maxRuns
// queued survivors on the key, nothing more.
func TestConcurrency_CancelQueuedExcept_WithParentConcurrencyKeepsOnlyMaxRuns(t *testing.T) {
	const (
		maxRuns = 2
		numRuns = 8 // 2 running + 2 queued survivors + 4 cancelled
	)

	cases := []struct {
		name     string
		strategy string
	}{
		{"except_newest", "CANCEL_QUEUED_EXCEPT_NEWEST"},
		{"except_oldest", "CANCEL_QUEUED_EXCEPT_OLDEST"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			runWithDatabase(t, func(conf *database.Layer) error {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				requireSchedulerSchema(t, ctx, conf)

				r := conf.V1
				queries := sqlcv1.New()

				tenantId := uuid.New()
				_, err := r.Tenant().CreateTenant(ctx, &repo.CreateTenantOpts{
					ID:   &tenantId,
					Name: "cqe-parent-" + tc.name,
					Slug: fmt.Sprintf("cqe-parent-%s-%s", tc.name, tenantId.String()),
				})
				require.NoError(t, err)

				desc := "parent concurrency cancel-queued-except test"
				strategy := tc.strategy
				mr := int32(maxRuns)

				wfVersion, err := r.Workflows().PutWorkflowVersion(ctx, tenantId, &repo.CreateWorkflowVersionOpts{
					Name:        "cqe-parent-" + tc.name,
					Description: &desc,
					Tasks: []repo.CreateStepOpts{
						{ReadableId: "task-a", Action: "test:run"},
						{ReadableId: "task-b", Action: "test:run"},
					},
					Concurrency: []repo.CreateConcurrencyOpts{
						{MaxRuns: &mr, LimitStrategy: &strategy, Expression: "input.my_id"},
					},
				})
				require.NoError(t, err)

				strategies, err := queries.ListActiveConcurrencyStrategies(ctx, conf.Pool, tenantId)
				require.NoError(t, err)
				require.NotEmpty(t, strategies)

				// Pick one child strategy derived from the workflow-level concurrency (it carries a
				// parent_strategy_id). Its step is the one we pile tasks onto.
				var child *sqlcv1.V1StepConcurrency
				for _, st := range strategies {
					if st.ParentStrategyID.Valid && st.Strategy == sqlcv1.V1ConcurrencyStrategy(tc.strategy) {
						child = st
						break
					}
				}
				require.NotNil(t, child, "expected a child %s strategy with a parent_strategy_id", tc.strategy)

				// numRuns workflow runs, each contributing one task for the child strategy's step,
				// all sharing a single concurrency key.
				taskParams := newCreateTasksParams(numRuns)
				for i := 0; i < numRuns; i++ {
					taskParams.Tenantids[i] = tenantId
					taskParams.Queues[i] = "default"
					taskParams.Actionids[i] = "test:run"
					taskParams.Stepids[i] = child.StepID
					taskParams.Stepreadableids[i] = "task-a"
					taskParams.Workflowids[i] = wfVersion.WorkflowVersion.WorkflowId
					taskParams.Scheduletimeouts[i] = "5m"
					taskParams.Priorities[i] = 1
					taskParams.Stickies[i] = string(sqlcv1.V1StickyStrategyNONE)
					taskParams.Externalids[i] = uuid.New()
					taskParams.Displaynames[i] = fmt.Sprintf("run-%d", i)
					taskParams.Additionalmetadatas[i] = []byte(`{}`)
					taskParams.InitialStates[i] = string(sqlcv1.V1TaskInitialStateQUEUED)
					taskParams.Concurrencyparentstrategyids[i] = []pgtype.Int8{child.ParentStrategyID}
					taskParams.ConcurrencyStrategyIds[i] = []int64{child.ID}
					taskParams.ConcurrencyKeys[i] = []string{"test-key"}
					taskParams.WorkflowVersionIds[i] = wfVersion.WorkflowVersion.ID
					taskParams.WorkflowRunIds[i] = uuid.New()
					taskParams.IsDagOrchestrators[i] = false
				}

				tasks, err := queries.CreateTasks(ctx, conf.Pool, taskParams)
				require.NoError(t, err)
				require.Len(t, tasks, numRuns)

				res, err := r.Scheduler().Concurrency().RunConcurrencyStrategy(ctx, tenantId, child)
				require.NoError(t, err)

				require.Len(t, res.Queued, maxRuns,
					"%s (child): exactly maxRuns tasks promoted to running", tc.strategy)

				running, queued := countConcurrencySlots(t, ctx, conf, child.ID)
				require.Equal(t, maxRuns, running,
					"%s (child): exactly maxRuns slots filled", tc.strategy)
				require.Equal(t, maxRuns, queued,
					"%s (child): exactly maxRuns slots kept queued - EXCEPT_OLDEST must not spare 2*maxRuns, "+
						"EXCEPT_NEWEST must not cancel a slot it is supposed to spare", tc.strategy)
				require.Len(t, res.Cancelled, numRuns-2*maxRuns,
					"%s (child): everything outside the maxRuns running + maxRuns queued bands is cancelled", tc.strategy)

				return nil
			})
		})
	}
}
