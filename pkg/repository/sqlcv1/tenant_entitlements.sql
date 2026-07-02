-- name: GetTenantEntitlement :one
SELECT *
FROM tenant_entitlement
WHERE tenant_id = @tenantId::uuid;

-- name: UpsertTenantEntitlement :one
INSERT INTO tenant_entitlement (tenant_id, audit_logs, prometheus_metrics)
VALUES (@tenantId::uuid, @auditLogs::boolean, @prometheusMetrics::boolean)
ON CONFLICT (tenant_id) DO UPDATE
SET
    audit_logs = EXCLUDED.audit_logs,
    prometheus_metrics = EXCLUDED.prometheus_metrics,
    updated_at = NOW()
RETURNING *;

-- name: AnyTenantHasAuditLogs :one
SELECT EXISTS (
    SELECT 1
    FROM tenant_entitlement
    WHERE
        tenant_id = ANY(@tenantIds::uuid[])
        AND audit_logs = TRUE
) AS has_audit_logs;
