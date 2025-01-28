-- name: CountWorkflowRuns :one
WITH runs AS (
    SELECT runs."id", runs."createdAt", runs."finishedAt", runs."startedAt", runs."duration"
    FROM
        "WorkflowRun" as runs
    LEFT JOIN
        "WorkflowRunTriggeredBy" as runTriggers ON runTriggers."parentId" = runs."id"
    LEFT JOIN
        "Event" as events ON
            runTriggers."eventId" = events."id"
            AND (
                sqlc.narg('eventId')::uuid IS NULL OR
                events."id" = sqlc.narg('eventId')::uuid
            )
    LEFT JOIN
        "WorkflowVersion" as workflowVersion ON
            runs."workflowVersionId" = workflowVersion."id"
            AND (
                sqlc.narg('workflowVersionId')::uuid IS NULL OR
                workflowVersion."id" = sqlc.narg('workflowVersionId')::uuid
            )
            AND (
                sqlc.narg('kinds')::text[] IS NULL OR
                workflowVersion."kind" = ANY(cast(sqlc.narg('kinds')::text[] as "WorkflowKind"[]))
            )
    LEFT JOIN
        "Workflow" as workflow ON
            workflowVersion."workflowId" = workflow."id"
            AND (
                sqlc.narg('workflowId')::uuid IS NULL OR
                workflow."id" = sqlc.narg('workflowId')::uuid
            )
    WHERE
        runs."tenantId" = $1 AND
        runs."deletedAt" IS NULL AND
        workflowVersion."deletedAt" IS NULL AND
        workflow."deletedAt" IS NULL AND
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
            sqlc.narg('groupKey')::text IS NULL OR
            runs."concurrencyGroupId" = sqlc.narg('groupKey')::text
        ) AND
        (
            sqlc.narg('statuses')::text[] IS NULL OR
            runs."status" = ANY(cast(sqlc.narg('statuses')::text[] as "WorkflowRunStatus"[]))
        ) AND
        (
            sqlc.narg('createdAfter')::timestamp IS NULL OR
            runs."createdAt" > sqlc.narg('createdAfter')::timestamp
        ) AND
        (
            sqlc.narg('createdBefore')::timestamp IS NULL OR
            runs."createdAt" < sqlc.narg('createdBefore')::timestamp
        ) AND
        (
            sqlc.narg('finishedAfter')::timestamp IS NULL OR
            runs."finishedAt" > sqlc.narg('finishedAfter')::timestamp OR
            runs."finishedAt" IS NULL
        ) AND
        (
            sqlc.narg('finishedBefore')::timestamp IS NULL OR
            runs."finishedAt" <= sqlc.narg('finishedBefore')::timestamp
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
    LIMIT 10000
)
SELECT
    count(runs) AS total
FROM
    runs;

-- name: WorkflowRunsMetricsCount :one
SELECT
    COUNT(CASE WHEN runs."status" = 'PENDING' THEN 1 END) AS "PENDING",
    COUNT(CASE WHEN runs."status" = 'RUNNING' OR runs."status" = 'CANCELLING' THEN 1 END) AS "RUNNING",
    COUNT(CASE WHEN runs."status" = 'SUCCEEDED' THEN 1 END) AS "SUCCEEDED",
    COUNT(CASE WHEN runs."status" = 'FAILED' THEN 1 END) AS "FAILED",
    COUNT(CASE WHEN runs."status" = 'QUEUED' THEN 1 END) AS "QUEUED",
    COUNT(CASE WHEN runs."status" = 'CANCELLED' THEN 1 END) AS "CANCELLED"
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
    (
        sqlc.narg('createdAfter')::timestamp IS NULL OR
        runs."createdAt" > sqlc.narg('createdAfter')::timestamp
    ) AND
    (
        sqlc.narg('createdBefore')::timestamp IS NULL OR
        runs."createdAt" < sqlc.narg('createdBefore')::timestamp
    ) AND
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
    "Event" as events ON
        runTriggers."eventId" = events."id"
        AND (
            sqlc.narg('eventId')::uuid IS NULL OR
            events."id" = sqlc.narg('eventId')::uuid
        )
JOIN
    "WorkflowVersion" as workflowVersion ON
        runs."workflowVersionId" = workflowVersion."id"
        AND (
            sqlc.narg('workflowVersionId')::uuid IS NULL OR
            workflowVersion."id" = sqlc.narg('workflowVersionId')::uuid
        )
        AND (
            sqlc.narg('kinds')::text[] IS NULL OR
            workflowVersion."kind" = ANY(cast(sqlc.narg('kinds')::text[] as "WorkflowKind"[]))
        )
JOIN
    "Workflow" as workflow ON
        workflowVersion."workflowId" = workflow."id"
        AND (
            sqlc.narg('workflowId')::uuid IS NULL OR
            workflow."id" = sqlc.narg('workflowId')::uuid
        )
