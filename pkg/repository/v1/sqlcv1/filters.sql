-- name: CreateFilter :one
WITH latest_version AS (
    SELECT DISTINCT ON (workflowVersions."workflowId")
        workflowVersions."id" AS workflowVersionId,
        workflowVersions."workflowId",
        workflowVersions."order"
    FROM
        "WorkflowVersion" as workflowVersions
    WHERE
        workflowVersions."workflowId" = @workflowId::UUID AND
        workflowVersions."deletedAt" IS NULL
    ORDER BY
        workflowVersions."workflowId", workflowVersions."order" DESC
)

INSERT INTO v1_filter (
    tenant_id,
    workflow_id,
    workflow_version_id,
    resource_hint,
    expression,
    payload
)

SELECT
    @tenantId::UUID AS tenant_id,
    @workflowId::UUID AS workflow_id,
    v.workflowVersionId AS workflow_version_id,
    @resourceHint::TEXT AS resource_hint,
    @expression::TEXT AS expression,
    COALESCE(@payload::JSONB, '{}'::JSONB) AS payload
FROM latest_version v
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