-- name: CountWorkflowRuns :one
WITH runs AS (
    SELECT runs."id", runs."createdAt"
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
    runs."deletedAt" IS NULL AND
    workflowVersion."deletedAt" IS NULL AND
    workflow."deletedAt" IS NULL AND
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
        sqlc.narg('additionalMetadata')::jsonb IS NULL OR
        runs."additionalMetadata" @> sqlc.narg('additionalMetadata')::jsonb
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
    ) AND
    (
        sqlc.narg('createdAfter')::timestamp IS NULL OR
        runs."createdAt" > sqlc.narg('createdAfter')::timestamp
    ) AND
    (
        sqlc.narg('finishedAfter')::timestamp IS NULL OR
        runs."finishedAt" > sqlc.narg('finishedAfter')::timestamp
    )
    ORDER BY
        case when @orderBy = 'createdAt ASC' THEN runs."createdAt" END ASC ,
        case when @orderBy = 'createdAt DESC' then runs."createdAt" END DESC,
        runs."id" ASC
    LIMIT 10000
)
SELECT
    count(runs) AS total
FROM
    runs;

-- name: WorkflowRunsMetricsCount :one
SELECT
    COUNT(CASE WHEN runs."status" = 'PENDING' THEN 1 END) AS "PENDING",
    COUNT(CASE WHEN runs."status" = 'RUNNING' THEN 1 END) AS "RUNNING",
    COUNT(CASE WHEN runs."status" = 'SUCCEEDED' THEN 1 END) AS "SUCCEEDED",
    COUNT(CASE WHEN runs."status" = 'FAILED' THEN 1 END) AS "FAILED",
    COUNT(CASE WHEN runs."status" = 'QUEUED' THEN 1 END) AS "QUEUED"
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
    runs."tenantId" = @tenantId::uuid AND
    runs."deletedAt" IS NULL AND
    workflowVersion."deletedAt" IS NULL AND
    workflow."deletedAt" IS NULL AND
    (
        sqlc.narg('workflowId')::uuid IS NULL OR
        workflow."id" = sqlc.narg('workflowId')::uuid
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
        sqlc.narg('additionalMetadata')::jsonb IS NULL OR
        runs."additionalMetadata" @> sqlc.narg('additionalMetadata')::jsonb
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
    runs."deletedAt" IS NULL AND
    workflowVersion."deletedAt" IS NULL AND
    workflow."deletedAt" IS NULL AND
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
        sqlc.narg('additionalMetadata')::jsonb IS NULL OR
        runs."additionalMetadata" @> sqlc.narg('additionalMetadata')::jsonb
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
    ) AND
    (
        sqlc.narg('createdAfter')::timestamp IS NULL OR
        runs."createdAt" > sqlc.narg('createdAfter')::timestamp
    ) AND
    (
        sqlc.narg('finishedAfter')::timestamp IS NULL OR
        runs."finishedAt" > sqlc.narg('finishedAfter')::timestamp
    )
ORDER BY
    case when @orderBy = 'createdAt ASC' THEN runs."createdAt" END ASC ,
    case when @orderBy = 'createdAt DESC' THEN runs."createdAt" END DESC,
    case when @orderBy = 'finishedAt ASC' THEN runs."finishedAt" END ASC ,
    case when @orderBy = 'finishedAt DESC' THEN runs."finishedAt" END DESC,
    case when @orderBy = 'startedAt ASC' THEN runs."startedAt" END ASC ,
    case when @orderBy = 'startedAt DESC' THEN runs."startedAt" END DESC,
    case when @orderBy = 'duration ASC' THEN runs."duration" END ASC NULLS FIRST,
    case when @orderBy = 'duration DESC' THEN runs."duration" END DESC NULLS LAST,
    runs."id" ASC
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
        r2."deletedAt" IS NULL AND
        workflowVersion."deletedAt" IS NULL AND
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
        "tenantId" = @tenantId::uuid AND
        "deletedAt" IS NULL
)
UPDATE "WorkflowRun" workflowRun
SET "status" = CASE
    -- Final states are final, cannot be updated. We also can't move out of a queued state
    WHEN "status" IN ('SUCCEEDED', 'FAILED', 'QUEUED') THEN "status"
    -- When the GetGroupKeyRun failed or been cancelled, then the workflow is failed
    WHEN groupKeyRun.groupKeyRunStatus IN ('FAILED', 'CANCELLED') THEN 'FAILED'
    WHEN groupKeyRun.output IS NOT NULL THEN 'QUEUED'
    ELSE "status"
