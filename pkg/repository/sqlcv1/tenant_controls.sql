-- name: ListTenantControls :many
SELECT *
FROM v1_tenant_control;

-- name: SetTenantControl :one
-- v1_tenant_control_paused_at_trigger keeps paused_at
INSERT INTO v1_tenant_control (tenant_id, ingest_per_second, paused, reason)
VALUES (@tenantId::uuid, sqlc.narg('ingestPerSecond')::integer, @paused::boolean, @reason::text)
ON CONFLICT (tenant_id) DO UPDATE
SET
    ingest_per_second = EXCLUDED.ingest_per_second,
    paused = EXCLUDED.paused,
    reason = EXCLUDED.reason,
    updated_at = NOW()
RETURNING *;

-- name: ClearTenantControl :execrows
DELETE FROM v1_tenant_control
WHERE tenant_id = @tenantId::uuid;
