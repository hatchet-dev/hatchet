package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(up20260203170921, down20260203170921)
}

const backfillSlotsBatchSize = 10_000

var zeroUUID = uuid.Nil

func up20260203170921(ctx context.Context, db *sql.DB) error {
	if err := backfillWorkerSlotConfigs(ctx, db); err != nil {
		return err
	}

	if err := backfillStepSlotRequests(ctx, db); err != nil {
		return err
	}

	if err := backfillTaskRuntimeSlots(ctx, db); err != nil {
		return err
	}

	return nil
}

// down20260203170921 is intentionally a no-op.
//
// By the time this migration runs, new services may already be writing to these
// tables. Deleting rows here would risk removing valid post-cutover data.
func down20260203170921(ctx context.Context, db *sql.DB) error {
	return nil
}

func backfillWorkerSlotConfigs(ctx context.Context, db *sql.DB) error {
	lastTenantID := zeroUUID
	lastWorkerID := zeroUUID

	for {
		var (
			n            int
			nextTenantID uuid.NullUUID
			nextWorkerID uuid.NullUUID
		)

		err := db.QueryRowContext(ctx, `
WITH batch AS (
	SELECT
		"tenantId" AS tenant_id,
		"id" AS worker_id,
		"maxRuns" AS max_units
	FROM "Worker"
	WHERE "maxRuns" IS NOT NULL
	  AND ("tenantId", "id") > ($1::uuid, $2::uuid)
	ORDER BY "tenantId", "id"
	LIMIT $3
),
ins AS (
	INSERT INTO v1_worker_slot_config (tenant_id, worker_id, slot_type, max_units)
	SELECT
		tenant_id,
		worker_id,
		'default'::text,
		max_units
	FROM batch
	ON CONFLICT (tenant_id, worker_id, slot_type) DO NOTHING
)
SELECT
	(SELECT COUNT(*) FROM batch) AS n,
	(SELECT tenant_id FROM batch ORDER BY tenant_id DESC, worker_id DESC LIMIT 1) AS last_tenant_id,
	(SELECT worker_id FROM batch ORDER BY tenant_id DESC, worker_id DESC LIMIT 1) AS last_worker_id;
`, lastTenantID, lastWorkerID, backfillSlotsBatchSize).Scan(&n, &nextTenantID, &nextWorkerID)
		if err != nil {
			return fmt.Errorf("backfill v1_worker_slot_config: %w", err)
		}

		if n == 0 {
			return nil
		}

		if !nextTenantID.Valid || !nextWorkerID.Valid {
			return fmt.Errorf("backfill v1_worker_slot_config: expected last keys for non-empty batch")
		}

		lastTenantID = nextTenantID.UUID
		lastWorkerID = nextWorkerID.UUID
	}
}

func backfillStepSlotRequests(ctx context.Context, db *sql.DB) error {
	lastTenantID := zeroUUID
	lastStepID := zeroUUID

	for {
		var (
			n          int
			nextTenant uuid.NullUUID
			nextStep   uuid.NullUUID
		)

		err := db.QueryRowContext(ctx, `
WITH batch AS (
	SELECT
		"tenantId" AS tenant_id,
		"id" AS step_id,
		"isDurable" AS is_durable
	FROM "Step"
	WHERE ("tenantId", "id") > ($1::uuid, $2::uuid)
	ORDER BY "tenantId", "id"
	LIMIT $3
),
ins AS (
	INSERT INTO v1_step_slot_request (tenant_id, step_id, slot_type, units)
	SELECT
		tenant_id,
		step_id,
		CASE WHEN is_durable THEN 'durable'::text ELSE 'default'::text END,
		1
	FROM batch
	ON CONFLICT (tenant_id, step_id, slot_type) DO NOTHING
)
SELECT
	(SELECT COUNT(*) FROM batch) AS n,
	(SELECT tenant_id FROM batch ORDER BY tenant_id DESC, step_id DESC LIMIT 1) AS last_tenant_id,
	(SELECT step_id FROM batch ORDER BY tenant_id DESC, step_id DESC LIMIT 1) AS last_step_id;
`, lastTenantID, lastStepID, backfillSlotsBatchSize).Scan(&n, &nextTenant, &nextStep)
		if err != nil {
			return fmt.Errorf("backfill v1_step_slot_request: %w", err)
		}

		if n == 0 {
			return nil
		}

		if !nextTenant.Valid || !nextStep.Valid {
			return fmt.Errorf("backfill v1_step_slot_request: expected last keys for non-empty batch")
		}

		lastTenantID = nextTenant.UUID
		lastStepID = nextStep.UUID
	}
}

func backfillTaskRuntimeSlots(ctx context.Context, db *sql.DB) error {
	var (
		lastTaskID         int64
		lastTaskInsertedAt = time.Unix(0, 0).UTC()
		lastRetryCount     int32
	)

	for {
		var (
			n              int
			nextTaskID     sql.NullInt64
			nextInsertedAt sql.NullTime
			nextRetry      sql.NullInt32
		)

		err := db.QueryRowContext(ctx, `
WITH batch AS (
	SELECT
		tenant_id,
		task_id,
		task_inserted_at,
		retry_count,
		worker_id
	FROM v1_task_runtime
	WHERE worker_id IS NOT NULL
	  AND (task_id, task_inserted_at, retry_count) > ($1::bigint, $2::timestamptz, $3::int)
	ORDER BY task_id, task_inserted_at, retry_count
	LIMIT $4
),
ins AS (
	INSERT INTO v1_task_runtime_slot (
		tenant_id,
		task_id,
		task_inserted_at,
		retry_count,
		worker_id,
		slot_type,
		units
	)
	SELECT
		tenant_id,
		task_id,
		task_inserted_at,
		retry_count,
		worker_id,
		'default'::text,
		1
	FROM batch
	ON CONFLICT (task_id, task_inserted_at, retry_count, slot_type) DO NOTHING
)
SELECT
	(SELECT COUNT(*) FROM batch) AS n,
	(SELECT task_id FROM batch ORDER BY task_id DESC, task_inserted_at DESC, retry_count DESC LIMIT 1) AS last_task_id,
	(SELECT task_inserted_at FROM batch ORDER BY task_id DESC, task_inserted_at DESC, retry_count DESC LIMIT 1) AS last_task_inserted_at,
	(SELECT retry_count FROM batch ORDER BY task_id DESC, task_inserted_at DESC, retry_count DESC LIMIT 1) AS last_retry_count;
`, lastTaskID, lastTaskInsertedAt, lastRetryCount, backfillSlotsBatchSize).Scan(&n, &nextTaskID, &nextInsertedAt, &nextRetry)
		if err != nil {
			return fmt.Errorf("backfill v1_task_runtime_slot: %w", err)
		}

		if n == 0 {
			return nil
		}

		if !nextTaskID.Valid || !nextInsertedAt.Valid || !nextRetry.Valid {
			return fmt.Errorf("backfill v1_task_runtime_slot: expected last keys for non-empty batch")
		}

		lastTaskID = nextTaskID.Int64
		lastTaskInsertedAt = nextInsertedAt.Time
		lastRetryCount = nextRetry.Int32
	}
}