WHERE
    runs."tenantId" = $1 AND
    runs."deletedAt" IS NULL AND
    workflowVersion."deletedAt" IS NULL AND
    workflow."deletedAt" IS NULL AND
    (
        sqlc.narg('eventId')::uuid IS NULL OR
        events."id" = sqlc.narg('eventId')::uuid
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
        sqlc.narg('groupKey')::text IS NULL OR
        runs."concurrencyGroupId" = sqlc.narg('groupKey')::text
    ) AND
    (
        sqlc.narg('statuses')::text[] IS NULL OR
        runs."status" = ANY(cast(sqlc.narg('statuses')::text[] as "WorkflowRunStatus"[]))
    ) AND
    (
        sqlc.narg('createdAfter')::timestamp IS NULL OR
        runs."createdAt" > sqlc.narg('createdAfter')::timestamp
    ) AND
    (
        sqlc.narg('createdBefore')::timestamp IS NULL OR
        runs."createdAt" < sqlc.narg('createdBefore')::timestamp
    ) AND
    (
        sqlc.narg('finishedAfter')::timestamp IS NULL OR
        runs."finishedAt" > sqlc.narg('finishedAfter')::timestamp OR
        runs."finishedAt" IS NULL
    ) AND
    (
        sqlc.narg('finishedBefore')::timestamp IS NULL OR
        runs."finishedAt" <= sqlc.narg('finishedBefore')::timestamp
    )
ORDER BY
    case when @orderBy = 'createdAt ASC' THEN (runs."createdAt", runs."insertOrder") END ASC ,
    case when @orderBy = 'createdAt DESC' THEN (runs."createdAt", runs."insertOrder") END DESC,
    case when @orderBy = 'finishedAt ASC' THEN runs."finishedAt" END ASC ,
    case when @orderBy = 'finishedAt DESC' THEN runs."finishedAt" END DESC,
    case when @orderBy = 'startedAt ASC' THEN (runs."startedAt" ,runs."insertOrder") END ASC ,
    case when @orderBy = 'startedAt DESC' THEN (runs."startedAt" ,runs."insertOrder") END DESC,
    case when @orderBy = 'duration ASC' THEN runs."duration" END ASC NULLS FIRST,
    case when @orderBy = 'duration DESC' THEN runs."duration" END DESC NULLS LAST,
    runs."id" ASC
OFFSET
    COALESCE(sqlc.narg('offset'), 0)
LIMIT
    COALESCE(sqlc.narg('limit'), 50);

-- name: LockWorkflowRunsForQueueing :many
-- Locks any workflow runs which are in a RUNNING or QUEUED state, and have a matching concurrencyGroupId in a QUEUED state
WITH queued_wrs AS (
    SELECT
        DISTINCT ON (wr."concurrencyGroupId")
        wr."concurrencyGroupId"
    FROM
        "WorkflowRun" wr
    LEFT JOIN
        "WorkflowVersion" workflowVersion ON wr."workflowVersionId" = workflowVersion."id"
    WHERE
        wr."tenantId" = @tenantId::uuid AND
        wr."deletedAt" IS NULL AND
        workflowVersion."deletedAt" IS NULL AND
        wr."status" = 'QUEUED' AND
        workflowVersion."id" = @workflowVersionId::uuid
)
SELECT
    wr.*
FROM
    "WorkflowRun" wr
LEFT JOIN
    "WorkflowVersion" workflowVersion ON wr."workflowVersionId" = workflowVersion."id"
WHERE
    wr."tenantId" = @tenantId::uuid AND
    wr."deletedAt" IS NULL AND
    workflowVersion."deletedAt" IS NULL AND
    (wr."status" = 'QUEUED' OR wr."status" = 'RUNNING') AND
    workflowVersion."id" = @workflowVersionId::uuid AND
    wr."concurrencyGroupId" IN (SELECT "concurrencyGroupId" FROM queued_wrs)
ORDER BY
    wr."createdAt" ASC, wr."insertOrder" ASC
FOR UPDATE;

-- name: MarkWorkflowRunsCancelling :exec
UPDATE
    "WorkflowRun"
SET
    "status" = 'CANCELLING'
WHERE
    "tenantId" = @tenantId::uuid AND
    "id" = ANY(@ids::uuid[]) AND
    ("status" = 'PENDING' OR "status" = 'QUEUED' OR "status" = 'RUNNING');


