-- name: ListFilters :many
WITH inputs AS (
    SELECT
        UNNEST(@tenantIds::UUID[]) AS tenant_id,
        UNNEST(@workflowIds::UUID[]) AS workflow_id,
        -- NOTE: this is nullable, so sqlc doesn't support casting to a type
        UNNEST(@scopes::TEXT[]) AS scope
)

SELECT f.*
FROM v1_filter f
JOIN inputs i ON (f.tenant_id, f.workflow_id, f.scope) = (i.tenant_id, i.workflow_id, i.scope)
LIMIT COALESCE(sqlc.narg('filterLimit')::BIGINT, 20000)
OFFSET COALESCE(sqlc.narg('filterOffset')::BIGINT, 0)
;
