-- name: ListWorkflowsForEvents :many
-- Get all of the latest workflow versions
WITH latest_versions AS (
    SELECT DISTINCT ON("workflowId")
        "workflowId",
        workflowVersions."id" AS "workflowVersionId",
        workflow."name" AS "workflowName"
    FROM
        "WorkflowVersion" as workflowVersions
    JOIN
        "Workflow" as workflow ON workflow."id" = workflowVersions."workflowId"
    WHERE
        workflow."tenantId" = @tenantId::uuid
        AND workflowVersions."deletedAt" IS NULL
    ORDER BY "workflowId", "order" DESC
)
-- select the workflow versions that have the event trigger
SELECT
    latest_versions."workflowVersionId",
    latest_versions."workflowId",
    latest_versions."workflowName",
    eventRef."eventKey" as "eventKey"
FROM
    latest_versions
JOIN
    "WorkflowTriggers" as triggers ON triggers."workflowVersionId" = latest_versions."workflowVersionId"
JOIN
    "WorkflowTriggerEventRef" as eventRef ON eventRef."parentId" = triggers."id"
WHERE
    eventRef."eventKey" = ANY(@eventKeys::text[]);

-- name: ListWorkflowsByNames :many
SELECT DISTINCT ON("workflowId")
    "workflowId",
    workflowVersions."id" AS "workflowVersionId",
    workflow."name" AS "workflowName"
FROM
    "WorkflowVersion" as workflowVersions
JOIN
    "Workflow" as workflow ON workflow."id" = workflowVersions."workflowId"
WHERE
    workflow."tenantId" = @tenantId::uuid
    AND workflowVersions."deletedAt" IS NULL
    AND workflow."name" = ANY(@workflowNames::text[])
ORDER BY "workflowId", "order" DESC;