END,
"finishedAt" = CASE
    -- Final states are final, cannot be updated
    WHEN "finishedAt" IS NOT NULL THEN "finishedAt"
    -- When one job run has failed or been cancelled, then the workflow is failed
    WHEN groupKeyRun.groupKeyRunStatus IN ('FAILED', 'CANCELLED') THEN NOW()
    ELSE "finishedAt"
END,
"duration" = CASE
    -- duration is final, cannot be changed
    WHEN "duration" IS NOT NULL THEN "duration"
    WHEN "startedAt" IS NOT NULL AND groupKeyRun.groupKeyRunStatus IN ('FAILED', 'CANCELLED') THEN
                EXTRACT(EPOCH FROM (NOW() - "startedAt")) * 1000
    ELSE "duration"
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
    JOIN "Job" as job ON runs."jobId" = job."id"
    WHERE
        "workflowRunId" = (
            SELECT "workflowRunId"
            FROM "JobRun"
            WHERE "id" = @jobRunId::uuid
        ) AND
        runs."deletedAt" IS NULL AND
        runs."tenantId" = @tenantId::uuid AND
        -- we should not include onFailure jobs in the calculation
        job."kind" = 'DEFAULT'
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
END,
"finishedAt" = CASE
    -- Final states are final, cannot be updated
    WHEN "finishedAt" IS NOT NULL THEN "finishedAt"
    -- We check for running first, because if a job run is running, then the workflow is not finished
    WHEN j.runningRuns > 0 THEN NULL
    -- When one job run has failed or been cancelled, then the workflow is failed
    WHEN j.failedRuns > 0 OR j.cancelledRuns > 0 OR j.succeededRuns > 0 THEN NOW()
    ELSE "finishedAt"
END,
"startedAt" = CASE
    -- Started at is final, cannot be changed
    WHEN "startedAt" IS NOT NULL THEN "startedAt"
    -- If a job is running or in a final state, then the workflow has started
    WHEN j.runningRuns > 0 OR j.succeededRuns > 0 OR j.failedRuns > 0 OR j.cancelledRuns > 0 THEN NOW()
    ELSE "startedAt"
END,
"duration" = CASE
    -- duration is final, cannot be changed
    WHEN "duration" IS NOT NULL THEN "duration"
    -- We check for running first, because if a job run is running, then the workflow is not finished
    WHEN j.runningRuns > 0 THEN NULL
    -- When one job run has failed or been cancelled, then the workflow is failed
    WHEN j.failedRuns > 0 OR j.cancelledRuns > 0 OR j.succeededRuns > 0 THEN
                    EXTRACT(EPOCH FROM (NOW() - "startedAt")) * 1000
    ELSE "duration"
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
    "status" = CASE
    -- Final states are final, cannot be updated
        WHEN "status" IN ('SUCCEEDED', 'FAILED') THEN "status"
        ELSE COALESCE(sqlc.narg('status')::"WorkflowRunStatus", "status")
    END,
    "error" = COALESCE(sqlc.narg('error')::text, "error"),
    "startedAt" = COALESCE(sqlc.narg('startedAt')::timestamp, "startedAt"),
    "finishedAt" = COALESCE(sqlc.narg('finishedAt')::timestamp, "finishedAt"),
    "duration" =
        EXTRACT(EPOCH FROM (COALESCE(sqlc.narg('finishedAt')::timestamp, "finishedAt") - COALESCE(sqlc.narg('startedAt')::timestamp, "startedAt")) * 1000)

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
    "finishedAt" = COALESCE(sqlc.narg('finishedAt')::timestamp, "finishedAt"),
    "duration" = COALESCE(sqlc.narg('finishedAt')::timestamp, "finishedAt") - COALESCE(sqlc.narg('startedAt')::timestamp, "startedAt")
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
    "parentStepRunId",
    "additionalMetadata"
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
    sqlc.narg('parentStepRunId')::uuid,
    @additionalMetadata::jsonb
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
    runs."deletedAt" IS NULL AND
    workflowVersion."deletedAt" IS NULL AND
    workflow."deletedAt" IS NULL AND
    runs."id" = ANY(@ids::uuid[]) AND
    runs."tenantId" = @tenantId::uuid;


