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
    @isDeclarative::BOOLEAN
)
ON CONFLICT (tenant_id, workflow_id, scope, expression) DO UPDATE
SET
    payload = EXCLUDED.payload,
    is_declarative = EXCLUDED.is_declarative,
    updated_at = NOW()
WHERE v1_filter.tenant_id = @tenantId::UUID
  AND v1_filter.workflow_id = @workflowId::UUID
  AND v1_filter.scope = @scope::TEXT
  AND v1_filter.expression = @expression::TEXT
RETURNING *;

-- name: DangerouslyBulkUpsertDeclarativeFilters :many
-- IMPORTANT: This query overwrites all existing declarative filters for a workflow.
-- it's intended to be used when the workflow version is created.
WITH inputs AS (
    SELECT
        UNNEST(@scopes::TEXT[]) AS scope,
        UNNEST(@expressions::TEXT[]) AS expression,
        UNNEST(@payloads::JSONB[]) AS payload,
        UNNEST(@isDeclaratives::BOOLEAN[]) AS is_declarative
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
    is_declarative
FROM inputs
ON CONFLICT (tenant_id, workflow_id, scope, expression) DO UPDATE
SET
    payload = EXCLUDED.payload,
    is_declarative = EXCLUDED.is_declarative,
    updated_at = NOW()
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
