//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The trigger v1_step_slot_request_insert_trigger inserts a one-slot row ({default: 1}, or
// {durable: 1} for a durable step) on Step insert. This checks that a registered slot cost
// overwrites that row and persists.
func TestSlotCostPersistsThroughWritePath(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx := context.Background()
	repo := newWorkflowTestRepository(pool)

	// The overwrite only matters while this trigger exists, so require it to be present.
	requireTriggerExists(ctx, t, pool, "v1_step_slot_request_insert_trigger")

	desc := "slot-cost-writepath"
	opts := &CreateWorkflowVersionOpts{
		Name:        "slot-cost-writepath",
		Description: &desc,
		Tasks: []CreateStepOpts{
			{ReadableId: "heavy", Action: "integration:heavy", SlotRequests: map[string]int32{SlotTypeDefault: 5}},
			{ReadableId: "light", Action: "integration:light", SlotRequests: map[string]int32{SlotTypeDefault: 1}},
			{ReadableId: "plain", Action: "integration:plain"},
			{ReadableId: "dur", Action: "integration:dur", IsDurable: true},
		},
	}

	_, err := repo.PutWorkflowVersion(ctx, internalTenantId, opts)
	require.NoError(t, err)

	got := readSlotRequestsByStep(ctx, t, pool)

	assert.Equal(t, map[string]int32{SlotTypeDefault: 5}, got["heavy"], "explicit slot cost 5 should persist")
	assert.Equal(t, map[string]int32{SlotTypeDefault: 1}, got["light"], "explicit slot cost 1 should persist")
	assert.Equal(t, map[string]int32{SlotTypeDefault: 1}, got["plain"], "no explicit cost stays at one default slot")
	assert.Equal(t, map[string]int32{SlotTypeDurable: 1}, got["dur"], "durable default is unchanged")
}

func requireTriggerExists(ctx context.Context, t *testing.T, pool *pgxpool.Pool, name string) {
	t.Helper()

	var exists bool
	err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = $1)`, name).Scan(&exists)
	require.NoError(t, err)
	require.True(t, exists, "test premise requires the compatibility trigger %q to be present", name)
}

func readSlotRequestsByStep(ctx context.Context, t *testing.T, pool *pgxpool.Pool) map[string]map[string]int32 {
	t.Helper()

	rows, err := pool.Query(ctx, `
		SELECT s."readableId", r.slot_type, r.units
		FROM v1_step_slot_request r
		JOIN "Step" s ON s.id = r.step_id
		WHERE r.tenant_id = $1
	`, internalTenantId)
	require.NoError(t, err)
	defer rows.Close()

	out := map[string]map[string]int32{}
	for rows.Next() {
		var readableID, slotType string
		var units int32
		require.NoError(t, rows.Scan(&readableID, &slotType, &units))
		if out[readableID] == nil {
			out[readableID] = map[string]int32{}
		}
		out[readableID][slotType] = units
	}
	require.NoError(t, rows.Err())

	return out
}