-- name: GetChildWorkflowRun :one
SELECT
    *
FROM
    "WorkflowRun"
WHERE
    "parentId" = @parentId::uuid AND
    "deletedAt" IS NULL AND
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

-- name: SoftDeleteExpiredWorkflowRunsWithDependencies :one
WITH for_delete AS (
    SELECT
        "id"
    FROM "WorkflowRun" wr2
    WHERE
        wr2."tenantId" = @tenantId::uuid AND
        wr2."status" = ANY(cast(sqlc.narg('statuses')::text[] as "WorkflowRunStatus"[])) AND
        wr2."createdAt" < @createdBefore::timestamp AND
        "deletedAt" IS NULL
    ORDER BY "createdAt" ASC
    LIMIT sqlc.arg('limit') +1
    FOR UPDATE SKIP LOCKED
),
expired_with_limit AS (
    SELECT
        for_delete."id" as "id"
    FROM for_delete
    LIMIT sqlc.arg('limit')
),
has_more AS (
    SELECT
        CASE
            WHEN COUNT(*) > sqlc.arg('limit') THEN TRUE
            ELSE FALSE
        END as has_more
    FROM for_delete
),
job_runs_to_delete AS (
    SELECT
        "id"
    FROM
        "JobRun"
    WHERE
        "workflowRunId" IN (SELECT "id" FROM expired_with_limit)
        AND "deletedAt" IS NULL
), step_runs_to_delete AS (
    SELECT
        "id"
    FROM
        "StepRun"
    WHERE
        "jobRunId" IN (SELECT "id" FROM job_runs_to_delete)
        AND "deletedAt" IS NULL
), update_step_runs AS (
    UPDATE
        "StepRun"
    SET
        "deletedAt" = CURRENT_TIMESTAMP
    WHERE
        "id" IN (SELECT "id" FROM step_runs_to_delete)
), update_job_runs AS (
    UPDATE
        "JobRun" jr
    SET
        "deletedAt" = CURRENT_TIMESTAMP
    WHERE
        jr."id" IN (SELECT "id" FROM job_runs_to_delete)
)
UPDATE
    "WorkflowRun" wr
SET
    "deletedAt" = CURRENT_TIMESTAMP
WHERE
    "id" IN (SELECT "id" FROM expired_with_limit) AND
    wr."tenantId" = @tenantId::uuid
RETURNING
    (SELECT has_more FROM has_more) as has_more;


-- name: ListActiveQueuedWorkflowVersions :many
WITH QueuedRuns AS (
    SELECT DISTINCT ON (wr."workflowVersionId")
        wr."workflowVersionId",
        w."tenantId",
        wr."status",
        wr."id",
        wr."concurrencyGroupId"
    FROM "WorkflowRun" wr
    JOIN "WorkflowVersion" wv ON wv."id" = wr."workflowVersionId"
    JOIN "Workflow" w ON w."id" = wv."workflowId"
    WHERE wr."status" = 'QUEUED'
		AND wr."concurrencyGroupId" IS NOT NULL
    ORDER BY wr."workflowVersionId"
)
SELECT
    q."workflowVersionId",
    q."tenantId",
    q."status",
    q."id",
    q."concurrencyGroupId"
FROM QueuedRuns q
GROUP BY q."workflowVersionId", q."tenantId", q."concurrencyGroupId", q."status", q."id";
