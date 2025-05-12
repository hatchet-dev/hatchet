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
WITH inputs AS (
    SELECT
        UNNEST(@tenantIds::UUID[]) AS tenant_id,
        UNNEST(@workflowIds::UUID[]) AS workflow_id,
        UNNEST(@workflowVersionIds::UUID[]) AS workflow_version_id,
        UNNEST(@resourceHints::TEXT[]) AS resource_hint
)

SELECT *
FROM v1_filter f
JOIN inputs i ON (f.tenant_id, f.workflow_id, f.workflow_version_id, f.resource_hint) = (i.tenant_id, i.workflow_id, i.workflow_version_id, i.resource_hint)
;