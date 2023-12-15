-- name: CountWorkflowRuns :one
SELECT
    count(runs) OVER() AS total
FROM
    "WorkflowRun" as runs
LEFT JOIN
    "WorkflowRunTriggeredBy" as runTriggers ON runTriggers."parentId" = runs."id"
LEFT JOIN
    "Event" as events ON runTriggers."eventId" = events."id"
LEFT JOIN
    "WorkflowVersion" as workflowVersion ON runs."workflowVersionId" = workflowVersion."id"
LEFT JOIN
    "Workflow" as workflow ON workflowVersion."workflowId" = workflow."id"
WHERE
    runs."tenantId" = $1 AND
    (
        sqlc.narg('workflowId')::uuid IS NULL OR
        workflow."id" = sqlc.narg('workflowId')::uuid
    ) AND
    (
        sqlc.narg('eventId')::uuid IS NULL OR
        events."id" = sqlc.narg('eventId')::uuid
    );

-- name: ListWorkflowRuns :many
SELECT
    sqlc.embed(runs), 
    sqlc.embed(workflow), 
    sqlc.embed(runTriggers), 
    sqlc.embed(workflowVersion), 
    -- waiting on https://github.com/sqlc-dev/sqlc/pull/2858 for nullable events field
    events.id, events.key, events."createdAt", events."updatedAt"
FROM
    "WorkflowRun" as runs 
LEFT JOIN
    "WorkflowRunTriggeredBy" as runTriggers ON runTriggers."parentId" = runs."id"
LEFT JOIN
    "Event" as events ON runTriggers."eventId" = events."id"
LEFT JOIN
    "WorkflowVersion" as workflowVersion ON runs."workflowVersionId" = workflowVersion."id"
LEFT JOIN
    "Workflow" as workflow ON workflowVersion."workflowId" = workflow."id"
WHERE
    runs."tenantId" = $1 AND
    (
        sqlc.narg('workflowId')::uuid IS NULL OR
        workflow."id" = sqlc.narg('workflowId')::uuid
    ) AND
    (
        sqlc.narg('eventId')::uuid IS NULL OR
        events."id" = sqlc.narg('eventId')::uuid
    )
ORDER BY
    case when @orderBy = 'createdAt ASC' THEN runs."createdAt" END ASC ,
    case when @orderBy = 'createdAt DESC' then runs."createdAt" END DESC
OFFSET
    COALESCE(sqlc.narg('offset'), 0)
LIMIT
    COALESCE(sqlc.narg('limit'), 50);

-- name: ResolveWorkflowRunStatus :one
WITH jobRuns AS (
    SELECT sum(case when runs."status" = 'PENDING' then 1 else 0 end) AS pendingRuns,
        sum(case when runs."status" = 'RUNNING' then 1 else 0 end) AS runningRuns,
        sum(case when runs."status" = 'SUCCEEDED' then 1 else 0 end) AS succeededRuns,
        sum(case when runs."status" = 'FAILED' then 1 else 0 end) AS failedRuns,
        sum(case when runs."status" = 'CANCELLED' then 1 else 0 end) AS cancelledRuns
    FROM "JobRun" as runs
    WHERE
        "workflowRunId" = (
            SELECT "workflowRunId"
            FROM "JobRun"
            WHERE "id" = @jobRunId::uuid
        ) AND
        "tenantId" = @tenantId::uuid
)
UPDATE "WorkflowRun"
SET "status" = CASE 
    -- Final states are final, cannot be updated
    WHEN "status" IN ('SUCCEEDED', 'FAILED') THEN "status"
    -- We check for running first, because if a job run is running, then the workflow is running
    WHEN j.runningRuns > 0 THEN 'RUNNING'
    -- When at least one job run has failed or been cancelled, then the workflow is failed
    WHEN j.failedRuns > 0 OR j.cancelledRuns > 0 THEN 'FAILED'
    -- When all job runs have succeeded, then the workflow is succeeded
    WHEN j.succeededRuns > 0 AND j.pendingRuns = 0 AND j.runningRuns = 0 AND j.failedRuns = 0 AND j.cancelledRuns = 0 THEN 'SUCCEEDED'
    ELSE "status"
END, "finishedAt" = CASE 
    -- Final states are final, cannot be updated
    WHEN "finishedAt" IS NOT NULL THEN "finishedAt"
    -- We check for running first, because if a job run is running, then the workflow is not finished
    WHEN j.runningRuns > 0 THEN NULL
    -- When one job run has failed or been cancelled, then the workflow is failed
    WHEN j.failedRuns > 0 OR j.cancelledRuns > 0 OR j.succeededRuns > 0 THEN NOW()
    ELSE "finishedAt"
END, "startedAt" = CASE 
    -- Started at is final, cannot be changed
    WHEN "startedAt" IS NOT NULL THEN "startedAt"
    -- If a job is running, then the workflow has started
    WHEN j.runningRuns > 0 THEN NOW()
    ELSE "startedAt"
END
FROM
    jobRuns j
WHERE "id" = (
    SELECT "workflowRunId"
    FROM "JobRun"
    WHERE "id" = @jobRunId::uuid
) AND "tenantId" = @tenantId::uuid
RETURNING "WorkflowRun".*;