-- name: PopWorkflowRunsRoundRobin :many
WITH workflow_runs AS (
    SELECT
        r2.id,
        r2."status",
        r2."concurrencyGroupId",
        row_number() OVER (PARTITION BY r2."concurrencyGroupId" ORDER BY r2."createdAt", r2.id) AS "rn",
        -- we order by r2.id as a second parameter to get a pseudo-random, stable order
        row_number() OVER (ORDER BY r2."createdAt", r2.id) AS "seqnum"
    FROM
        "WorkflowRun" r2
    LEFT JOIN
        "WorkflowVersion" workflowVersion ON r2."workflowVersionId" = workflowVersion."id"
    WHERE
        r2."tenantId" = @tenantId::uuid AND
        r2."deletedAt" IS NULL AND
        workflowVersion."deletedAt" IS NULL AND
        (r2."status" = 'QUEUED' OR r2."status" = 'RUNNING') AND
        workflowVersion."id" = @workflowVersionId::uuid
), eligible_runs_per_group AS (
    SELECT
        id,
        "concurrencyGroupId",
        "rn",
        "seqnum"
    FROM workflow_runs
    WHERE "rn" <= (@maxRuns::int) -- we limit the number of runs per group to maxRuns
), eligible_runs AS (
    SELECT
        wr."id"
    FROM "WorkflowRun" wr
    WHERE
        wr."id" IN (
            SELECT
                id
            FROM eligible_runs_per_group
            ORDER BY "rn", "seqnum" ASC
            LIMIT (@maxRuns::int) * (SELECT COUNT(DISTINCT "concurrencyGroupId") FROM workflow_runs)
        )
        AND wr."status" = 'QUEUED'
    LIMIT 500
    FOR UPDATE SKIP LOCKED
)
UPDATE "WorkflowRun"
SET
    "status" = 'RUNNING'
FROM
    eligible_runs
WHERE
    "WorkflowRun".id = eligible_runs.id
RETURNING
    "WorkflowRun".*;


-- name: UpdateWorkflowRunGroupKeyFromExpr :one
UPDATE "WorkflowRun" wr
SET "error" = CASE
    -- Final states are final, cannot be updated. We also can't move out of a queued state
    WHEN "status" IN ('SUCCEEDED', 'FAILED', 'QUEUED') THEN "error"
    WHEN sqlc.narg('error')::text IS NOT NULL THEN sqlc.narg('error')::text
    ELSE "error"
END,
"status" = CASE
    -- Final states are final, cannot be updated. We also can't move out of a queued state
    WHEN "status" IN ('SUCCEEDED', 'FAILED', 'QUEUED') THEN "status"
    -- When the concurrency expression errored, then the workflow is failed
    WHEN sqlc.narg('error')::text IS NOT NULL THEN 'FAILED'
    -- When the expression evaluated successfully, then queue the workflow run
    ELSE 'QUEUED'
END,
"concurrencyGroupId" = CASE
    WHEN sqlc.narg('concurrencyGroupId')::text IS NOT NULL THEN sqlc.narg('concurrencyGroupId')::text
    ELSE "concurrencyGroupId"
END
WHERE
    wr."id" = @workflowRunId::uuid
RETURNING wr."id";

-- name: UpdateWorkflowRunGroupKeyFromRun :one
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

-- name: ResolveWorkflowRunStatus :many
WITH jobRuns AS (
    SELECT
        runs."workflowRunId",
        sum(case when runs."status" = 'PENDING' then 1 else 0 end) AS pendingRuns,
        sum(case when runs."status" = 'RUNNING' then 1 else 0 end) AS runningRuns,
        sum(case when runs."status" = 'SUCCEEDED' then 1 else 0 end) AS succeededRuns,
        sum(case when runs."status" = 'FAILED' then 1 else 0 end) AS failedRuns,
        sum(case when runs."status" = 'CANCELLED' then 1 else 0 end) AS cancelledRuns
    FROM "JobRun" as runs
    JOIN "Job" as job ON runs."jobId" = job."id"
    WHERE
        runs."workflowRunId" = ANY(
            SELECT "workflowRunId"
            FROM "JobRun"
            WHERE "id" = ANY(@jobRunIds::uuid[])
        ) AND
        runs."deletedAt" IS NULL AND
        runs."tenantId" = @tenantId::uuid AND
        -- we should not include onFailure jobs in the calculation
        job."kind" = 'DEFAULT'
    GROUP BY runs."workflowRunId"
), updated_workflow_runs AS (
    UPDATE "WorkflowRun" wr
    SET "status" = CASE
        -- Final states are final, cannot be updated
        WHEN "status" IN ('SUCCEEDED', 'FAILED') THEN "status"
        -- We check for cancelled first, because if a job run is cancelled, then the workflow is cancelled
        WHEN j.cancelledRuns > 0 THEN 'CANCELLED'
        -- Then we check for running, because if a job run is running, then the workflow is running
        WHEN j.runningRuns > 0 THEN 'RUNNING'
        -- Then we check for failed, because if a job run has failed, then the workflow is failed
        WHEN j.failedRuns > 0 THEN 'FAILED'
        -- Then we check for succeeded, because if all job runs have succeeded, then the workflow is succeeded
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
    WHERE
        wr."id" = j."workflowRunId"
        AND "tenantId" = @tenantId::uuid
    RETURNING wr."id", wr."status", wr."tenantId"
)
-- Return distinct workflow run ids in a final state
SELECT DISTINCT "id", "status", "tenantId"
FROM updated_workflow_runs
WHERE "status" IN ('SUCCEEDED', 'FAILED');

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
    "additionalMetadata",
    "priority"
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
    @additionalMetadata::jsonb,
    sqlc.narg('priority')::int
) RETURNING *;


