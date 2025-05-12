-- name: CreateFilter :one
INSERT INTO v1_filter (
    tenant_id,
    workflow_id,
    resource_hint,
    expression,
    payload
) VALUES (
    @tenantId::UUID,
    @workflowId::UUID,
    @resourceHint::TEXT,
    @expression::TEXT,
    @payload::JSONB
)
RETURNING *
;

-- name: ListFilters :many
WITH inputs AS (
    SELECT
        UNNEST(@tenantIds::UUID[]) AS tenant_id,
        UNNEST(@workflowIds::UUID[]) AS workflow_id,
        UNNEST(@workflowVersionIds::UUID[]) AS workflow_version_id,
        UNNEST(@resourceHints::TEXT[]) AS resource_hint
)

SELECT f.*
FROM v1_filter f
JOIN inputs i ON (f.tenant_id, f.workflow_id, f.workflow_version_id, f.resource_hint) = (i.tenant_id, i.workflow_id, i.workflow_version_id, i.resource_hint)
;