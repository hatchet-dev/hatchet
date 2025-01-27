-- name: ListWorkflowsForEvents :many
-- Get all of the latest workflow versions
WITH latest_versions AS (
    SELECT DISTINCT ON("workflowId")
        workflowVersions."id" AS "workflowVersionId"
    FROM
        "WorkflowVersion" as workflowVersions
    JOIN
        "Workflow" as workflow ON workflow."id" = workflowVersions."workflowId"
    WHERE
        workflow."tenantId" = @tenantId::uuid
        AND workflowVersions."deletedAt" IS NULL
    ORDER BY "workflowId", "order" DESC
), events AS (
    SELECT
        unnest(@eventIds::uuid[]) AS "eventId",
        unnest(@eventKeys::text[]) AS "eventKey"
)
-- select the workflow versions that have the event trigger
SELECT
    latest_versions."workflowVersionId",
    events."eventId"::uuid as "eventId",
    events."eventKey"::text as "eventKey"
FROM
    latest_versions
JOIN
    "WorkflowTriggers" as triggers ON triggers."workflowVersionId" = latest_versions."workflowVersionId"
JOIN
    "WorkflowTriggerEventRef" as eventRef ON eventRef."parentId" = triggers."id"
JOIN
    events ON events."eventKey" = eventRef."eventKey";