-- name: CreateWorkflowRuns :copyfrom
INSERT INTO "WorkflowRun" (
    "id",
    "displayName",
    "tenantId",
    "workflowVersionId",
    "status",
    "childIndex",
    "childKey",
    "parentId",
    "parentStepRunId",
    "additionalMetadata",
    "priority",
    "insertOrder"
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10,
    $11,
    $12

);


-- name: GetWorkflowRunsInsertedInThisTxn :many
SELECT * FROM "WorkflowRun"
WHERE xmin::text = (txid_current() % (2^32)::bigint)::text
AND ("createdAt" = CURRENT_TIMESTAMP::timestamp(3))
ORDER BY "insertOrder" ASC;

-- name: CreateWorkflowRunDedupe :one
WITH workflow_id AS (
    SELECT w."id" FROM "Workflow" w
    JOIN "WorkflowVersion" wv ON wv."workflowId" = w."id"
    WHERE wv."id" = @workflowVersionId::uuid
)
INSERT INTO "WorkflowRunDedupe" (
    "createdAt",
    "updatedAt",
    "tenantId",
    "workflowId",
    "workflowRunId",
    "value"
) VALUES (
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    @tenantId::uuid,
    (SELECT "id" FROM workflow_id),
    @workflowRunId::uuid,
    sqlc.narg('value')::text
) RETURNING *;



-- name: CreateWorkflowRunStickyState :one
WITH workflow_version AS (
    SELECT "sticky"
    FROM "WorkflowVersion"
    WHERE "id" = @workflowVersionId::uuid
)
INSERT INTO "WorkflowRunStickyState" (
    "createdAt",
    "updatedAt",
    "tenantId",
    "workflowRunId",
    "desiredWorkerId",
    "strategy"
)
SELECT
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    @tenantId::uuid,
    @workflowRunId::uuid,
    sqlc.narg('desiredWorkerId')::uuid,
    workflow_version."sticky"
FROM workflow_version
WHERE workflow_version."sticky" IS NOT NULL
RETURNING *;

-- name: CreateMultipleWorkflowRunStickyStates :exec
WITH input_rows AS (
    SELECT
        UNNEST(@tenantId::uuid[]) as "tenantId",
        UNNEST(@workflowRunIds::uuid[]) as "workflowRunId",
        UNNEST(@desiredWorkerIds::uuid[]) as "desiredWorkerId",
        UNNEST(@workflowVersionIds::uuid[]) as "workflowVersionId"
), valid_rows AS (
    SELECT
        ir."tenantId",
        ir."workflowRunId",
        ir."desiredWorkerId",
        ir."workflowVersionId",
        wv."sticky"
    FROM
        input_rows ir
    JOIN
        "WorkflowVersion" wv ON wv."id" = ir."workflowVersionId"
    WHERE
        wv."sticky" IS NOT NULL
)
INSERT INTO "WorkflowRunStickyState" (
    "createdAt",
    "updatedAt",
    "tenantId",
    "workflowRunId",
    "desiredWorkerId",
    "strategy"
)
SELECT
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    vr."tenantId",
    vr."workflowRunId",
    vr."desiredWorkerId",
    vr."sticky"
FROM valid_rows vr;

-- name: GetWorkflowRunAdditionalMeta :one
SELECT
    "additionalMetadata",
    "id"
FROM
    "WorkflowRun"
WHERE
    "id" = @workflowRunId::uuid AND
    "tenantId" = @tenantId::uuid;

-- name: GetWorkflowRunStickyStateForUpdate :one
SELECT
    *
FROM
    "WorkflowRunStickyState"
WHERE
    "workflowRunId" = @workflowRunId::uuid AND
    "tenantId" = @tenantId::uuid
FOR UPDATE;

-- name: UpdateWorkflowRunStickyState :exec
UPDATE "WorkflowRunStickyState"
SET
    "updatedAt" = CURRENT_TIMESTAMP,
    "desiredWorkerId" = sqlc.narg('desiredWorkerId')::uuid
WHERE
    "workflowRunId" = @workflowRunId::uuid AND
    "tenantId" = @tenantId::uuid;

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
    "cronName",
    "scheduledId"
) VALUES (
    gen_random_uuid(),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    NULL,
    @tenantId::uuid,
    @workflowRunId::uuid,
    sqlc.narg('eventId')::uuid,
    sqlc.narg('cronParentId')::uuid,
    sqlc.narg('cronSchedule')::text,
    sqlc.narg('cronName')::text,
    sqlc.narg('scheduledId')::uuid
) RETURNING *;

