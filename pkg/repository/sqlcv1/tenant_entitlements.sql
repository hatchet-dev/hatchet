-- name: GetTenantEntitlement :one
SELECT *
FROM tenant_entitlement
WHERE tenant_id = @tenantId::uuid;

-- name: UpsertTenantEntitlement :one
INSERT INTO tenant_entitlement (tenant_id, audit_logs, prometheus_metrics, strict_additional_metadata_filters, dag_operator, durable_streams, serverless_operator, sandbox_operator)
VALUES (@tenantId::uuid, @auditLogs::boolean, @prometheusMetrics::boolean, @strictAdditionalMetadataFilters::boolean, @dagOperator::boolean, @durableStreams::boolean, @serverlessOperator::boolean, @sandboxOperator::boolean)
ON CONFLICT (tenant_id) DO UPDATE
SET
    audit_logs = EXCLUDED.audit_logs,
    prometheus_metrics = EXCLUDED.prometheus_metrics,
    strict_additional_metadata_filters = EXCLUDED.strict_additional_metadata_filters,
    dag_operator = EXCLUDED.dag_operator,
    durable_streams = EXCLUDED.durable_streams,
    serverless_operator = EXCLUDED.serverless_operator,
    sandbox_operator = EXCLUDED.sandbox_operator,
    updated_at = NOW()
RETURNING *;

-- name: UpdateTenantLabsEntitlements :one
INSERT INTO tenant_entitlement (tenant_id, durable_streams, serverless_operator, sandbox_operator)
VALUES (
    @tenantId::uuid,
    COALESCE(sqlc.narg('durableStreams')::boolean, FALSE),
    COALESCE(sqlc.narg('serverlessOperator')::boolean, FALSE),
    COALESCE(sqlc.narg('sandboxOperator')::boolean, FALSE)
)
ON CONFLICT (tenant_id) DO UPDATE
SET
    durable_streams = COALESCE(sqlc.narg('durableStreams')::boolean, tenant_entitlement.durable_streams),
    serverless_operator = COALESCE(sqlc.narg('serverlessOperator')::boolean, tenant_entitlement.serverless_operator),
    sandbox_operator = COALESCE(sqlc.narg('sandboxOperator')::boolean, tenant_entitlement.sandbox_operator),
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
