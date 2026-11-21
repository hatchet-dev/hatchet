//go:build !e2e && !load && !rampup && !integration

package repository

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// backlogSeedStatements insert $2 rows for tenant $1 into one table, with task ids starting
// after $3. Only the columns the tables require are set.
var backlogSeedStatements = map[string]string{
	"v1_queue_item": `
		INSERT INTO v1_queue_item (tenant_id, queue, task_id, task_inserted_at, external_id, action_id, step_id, workflow_id, workflow_run_id, sticky)
		SELECT $1, 'default', $3::bigint + i, NOW(), gen_random_uuid(), 'test:run', gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), 'NONE'
		FROM generate_series(1, $2::bigint) i`,
	"v1_rate_limited_queue_items": `
		INSERT INTO v1_rate_limited_queue_items (requeue_after, tenant_id, queue, task_id, task_inserted_at, external_id, action_id, step_id, workflow_id, workflow_run_id, sticky)
		SELECT NOW() + INTERVAL '1 minute', $1, 'default', $3::bigint + i, NOW(), gen_random_uuid(), 'test:run', gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), 'NONE'
		FROM generate_series(1, $2::bigint) i`,
	"v1_batched_queue_item": `
		INSERT INTO v1_batched_queue_item (tenant_id, queue, task_id, task_inserted_at, external_id, action_id, step_id, workflow_id, workflow_run_id, sticky, batch_key)
		SELECT $1, 'default', $3::bigint + i, NOW(), gen_random_uuid(), 'test:run', gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), 'NONE', 'batch'
		FROM generate_series(1, $2::bigint) i`,
	"v1_paused_workflow_queue_item": `
		INSERT INTO v1_paused_workflow_queue_item (tenant_id, queue, task_id, task_inserted_at, external_id, action_id, step_id, workflow_id, workflow_run_id, sticky)
		SELECT $1, 'default', $3::bigint + i, NOW(), gen_random_uuid(), 'test:run', gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), 'NONE'
		FROM generate_series(1, $2::bigint) i`,
	"v1_retry_queue_item": `
		INSERT INTO v1_retry_queue_item (task_id, task_inserted_at, task_retry_count, retry_after, tenant_id)
		SELECT $3::bigint + i, NOW(), 0, NOW() + INTERVAL '1 minute', $1
		FROM generate_series(1, $2::bigint) i`,
	"v1_concurrency_slot unfilled": concurrencySlotSeedStatement(false),
	"v1_concurrency_slot filled":   concurrencySlotSeedStatement(true),
	"v1_workflow_concurrency_slot": `
		INSERT INTO v1_workflow_concurrency_slot (sort_id, tenant_id, workflow_id, workflow_version_id, workflow_run_id, strategy_id, key)
		SELECT $3::bigint + i, $1, gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), 1, 'key'
		FROM generate_series(1, $2::bigint) i`,
}

func concurrencySlotSeedStatement(filled bool) string {
	isFilled := "FALSE"
	if filled {
		isFilled = "TRUE"
	}

	return `
		INSERT INTO v1_concurrency_slot (task_id, task_inserted_at, task_retry_count, external_id, tenant_id, workflow_id, workflow_version_id, workflow_run_id, strategy_id, key, is_filled, queue_to_notify, schedule_timeout_at)
		SELECT $3::bigint + i, NOW(), 0, gen_random_uuid(), $1, gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), 1, 'key', ` + isFilled + `, 'default', NOW() + INTERVAL '5 minutes'
		FROM generate_series(1, $2::bigint) i`
}

var backlogSeedTaskIds atomic.Int64

func seedBacklogRows(t *testing.T, ctx context.Context, pool *pgxpool.Pool, table string, tenantId uuid.UUID, n int64) {
	t.Helper()

	stmt, ok := backlogSeedStatements[table]
	require.True(t, ok, "no seed statement for %s", table)

	firstTaskId := backlogSeedTaskIds.Add(n) - n

	_, err := pool.Exec(ctx, stmt, tenantId, n, firstTaskId)
	require.NoError(t, err)
}

