//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
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

	announcements := 0

	workflows := &workflowRepository{sharedRepository: registering}
	workflows.RegisterVersionCreatedCallback(func(_ context.Context, tenantId uuid.UUID) {
		announcements++
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

	// Every worker start re-registers its workflows; an unchanged one must not flush every engine's cache.
	_, err = workflows.PutWorkflowVersion(ctx, internalTenantId, opts)
	require.NoError(t, err)
	assert.Equal(t, 1, announcements, "re-registering an unchanged workflow should not announce a new version")

	// A lookup whose query started before a registration commits must not cache what it read.
	const laterEventKey = "user:deleted"

	laterOpts := minimalWorkflowOpts("event-key-workflows-later", "v1", nil)
	laterOpts.EventTriggers = []string{laterEventKey}

	overlapping := registerDuringQuery{DBTX: pool, register: func() {
		_, err := workflows.PutWorkflowVersion(ctx, internalTenantId, laterOpts)
		require.NoError(t, err)
	}}

	rows, err := other.listWorkflowsForEvents(ctx, overlapping, internalTenantId, []string{laterEventKey})
	require.NoError(t, err)
	require.Empty(t, rows, "the lookup read Postgres before the workflow was committed")

	rows, err = other.listWorkflowsForEvents(ctx, pool, internalTenantId, []string{laterEventKey})
	require.NoError(t, err)
	assert.Len(t, rows, 1, "the next lookup should read Postgres rather than the overlapping lookup's result")
}

// registerDuringQuery registers a workflow after a query has started, so the query's result
// predates the commit.
type registerDuringQuery struct {
	sqlcv1.DBTX
	register func()
}

func (r registerDuringQuery) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	rows, err := r.DBTX.Query(ctx, sql, args...)
	r.register()

	return rows, err
}