-- name: CreateWorkflowRunTriggeredBys :copyfrom
INSERT INTO "WorkflowRunTriggeredBy" (
    "id",
    "tenantId",
    "parentId",
    "eventId",
    "cronParentId",
    "cronSchedule",
    "cronName",
    "scheduledId"
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8
);

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

-- name: CreateGetGroupKeyRuns :copyfrom

INSERT INTO "GetGroupKeyRun" (
    "id",
    "tenantId",
    "workflowRunId",
    "input",
    "requeueAfter",
    "scheduleTimeoutAt",
    "status"

) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7

);

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

-- name: CreateManyJobRuns :many

WITH input_data AS (
    SELECT
        UNNEST(@tenantIds::uuid[]) AS tenantId,
        UNNEST(@workflowRunIds::uuid[]) AS workflowRunId,
        UNNEST(@workflowVersionIds::uuid[]) AS workflowVersionId
)
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
    input_data.tenantId,
    input_data.workflowRunId,
    "Job"."id",
    'PENDING'
FROM
    input_data
JOIN
    "Job"
ON
    "Job"."workflowVersionId" = input_data.workflowVersionId
RETURNING "JobRun"."id", "JobRun"."workflowRunId", "JobRun"."tenantId";



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

-- name: CreateJobRunLookupDatas :many

WITH input_data AS (
    SELECT
        UNNEST(COALESCE(@ids::uuid[], ARRAY[gen_random_uuid()])) AS id,
        UNNEST(@jobRunIds::uuid[]) AS jobRunId,
        UNNEST(@tenantIds::uuid[]) AS tenantId,
        UNNEST(@triggeredBys::text[]) AS triggeredBy,
        UNNEST(COALESCE(@inputs::jsonb[], ARRAY[ '{}'::jsonb ])) AS input
)
INSERT INTO "JobRunLookupData" (
    "id",
    "createdAt",
    "updatedAt",
    "deletedAt",
    "jobRunId",
    "tenantId",
    "data"
)
SELECT
    COALESCE(input_data.id, gen_random_uuid()),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    NULL,
    input_data.jobRunId,
    input_data.tenantId,
    jsonb_build_object(
        'input', input_data.input,
        'triggered_by', input_data.triggeredBy,
        'steps', '{}'::jsonb
    )
FROM input_data
RETURNING *;


-- name: CreateStepRun :one
INSERT INTO "StepRun" (
    "id",
    "createdAt",
    "updatedAt",
    "tenantId",
    "jobRunId",
    "stepId",
    "status",
    "requeueAfter",
    "queue",
    "priority"
)
SELECT
    gen_random_uuid(),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    @tenantId::uuid,
    @jobRunId::uuid,
    @stepId::uuid,
    'PENDING', -- default status
    CURRENT_TIMESTAMP + INTERVAL '5 seconds',
    sqlc.narg('queue')::text,
    sqlc.narg('priority')::int
RETURNING "id";

-- name: CreateStepRuns :copyfrom

INSERT INTO "StepRun" (
    "id",
    "tenantId",
    "jobRunId",
    "stepId",
    "status",
    "requeueAfter",
    "queue",
    "priority"
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8

);

-- name: GetStepsForWorkflowVersion :many

SELECT
    "Step".*  from "Step"
JOIN "Job" j ON "Step"."jobId" = j."id"
WHERE
    j."workflowVersionId" = ANY(@workflowVersionIds::uuid[]);

-- name: ListStepsForJob :many
WITH job_id AS (
    SELECT "jobId"
    FROM "JobRun"
    WHERE "id" = @jobRunId::uuid
)
SELECT
    s."id",
    s."actionId"
FROM
    "Step" s, job_id
WHERE
    s."jobId" = job_id."jobId";

-- name: CreateStepRunsForJobRunIds :many
WITH job_ids AS (
    SELECT DISTINCT "jobId", "id" as jobRunId, "tenantId"
    FROM "JobRun"
    WHERE "id" = ANY(@jobRunIds::uuid[])
),
steps AS (
    SELECT
        s."id" as step_id,
        s."actionId",
        s."jobId",
        j.jobRunId,
        j."tenantId"
    FROM "Step" s
    JOIN job_ids j ON s."jobId" = j."jobId"
)
INSERT INTO "StepRun" (
    "id",
    "tenantId",
    "priority",
    "status",
    "jobRunId",
    "stepId",
    "queue"
)
SELECT
    gen_random_uuid() as id,
    s."tenantId" as tenantId,
    @priority::int4 as priority,
    'PENDING' as status,
    s.jobRunId as jobRunId,
    step_id as stepId,
    s."actionId" as queue
FROM steps s
RETURNING
    "id";



