//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRetriedRegularParentSpawnsTheSameChildWhenDeterministicIdsAreToggled(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repos := newUserEventScopeTestRepositories(t, pool)
	workflows := &workflowRepository{sharedRepository: repos.shared}
	triggers := newTriggerRepository(repos.shared)

	_, err := workflows.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts("regular-parent", "v1", nil))
	require.NoError(t, err)

	_, err = workflows.PutWorkflowVersion(ctx, internalTenantId, minimalWorkflowOpts("regular-child", "v1", nil))
	require.NoError(t, err)

	parents, _, _, _, err := triggers.TriggerFromWorkflowNames(ctx, internalTenantId, []*WorkflowNameTriggerOpts{{
		TriggerTaskData: &TriggerTaskData{WorkflowName: "regular-parent", Data: []byte(`{}`)},
		ExternalId:      uuid.New(),
	}})
	require.NoError(t, err)
	require.Len(t, parents, 1)

	parent := parents[0]
	parentInsertedAt := parent.InsertedAt.Time
	childIndex := int64(0)

	requestChild := func() *WorkflowNameTriggerOpts {
		opt := &WorkflowNameTriggerOpts{TriggerTaskData: &TriggerTaskData{
			WorkflowName:         "regular-child",
			Data:                 []byte(`{}`),
			ParentExternalId:     &parent.ExternalID,
			ParentTaskId:         &parent.ID,
			ParentTaskInsertedAt: &parentInsertedAt,
			ChildIndex:           &childIndex,
		}}

		require.NoError(t, triggers.PopulateExternalIdsForWorkflow(ctx, internalTenantId, []*WorkflowNameTriggerOpts{opt}))

		return opt
	}

	triggerRequested := func(opts ...*WorkflowNameTriggerOpts) int {
		children, _, _, _, err := triggers.TriggerFromWorkflowNames(ctx, internalTenantId, opts)
		require.NoError(t, err)

		return len(children)
	}

	repos.shared.childRunDirectSpawn = true

	firstRequest, requestMadeBeforeTheFirstWasTriggered := requestChild(), requestChild()
	require.False(t, firstRequest.ShouldSkip)
	require.False(t, requestMadeBeforeTheFirstWasTriggered.ShouldSkip)
	require.Equal(t, firstRequest.ExternalId, requestMadeBeforeTheFirstWasTriggered.ExternalId)
	require.Equal(t, 1, triggerRequested(firstRequest, requestMadeBeforeTheFirstWasTriggered))

	retriedRequest := requestChild()
	require.True(t, retriedRequest.ShouldSkip)
	require.Equal(t, firstRequest.ExternalId, retriedRequest.ExternalId)

	repos.shared.childRunDirectSpawn = false

	retriedRequestAfterDisablement := requestChild()
	require.Equal(t, firstRequest.ExternalId, retriedRequestAfterDisablement.ExternalId)
	require.Equal(t, 0, triggerRequested(retriedRequestAfterDisablement))
}
