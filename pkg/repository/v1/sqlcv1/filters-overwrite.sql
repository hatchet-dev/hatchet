-- name: ListFilters :many
WITH inputs AS (
    SELECT
        UNNEST(@tenantIds::UUID[]) AS tenant_id,
        UNNEST(@workflowIds::UUID[]) AS workflow_id,
        UNNEST(@workflowVersionIds::UUID[]) AS workflow_version_id,
        -- NOTE: this is nullable, so sqlc doesn't support casting to a type
        UNNEST(@resourceHints::TEXT[]) AS resource_hint
)

SELECT f.*
FROM v1_filter f
JOIN inputs i ON (f.tenant_id, f.workflow_id, f.resource_hint) = (i.tenant_id, i.workflow_id, i.resource_hint)
;
