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
ON CONFLICT (tenant_id, workflow_id, resource_hint, expression) DO UPDATE
SET
    payload = EXCLUDED.payload,
    resource_hint = EXCLUDED.resource_hint,
    expression = EXCLUDED.expression,
    updated_at = NOW()
WHERE v1_filter.tenant_id = @tenantId::UUID
  AND v1_filter.workflow_id = @workflowId::UUID
  AND v1_filter.resource_hint = @resourceHint::TEXT
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