//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestPutDynamicDagWorkflowVersionCopiesRegisteredTasks(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo := newWorkflowTestRepository(pool)
	repo.tenantIdWorkflowNameCache = expirable.NewLRU(10, func(string, *sqlcv1.ListWorkflowsByNamesRow) {}, time.Millisecond)
	repo.stepsInWorkflowVersionCache = expirable.NewLRU(10, func(uuid.UUID, []*sqlcv1.ListStepsByWorkflowVersionIdsRow) {}, time.Millisecond)
	repo.stepIdMatchConditionsCache = expirable.NewLRU(10, func(uuid.UUID, []*sqlcv1.V1StepMatchCondition) {}, time.Millisecond)

	retries := 3
	orGroup := uuid.New()
	eventKey := "user:create"
	maxRuns := int32(2)

	_, err := repo.PutWorkflowVersion(ctx, internalTenantId, &CreateWorkflowVersionOpts{
		Name:        "dyn-a",
		Concurrency: []CreateConcurrencyOpts{{Expression: "input.x", MaxRuns: &maxRuns}},
		Tasks: []CreateStepOpts{{
			ReadableId: "dyn-a", Action: "svc:dyn-a", Retries: &retries,
			TriggerConditions: []CreateStepMatchConditionOpt{
				{MatchConditionKind: "USER_EVENT", EventKey: &eventKey, ReadableDataKey: "e1", Action: "QUEUE", OrGroupId: orGroup},
				{MatchConditionKind: "USER_EVENT", EventKey: &eventKey, ReadableDataKey: "e2", Action: "QUEUE", OrGroupId: orGroup},
				{MatchConditionKind: "USER_EVENT", EventKey: &eventKey, ReadableDataKey: "e3", Action: "SKIP", OrGroupId: uuid.New()},
			},
		}},
	})
	require.NoError(t, err)

	_, err = repo.PutWorkflowVersion(ctx, internalTenantId, &CreateWorkflowVersionOpts{
		Name:  "dyn-b",
		Tasks: []CreateStepOpts{{ReadableId: "dyn-b", Action: "svc:dyn-b"}},
	})
	require.NoError(t, err)

	nodes := []DynamicDagNodeOpts{{TaskName: "dyn-a"}, {TaskName: "dyn-b", ParentTaskNames: []string{"dyn-a"}}}

	versionId, err := repo.PutDynamicDagWorkflowVersion(ctx, internalTenantId, "dyn-dag", nodes)
	require.NoError(t, err)

	steps, err := repo.ListStepsByWorkflowVersionId(ctx, internalTenantId, versionId)
	require.NoError(t, err)
	require.Len(t, steps, 3)

	byReadableId := map[string]*sqlcv1.ListStepsByWorkflowVersionIdsRow{}
	for _, s := range steps {
		byReadableId[s.ReadableId.String] = s
	}

	a, b := byReadableId["dyn-a"], byReadableId["dyn-b"]
	require.NotNil(t, a)
	require.NotNil(t, b)
	require.Equal(t, "svc:dyn-a", a.ActionId)
	require.EqualValues(t, 3, a.Retries)
	require.Equal(t, []uuid.UUID{a.ID}, b.Parents)

	conds, err := repo.ListStepMatchConditions(ctx, internalTenantId, []uuid.UUID{a.ID})
	require.NoError(t, err)
	require.Len(t, conds, 3)
	groups := map[uuid.UUID]int{}
	for _, c := range conds {
		groups[c.OrGroupID]++
	}
	require.Len(t, groups, 2)

	strategies, err := repo.queries.ListConcurrencyStrategiesByStepId(ctx, repo.pool.ForShared(), sqlcv1.ListConcurrencyStrategiesByStepIdParams{
		Tenantid: internalTenantId, Stepids: []uuid.UUID{a.ID},
	})
	require.NoError(t, err)
	require.Len(t, strategies, 1)

	again, err := repo.PutDynamicDagWorkflowVersion(ctx, internalTenantId, "dyn-dag", nodes)
	require.NoError(t, err)
	require.Equal(t, versionId, again)

	_, err = repo.PutDynamicDagWorkflowVersion(ctx, internalTenantId, "dyn-b", []DynamicDagNodeOpts{{TaskName: "dyn-a"}})
	require.ErrorContains(t, err, "is not a dynamic dag")
}
