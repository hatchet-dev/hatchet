-- name: CreateFilter :one
INSERT INTO v1_filter (
    tenant_id,
    workflow_id,
    scope,
    expression,
    payload
) VALUES (
    @tenantId::UUID,
    @workflowId::UUID,
    @scope::TEXT,
    @expression::TEXT,
    @payload::JSONB
)
ON CONFLICT (tenant_id, workflow_id, scope, expression) DO UPDATE
SET
    payload = EXCLUDED.payload,
    updated_at = NOW()
WHERE v1_filter.tenant_id = @tenantId::UUID
  AND v1_filter.workflow_id = @workflowId::UUID
  AND v1_filter.scope = @scope::TEXT
  AND v1_filter.expression = @expression::TEXT
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
