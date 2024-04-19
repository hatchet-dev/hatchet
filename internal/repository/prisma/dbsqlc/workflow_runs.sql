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
        sqlc.narg('workflowVersionId')::uuid IS NULL OR
        workflowVersion."id" = sqlc.narg('workflowVersionId')::uuid
    ) AND
    (
        sqlc.narg('workflowId')::uuid IS NULL OR
        workflow."id" = sqlc.narg('workflowId')::uuid
    ) AND
    (
        sqlc.narg('ids')::uuid[] IS NULL OR
        runs."id" = ANY(sqlc.narg('ids')::uuid[])
    ) AND
    (
        sqlc.narg('parentId')::uuid IS NULL OR
        runs."parentId" = sqlc.narg('parentId')::uuid
    ) AND
    (
        sqlc.narg('parentStepRunId')::uuid IS NULL OR
        runs."parentStepRunId" = sqlc.narg('parentStepRunId')::uuid
    ) AND
    (
        sqlc.narg('eventId')::uuid IS NULL OR
        events."id" = sqlc.narg('eventId')::uuid
    ) AND
    (
    sqlc.narg('groupKey')::text IS NULL OR
    runs."concurrencyGroupId" = sqlc.narg('groupKey')::text
    ) AND
    (
        sqlc.narg('statuses')::text[] IS NULL OR
        "status" = ANY(cast(sqlc.narg('statuses')::text[] as "WorkflowRunStatus"[]))
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
        sqlc.narg('workflowVersionId')::uuid IS NULL OR
        workflowVersion."id" = sqlc.narg('workflowVersionId')::uuid
    ) AND
    (
        sqlc.narg('workflowId')::uuid IS NULL OR
        workflow."id" = sqlc.narg('workflowId')::uuid
    ) AND
    (
        sqlc.narg('ids')::uuid[] IS NULL OR
        runs."id" = ANY(sqlc.narg('ids')::uuid[])
    ) AND
    (
        sqlc.narg('parentId')::uuid IS NULL OR
        runs."parentId" = sqlc.narg('parentId')::uuid
    ) AND
    (
        sqlc.narg('parentStepRunId')::uuid IS NULL OR
        runs."parentStepRunId" = sqlc.narg('parentStepRunId')::uuid
    ) AND
    (
        sqlc.narg('eventId')::uuid IS NULL OR
        events."id" = sqlc.narg('eventId')::uuid
    ) AND
    (
    sqlc.narg('groupKey')::text IS NULL OR
    runs."concurrencyGroupId" = sqlc.narg('groupKey')::text
    ) AND
    (
        sqlc.narg('statuses')::text[] IS NULL OR
        "status" = ANY(cast(sqlc.narg('statuses')::text[] as "WorkflowRunStatus"[]))
    )
ORDER BY
    case when @orderBy = 'createdAt ASC' THEN runs."createdAt" END ASC ,
    case when @orderBy = 'createdAt DESC' then runs."createdAt" END DESC
OFFSET
    COALESCE(sqlc.narg('offset'), 0)
LIMIT
    COALESCE(sqlc.narg('limit'), 50);

-- name: PopWorkflowRunsRoundRobin :many
WITH workflow_runs AS (
    SELECT
        r2.id,
        r2."status",
        row_number() OVER (PARTITION BY r2."concurrencyGroupId" ORDER BY r2."createdAt") AS rn,
        row_number() over (order by r2."createdAt" ASC) as seqnum
    FROM
        "WorkflowRun" r2
    LEFT JOIN
        "WorkflowVersion" workflowVersion ON r2."workflowVersionId" = workflowVersion."id"
    WHERE
        r2."tenantId" = @tenantId::uuid AND
        (r2."status" = 'QUEUED' OR r2."status" = 'RUNNING') AND
        workflowVersion."workflowId" = @workflowId::uuid
    ORDER BY
        rn, seqnum ASC
), min_rn AS (
    SELECT
        MIN(rn) as min_rn
    FROM
        workflow_runs
), total_group_count AS ( -- counts the number of groups
    SELECT
        COUNT(*) as count
    FROM
        workflow_runs
    WHERE
        rn = (SELECT min_rn FROM min_rn)
), eligible_runs AS (
    SELECT
        id
    FROM
        "WorkflowRun" wr
    WHERE
        wr."id" IN (
            SELECT
                id
            FROM
                workflow_runs
            ORDER BY
                rn, seqnum ASC
            LIMIT
                -- We can run up to maxRuns per group, so we multiple max runs by the number of groups, then subtract the 
                -- total number of running workflows.
                (@maxRuns::int) * (SELECT count FROM total_group_count)
        ) AND
        wr."status" = 'QUEUED'
    FOR UPDATE SKIP LOCKED
)
UPDATE "WorkflowRun"
SET
    "status" = 'RUNNING'
