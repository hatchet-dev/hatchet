-- name: ListTenantControls :many
SELECT *
FROM v1_tenant_control;

-- name: LockTenantControl :one
SELECT *
FROM v1_tenant_control
WHERE tenant_id = @tenantId::uuid
FOR UPDATE;

-- name: SetTenantControl :one
-- paused_at is set when paused turns true and kept while it stays true.
INSERT INTO v1_tenant_control (tenant_id, paused, paused_at, db_conn_percent, db_time_percent, ingest_per_second, reason)
VALUES (
    @tenantId::uuid,
    @paused::boolean,
    CASE WHEN @paused::boolean THEN NOW() END,
    sqlc.narg('dbConnPercent')::integer,
    sqlc.narg('dbTimePercent')::integer,
    sqlc.narg('ingestPerSecond')::integer,
    @reason::text
)
ON CONFLICT (tenant_id) DO UPDATE
SET
    paused = EXCLUDED.paused,
    paused_at = CASE WHEN EXCLUDED.paused THEN COALESCE(v1_tenant_control.paused_at, EXCLUDED.paused_at) END,
    db_conn_percent = EXCLUDED.db_conn_percent,
    db_time_percent = EXCLUDED.db_time_percent,
    ingest_per_second = EXCLUDED.ingest_per_second,
    reason = EXCLUDED.reason,
    updated_at = NOW()
RETURNING *;

-- name: ClearTenantControl :execrows
DELETE FROM v1_tenant_control
WHERE tenant_id = @tenantId::uuid;

-- name: ShiftScheduleTimeouts :exec
-- Moves the schedule deadline of every queued item of the tenant forward by the time since
-- pausedAt, so the time a tenant spends paused does not count toward its schedule timeouts.
WITH queue_items AS (
    UPDATE v1_queue_item
    SET schedule_timeout_at = schedule_timeout_at + (NOW() - @pausedAt::timestamptz)
    WHERE tenant_id = @tenantId::uuid AND schedule_timeout_at IS NOT NULL
), concurrency_slots AS (
    UPDATE v1_concurrency_slot
    SET schedule_timeout_at = schedule_timeout_at + (NOW() - @pausedAt::timestamptz)
    WHERE tenant_id = @tenantId::uuid
), batched_queue_items AS (
    UPDATE v1_batched_queue_item
    SET schedule_timeout_at = schedule_timeout_at + (NOW() - @pausedAt::timestamptz)
    WHERE tenant_id = @tenantId::uuid AND schedule_timeout_at IS NOT NULL
)
UPDATE v1_rate_limited_queue_items
SET schedule_timeout_at = schedule_timeout_at + (NOW() - @pausedAt::timestamptz)
WHERE tenant_id = @tenantId::uuid AND schedule_timeout_at IS NOT NULL;