-- name: LinkStepRunParents :exec
WITH step_runs AS (
    SELECT "id", "stepId", "jobRunId"
    FROM "StepRun"
    WHERE "id" = ANY(@stepRunIds::uuid[])
), parent_child_step_runs AS (
    SELECT
        parent_run."id" AS "A",
        child_run."id" AS "B"
    FROM
        "_StepOrder" AS step_order
    JOIN
        step_runs AS parent_run ON parent_run."stepId" = step_order."A"
    JOIN
        step_runs AS child_run ON child_run."stepId" = step_order."B" AND child_run."jobRunId" = parent_run."jobRunId"
)
INSERT INTO "_StepRunOrder" ("A", "B")
SELECT
    parent_child_step_runs."A" AS "A",
    parent_child_step_runs."B" AS "B"
FROM
    parent_child_step_runs;

-- name: GetWorkflowRun :many
SELECT
    sqlc.embed(runs),
    sqlc.embed(runTriggers),
    sqlc.embed(workflowVersion),
    workflow."name" as "workflowName",
    -- waiting on https://github.com/sqlc-dev/sqlc/pull/2858 for nullable fields
    wc."limitStrategy" as "concurrencyLimitStrategy",
    wc."maxRuns" as "concurrencyMaxRuns",
    workflow."isPaused" as "isPaused",
    wc."concurrencyGroupExpression" as "concurrencyGroupExpression",
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

-- name: GetChildWorkflowRunsByIndex :many
SELECT
    wr.*
FROM
    "WorkflowRun" wr
WHERE
    (wr."parentId", wr."parentStepRunId", wr."childIndex") IN (
        SELECT
            UNNEST(@parentIds::uuid[]),
            UNNEST(@parentStepRunIds::uuid[]),
            UNNEST(@childIndexes::int[])
    )
    AND wr."deletedAt" IS NULL;

-- name: GetChildWorkflowRunsByKey :many
SELECT
    wr.*
FROM
    "WorkflowRun" wr
WHERE
    (wr."parentId", wr."parentStepRunId", wr."childKey") IN (
        SELECT
            UNNEST(@parentIds::uuid[]),
            UNNEST(@parentStepRunIds::uuid[]),
            UNNEST(@childKeys::text[])
    )
    AND wr."deletedAt" IS NULL;

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
    WHERE
        wr."tenantId" = @tenantId::uuid
        AND wr."status" = 'QUEUED'
		AND wr."concurrencyGroupId" IS NOT NULL
        AND wr."deletedAt" IS NULL
        AND wv."deletedAt" IS NULL
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

-- name: ReplayWorkflowRunResetJobRun :one
UPDATE
    "JobRun"
SET
    -- We set this to pending so that the entire workflow starts fresh, and we
    -- don't accidentally trigger on failure jobs
    "status" = 'PENDING',
    "updatedAt" = CURRENT_TIMESTAMP,
    "startedAt" = NULL,
    "finishedAt" = NULL,
    "timeoutAt" = NULL,
    "cancelledAt" = NULL,
    "cancelledReason" = NULL,
    "cancelledError" = NULL
WHERE
    "id" = @jobRunId::uuid
RETURNING *;

-- name: GetWorkflowRunInput :one
SELECT jld."data" AS lookupData
FROM "JobRun" jr
JOIN "JobRunLookupData" jld ON jr."id" = jld."jobRunId"
WHERE jld."data" ? 'input' AND jr."workflowRunId" = @workflowRunId::uuid
LIMIT 1;

-- name: GetWorkflowRunById :one
SELECT
    r.*,
    sqlc.embed(wv),
    sqlc.embed(w),
    sqlc.embed(tb)
FROM
    "WorkflowRun" r
JOIN
    "WorkflowVersion" as wv ON
        r."workflowVersionId" = wv."id"
JOIN "Workflow" as w ON
    wv."workflowId" = w."id"
JOIN "WorkflowRunTriggeredBy" as tb ON
    r."id" = tb."parentId"
WHERE
    r."id" = @workflowRunId::uuid AND
    r."tenantId" = @tenantId::uuid;


-- name: GetWorkflowRunByIds :many
SELECT
    r.*,
    sqlc.embed(wv),
    sqlc.embed(w),
    sqlc.embed(tb)
FROM
    "WorkflowRun" r
JOIN
    "WorkflowVersion" as wv ON
        r."workflowVersionId" = wv."id"
JOIN "Workflow" as w ON
    wv."workflowId" = w."id"
JOIN "WorkflowRunTriggeredBy" as tb ON
    r."id" = tb."parentId"
WHERE
    r."id" = ANY(@workflowRunIds::uuid[]) AND
    r."tenantId" = @tenantId::uuid;

-- name: GetWorkflowRunTrigger :one
SELECT *
FROM
    "WorkflowRunTriggeredBy"
WHERE
    "parentId" = @runId::uuid AND
    "tenantId" = @tenantId::uuid;


