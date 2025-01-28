-- name: GetWorkflowStartData :many
-- If the workflow has multiple steps, this does not return any data
-- If the workflow has a single step, this returns step data along with workflowVersionId and workflowName
WITH workflow_versions_with_steps AS (
    SELECT
        s.*,
        wv."id" as "workflowVersionId",
        w."name" as "workflowName",
        w."id" as "workflowId"
    FROM
        "WorkflowVersion" as wv
    JOIN
        "Workflow" as w ON w."id" = wv."workflowId"
    JOIN
        "Job" j ON j."workflowVersionId" = wv."id"
    JOIN
        "Step" s ON s."jobId" = j."id"
    WHERE
        wv."id" = ANY(@ids::uuid[])
        AND w."tenantId" = @tenantId::uuid
        AND w."deletedAt" IS NULL 
        AND wv."deletedAt" IS NULL
), step_counts AS (
    SELECT
        DISTINCT ON (wv."workflowVersionId") wv."workflowVersionId" as "workflowVersionId",
        COUNT(wv."id") as "numSteps"
    FROM
        workflow_versions_with_steps as wv
    GROUP BY
        wv."workflowVersionId"
)
SELECT
    wv.*
FROM
    workflow_versions_with_steps as wv
JOIN
    step_counts sc ON sc."workflowVersionId" = wv."workflowVersionId"
WHERE
    sc."numSteps" = 1;