FROM
    eligible_runs
WHERE
    "WorkflowRun".id = eligible_runs.id AND
    "WorkflowRun"."status" = 'QUEUED'
RETURNING
    "WorkflowRun".*;

-- name: UpdateWorkflowRunGroupKey :one
WITH groupKeyRun AS (
    SELECT "id", "status" as groupKeyRunStatus, "output", "workflowRunId"
    FROM "GetGroupKeyRun" as groupKeyRun
    WHERE
        "id" = @groupKeyRunId::uuid AND
        "tenantId" = @tenantId::uuid
)
UPDATE "WorkflowRun" workflowRun
SET "status" = CASE 
    -- Final states are final, cannot be updated. We also can't move out of a queued state
    WHEN "status" IN ('SUCCEEDED', 'FAILED', 'QUEUED') THEN "status"
    -- When the GetGroupKeyRun failed or been cancelled, then the workflow is failed
    WHEN groupKeyRun.groupKeyRunStatus IN ('FAILED', 'CANCELLED') THEN 'FAILED'
    WHEN groupKeyRun.output IS NOT NULL THEN 'QUEUED'
    ELSE "status"
END, "finishedAt" = CASE 
    -- Final states are final, cannot be updated
    WHEN "finishedAt" IS NOT NULL THEN "finishedAt"
    -- When one job run has failed or been cancelled, then the workflow is failed
    WHEN groupKeyRun.groupKeyRunStatus IN ('FAILED', 'CANCELLED') THEN NOW()
    ELSE "finishedAt"
END, 
"concurrencyGroupId" = groupKeyRun."output"
FROM
    groupKeyRun
WHERE 
workflowRun."id" = groupKeyRun."workflowRunId" AND
workflowRun."tenantId" = @tenantId::uuid
RETURNING workflowRun.*;

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

-- name: UpdateWorkflowRun :one
UPDATE
    "WorkflowRun"
SET
    "status" = COALESCE(sqlc.narg('status')::"WorkflowRunStatus", "status"),
    "error" = COALESCE(sqlc.narg('error')::text, "error"),
    "startedAt" = COALESCE(sqlc.narg('startedAt')::timestamp, "startedAt"),
    "finishedAt" = COALESCE(sqlc.narg('finishedAt')::timestamp, "finishedAt")
WHERE 
    "id" = @id::uuid AND
    "tenantId" = @tenantId::uuid
RETURNING "WorkflowRun".*;

-- name: UpdateManyWorkflowRun :many
UPDATE
    "WorkflowRun"
SET
    "status" = COALESCE(sqlc.narg('status')::"WorkflowRunStatus", "status"),
    "error" = COALESCE(sqlc.narg('error')::text, "error"),
    "startedAt" = COALESCE(sqlc.narg('startedAt')::timestamp, "startedAt"),
    "finishedAt" = COALESCE(sqlc.narg('finishedAt')::timestamp, "finishedAt")
WHERE 
    "tenantId" = @tenantId::uuid AND
    "id" = ANY(@ids::uuid[])
RETURNING "WorkflowRun".*;

-- name: CreateWorkflowRun :one
INSERT INTO "WorkflowRun" (
    "id",
    "createdAt",
    "updatedAt",
    "deletedAt",
    "displayName",
    "tenantId",
    "workflowVersionId",
    "status",
    "error",
    "startedAt",
    "finishedAt",
    "childIndex",
    "childKey",
    "parentId",
    "parentStepRunId"
) VALUES (
    COALESCE(sqlc.narg('id')::uuid, gen_random_uuid()),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    NULL, -- assuming deletedAt is not set on creation
    sqlc.narg('displayName')::text,
    @tenantId::uuid,
    @workflowVersionId::uuid,
    'PENDING', -- default status
    NULL, -- assuming error is not set on creation
    NULL, -- assuming startedAt is not set on creation
    NULL,  -- assuming finishedAt is not set on creation
    sqlc.narg('childIndex')::int,
    sqlc.narg('childKey')::text,
    sqlc.narg('parentId')::uuid,
    sqlc.narg('parentStepRunId')::uuid
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
    @workflowRunId::uuid, -- assuming parentId is the workflowRunId
    sqlc.narg('eventId')::uuid, -- NULL if not provided
    sqlc.narg('cronParentId')::uuid, -- NULL if not provided
    sqlc.narg('cron')::text, -- NULL if not provided
    sqlc.narg('scheduledId')::uuid -- NULL if not provided
) RETURNING *;