-- name: GetStepsForJobs :many
SELECT
	j."id" as "jobId",
    sqlc.embed(s),
    (
        SELECT array_agg(so."A")::uuid[]  -- Casting the array_agg result to uuid[]
        FROM "_StepOrder" so
        WHERE so."B" = s."id"
    ) AS "parents"
FROM "Job" j
JOIN "Step" s ON s."jobId" = j."id"
WHERE
    j."id" = ANY(@jobIds::uuid[])
    AND j."tenantId" = @tenantId::uuid
    AND j."deletedAt" IS NULL;

-- name: ListChildWorkflowRunCounts :many
SELECT
    wr."parentStepRunId",
    COUNT(wr."id") as "count"
FROM
    "WorkflowRun" wr
WHERE
    wr."parentStepRunId" = ANY(@stepRunIds::uuid[])
GROUP BY
    wr."parentStepRunId";


-- We grab the output for each step run here which could potentially be very large

-- name: GetStepRunsForJobRunsWithOutput :many
SELECT
	sr."id",
	sr."createdAt",
	sr."updatedAt",
	sr."status",
    sr."jobRunId",
    sr."stepId",
    sr."tenantId",
    sr."startedAt",
    sr."finishedAt",
    sr."cancelledAt",
    sr."cancelledError",
    sr."cancelledReason",
    sr."timeoutAt",
    sr."error",
    sr."workerId",
    sr."output"
FROM "StepRun" sr
WHERE
	sr."jobRunId" = ANY(@jobIds::uuid[])
    AND sr."tenantId" = @tenantId::uuid
    AND sr."deletedAt" IS NULL
ORDER BY sr."order" DESC;

-- name: BulkCreateWorkflowRunEvent :exec
WITH input_values AS (
    SELECT
        unnest(@timeSeen::timestamp[]) AS "timeFirstSeen",
        unnest(@timeSeen::timestamp[]) AS "timeLastSeen",
        unnest(@workflowRunIds::uuid[]) AS "workflowRunId",
        unnest(cast(@reasons::text[] as"StepRunEventReason"[])) AS "reason",
        unnest(cast(@severities::text[] as "StepRunEventSeverity"[])) AS "severity",
        unnest(@messages::text[]) AS "message",
        1 AS "count",
        unnest(@data::jsonb[]) AS "data"
),
updated AS (
    UPDATE "StepRunEvent"
    SET
        "timeLastSeen" = input_values."timeLastSeen",
        "message" = input_values."message",
        "count" = "StepRunEvent"."count" + 1,
        "data" = input_values."data"
    FROM input_values
    WHERE
        "StepRunEvent"."workflowRunId" = input_values."workflowRunId"
        AND "StepRunEvent"."reason" = input_values."reason"
        AND "StepRunEvent"."severity" = input_values."severity"
        AND "StepRunEvent"."id" = (
            SELECT "id"
            FROM "StepRunEvent"
            WHERE "workflowRunId" = input_values."workflowRunId"
            ORDER BY "id" DESC
            LIMIT 1
        )
    RETURNING "StepRunEvent".*
)
INSERT INTO "StepRunEvent" (
    "timeFirstSeen",
    "timeLastSeen",
    "workflowRunId",
    "reason",
    "severity",
    "message",
    "count",
    "data"
)
SELECT
    "timeFirstSeen",
    "timeLastSeen",
    "workflowRunId",
    "reason",
    "severity",
    "message",
    "count",
    "data"
FROM input_values
WHERE NOT EXISTS (
    SELECT 1 FROM updated WHERE "workflowRunId" = input_values."workflowRunId"
);

-- name: ListWorkflowRunEventsByWorkflowRunId :many
SELECT
    sre.*
FROM
    "StepRunEvent" sre
WHERE
    sre."workflowRunId" = @workflowRunId::uuid
ORDER BY
    sre."id" DESC;

-- name: GetFailureDetails :many
SELECT
	wr."status",
	wr."id",
	jr."status" as "jrStatus",
	sr."status" as "srStatus",
	sr."cancelledReason",
	sr."error"
FROM "WorkflowRun" wr
JOIN
	"JobRun" jr on jr."workflowRunId" = wr."id"
JOIN
	"StepRun" sr on sr."jobRunId" = jr."id"
WHERE
	wr."status" = 'FAILED' AND
    sr."status" IN ('FAILED', 'CANCELLED') AND
    (
        sr."cancelledReason" IS NULL OR
        sr."cancelledReason" NOT IN ('CANCELLED_BY_USER', 'PREVIOUS_STEP_TIMED_OUT', 'PREVIOUS_STEP_FAILED', 'PREVIOUS_STEP_CANCELLED', 'CANCELLED_BY_CONCURRENCY_LIMIT')
    ) AND
	wr."id" = @workflowRunId::uuid AND
    wr."tenantId" = @tenantId::uuid;

