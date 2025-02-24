-- name: ListStepsByWorkflowVersionIds :many
WITH steps AS (
    SELECT
        s.*,
        wv."id" as "workflowVersionId",
        w."name" as "workflowName",
        w."id" as "workflowId",
        j."kind" as "jobKind"
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
), step_orders AS (
    SELECT
        so."B" as "stepId",
        array_agg(so."A")::uuid[] as "parents"
    FROM
        steps
    JOIN
        "_StepOrder" so ON so."B" = steps."id"
    GROUP BY
        so."B"
)
SELECT
    s.*,
    COALESCE(so."parents", '{}'::uuid[]) as "parents"
FROM
    steps s
LEFT JOIN
    step_orders so ON so."stepId" = s."id";

-- name: ListStepsByIds :many
SELECT
    s.*,
    wv."id" as "workflowVersionId",
    w."name" as "workflowName",
    w."id" as "workflowId",
    COUNT(sc.id) as "concurrencyCount"
FROM
    "Step" s
JOIN
    "Job" j ON j."id" = s."jobId"
JOIN
    "WorkflowVersion" wv ON wv."id" = j."workflowVersionId"
JOIN
    "Workflow" w ON w."id" = wv."workflowId"
LEFT JOIN
    v2_step_concurrency sc ON sc.workflow_id = w."id" AND sc.step_id = s."id"
WHERE
    s."id" = ANY(@ids::uuid[])
    AND w."tenantId" = @tenantId::uuid
    AND w."deletedAt" IS NULL
    AND wv."deletedAt" IS NULL
GROUP BY
    s."id", wv."id", w."name", w."id";

-- name: ListStepExpressions :many
SELECT
    *
FROM
    "StepExpression"
WHERE
    "stepId" = ANY(@stepIds::uuid[]);