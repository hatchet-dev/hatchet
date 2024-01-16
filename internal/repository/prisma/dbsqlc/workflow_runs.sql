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
    -- If a job is running or in a final state, then the workflow has started
    WHEN j.runningRuns > 0 OR j.succeededRuns > 0 OR j.failedRuns > 0 OR j.cancelledRuns > 0 THEN NOW()
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

-- name: CreateWorkflowRun :one
INSERT INTO "WorkflowRun" (
    "id",
    "createdAt",
    "updatedAt",
    "deletedAt",
    "tenantId",
    "workflowVersionId",
    "status",
    "error",
    "startedAt",
    "finishedAt"
) VALUES (
    COALESCE(sqlc.narg('id')::uuid, gen_random_uuid()),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    NULL, -- assuming deletedAt is not set on creation
    @tenantId::uuid,
    @workflowVersionId::uuid,
    'PENDING', -- default status
    NULL, -- assuming error is not set on creation
    NULL, -- assuming startedAt is not set on creation
    NULL  -- assuming finishedAt is not set on creation
) RETURNING *;

-- name: CreateWorkflowRunTriggeredBy :one
INSERT INTO "WorkflowRunTriggeredBy" (
    "id",
    "createdAt",
    "updatedAt",
    "deletedAt",
    "tenantId",
    "parentId",
    "eventId",
    "cronParentId",
    "cronSchedule",
    "scheduledId"
) VALUES (
    gen_random_uuid(), -- Generates a new UUID for id
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    NULL, -- assuming deletedAt is not set on creation
    @tenantId::uuid,
    @workflowRunId::text, -- assuming parentId is the workflowRunId
    sqlc.narg('eventId')::uuid, -- NULL if not provided
    sqlc.narg('cronParentId')::uuid, -- NULL if not provided
    sqlc.narg('cron')::text, -- NULL if not provided
    sqlc.narg('scheduledId')::uuid -- NULL if not provided
) RETURNING *;

-- name: CreateJobRun :one
INSERT INTO "JobRun" (
    "id",
    "createdAt",
    "updatedAt",
    "deletedAt",
    "tenantId",
    "workflowRunId",
    "jobId",
    "tickerId",
    "status",
    "result",
    "startedAt",
    "finishedAt",
    "timeoutAt",
    "cancelledAt",
    "cancelledReason",
    "cancelledError"
) VALUES (
    COALESCE(sqlc.narg('id')::uuid, gen_random_uuid()),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    NULL,
    @tenantId::uuid,
    @workflowRunId::text,
    @jobId::uuid,
    NULL,
    'PENDING', -- default status
    NULL,
    NULL,
    NULL,
    NULL,
    NULL,
    NULL,
    NULL
) RETURNING *;

-- name: CreateJobRunLookupData :one
INSERT INTO "JobRunLookupData" (
    "id",
    "createdAt",
    "updatedAt",
    "deletedAt",
    "jobRunId",
    "tenantId",
    "data"
) VALUES (
    COALESCE(sqlc.narg('id')::uuid, gen_random_uuid()),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    NULL,
    @jobRunId::uuid,
    @tenantId::uuid,
    jsonb_build_object(
        'input', COALESCE(sqlc.narg('input')::jsonb, '{}'::jsonb),
        'triggered_by', @triggeredBy::text,
        'steps', '{}'::jsonb
    )
) RETURNING *;

-- name: CreateStepRun :one
INSERT INTO "StepRun" (
    "id",
    "createdAt",
    "updatedAt",
    "deletedAt",
    "tenantId",
    "jobRunId",
    "stepId",
    "workerId",
    "tickerId",
    "status",
    "input",
    "output",
    "requeueAfter",
    "scheduleTimeoutAt",
    "error",
    "startedAt",
    "finishedAt",
    "timeoutAt",
    "cancelledAt",
    "cancelledReason",
    "cancelledError"
) VALUES (
    COALESCE(sqlc.narg('id')::uuid, gen_random_uuid()),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    NULL,
    @tenantId::uuid,
    @jobRunId::uuid,
    @stepId::uuid,
    NULL,
    NULL,
    'PENDING', -- default status
    NULL,
    NULL,
    @requeueAfter::timestamp,
    @scheduleTimeoutAt::timestamp,
    NULL,
    NULL,
    NULL,
    NULL,
    NULL,
    NULL,
    NULL
) RETURNING *;

-- name: LinkStepRunParents :exec
INSERT INTO "_StepRunOrder" ("A", "B")
SELECT 
    parent_run."id" AS "A",
    child_run."id" AS "B"
FROM 
    "_StepOrder" AS step_order
JOIN 
    "StepRun" AS parent_run ON parent_run."stepId" = step_order."A" AND parent_run."jobRunId" = @jobRunId::uuid
JOIN 
    "StepRun" AS child_run ON child_run."stepId" = step_order."B" AND child_run."jobRunId" = @jobRunId::uuid;

-- name: ListStartableStepRuns :many
SELECT 
    child_run.*
FROM 
    "StepRun" AS child_run
JOIN 
    "_StepRunOrder" AS step_run_order ON step_run_order."B" = child_run."id"
WHERE 
    child_run."tenantId" = @tenantId::uuid
    AND child_run."jobRunId" = @jobRunId::uuid
    AND child_run."status" = 'PENDING'
    AND step_run_order."A" = @parentStepRunId::uuid
    AND NOT EXISTS (
        SELECT 1
        FROM "_StepRunOrder" AS parent_order
        JOIN "StepRun" AS parent_run ON parent_order."A" = parent_run."id"
        WHERE 
            parent_order."B" = child_run."id"
            AND parent_run."status" != 'SUCCEEDED'
    );