-- name: ListScheduledWorkflows :many
SELECT
    w."name",
    w."id" as "workflowId",
    v."id" as "workflowVersionId",
    w."tenantId",
    t.*,
    wr."createdAt" as "workflowRunCreatedAt",
    wr."status" as "workflowRunStatus",
    wr."id" as "workflowRunId",
    wr."displayName" as "workflowRunName"
FROM "WorkflowTriggerScheduledRef" t
JOIN "WorkflowVersion" v ON t."parentId" = v."id"
JOIN "Workflow" w on v."workflowId" = w."id"
LEFT JOIN "WorkflowRunTriggeredBy" tb ON t."id" = tb."scheduledId"
LEFT JOIN "WorkflowRun" wr ON tb."parentId" = wr."id"
WHERE v."deletedAt" IS NULL
	AND w."tenantId" = @tenantId::uuid
    AND (@scheduleId::uuid IS NULL OR t."id" = @scheduleId::uuid)
    AND (@workflowId::uuid IS NULL OR w."id" = @workflowId::uuid)
    AND (@parentWorkflowRunId::uuid IS NULL OR t."id" = @parentWorkflowRunId::uuid)
    AND (@parentStepRunId::uuid IS NULL OR t."parentStepRunId" = @parentStepRunId::uuid)
    AND (sqlc.narg('additionalMetadata')::jsonb IS NULL OR
        t."additionalMetadata" @> sqlc.narg('additionalMetadata')::jsonb)
    AND (
        sqlc.narg('statuses')::text[] IS NULL OR
        wr."status" = ANY(cast(sqlc.narg('statuses')::text[] as "WorkflowRunStatus"[]))
        or (
            @includeScheduled::boolean IS TRUE AND
            wr."status" IS NULL
        )
    )
ORDER BY
    case when @orderBy = 'triggerAt ASC' THEN t."triggerAt" END ASC ,
    case when @orderBy = 'triggerAt DESC' THEN t."triggerAt" END DESC,
    case when @orderBy = 'createdAt ASC' THEN t."createdAt" END ASC ,
    case when @orderBy = 'createdAt DESC' THEN t."createdAt" END DESC,
    t."id" ASC
OFFSET
    COALESCE(sqlc.narg('offset'), 0)
LIMIT
    COALESCE(sqlc.narg('limit'), 50);

-- name: CountScheduledWorkflows :one
SELECT count(*)
FROM "WorkflowTriggerScheduledRef" t
JOIN "WorkflowVersion" v ON t."parentId" = v."id"
JOIN "Workflow" w on v."workflowId" = w."id"
LEFT JOIN "WorkflowRunTriggeredBy" tb ON t."id" = tb."scheduledId"
LEFT JOIN "WorkflowRun" wr ON tb."parentId" = wr."id"
WHERE v."deletedAt" IS NULL
	AND w."tenantId" = @tenantId::uuid
    AND (@scheduleId::uuid IS NULL OR t."id" = @scheduleId::uuid)
    AND (@workflowId::uuid IS NULL OR w."id" = @workflowId::uuid)
    AND (@parentWorkflowRunId::uuid IS NULL OR t."id" = @parentWorkflowRunId::uuid)
    AND (@parentStepRunId::uuid IS NULL OR t."parentStepRunId" = @parentStepRunId::uuid)
    AND (sqlc.narg('additionalMetadata')::jsonb IS NULL OR
        t."additionalMetadata" @> sqlc.narg('additionalMetadata')::jsonb)
    AND (
        sqlc.narg('statuses')::text[] IS NULL OR
        wr."status" = ANY(cast(sqlc.narg('statuses')::text[] as "WorkflowRunStatus"[]))
        or (
            @includeScheduled::boolean IS TRUE AND
            wr."status" IS NULL
        )
    );

-- name: UpdateScheduledWorkflow :exec
UPDATE "WorkflowTriggerScheduledRef"
SET "triggerAt" = @triggerAt::timestamp
WHERE
    "id" = @scheduleId::uuid;

-- name: DeleteScheduledWorkflow :exec
DELETE FROM "WorkflowTriggerScheduledRef"
WHERE
    "id" = @scheduleId::uuid;

-- name: GetUpstreamErrorsForOnFailureStep :many
WITH workflow_run AS (
    SELECT wr.*
    FROM "WorkflowRun" wr
    JOIN "JobRun" jr ON wr."id" = jr."workflowRunId"
    JOIN "StepRun" sr ON jr."id" = sr."jobRunId"
    WHERE sr."id" = @onFailureStepRunId::uuid
)
SELECT
    sr."id" AS "stepRunId",
    s."readableId" AS "stepReadableId",
    sr."error" AS "stepRunError"
FROM workflow_run wr
JOIN "JobRun" jr ON wr."id" = jr."workflowRunId"
JOIN "StepRun" sr ON jr."id" = sr."jobRunId"
JOIN "Step" s ON sr."stepId" = s."id"
WHERE sr."error" IS NOT NULL;
