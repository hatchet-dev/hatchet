//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Each engine caches the workflows an event key triggers, including that it triggers none. Two
// repositories on one database stand in for two engines: the one whose admin service registers a
// workflow, and another that handles events. The callback stands in for the pubsub fan-out.
func TestEventTriggersWorkflowCreatedRightAfterItsKeyWasLookedUp(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	t.Cleanup(cleanup)

	ctx := context.Background()

	const eventKey = "user:created"

	registering := newBatchTestRepository(pool).sharedRepository
	other := newBatchTestRepository(pool).sharedRepository

	workflows := &workflowRepository{sharedRepository: registering}
	workflows.RegisterVersionCreatedCallback(func(_ context.Context, tenantId uuid.UUID) {
		other.InvalidateEventKeyWorkflows(tenantId)
	})

	engines := map[string]*sharedRepository{"registering engine": registering, "other engine": other}

	pushEvent := func(shared *sharedRepository) []*V1TaskWithPayload {
		result, err := newTriggerRepository(shared).TriggerFromEvents(ctx, internalTenantId, []EventTriggerOpts{
			{
				ExternalId: uuid.New(),
				SeenAt:     time.Now(),
				Key:        eventKey,
				Data:       []byte(`{}`),
			},
		})
		require.NoError(t, err)

		return result.Tasks
	}

	for name, shared := range engines {
		require.Empty(t, pushEvent(shared), "%s: no workflow listens on the key yet", name)
	}

	opts := minimalWorkflowOpts("event-key-workflows", "v1", nil)
	opts.EventTriggers = []string{eventKey}

	_, err := workflows.PutWorkflowVersion(ctx, internalTenantId, opts)
	require.NoError(t, err)

	for name, shared := range engines {
		assert.Len(t, pushEvent(shared), 1, "%s should trigger the workflow registered after the last event", name)
	}
}
