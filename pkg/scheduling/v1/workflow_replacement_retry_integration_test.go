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

func TestConcurrency_WorkflowReplacementDuringRetryBackoff(t *testing.T) {
	for _, standalone := range []bool{false, true} {
		for _, expired := range []bool{false, true} {
			t.Run(fmt.Sprintf("standalone_%t/retry_due_%t", standalone, expired), func(t *testing.T) {
				runWithDatabase(t, func(conf *database.Layer) error {
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()
					requireSchedulerSchema(t, ctx, conf)
					tenantID := uuid.New()
					_, err := conf.V1.Tenant().CreateTenant(ctx, &repo.CreateTenantOpts{
						ID: &tenantID, Name: "replacement-retry", Slug: fmt.Sprintf("replacement-retry-%s", tenantID),
					})
					require.NoError(t, err)
					strategy := "CANCEL_IN_PROGRESS"
					maxRuns := int32(1)
					retries := 1
					description := "workflow replacement during retry backoff"
					tasks := []repo.CreateStepOpts{
						{ReadableId: "work", Action: "test:work", Retries: &retries},
					}
					if !standalone {
						tasks = append(tasks, repo.CreateStepOpts{ReadableId: "publish", Action: "test:publish", Parents: []string{"work"}})
					}
					version, err := conf.V1.Workflows().PutWorkflowVersion(ctx, tenantID, &repo.CreateWorkflowVersionOpts{
						Name:        "replacement-retry",
						Description: &description,
						Concurrency: []repo.CreateConcurrencyOpts{{MaxRuns: &maxRuns, LimitStrategy: &strategy, Expression: "input.key"}},
						Tasks:       tasks,
					})
					require.NoError(t, err)
					queries := sqlcv1.New()
					strategies, err := queries.ListActiveConcurrencyStrategies(ctx, conf.Pool, tenantID)
					require.NoError(t, err)
					require.NotEmpty(t, strategies)
					child := strategies[0]
					for _, candidate := range strategies {
						var readableID string
						err = conf.Pool.QueryRow(ctx, `SELECT "readableId" FROM "Step" WHERE id = $1`, candidate.StepID).Scan(&readableID)
						require.NoError(t, err)
						if readableID == "work" {
							child = candidate
						}
					}
					require.Equal(t, !standalone, child.ParentStrategyID.Valid)

					create := func(key string) *repo.TaskIdInsertedAtRetryCount {
						params := newCreateTasksParams(1)
						params.Tenantids[0] = tenantID
						params.Queues[0] = "default"
						params.Actionids[0] = "test:work"
						params.Stepids[0] = child.StepID
						params.Stepreadableids[0] = "work"
						params.Workflowids[0] = version.WorkflowVersion.WorkflowId
						params.Scheduletimeouts[0] = "5m"
						params.Priorities[0] = 1
						params.Stickies[0] = "NONE"
						params.Externalids[0] = uuid.New()
						params.Displaynames[0] = "replacement-retry"
						params.Additionalmetadatas[0] = []byte(`{}`)
						params.InitialStates[0] = "QUEUED"
						params.Concurrencyparentstrategyids[0] = []pgtype.Int8{child.ParentStrategyID}
						params.ConcurrencyStrategyIds[0] = []int64{child.ID}
						params.ConcurrencyKeys[0] = []string{key}
						params.WorkflowVersionIds[0] = version.WorkflowVersion.ID
						params.WorkflowRunIds[0] = params.Externalids[0]
						params.RetryBackoffFactor[0] = pgtype.Float8{Float64: 30, Valid: true}
						tasks, err := queries.CreateTasks(ctx, conf.Pool, params)
						require.NoError(t, err)
						require.Len(t, tasks, 1)
						return &repo.TaskIdInsertedAtRetryCount{Id: tasks[0].ID, InsertedAt: tasks[0].InsertedAt, RetryCount: 0}
					}
					old := create("same-key")
					concurrency := conf.V1.Scheduler().Concurrency()
					admitted, err := concurrency.RunConcurrencyStrategy(ctx, tenantID, child)
					require.NoError(t, err)
					require.Len(t, admitted.Queued, 1)
					_, err = conf.Pool.Exec(ctx, `INSERT INTO v1_task_runtime (task_id, task_inserted_at, retry_count, tenant_id, timeout_at) VALUES ($1, $2, 0, $3, NOW() + INTERVAL '5 minutes')`, old.Id, old.InsertedAt, tenantID)
					require.NoError(t, err)
					failed, err := conf.V1.Tasks().FailTasks(ctx, tenantID, []repo.FailTaskOpts{{TaskIdInsertedAtRetryCount: old, IsAppError: true, ErrorMessage: "retryable"}})
					require.NoError(t, err)
					require.Len(t, failed.RetriedTasks, 1)
					var pending int
					err = conf.Pool.QueryRow(ctx, "SELECT count(*) FROM v1_retry_queue_item WHERE task_id = $1", old.Id).Scan(&pending)
					require.NoError(t, err)
					require.Equal(t, 1, pending)
					// Move the stored deadline instead of sleeping so both sides of retry
					// admission are exercised deterministically.
					if expired {
						_, err = conf.Pool.Exec(ctx, "UPDATE v1_retry_queue_item SET retry_after = NOW() - INTERVAL '1 second' WHERE task_id = $1", old.Id)
						require.NoError(t, err)
					}
					create("same-key")
					replaced, err := concurrency.RunConcurrencyStrategy(ctx, tenantID, child)
					require.NoError(t, err)
					require.Len(t, replaced.Queued, 1)
					require.Len(t, replaced.Cancelled, 1, "replacement must cancel the superseded pending retry")
					require.Equal(t, old.Id, replaced.Cancelled[0].Id)
					require.Equal(t, int32(1), replaced.Cancelled[0].RetryCount)
					err = conf.Pool.QueryRow(ctx, "SELECT count(*) FROM v1_retry_queue_item WHERE task_id = $1", old.Id).Scan(&pending)
					require.NoError(t, err)
					require.Zero(t, pending, "replacement must remove the retry before asynchronous cancellation delivery")
					_, err = conf.V1.Tasks().CancelTasks(ctx, tenantID, []repo.TaskIdInsertedAtRetryCount{*replaced.Cancelled[0].TaskIdInsertedAtRetryCount})
					require.NoError(t, err)
					// A different key can still fail, wait in backoff, and retry normally.
					independent := create("other-key")
					admitted, err = concurrency.RunConcurrencyStrategy(ctx, tenantID, child)
					require.NoError(t, err)
					require.Len(t, admitted.Queued, 1)
					_, err = conf.Pool.Exec(ctx, `INSERT INTO v1_task_runtime (task_id, task_inserted_at, retry_count, tenant_id, timeout_at) VALUES ($1, $2, 0, $3, NOW() + INTERVAL '5 minutes')`, independent.Id, independent.InsertedAt, tenantID)
					require.NoError(t, err)
					failed, err = conf.V1.Tasks().FailTasks(ctx, tenantID, []repo.FailTaskOpts{{TaskIdInsertedAtRetryCount: independent, IsAppError: true}})
					require.NoError(t, err)
					require.Len(t, failed.RetriedTasks, 1)
					waiting, err := concurrency.RunConcurrencyStrategy(ctx, tenantID, child)
					require.NoError(t, err)
					require.Empty(t, waiting.Queued, "the retained slot must not admit a retry before its deadline")
					require.Empty(t, waiting.Cancelled)
					_, err = conf.Pool.Exec(ctx, "UPDATE v1_retry_queue_item SET retry_after = NOW() - INTERVAL '1 second' WHERE task_id = $1", independent.Id)
					require.NoError(t, err)
					_, err = conf.Pool.Exec(ctx, "DELETE FROM v1_retry_queue_item WHERE task_id = $1", independent.Id)
					require.NoError(t, err)
					admitted, err = concurrency.RunConcurrencyStrategy(ctx, tenantID, child)
					require.NoError(t, err)
					require.Len(t, admitted.Queued, 1)
					require.Equal(t, independent.Id, admitted.Queued[0].Id)
					require.Equal(t, int32(1), admitted.Queued[0].RetryCount)
					return nil
				})
			})
		}
	}
}
