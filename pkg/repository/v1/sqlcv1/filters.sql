-- name: CreateFilter :one
INSERT INTO v1_filter (
    tenant_id,
    workflow_id,
    workflow_version_id,
    resource_hint,
    expression,
    payload
) VALUES (
    @tenantId::UUID,
    @workflowId::UUID,
    @workflowVersionId::UUID,
    @resourceHint::TEXT,
    @expression::TEXT,
    COALESCE(@payload::JSONB, '{}'::JSONB)
)
RETURNING *
;

-- name: ListFilters :many
SELECT *
FROM v1_filter
WHERE tenant_id = @tenantId::UUID
  AND workflow_id = @workflowId::UUID
  AND workflow_version_id = @workflowVersionId::UUID
  AND resource_hint = @resourceHint::TEXT
;