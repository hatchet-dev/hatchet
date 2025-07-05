-- name: CreateFilter :one
INSERT INTO v1_filter (
    tenant_id,
    workflow_id,
    scope,
    expression,
    payload,
    is_declarative
) VALUES (
    @tenantId::UUID,
    @workflowId::UUID,
    @scope::TEXT,
    @expression::TEXT,
    @payload::JSONB,
    false
)
RETURNING *;

-- name: BulkUpsertDeclarativeFilters :many
-- IMPORTANT: This query overwrites all existing declarative filters for a workflow.
-- it's intended to be used when the workflow version is created.
WITH inputs AS (
    SELECT
        UNNEST(@scopes::TEXT[]) AS scope,
        UNNEST(@expressions::TEXT[]) AS expression,
        UNNEST(@payloads::JSONB[]) AS payload
), deletions AS (
    DELETE FROM v1_filter
    WHERE
        tenant_id = @tenantId::UUID
        AND workflow_id = @workflowId::UUID
        AND is_declarative
)

INSERT INTO v1_filter (
    tenant_id,
    workflow_id,
    scope,
    expression,
    payload,
    is_declarative
)
SELECT
    @tenantId::UUID,
    @workflowId::UUID,
    scope,
    expression,
    payload,
    true
FROM inputs
RETURNING *;

-- name: DeleteFilter :one
DELETE FROM v1_filter
WHERE
    tenant_id = @tenantId::UUID
    AND id = @id::UUID
RETURNING *;

-- name: GetFilter :one
SELECT *
FROM v1_filter
WHERE
    tenant_id = @tenantId::UUID
    AND id = @id::UUID;

-- name: UpdateFilter :one
UPDATE v1_filter
SET
    scope = COALESCE(sqlc.narg('scope')::TEXT, scope),
    expression = COALESCE(sqlc.narg('expression')::TEXT, expression),
    payload = COALESCE(sqlc.narg('payload')::JSONB, payload),
    updated_at = NOW()
WHERE
    tenant_id = @tenantId::UUID
    AND id = @id::UUID
RETURNING *;

-- name: ListFilterCountsForWorkflows :many
WITH inputs AS (
    SELECT UNNEST(@workflowIds::UUID[]) AS workflow_id
)

SELECT workflow_id, COUNT(*)
FROM v1_filter
WHERE
    tenant_id = @tenantId::UUID
    AND workflow_id = ANY(@workflowIds::UUID[])
GROUP BY workflow_id
;

-- name: ListFiltersForEventTriggers :many
WITH inputs AS (
    SELECT
        UNNEST(sqlc.narg(workflowIds)::UUID[]) AS workflow_id,
        UNNEST(sqlc.narg(scopes)::TEXT[]) AS scope
)

SELECT f.*
FROM v1_filter f
JOIN inputs i ON (f.workflow_id, f.scope) = (i.workflow_id, i.scope)
WHERE f.tenant_id = @tenantId::UUID
ORDER BY f.id DESC
LIMIT COALESCE(sqlc.narg('filterLimit')::BIGINT, 20000)
OFFSET COALESCE(sqlc.narg('filterOffset')::BIGINT, 0)
;

-- name: ListFilters :many
SELECT *
FROM v1_filter
WHERE
    tenant_id = @tenantId::UUID
    AND (
        sqlc.narg('workflowIds')::UUID[] IS NULL
        OR workflow_id = ANY(sqlc.narg('workflowIds')::UUID[])
    )
    AND (
        sqlc.narg('scopes')::TEXT[] IS NULL
        OR scope = ANY(sqlc.narg('scopes')::TEXT[])
    )
ORDER BY id DESC
LIMIT COALESCE(sqlc.narg('filterLimit')::BIGINT, 20000)
OFFSET COALESCE(sqlc.narg('filterOffset')::BIGINT, 0)
;