func TestGetTenantBacklogs(t *testing.T) {
	pool, cleanup := setupPostgresWithMigration(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	repo := createTaskRepository(pool)

	t.Run("each waiting table is reported as its own state", func(t *testing.T) {
		for table, want := range map[string]TenantBacklog{
			"v1_queue_item":                 {Queued: 3},
			"v1_concurrency_slot unfilled":  {Concurrency: 3},
			"v1_rate_limited_queue_items":   {RateLimited: 3},
			"v1_batched_queue_item":         {Batched: 3},
			"v1_paused_workflow_queue_item": {Paused: 3},
			"v1_retry_queue_item":           {Retry: 3},
		} {
			tenantId := uuid.New()
			seedBacklogRows(t, ctx, pool, table, tenantId, 3)

			backlogs, err := repo.GetTenantBacklogs(ctx, []uuid.UUID{tenantId}, 100)
			require.NoError(t, err)
			require.Equal(t, map[uuid.UUID]TenantBacklog{tenantId: want}, backlogs, table)
		}
	})

	t.Run("filled slots and workflow slots do not count", func(t *testing.T) {
		tenantId := uuid.New()
		seedBacklogRows(t, ctx, pool, "v1_concurrency_slot filled", tenantId, 4)
		seedBacklogRows(t, ctx, pool, "v1_workflow_concurrency_slot", tenantId, 5)
		seedBacklogRows(t, ctx, pool, "v1_concurrency_slot unfilled", tenantId, 2)

		backlogs, err := repo.GetTenantBacklogs(ctx, []uuid.UUID{tenantId}, 100)
		require.NoError(t, err)
		require.Equal(t, map[uuid.UUID]TenantBacklog{tenantId: {Concurrency: 2}}, backlogs)
	})

	t.Run("several tenants in one call, an empty one maps to zero", func(t *testing.T) {
		busy := uuid.New()
		other := uuid.New()
		empty := uuid.New()

		seedBacklogRows(t, ctx, pool, "v1_queue_item", busy, 5)
		seedBacklogRows(t, ctx, pool, "v1_retry_queue_item", busy, 1)
		seedBacklogRows(t, ctx, pool, "v1_paused_workflow_queue_item", other, 2)

		backlogs, err := repo.GetTenantBacklogs(ctx, []uuid.UUID{busy, other, empty}, 100)
		require.NoError(t, err)
		require.Equal(t, map[uuid.UUID]TenantBacklog{
			busy:  {Queued: 5, Retry: 1},
			other: {Paused: 2},
			empty: {},
		}, backlogs)
	})

	t.Run("the ceiling clamps each state", func(t *testing.T) {
		tenantId := uuid.New()
		seedBacklogRows(t, ctx, pool, "v1_queue_item", tenantId, 5)
		seedBacklogRows(t, ctx, pool, "v1_concurrency_slot unfilled", tenantId, 2)

		for ceiling, want := range map[int64]TenantBacklog{
			1:   {Queued: 1, Concurrency: 1},
			3:   {Queued: 3, Concurrency: 2},
			100: {Queued: 5, Concurrency: 2},
		} {
			backlogs, err := repo.GetTenantBacklogs(ctx, []uuid.UUID{tenantId}, ceiling)
			require.NoError(t, err)
			require.Equal(t, want, backlogs[tenantId], "ceiling %d", ceiling)
		}
	})

	t.Run("tenant lists longer than one statement are merged", func(t *testing.T) {
		tenantIds := make([]uuid.UUID, 0, tenantBacklogsChunkSize+1)
		for range tenantBacklogsChunkSize {
			tenantIds = append(tenantIds, uuid.New())
		}

		last := uuid.New()
		tenantIds = append(tenantIds, last)
		seedBacklogRows(t, ctx, pool, "v1_queue_item", last, 4)

		backlogs, err := repo.GetTenantBacklogs(ctx, tenantIds, 100)
		require.NoError(t, err)
		require.Len(t, backlogs, len(tenantIds))
		require.Equal(t, TenantBacklog{Queued: 4}, backlogs[last])
		require.Equal(t, TenantBacklog{}, backlogs[tenantIds[0]])
	})

	t.Run("every count walks a tenant_id-leading index and the slot count skips filled slots", func(t *testing.T) {
		tenantId := uuid.New()
		seedBacklogRows(t, ctx, pool, "v1_concurrency_slot filled", tenantId, 5_000)
		seedBacklogRows(t, ctx, pool, "v1_concurrency_slot unfilled", tenantId, 5)

		for _, table := range []string{"v1_queue_item", "v1_rate_limited_queue_items", "v1_batched_queue_item", "v1_paused_workflow_queue_item", "v1_retry_queue_item"} {
			seedBacklogRows(t, ctx, pool, table, tenantId, 50)
		}

		_, err := pool.Exec(ctx, "ANALYZE v1_queue_item, v1_concurrency_slot, v1_rate_limited_queue_items, v1_batched_queue_item, v1_paused_workflow_queue_item, v1_retry_queue_item")
		require.NoError(t, err)

		// a table with two suitable indexes may use either
		want := map[string][]string{
			"v1_queue_item":                 {"v1_queue_item_list_idx"},
			"v1_concurrency_slot":           {"v1_concurrency_slot_timeout_idx", "v1_concurrency_slot_workflow_unfilled_idx"},
			"v1_rate_limited_queue_items":   {"v1_rate_limited_queue_items_tenant_requeue_after_idx"},
			"v1_batched_queue_item":         {"v1_batched_queue_item_step_batch_id_idx", "v1_batched_queue_item_step_priority_idx"},
			"v1_paused_workflow_queue_item": {"v1_paused_workflow_queue_item_tenant_idx"},
			"v1_retry_queue_item":           {"v1_retry_queue_item_tenant_id_retry_after_idx"},
		}

		scans := tenantBacklogsPlanScans(t, ctx, pool, tenantId)
		require.Len(t, scans, len(want), "plan scans %v", scans)

		for table, index := range scans {
			require.Contains(t, want[table], index, "%s is counted with index %q", table, index)

			var leading, predicate string
			require.NoError(t, pool.QueryRow(ctx, `
				SELECT a.attname, COALESCE(pg_get_expr(i.indpred, i.indrelid), '')
				FROM pg_index i
				JOIN pg_class c ON c.oid = i.indexrelid
				JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = i.indkey[0]
				WHERE c.relname = $1`, index).Scan(&leading, &predicate))

			require.Equal(t, "tenant_id", leading, "%s leads with %s", index, leading)

			if table == "v1_concurrency_slot" {
				require.Equal(t, "(is_filled = false)", predicate, "%s must cover only unfilled slots", index)
			}
		}
	})
}

// errExplained stops GetTenantBacklogs once explainingDBTX has the plan.
var errExplained = errors.New("explained")

// explainingDBTX answers a query with an error after storing its JSON plan, so a test can
// explain a generated query with the arguments the repository passes it.
type explainingDBTX struct {
	sqlcv1.DBTX

	plan []byte
}

func (e *explainingDBTX) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if err := e.DBTX.QueryRow(ctx, "EXPLAIN (FORMAT JSON) "+sql, args...).Scan(&e.plan); err != nil {
		return nil, err
	}

	return nil, errExplained
}