-- name: CreateGetGroupKeyRun :one
INSERT INTO "GetGroupKeyRun" (
    "id",
    "createdAt",
    "updatedAt",
    "deletedAt",
    "tenantId",
    "workflowRunId",
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
    @workflowRunId::uuid,
    NULL,
    NULL,
    'PENDING', -- default status
    @input::jsonb,
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

-- name: CreateJobRuns :many
INSERT INTO "JobRun" (
    "id",
    "createdAt",
    "updatedAt",
    "tenantId",
    "workflowRunId",
    "jobId",
    "status"
) 
SELECT
    gen_random_uuid(),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    @tenantId::uuid,
    @workflowRunId::uuid,
    "id",
    'PENDING' -- default status
FROM
    "Job"
WHERE
    "workflowVersionId" = @workflowVersionId::uuid
RETURNING "id";

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

-- name: CreateStepRuns :exec
WITH job_id AS (
    SELECT "jobId"
    FROM "JobRun"
    WHERE "id" = @jobRunId::uuid
)
INSERT INTO "StepRun" (
    "id",
    "createdAt",
    "updatedAt",
    "tenantId",
    "jobRunId",
    "stepId",
    "status",
    "requeueAfter",
    "callerFiles"
) 
SELECT
    gen_random_uuid(),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    @tenantId::uuid,
    @jobRunId::uuid,
    "id",
    'PENDING', -- default status
    CURRENT_TIMESTAMP + INTERVAL '5 seconds',
    '{}'
FROM
    "Step", job_id
WHERE
    "Step"."jobId" = job_id."jobId";

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

-- name: GetWorkflowRun :many
SELECT
    sqlc.embed(runs), 
    sqlc.embed(runTriggers), 
    sqlc.embed(workflowVersion), 
    workflow."name" as "workflowName",
    -- waiting on https://github.com/sqlc-dev/sqlc/pull/2858 for nullable fields
    wc."limitStrategy" as "concurrencyLimitStrategy",
    wc."maxRuns" as "concurrencyMaxRuns",
    groupKeyRun."id" as "getGroupKeyRunId"
FROM
    "WorkflowRun" as runs
LEFT JOIN
    "WorkflowRunTriggeredBy" as runTriggers ON runTriggers."parentId" = runs."id"
LEFT JOIN
    "WorkflowVersion" as workflowVersion ON runs."workflowVersionId" = workflowVersion."id"
LEFT JOIN
    "Workflow" as workflow ON workflowVersion."workflowId" = workflow."id"
LEFT JOIN
    "WorkflowConcurrency" as wc ON wc."workflowVersionId" = workflowVersion."id"
LEFT JOIN
    "GetGroupKeyRun" as groupKeyRun ON groupKeyRun."workflowRunId" = runs."id"
WHERE
    runs."id" = ANY(@ids::uuid[]) AND
    runs."tenantId" = @tenantId::uuid;


-- name: GetChildWorkflowRun :one
SELECT
    *
FROM
    "WorkflowRun"
WHERE
    "parentId" = @parentId::uuid AND
    "parentStepRunId" = @parentStepRunId::uuid AND
    (
        -- if childKey is set, use that
        (sqlc.narg('childKey')::text IS NULL AND "childIndex" = @childIndex) OR
        (sqlc.narg('childKey')::text IS NOT NULL AND "childKey" = sqlc.narg('childKey')::text)
    );

-- name: GetScheduledChildWorkflowRun :one
SELECT
    *
FROM
    "WorkflowTriggerScheduledRef"
WHERE
    "parentId" = @parentId::uuid AND
    "parentStepRunId" = @parentStepRunId::uuid AND
    (
        -- if childKey is set, use that
        (sqlc.narg('childKey')::text IS NULL AND "childIndex" = @childIndex) OR
        (sqlc.narg('childKey')::text IS NOT NULL AND "childKey" = sqlc.narg('childKey')::text)
    );