-- name: GetTenantEntitlement :one
SELECT *
FROM tenant_entitlement
WHERE tenant_id = @tenantId::uuid;

-- name: UpsertTenantEntitlement :one
INSERT INTO tenant_entitlement (tenant_id, audit_logs, prometheus_metrics, strict_additional_metadata_filters, dag_operator)
VALUES (@tenantId::uuid, @auditLogs::boolean, @prometheusMetrics::boolean, @strictAdditionalMetadataFilters::boolean, @dagOperator::boolean)
ON CONFLICT (tenant_id) DO UPDATE
SET
    audit_logs = EXCLUDED.audit_logs,
    prometheus_metrics = EXCLUDED.prometheus_metrics,
    strict_additional_metadata_filters = EXCLUDED.strict_additional_metadata_filters,
    dag_operator = EXCLUDED.dag_operator,
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
