-- name: ListTenantControls :many
SELECT *
FROM v1_tenant_control;

-- name: SetTenantControl :one
INSERT INTO v1_tenant_control (tenant_id, ingest_per_second, reason)
VALUES (@tenantId::uuid, @ingestPerSecond::integer, @reason::text)
ON CONFLICT (tenant_id) DO UPDATE
SET
    ingest_per_second = EXCLUDED.ingest_per_second,
    reason = EXCLUDED.reason,
    updated_at = NOW()
RETURNING *;

-- name: ClearTenantControl :execrows
DELETE FROM v1_tenant_control
WHERE tenant_id = @tenantId::uuid;