type planNode struct {
	RelationName string     `json:"Relation Name"`
	IndexName    string     `json:"Index Name"`
	Plans        []planNode `json:"Plans"`
}

// tenantBacklogsPlanScans plans GetTenantBacklogs for one tenant under the planner settings it
// runs with, and returns the index each table is scanned with ("" for a scan without one).
func tenantBacklogsPlanScans(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantId uuid.UUID) map[string]string {
	t.Helper()

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx) // nolint: errcheck

	_, err = tx.Exec(ctx, tenantBacklogsPlannerSettings)
	require.NoError(t, err)

	db := &explainingDBTX{DBTX: tx}
	_, err = sqlcv1.New().GetTenantBacklogs(ctx, db, sqlcv1.GetTenantBacklogsParams{
		Tenantids: []uuid.UUID{tenantId},
		Ceiling:   250_000,
	})
	require.ErrorIs(t, err, errExplained)

	var explained []struct {
		Plan planNode `json:"Plan"`
	}
	require.NoError(t, json.Unmarshal(db.plan, &explained))
	require.Len(t, explained, 1)

	scans := make(map[string]string)

	var walk func(n planNode)
	walk = func(n planNode) {
		if n.RelationName != "" {
			scans[n.RelationName] = n.IndexName
		}

		for _, child := range n.Plans {
			walk(child)
		}
	}
	walk(explained[0].Plan)

	return scans
}
