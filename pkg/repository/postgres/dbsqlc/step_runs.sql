-- name: GetStepRun :one
SELECT
    "StepRun".*
FROM
    "StepRun"
WHERE
    "id" = @id::uuid AND
    "deletedAt" IS NULL;

-- name: GetStepRunDataForEngine :one
WITH expr_count AS (
    SELECT
        COUNT(*) AS "exprCount",
        sr."id" AS "id"
    FROM
        "StepRun" sr
    JOIN
        "Step" s ON sr."stepId" = s."id"
    JOIN
        "StepExpression" se ON s."id" = se."stepId"
    WHERE
        sr."id" = @id::uuid
    GROUP BY
        sr."id"
)
SELECT
    sr."input",
    sr."output",
    sr."error",
    jrld."data" AS "jobRunLookupData",
    wr."additionalMetadata",
    wr."childIndex",
    wr."childKey",
    wr."parentId",
    COALESCE(ec."exprCount", 0) AS "exprCount"
FROM
    "StepRun" sr
JOIN
    "JobRun" jr ON sr."jobRunId" = jr."id"
JOIN
    "JobRunLookupData" jrld ON jr."id" = jrld."jobRunId"
JOIN
    -- Take advantage of composite index on "JobRun"("workflowRunId", "tenantId")
    "WorkflowRun" wr ON jr."workflowRunId" = wr."id" AND wr."tenantId" = @tenantId::uuid
LEFT JOIN
    expr_count ec ON sr."id" = ec."id"
WHERE
    sr."id" = @id::uuid AND
    sr."tenantId" = @tenantId::uuid;

-- name: GetStepRunBulkDataForEngine :many
SELECT
    sr."id" AS "SR_id",
    sr."retryCount" AS "SR_retryCount",
    sr."input",
    sr."output",
    sr."error",
    sr."status",
    COALESCE(sr."priority", 1) AS "priority",
    jr."id" AS "jobRunId",
    jr."status" AS "jobRunStatus",
    jr."status" AS "jobRunStatus",
    jr."workflowRunId" AS "workflowRunId",
    jrld."data" AS "jobRunLookupData",
    wr."additionalMetadata",
    wr."childIndex",
    wr."childKey",
    wr."parentId",
    jr."id" AS "jobRunId",
    s."id" AS "stepId",
    s."retries" AS "stepRetries",
    s."timeout" AS "stepTimeout",
    s."scheduleTimeout" AS "stepScheduleTimeout",
    s."readableId" AS "stepReadableId",
    s."customUserData" AS "stepCustomUserData",
    j."name" AS "jobName",
    j."id" AS "jobId",
    j."kind" AS "jobKind",
    j."workflowVersionId" AS "workflowVersionId",
    a."actionId" AS "actionId"
FROM
    "StepRun" sr
JOIN
    "Step" s ON sr."stepId" = s."id"
JOIN
    "Action" a ON s."actionId" = a."actionId" AND s."tenantId" = a."tenantId"
JOIN
    "JobRun" jr ON sr."jobRunId" = jr."id"
JOIN
    "Job" j ON jr."jobId" = j."id"
JOIN
    "JobRunLookupData" jrld ON jr."id" = jrld."jobRunId"
JOIN
    -- Take advantage of composite index on "JobRun"("workflowRunId", "tenantId")
    "WorkflowRun" wr ON jr."workflowRunId" = wr."id" AND wr."tenantId" = @tenantId::uuid
WHERE
    sr."id" = ANY(@ids::uuid[])
    AND sr."tenantId" = @tenantId::uuid;

-- name: ListStepRunExpressionEvals :many
SELECT
    *
FROM
    "StepRunExpressionEval" sre
WHERE
    "stepRunId" = ANY(@stepRunIds::uuid[]);

-- name: CreateStepRunExpressionEvalStrs :exec
INSERT INTO "StepRunExpressionEval" (
    "key",
    "stepRunId",
    "valueStr",
    "kind"
) VALUES (
    unnest(@keys::text[]),
    @stepRunId::uuid,
    unnest(@valuesStr::text[]),
    unnest(cast(@kinds::text[] as"StepExpressionKind"[]))
) ON CONFLICT ("key", "stepRunId", "kind") DO UPDATE
SET
    "valueStr" = EXCLUDED."valueStr",
    "valueInt" = EXCLUDED."valueInt";

-- name: CreateStepRunExpressionEvalInts :exec
INSERT INTO "StepRunExpressionEval" (
    "key",
    "stepRunId",
    "valueInt",
    "kind"
) VALUES (
    unnest(@keys::text[]),
    @stepRunId::uuid,
    unnest(@valuesInt::int[]),
    unnest(cast(@kinds::text[] as"StepExpressionKind"[]))
) ON CONFLICT ("key", "stepRunId", "kind") DO UPDATE
SET
    "valueStr" = EXCLUDED."valueStr",
    "valueInt" = EXCLUDED."valueInt";

-- name: GetStepExpressions :many
SELECT
    *
FROM
    "StepExpression"
WHERE
    "stepId" = @stepId::uuid;

-- name: GetStepRunMeta :one
SELECT
    jr."workflowRunId" AS "workflowRunId",
    sr."retryCount" AS "retryCount",
    s."retries" as "retries"
FROM "StepRun" sr
JOIN "Step" s ON sr."stepId" = s."id"
JOIN "JobRun" jr ON sr."jobRunId" = jr."id"
WHERE sr."id" = @stepRunId::uuid
AND sr."tenantId" = @tenantId::uuid;

-- name: GetStepRunForEngine :many
WITH child_count AS (
    SELECT
        COUNT(*) AS "childCount",
        sr."id" AS "id"
    FROM
        "StepRun" sr
    LEFT JOIN
        "_StepRunOrder" AS step_run_order ON sr."id" = step_run_order."A"
    WHERE
        sr."id" = ANY(@ids::uuid[])
        AND step_run_order IS NOT NULL
    GROUP BY
        sr."id"
)
SELECT
    DISTINCT ON (sr."id")
    sr."id" AS "SR_id",
    sr."createdAt" AS "SR_createdAt",
    sr."updatedAt" AS "SR_updatedAt",
    sr."deletedAt" AS "SR_deletedAt",
    sr."tenantId" AS "SR_tenantId",
    sr."queue" AS "SR_queue",
    sr."order" AS "SR_order",
    sqi."workerId" AS "SR_workerId",
    sr."tickerId" AS "SR_tickerId",
    sr."status" AS "SR_status",
    sr."requeueAfter" AS "SR_requeueAfter",
    sr."scheduleTimeoutAt" AS "SR_scheduleTimeoutAt",
    sr."startedAt" AS "SR_startedAt",
    sr."finishedAt" AS "SR_finishedAt",
    sr."timeoutAt" AS "SR_timeoutAt",
    sr."cancelledAt" AS "SR_cancelledAt",
    sr."cancelledReason" AS "SR_cancelledReason",
    sr."cancelledError" AS "SR_cancelledError",
    sr."callerFiles" AS "SR_callerFiles",
    sr."gitRepoBranch" AS "SR_gitRepoBranch",
    sr."retryCount" AS "SR_retryCount",
    sr."semaphoreReleased" AS "SR_semaphoreReleased",
    sr."priority" AS "SR_priority",
    COALESCE(cc."childCount", 0) AS "SR_childCount",
    -- TODO: everything below this line is cacheable and should be moved to a separate query
    jr."id" AS "jobRunId",
    s."id" AS "stepId",
    s."retries" AS "stepRetries",
    s."timeout" AS "stepTimeout",
    s."scheduleTimeout" AS "stepScheduleTimeout",
    s."readableId" AS "stepReadableId",
    s."customUserData" AS "stepCustomUserData",
    s."retryBackoffFactor" AS "stepRetryBackoffFactor",
    s."retryMaxBackoff" AS "stepRetryMaxBackoff",
    j."name" AS "jobName",
    j."id" AS "jobId",
    j."kind" AS "jobKind",
    j."workflowVersionId" AS "workflowVersionId",
    jr."status" AS "jobRunStatus",
    jr."workflowRunId" AS "workflowRunId",
    a."actionId" AS "actionId",
    sticky."strategy" AS "stickyStrategy",
    sticky."desiredWorkerId" AS "desiredWorkerId"
FROM
    "StepRun" sr
LEFT JOIN
    child_count cc ON sr."id" = cc."id"
JOIN
    "Step" s ON sr."stepId" = s."id"
JOIN
    "Action" a ON s."actionId" = a."actionId" AND s."tenantId" = a."tenantId"
JOIN
    "JobRun" jr ON sr."jobRunId" = jr."id"
JOIN
    "Job" j ON jr."jobId" = j."id"
LEFT JOIN
    "SemaphoreQueueItem" sqi ON sr."id" = sqi."stepRunId"
LEFT JOIN
    "WorkflowRunStickyState" sticky ON jr."workflowRunId" = sticky."workflowRunId"
WHERE
    sr."id" = ANY(@ids::uuid[]) AND
    sr."deletedAt" IS NULL AND
    jr."deletedAt" IS NULL AND
    (
        sqlc.narg('tenantId')::uuid IS NULL OR
        sr."tenantId" = sqlc.narg('tenantId')::uuid
    );

-- name: ListInitialStepRuns :many
SELECT
    DISTINCT ON (child_run."id")
    child_run."id" AS "id"
FROM
    "StepRun" AS child_run
LEFT JOIN
    "_StepRunOrder" AS step_run_order ON step_run_order."B" = child_run."id"
WHERE
    child_run."jobRunId" = @jobRunId::uuid
    AND (
        child_run."status" = 'PENDING' OR
        child_run."status" = 'BACKOFF'
    )
    AND step_run_order."A" IS NULL;

-- name: ListStartableStepRunsManyParents :many
SELECT
    DISTINCT ON (child_run."id")
    child_run."id" AS "id"
FROM
    "StepRun" AS parent_run
LEFT JOIN
    "_StepRunOrder" AS step_run_order ON step_run_order."A" = parent_run."id"
JOIN
    "StepRun" AS child_run ON step_run_order."B" = child_run."id"
WHERE
    parent_run."id" = @parentStepRunId::uuid
    AND (
        child_run."status" = 'PENDING' OR
        child_run."status" = 'BACKOFF'
    )
    -- we look for whether the step run is startable by ensuring that all parent step runs have succeeded
    AND NOT EXISTS (
        SELECT 1
        FROM "_StepRunOrder" AS parent_order
        JOIN "StepRun" AS parent_run ON parent_order."A" = parent_run."id"
        WHERE
            parent_order."B" = child_run."id"
            AND parent_run."status" != 'SUCCEEDED'
    )
    -- AND we ensure that there's at least 2 parent step runs
    AND EXISTS (
        SELECT 1
        FROM "_StepRunOrder" AS parent_order
        JOIN "StepRun" AS parent_run ON parent_order."A" = parent_run."id"
        WHERE
            parent_order."B" = child_run."id"
        OFFSET 1
    );

-- name: ListStartableStepRunsSingleParent :many
SELECT
    DISTINCT ON (child_run."id")
    child_run."id" AS "id"
FROM
    "StepRun" AS parent_run
LEFT JOIN
    "_StepRunOrder" AS step_run_order ON step_run_order."A" = parent_run."id"
JOIN
    "StepRun" AS child_run ON step_run_order."B" = child_run."id"
WHERE
    parent_run."id" = @parentStepRunId::uuid
    AND (
        child_run."status" = 'PENDING' OR
        child_run."status" = 'BACKOFF'
    )
    -- we look for whether the step run is startable ASSUMING that parentStepRunId has succeeded,
    -- but we only have one parent step run
    AND NOT EXISTS (
        SELECT 1
        FROM "_StepRunOrder" AS parent_order
        JOIN "StepRun" AS parent_run ON parent_order."A" = parent_run."id"
        WHERE
            parent_order."B" = child_run."id"
            AND parent_run."id" != @parentStepRunId::uuid
    );

-- name: ListStepRuns :many
SELECT
    DISTINCT ON ("StepRun"."order","StepRun"."id")
    "StepRun"."id"
FROM
    "StepRun"
JOIN
    "JobRun" ON "StepRun"."jobRunId" = "JobRun"."id"
WHERE
    "StepRun"."deletedAt" IS NULL AND
    "JobRun"."deletedAt" IS NULL AND
    (
        sqlc.narg('tenantId')::uuid IS NULL OR
        "StepRun"."tenantId" = sqlc.narg('tenantId')::uuid
    )
    AND (
        sqlc.narg('status')::"StepRunStatus" IS NULL OR
        "StepRun"."status" = sqlc.narg('status')::"StepRunStatus"
    )
    AND (
        sqlc.narg('workflowRunIds')::uuid[] IS NULL OR
        "JobRun"."workflowRunId" = ANY(sqlc.narg('workflowRunIds')::uuid[])
    )
    AND (
        sqlc.narg('jobRunId')::uuid IS NULL OR
        "StepRun"."jobRunId" = sqlc.narg('jobRunId')::uuid
    )
    AND (
        sqlc.narg('tickerId')::uuid IS NULL OR
        "StepRun"."tickerId" = sqlc.narg('tickerId')::uuid
    )

ORDER BY "StepRun"."order" ASC    ;

-- name: ListStepRunsToCancel :many
SELECT
    DISTINCT ON ("StepRun"."id")
    "StepRun"."id"
FROM
    "StepRun"
JOIN
    "JobRun" ON "StepRun"."jobRunId" = "JobRun"."id"
WHERE
    "StepRun"."deletedAt" IS NULL AND
    "JobRun"."deletedAt" IS NULL AND
    "StepRun"."tenantId" = @tenantId::uuid AND
    "StepRun"."jobRunId" = @jobRunId::uuid AND
    "StepRun"."status" = ANY(ARRAY['PENDING', 'PENDING_ASSIGNMENT', 'ASSIGNED', 'RUNNING', 'CANCELLING', 'BACKOFF']::"StepRunStatus"[]);

-- name: QueueStepRun :exec
UPDATE
    "StepRun"
SET
    "finishedAt" = NULL,
    "status" = 'PENDING_ASSIGNMENT',
    "input" = COALESCE(sqlc.narg('input')::jsonb, "input"),
    "output" = NULL,
    "error" = NULL,
    "cancelledAt" = NULL,
    "cancelledReason" = NULL,
    "retryCount" = CASE
        WHEN sqlc.narg('isRetry')::boolean IS NOT NULL THEN "retryCount" + 1
        ELSE "retryCount"
    END,
    "semaphoreReleased" = false
WHERE
  "id" = @id::uuid AND
  "tenantId" = @tenantId::uuid;

-- name: QueueStepRunBulkWithInput :exec
WITH input AS (
    SELECT
        unnest(@ids::uuid[]) AS "id",
        unnest(@inputs::jsonb[]) AS "input",
        unnest(@retryCounts::int[]) AS "retryCount"
)
UPDATE
    "StepRun" sr
SET
    "finishedAt" = NULL,
    "status" = 'PENDING_ASSIGNMENT',
    "input" = COALESCE(input."input", sr."input"),
    "output" = NULL,
    "error" = NULL,
    "cancelledAt" = NULL,
    "cancelledReason" = NULL,
    "retryCount" = input."retryCount",
    "semaphoreReleased" = false
FROM
    input
WHERE
    sr."id" = input."id";

-- name: QueueStepRunBulkNoInput :exec
WITH input AS (
    SELECT
        unnest(@ids::uuid[]) AS "id",
        unnest(@retryCounts::int[]) AS "retryCount"
)
UPDATE
    "StepRun" sr
SET
    "finishedAt" = NULL,
    "status" = 'PENDING_ASSIGNMENT',
    "output" = NULL,
    "error" = NULL,
    "cancelledAt" = NULL,
    "cancelledReason" = NULL,
    "retryCount" = input."retryCount",
    "semaphoreReleased" = false
FROM
    input
WHERE
    sr."id" = input."id";

-- name: ManualReleaseSemaphore :exec
UPDATE
    "StepRun"
SET
    -- note that workerId has already been removed via SemaphoreQueueItem
    "semaphoreReleased" = true
WHERE
    "id" = @stepRunId::uuid AND
    "tenantId" = @tenantId::uuid;

-- name: BulkStartStepRun :exec
UPDATE
    "StepRun"
SET
    "status" = CASE
        -- Final states are final, cannot be updated, and we cannot go from cancelling to a non-final state
        WHEN "status" IN ('SUCCEEDED', 'FAILED', 'CANCELLED', 'CANCELLING') THEN "status"
        ELSE 'RUNNING'
    END,
    "startedAt" = input."startedAt"
FROM (
    SELECT
        unnest(@stepRunIds::uuid[]) AS "id",
        unnest(@startedAts::timestamp[]) AS "startedAt"
    ) AS input
WHERE
    "StepRun"."id" = input."id";

-- name: ValidatesAsJson :exec
SELECT @input::jsonb AS "is_valid";

-- name: BulkFinishStepRun :exec
UPDATE
    "StepRun"
SET
    "status" = CASE
        WHEN "status" IN ('SUCCEEDED', 'FAILED', 'CANCELLED') THEN "status"
        ELSE 'SUCCEEDED'
    END,
    "finishedAt" = input."finishedAt",
    "output" = input."output"::jsonb
FROM (
    SELECT
        unnest(@stepRunIds::uuid[]) AS "id",
        unnest(@finishedAts::timestamp[]) AS "finishedAt",
        unnest(@outputs::jsonb[]) AS "output"
    ) AS input
WHERE
    "StepRun"."id" = input."id";

-- name: BulkCancelStepRun :exec
WITH input AS (
    SELECT
        unnest(@stepRunIds::uuid[]) AS "id",
        unnest(@finishedAts::timestamp[]) AS "finishedAt",
        unnest(@cancelledAts::timestamp[]) AS "cancelledAt",
        unnest(@cancelledReasons::text[]) AS "cancelledReason",
        unnest(@cancelledErrors::text[]) AS "cancelledError"
)
UPDATE "StepRun"
SET
    "status" = CASE
        -- Final states are final, cannot be updated
        WHEN "status" IN ('SUCCEEDED', 'FAILED', 'CANCELLED') THEN "status"
        ELSE 'CANCELLED'
    END,
    "finishedAt" = input."finishedAt",
    "cancelledAt" = input."cancelledAt",
    "cancelledReason" = input."cancelledReason",
    "cancelledError" = input."cancelledError"
FROM input
WHERE "StepRun"."id" = input."id";


-- name: BulkFailStepRun :exec
UPDATE
    "StepRun"
SET
    "status" = CASE
        -- Final states are final, cannot be updated
        WHEN "status" IN ('SUCCEEDED', 'FAILED', 'CANCELLED') THEN "status"
        ELSE 'FAILED'
    END,
    "finishedAt" = input."finishedAt",
    "error" = input."error"::text
FROM (
    SELECT
        unnest(@stepRunIds::uuid[]) AS "id",
        unnest(@finishedAts::timestamp[]) AS "finishedAt",
        unnest(@errors::text[]) AS "error"
    ) AS input
WHERE
    "StepRun"."id" = input."id";


-- name: BulkBackoffStepRun :exec
UPDATE
    "StepRun" sr
SET
    "status" = 'BACKOFF'
FROM (
    SELECT "id"
    FROM "StepRun" sr2
    WHERE sr2."id" = ANY(@stepRunIds::uuid[])
    ORDER BY  sr2."id"
    FOR UPDATE
) upd
WHERE sr."id" = upd."id";



-- name: BulkRetryStepRun :exec
UPDATE
    "StepRun" sr
SET
    "status" = 'PENDING_ASSIGNMENT'
FROM (
    SELECT "id"
    FROM "StepRun" sr2
    WHERE sr2."id" = ANY(@stepRunIds::uuid[])
    ORDER BY  sr2."id"
    FOR UPDATE
) upd
WHERE sr."id" = upd."id";

-- name: ResolveLaterStepRuns :many
WITH RECURSIVE currStepRun AS (
  SELECT "id", "status", "cancelledReason"
  FROM "StepRun"
  WHERE
    "id" = @stepRunId::uuid
), childStepRuns AS (
  SELECT sr."id", sr."status"
  FROM "StepRun" sr
  JOIN "_StepRunOrder" sro ON sr."id" = sro."B"
  WHERE sro."A" = (SELECT "id" FROM currStepRun)

  UNION ALL

  SELECT sr."id", sr."status"
  FROM "StepRun" sr
  JOIN "_StepRunOrder" sro ON sr."id" = sro."B"
  JOIN childStepRuns csr ON sro."A" = csr."id"
)
UPDATE
    "StepRun" as sr
SET  "status" = CASE
    -- When the step is in a final state, it cannot be updated
    WHEN sr."status" IN ('SUCCEEDED', 'FAILED', 'CANCELLED') THEN sr."status"
    -- When the given step run has failed or been cancelled, then all child step runs are cancelled
    WHEN @status::"StepRunStatus" IN ('FAILED', 'CANCELLED') THEN 'CANCELLED'
    ELSE sr."status"
    END,
    -- When the previous step run timed out, the cancelled reason is set
    "cancelledReason" = CASE
    -- When the step is in a final state, it cannot be updated
    WHEN sr."status" IN ('SUCCEEDED', 'FAILED', 'CANCELLED') THEN sr."cancelledReason"
    WHEN @status::"StepRunStatus" = 'CANCELLED' AND (SELECT "cancelledReason" FROM currStepRun) = 'TIMED_OUT'::text THEN 'PREVIOUS_STEP_TIMED_OUT'
    WHEN @status::"StepRunStatus" = 'FAILED' THEN 'PREVIOUS_STEP_FAILED'
    WHEN @status::"StepRunStatus" = 'CANCELLED' THEN 'PREVIOUS_STEP_CANCELLED'
    ELSE NULL
    END
FROM
    childStepRuns csr
WHERE
    sr."id" = csr."id"
RETURNING sr.*;

-- name: UpdateStepRunOverridesData :one
UPDATE
    "StepRun" AS sr
SET
    "updatedAt" = CURRENT_TIMESTAMP,
    "input" = jsonb_set("input", @fieldPath::text[], @jsonData::jsonb, true),
    "callerFiles" = jsonb_set("callerFiles", @overridesKey::text[], to_jsonb(@callerFile::text), true)
WHERE
    sr."tenantId" = @tenantId::uuid AND
    sr."id" = @stepRunId::uuid
RETURNING "input";

-- name: UpdateStepRunInputSchema :one
UPDATE
    "StepRun" sr
SET
    "inputSchema" = coalesce(sqlc.narg('inputSchema')::jsonb, '{}'),
    "updatedAt" = CURRENT_TIMESTAMP
WHERE
    sr."tenantId" = @tenantId::uuid AND
    sr."id" = @stepRunId::uuid
RETURNING "inputSchema";

-- name: ArchiveStepRunResultFromStepRun :one
WITH step_run_data AS (
    SELECT
        "id" AS step_run_id,
        "createdAt",
        "updatedAt",
        "deletedAt",
        "retryCount",
        "order",
        "input",
        "output",
        "error",
        "startedAt",
        "finishedAt",
        "timeoutAt",
        "cancelledAt",
        "cancelledReason",
        "cancelledError"
    FROM "StepRun"
    WHERE
        "id" = @stepRunId::uuid
        AND "tenantId" = @tenantId::uuid
        AND "deletedAt" IS NULL
)
INSERT INTO "StepRunResultArchive" (
    "id",
    "createdAt",
    "updatedAt",
    "deletedAt",
    "stepRunId",
    "retryCount",
    "input",
    "output",
    "error",
    "startedAt",
    "finishedAt",
    "timeoutAt",
    "cancelledAt",
    "cancelledReason",
    "cancelledError"
)
SELECT
    COALESCE(sqlc.arg('id')::uuid, gen_random_uuid()),
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    step_run_data."deletedAt",
    step_run_data.step_run_id,
    step_run_data."retryCount",
    step_run_data."input",
    step_run_data."output",
    COALESCE(sqlc.narg('error')::text, step_run_data."error"),
    step_run_data."startedAt",
    step_run_data."finishedAt",
    step_run_data."timeoutAt",
    step_run_data."cancelledAt",
    step_run_data."cancelledReason",
    step_run_data."cancelledError"
FROM step_run_data
RETURNING *;

-- name: ListStepRunsToReassign :many
WITH step_runs_on_inactive_workers AS (
    SELECT
        sr."id",
        sr."tenantId",
        sr."scheduleTimeoutAt",
        sr."retryCount",
        sr."internalRetryCount",
        sqi."workerId",
        s."actionId",
        s."id" AS "stepId",
        s."timeout" AS "stepTimeout",
        s."scheduleTimeout" AS "scheduleTimeout"
    FROM
        "Worker" w
    LEFT JOIN
        "SemaphoreQueueItem" sqi ON w."id" = sqi."workerId"
    JOIN
        "StepRun" sr ON sr."id" = sqi."stepRunId"
    JOIN
        "Step" s ON sr."stepId" = s."id"
    WHERE
        w."tenantId" = @tenantId::uuid
        AND w."lastHeartbeatAt" < NOW() - INTERVAL '30 seconds'),
step_runs_to_reassign AS (
    SELECT
        *
    FROM
        step_runs_on_inactive_workers
    WHERE
        "internalRetryCount" < @maxInternalRetryCount::int
),
step_runs_to_fail AS (
    SELECT
        *
    FROM
        step_runs_on_inactive_workers
    WHERE
        "internalRetryCount" >= @maxInternalRetryCount::int
),
deleted_sqis AS (
    DELETE FROM
        "SemaphoreQueueItem" sqi
    -- delete when step run id AND worker id tuples match
    USING
        step_runs_on_inactive_workers srs
    WHERE
        sqi."stepRunId" = srs."id"
        AND sqi."workerId" = srs."workerId"
),
deleted_tqis AS (
    DELETE FROM
        "TimeoutQueueItem" tqi
    -- delete when step run id AND retry count tuples match
    USING
        step_runs_on_inactive_workers srs
    WHERE
        tqi."stepRunId" = srs."id"
        AND tqi."retryCount" = srs."retryCount"
),
inserted_queue_items AS (
    INSERT INTO "QueueItem" (
        "stepRunId",
        "stepId",
        "actionId",
        "scheduleTimeoutAt",
        "stepTimeout",
        "priority",
        "isQueued",
        "tenantId",
        "queue"
    )
    SELECT
        srs."id",
        srs."stepId",
        srs."actionId",
        CURRENT_TIMESTAMP + COALESCE(convert_duration_to_interval(srs."scheduleTimeout"), INTERVAL '5 minutes'),
        srs."stepTimeout",
        -- Queue with priority 4 so that reassignment gets highest priority
        4,
        true,
        srs."tenantId",
        srs."actionId"
    FROM
        step_runs_to_reassign srs
),
updated_step_runs AS (
    UPDATE "StepRun" sr
    SET
        "status" = 'PENDING_ASSIGNMENT',
        "scheduleTimeoutAt" = CURRENT_TIMESTAMP + COALESCE(convert_duration_to_interval(srs."scheduleTimeout"), INTERVAL '5 minutes'),
        "updatedAt" = CURRENT_TIMESTAMP,
        "internalRetryCount" = sr."internalRetryCount" + 1
    FROM step_runs_to_reassign srs
    WHERE sr."id" = srs."id"
    RETURNING sr."id"
)
SELECT
    srs1."id",
    srs1."workerId",
    srs1."retryCount",
    'REASSIGNED' AS "operation"
FROM
    step_runs_to_reassign srs1
UNION ALL
SELECT
    srs2."id",
    srs2."workerId",
    srs2."retryCount",
    'FAILED' AS "operation"
FROM
    step_runs_to_fail srs2;

-- name: InternalRetryStepRuns :many
WITH step_runs AS (
    SELECT
        sr."id",
        sr."tenantId",
        sr."scheduleTimeoutAt",
        sr."retryCount",
        sr."internalRetryCount",
        s."actionId",
        s."id" AS "stepId",
        s."timeout" AS "stepTimeout",
        s."scheduleTimeout" AS "scheduleTimeout"
    FROM
        "StepRun" sr
    JOIN
        "Step" s ON sr."stepId" = s."id"
    WHERE
        sr."tenantId" = @tenantId::uuid
        AND sr."id" = ANY(@stepRunIds::uuid[])
),
step_runs_to_reassign AS (
    SELECT
        *
    FROM
        step_runs
    WHERE
        "internalRetryCount" < @maxInternalRetryCount::int
),
step_runs_to_fail AS (
    SELECT
        *
    FROM
        step_runs
    WHERE
        "internalRetryCount" >= @maxInternalRetryCount::int
),
deleted_sqis AS (
    DELETE FROM
        "SemaphoreQueueItem" sqi
    USING
        step_runs srs
    WHERE
        sqi."stepRunId" = srs."id"
),
deleted_tqis AS (
    DELETE FROM
        "TimeoutQueueItem" tqi
    -- delete when step run id AND retry count tuples match
    USING
        step_runs srs
    WHERE
        tqi."stepRunId" = srs."id"
        AND tqi."retryCount" = srs."retryCount"
),
inserted_queue_items AS (
    INSERT INTO "QueueItem" (
        "stepRunId",
        "stepId",
        "actionId",
        "scheduleTimeoutAt",
        "stepTimeout",
        "priority",
        "isQueued",
        "tenantId",
        "queue"
    )
    SELECT
        srs."id",
        srs."stepId",
        srs."actionId",
        CURRENT_TIMESTAMP + COALESCE(convert_duration_to_interval(srs."scheduleTimeout"), INTERVAL '5 minutes'),
        srs."stepTimeout",
        -- Queue with priority 4 so that reassignment gets highest priority
        4,
        true,
        srs."tenantId",
        srs."actionId"
    FROM
        step_runs_to_reassign srs
),
updated_step_runs AS (
    UPDATE "StepRun" sr
    SET
        "status" = 'PENDING_ASSIGNMENT',
        "scheduleTimeoutAt" = CURRENT_TIMESTAMP + COALESCE(convert_duration_to_interval(srs."scheduleTimeout"), INTERVAL '5 minutes'),
        "updatedAt" = CURRENT_TIMESTAMP,
        "internalRetryCount" = sr."internalRetryCount" + 1
    FROM step_runs_to_reassign srs
    WHERE sr."id" = srs."id"
    RETURNING sr."id"
)
SELECT
    srs1."id",
    srs1."retryCount",
    'REASSIGNED' AS "operation"
FROM
    step_runs_to_reassign srs1
UNION ALL
SELECT
    srs2."id",
    srs2."retryCount",
    'FAILED' AS "operation"
FROM
    step_runs_to_fail srs2;

-- name: ListStepRunsToTimeout :many
SELECT "id"
FROM "StepRun"
WHERE
    "status" = ANY(ARRAY['RUNNING', 'ASSIGNED']::"StepRunStatus"[])
    AND "timeoutAt" < NOW()
    AND "tenantId" = @tenantId::uuid
LIMIT 100;

-- name: RefreshTimeoutBy :one
WITH step_run AS (
    SELECT
        "id",
        "retryCount",
        "tenantId"
    FROM
        "StepRun"
    WHERE
        "id" = @stepRunId::uuid AND
        "tenantId" = @tenantId::uuid
)
INSERT INTO
    "TimeoutQueueItem" (
        "stepRunId",
        "retryCount",
        "timeoutAt",
        "tenantId",
        "isQueued"
    )
SELECT
    sr."id",
    sr."retryCount",
    NOW() + convert_duration_to_interval(sqlc.narg('incrementTimeoutBy')::text),
    sr."tenantId",
    true
FROM
    step_run sr
ON CONFLICT ("stepRunId", "retryCount") DO UPDATE
SET
    "timeoutAt" = "TimeoutQueueItem"."timeoutAt" + convert_duration_to_interval(sqlc.narg('incrementTimeoutBy')::text)
RETURNING "TimeoutQueueItem"."timeoutAt";

-- name: UpdateStepRunUnsetWorkerId :one
WITH oldsr AS (
    SELECT
        "id",
        "retryCount"
    FROM
        "StepRun"
    WHERE
        "id" = @stepRunId::uuid AND
        "tenantId" = @tenantId::uuid
), deleted_sqi AS (
    DELETE FROM
        "SemaphoreQueueItem" sqi
    WHERE
        sqi."stepRunId" = @stepRunId::uuid
    RETURNING sqi."workerId"
)
SELECT
    deleted_sqi."workerId" AS "workerId",
    oldsr."retryCount" AS "retryCount"
FROM
    deleted_sqi, oldsr;

-- name: VerifiedStepRunTenantIds :many
WITH input AS (
    SELECT
        unnest(@stepRunIds::uuid[]) AS "id",
        unnest(@tenantIds::uuid[]) AS "tenantId"
)
SELECT
    sr."id"
FROM "StepRun" sr
JOIN input ON sr."id" = input."id" AND sr."tenantId" = input."tenantId"
-- stable ordering as it minimizes the chance of deadlocks
ORDER BY sr."id";

-- name: UpdateStepRunUnsetWorkerIdBulk :exec
DELETE FROM
    "SemaphoreQueueItem"
WHERE
    "stepRunId" = ANY(@stepRunIds::uuid[]);

-- name: RemoveTimeoutQueueItems :exec
DELETE FROM
    "TimeoutQueueItem"
WHERE
    "stepRunId" = ANY(@stepRunIds::uuid[]);

-- name: CheckWorker :one
SELECT
    "id"
FROM
    "Worker"
WHERE
    "tenantId" = @tenantId::uuid
    AND "dispatcherId" IS NOT NULL
    AND "isActive" = true
    AND "isPaused" = false
    AND "lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
    AND "id" = @workerId::uuid;

-- name: GetWorkerDispatcherActions :many
WITH actions AS (
    SELECT
        "id",
        "actionId"
    FROM
        "Action"
    WHERE
        "tenantId" = @tenantId::uuid AND
        "actionId" = ANY(@actionIds::text[])
)
SELECT
    w."id",
    a."actionId",
    w."dispatcherId"
FROM
    "Worker" w
JOIN
    "_ActionToWorker" atw ON w."id" = atw."B"
JOIN
    actions a ON atw."A" = a."id"
WHERE
    w."tenantId" = @tenantId::uuid
    AND w."dispatcherId" IS NOT NULL
    AND w."lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
    AND w."isActive" = true
    AND w."isPaused" = false;

-- name: CreateWorkerAssignEvents :exec
INSERT INTO "WorkerAssignEvent" (
    "workerId",
    "assignedStepRuns"
)
SELECT
    input."workerId",
    input."assignedStepRuns"
FROM (
    SELECT
        unnest(@workerIds::uuid[]) AS "workerId",
        unnest(@assignedStepRuns::jsonb[]) AS "assignedStepRuns"
    ) AS input
RETURNING *;

-- name: UpdateStepRunsToAssigned :many
WITH input AS (
    SELECT
        "id",
        "stepTimeout",
        "workerId"
    FROM
        (
            SELECT
                unnest(@stepRunIds::uuid[]) AS "id",
                unnest(@stepRunTimeouts::text[]) AS "stepTimeout",
                unnest(@workerIds::uuid[]) AS "workerId"
        ) AS subquery
    ORDER BY "id"
), updated_step_runs AS (
    SELECT
        sr."id",
        sr."retryCount",
        sr."tenantId",
        CURRENT_TIMESTAMP + convert_duration_to_interval(input."stepTimeout") AS "timeoutAt"
    FROM
        input
    JOIN
        "StepRun" sr ON sr."id" = input."id"
    ORDER BY sr."id"
), assigned_step_runs AS (
    INSERT INTO "SemaphoreQueueItem" (
        "stepRunId",
        "workerId",
        "tenantId"
    )
    SELECT
        input."id",
        input."workerId",
        @tenantId::uuid
    FROM
        input
    ON CONFLICT ("stepRunId") DO NOTHING
    -- only return the step run ids that were successfully assigned
    RETURNING "stepRunId", "workerId"
), timeout_insert AS (
    -- bulk insert into timeout queue items
    INSERT INTO
        "TimeoutQueueItem" (
            "stepRunId",
            "retryCount",
            "timeoutAt",
            "tenantId",
            "isQueued"
        )
    SELECT
        sr."id",
        sr."retryCount",
        sr."timeoutAt",
        sr."tenantId",
        true
    FROM
        updated_step_runs sr
    JOIN
        assigned_step_runs asr ON sr."id" = asr."stepRunId"
    ON CONFLICT ("stepRunId", "retryCount") DO UPDATE
    SET
        "timeoutAt" = EXCLUDED."timeoutAt"
    RETURNING
        "stepRunId"
)
SELECT
    asr."stepRunId",
    asr."workerId"
FROM
    assigned_step_runs asr;

-- name: GetFinalizedStepRuns :many
SELECT
    "id", "status"
FROM
    "StepRun"
WHERE
    "id" = ANY(@stepRunIds::uuid[])
    AND "status" = ANY(ARRAY['SUCCEEDED', 'FAILED', 'CANCELLED', 'CANCELLING']::"StepRunStatus"[]);

-- name: BulkMarkStepRunsAsCancelling :many
UPDATE
    "StepRun" sr
SET
    "status" = CASE
        -- Final states are final, we cannot go from a final state to cancelling
        WHEN "status" IN ('SUCCEEDED', 'FAILED', 'CANCELLED') THEN "status"
        ELSE 'CANCELLING'
    END,
    "updatedAt" = CURRENT_TIMESTAMP
FROM (
    SELECT
        unnest(@stepRunIds::uuid[]) AS "id"
    ) AS input
WHERE
    sr."id" = input."id"
RETURNING sr."id";

-- name: GetDesiredLabels :many
SELECT
    "key",
    "strValue",
    "intValue",
    "required",
    "weight",
    "comparator",
    "stepId"
FROM
    "StepDesiredWorkerLabel"
WHERE
    "stepId" = ANY(@stepIds::uuid[]);

-- name: GetWorkerLabels :many
SELECT
    "key",
    "strValue",
    "intValue"
FROM
    "WorkerLabel"
WHERE
    "workerId" = @workerId::uuid;

-- name: UpsertDesiredWorkerLabel :one
INSERT INTO "StepDesiredWorkerLabel" (
    "createdAt",
    "updatedAt",
    "stepId",
    "key",
    "intValue",
    "strValue",
    "required",
    "weight",
    "comparator"
) VALUES (
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP,
    @stepId::uuid,
    @key::text,
    COALESCE(sqlc.narg('intValue')::int, NULL),
    COALESCE(sqlc.narg('strValue')::text, NULL),
    COALESCE(sqlc.narg('required')::boolean, false),
    COALESCE(sqlc.narg('weight')::int, 100),
    COALESCE(sqlc.narg('comparator')::"WorkerLabelComparator", 'EQUAL')
) ON CONFLICT ("stepId", "key") DO UPDATE
SET
    "updatedAt" = CURRENT_TIMESTAMP,
    "intValue" = COALESCE(sqlc.narg('intValue')::int, null),
    "strValue" = COALESCE(sqlc.narg('strValue')::text, null),
    "required" = COALESCE(sqlc.narg('required')::boolean, false),
    "weight" = COALESCE(sqlc.narg('weight')::int, 100),
    "comparator" = COALESCE(sqlc.narg('comparator')::"WorkerLabelComparator", 'EQUAL')
RETURNING *;

-- name: GetStepDesiredWorkerLabels :one
SELECT
    jsonb_agg(
        jsonb_build_object(
            'key', dwl."key",
            'strValue', dwl."strValue",
            'intValue', dwl."intValue",
            'required', dwl."required",
            'weight', dwl."weight",
            'comparator', dwl."comparator",
            'is_true', false
        )
    ) AS desired_labels
FROM
    "StepDesiredWorkerLabel" dwl
WHERE
    dwl."stepId" = @stepId::uuid;

-- name: CreateStepRunEvent :exec
WITH input_values AS (
    SELECT
        CURRENT_TIMESTAMP AS "timeFirstSeen",
        CURRENT_TIMESTAMP AS "timeLastSeen",
        @stepRunId::uuid AS "stepRunId",
        @jobRunid::uuid AS "jobRunId",
        @reason::"StepRunEventReason" AS "reason",
        @severity::"StepRunEventSeverity" AS "severity",
        @message::text AS "message",
        1 AS "count",
        sqlc.narg('data')::jsonb AS "data"
),
updated AS (
    UPDATE "StepRunEvent"
    SET
        "timeLastSeen" = CURRENT_TIMESTAMP,
        "message" = input_values."message",
        "count" = "StepRunEvent"."count" + 1,
        "data" = input_values."data"
    FROM input_values
    WHERE
        "StepRunEvent"."stepRunId" = input_values."stepRunId"
        AND "StepRunEvent"."reason" = input_values."reason"
        AND "StepRunEvent"."severity" = input_values."severity"
        AND "StepRunEvent"."id" = (
            SELECT "id"
            FROM "StepRunEvent"
            WHERE "stepRunId" = input_values."stepRunId"
            ORDER BY "id" DESC
            LIMIT 1
        )
    RETURNING "StepRunEvent".*
)
INSERT INTO "StepRunEvent" (
    "timeFirstSeen",
    "timeLastSeen",
    "stepRunId",
    "reason",
    "severity",
    "message",
    "count",
    "data"
)
SELECT
    "timeFirstSeen",
    "timeLastSeen",
    "stepRunId",
    "reason",
    "severity",
    "message",
    "count",
    "data"
FROM input_values
WHERE NOT EXISTS (
    SELECT 1 FROM updated WHERE "stepRunId" = input_values."stepRunId"
);

-- name: BulkCreateStepRunEvent :exec
WITH input_values AS (
    SELECT
        unnest(@timeSeen::timestamp[]) AS "timeFirstSeen",
        unnest(@timeSeen::timestamp[]) AS "timeLastSeen",
        unnest(@stepRunIds::uuid[]) AS "stepRunId",
        unnest(cast(@reasons::text[] as"StepRunEventReason"[])) AS "reason",
        unnest(cast(@severities::text[] as "StepRunEventSeverity"[])) AS "severity",
        unnest(@messages::text[]) AS "message",
        1 AS "count",
        unnest(@data::jsonb[]) AS "data"
),
matched_rows AS (
    SELECT DISTINCT ON (sre."stepRunId")
        sre."stepRunId", sre."reason", sre."severity", sre."id"
    FROM "StepRunEvent" sre
    WHERE
        sre."stepRunId" = ANY(@stepRunIds::uuid[])
    ORDER BY sre."stepRunId", sre."id" DESC
),
locked_rows AS (
    SELECT sre."id", iv.*
    FROM "StepRunEvent" sre
    JOIN
        matched_rows mr ON sre."id" = mr."id"
    JOIN
        input_values iv ON sre."stepRunId" = iv."stepRunId" AND sre."reason" = iv."reason" AND sre."severity" = iv."severity"
    ORDER BY "id"
    FOR UPDATE
),
updated AS (
    UPDATE "StepRunEvent"
    SET
        "timeLastSeen" = locked_rows."timeLastSeen",
        "message" = locked_rows."message",
        "count" = "StepRunEvent"."count" + 1,
        "data" = locked_rows."data"
    FROM locked_rows
    WHERE
        "StepRunEvent"."id" = locked_rows."id"
    RETURNING "StepRunEvent".*
)
INSERT INTO "StepRunEvent" (
    "timeFirstSeen",
    "timeLastSeen",
    "stepRunId",
    "reason",
    "severity",
    "message",
    "count",
    "data"
)
SELECT
    "timeFirstSeen",
    "timeLastSeen",
    "stepRunId",
    "reason",
    "severity",
    "message",
    "count",
    "data"
FROM input_values
WHERE NOT EXISTS (
    SELECT 1 FROM updated WHERE "stepRunId" = input_values."stepRunId" AND "reason" = input_values."reason" AND "severity" = input_values."severity"
);

-- name: CountStepRunEvents :one
SELECT
    count(*) OVER() AS total
FROM
    "StepRunEvent"
WHERE
    "stepRunId" = @stepRunId::uuid;

-- name: ListStepRunEvents :many
SELECT
    *
FROM
    "StepRunEvent"
WHERE
    "stepRunId" = @stepRunId::uuid
ORDER BY
    "id" DESC
OFFSET
    COALESCE(sqlc.narg('offset'), 0)
LIMIT
    COALESCE(sqlc.narg('limit'), 50);


-- name: ListStepRunEventsByWorkflowRunId :many
SELECT
    sre.*
FROM
    "StepRunEvent" sre
JOIN
    "StepRun" sr ON sr."id" = sre."stepRunId"
JOIN
    "JobRun" jr ON jr."id" = sr."jobRunId"
WHERE
    jr."workflowRunId" = @workflowRunId::uuid
    AND jr."tenantId" = @tenantId::uuid
    AND sre."id" > COALESCE(sqlc.narg('lastId'), 0)
    -- / TODO ID > Last ID
ORDER BY
    sre."id" DESC;

-- name: ReplayStepRunResetWorkflowRun :one
UPDATE
    "WorkflowRun"
SET
    "status" = 'PENDING',
    "updatedAt" = CURRENT_TIMESTAMP,
    "startedAt" = NULL,
    "finishedAt" = NULL,
    "duration" = NULL,
    "concurrencyGroupId" = NULL,
    "error" = NULL
WHERE
    "id" =  @workflowRunId::uuid
RETURNING *;


-- name: ReplayWorkflowRunResetGetGroupKeyRun :one
UPDATE
    "GetGroupKeyRun"
SET
    "status" = 'PENDING',
    "scheduleTimeoutAt" = NULL,
    "finishedAt" = NULL,
    "startedAt" = NULL,
    "timeoutAt" = NULL,
    "output" = NULL,
    "error" = NULL,
    "cancelledAt" = NULL,
    "cancelledReason" = NULL,
    "cancelledError" = NULL
WHERE
    "workflowRunId" = @workflowRunId::uuid
RETURNING *;


-- name: ReplayStepRunResetJobRun :one
UPDATE
    "JobRun"
SET
    "status" = 'RUNNING',
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

-- name: GetLaterStepRuns :many
WITH RECURSIVE currStepRun AS (
    SELECT *
    FROM "StepRun"
    WHERE
        "id" = @stepRunId::uuid
), childStepRuns AS (
    SELECT sr."id", sr."status"
    FROM "StepRun" sr
    JOIN "_StepRunOrder" sro ON sr."id" = sro."B"
    WHERE sro."A" = (SELECT "id" FROM currStepRun)

    UNION ALL

    SELECT sr."id", sr."status"
    FROM "StepRun" sr
    JOIN "_StepRunOrder" sro ON sr."id" = sro."B"
    JOIN childStepRuns csr ON sro."A" = csr."id"
)
SELECT
    sr.*
FROM
    "StepRun" sr
JOIN
    childStepRuns csr ON sr."id" = csr."id";

-- name: ReplayStepRunResetStepRuns :many
WITH RECURSIVE currStepRun AS (
    SELECT *
    FROM "StepRun"
    WHERE
        "id" = @stepRunId::uuid
), childStepRuns AS (
    SELECT sr."id", sr."status"
    FROM "StepRun" sr
    JOIN "_StepRunOrder" sro ON sr."id" = sro."B"
    WHERE sro."A" = (SELECT "id" FROM currStepRun)

    UNION ALL

    SELECT sr."id", sr."status"
    FROM "StepRun" sr
    JOIN "_StepRunOrder" sro ON sr."id" = sro."B"
    JOIN childStepRuns csr ON sro."A" = csr."id"
)
UPDATE
    "StepRun" as sr
SET
    "status" = 'PENDING',
    "scheduleTimeoutAt" = NULL,
    "finishedAt" = NULL,
    "startedAt" = NULL,
    "output" = NULL,
    "error" = NULL,
    "cancelledAt" = NULL,
    "cancelledReason" = NULL,
    "input" = CASE
        WHEN sr."id" = @stepRunId::uuid THEN COALESCE(sqlc.narg('input')::jsonb, "input")
        ELSE NULL
    END,
    "retryCount" = 0
FROM
    childStepRuns csr
WHERE
    sr."id" = csr."id" OR
    sr."id" = @stepRunId::uuid
RETURNING sr.*;

-- name: ResetStepRunsByIds :many
UPDATE
    "StepRun" as sr
SET
    "status" = 'PENDING',
    "scheduleTimeoutAt" = NULL,
    "finishedAt" = NULL,
    "startedAt" = NULL,
    "output" = NULL,
    "error" = NULL,
    "cancelledAt" = NULL,
    "cancelledReason" = NULL,
    "input" = NULL,
    "retryCount" = 0
WHERE
    sr."id" = ANY(@ids::uuid[]) AND
    sr."tenantId" = @tenantId::uuid
RETURNING sr.*;

-- name: ListNonFinalChildStepRuns :many
WITH RECURSIVE currStepRun AS (
    SELECT *
    FROM "StepRun"
    WHERE
        "id" = @stepRunId::uuid
), childStepRuns AS (
    SELECT sr."id", sr."status"
    FROM "StepRun" sr
    JOIN "_StepRunOrder" sro ON sr."id" = sro."B"
    WHERE sro."A" = (SELECT "id" FROM currStepRun)
        AND sr."deletedAt" IS NULL

    UNION ALL

    SELECT sr."id", sr."status"
    FROM "StepRun" sr
    JOIN "_StepRunOrder" sro ON sr."id" = sro."B"
    JOIN childStepRuns csr ON sro."A" = csr."id"
)
-- Select all child step runs that are not in a final state
SELECT
    sr.*
FROM
    "StepRun" sr
JOIN
    childStepRuns csr ON sr."id" = csr."id"
WHERE
    sr."deletedAt" IS NULL AND
    sr."status" NOT IN ('SUCCEEDED', 'FAILED', 'CANCELLED');

-- name: ListStepRunArchives :many
SELECT
    "StepRunResultArchive".*
FROM
    "StepRunResultArchive"
JOIN
    "StepRun" ON "StepRunResultArchive"."stepRunId" = "StepRun"."id"
WHERE
    "StepRunResultArchive"."stepRunId" = @stepRunId::uuid AND
    "StepRun"."tenantId" = @tenantId::uuid AND
    "StepRun"."deletedAt" IS NULL
ORDER BY
    "StepRunResultArchive"."createdAt" DESC
OFFSET
    COALESCE(sqlc.narg('offset'), 0)
LIMIT
    COALESCE(sqlc.narg('limit'), 50);

-- name: CountStepRunArchives :one
SELECT
    count(*) OVER() AS total
FROM
    "StepRunResultArchive"
WHERE
    "stepRunId" = @stepRunId::uuid;


-- name: ClearStepRunPayloadData :one
WITH for_delete AS (
    SELECT
        sr2."id"
    FROM "StepRun" sr2
    WHERE
        sr2."tenantId" = @tenantId::uuid AND
        sr2."deletedAt" IS NOT NULL AND
        (sr2."input" IS NOT NULL OR sr2."output" IS NOT NULL OR sr2."error" IS NOT NULL)
    ORDER BY "deletedAt" ASC
    LIMIT sqlc.arg('limit') + 1
),
deleted_with_limit AS (
    SELECT
        for_delete."id" as "id"
    FROM for_delete
    LIMIT sqlc.arg('limit')
),
deleted_archives AS (
    SELECT sra1."id" as "id"
    FROM "StepRunResultArchive" sra1
    WHERE
        sra1."stepRunId" IN (SELECT "id" FROM deleted_with_limit)
        AND (sra1."input" IS NOT NULL OR sra1."output" IS NOT NULL OR sra1."error" IS NOT NULL)
),
has_more AS (
    SELECT
        CASE
            WHEN COUNT(*) > sqlc.arg('limit') THEN TRUE
            ELSE FALSE
        END as has_more
    FROM for_delete
),
cleared_archives AS (
    UPDATE "StepRunResultArchive"
    SET
        "input" = NULL,
        "output" = NULL,
        "error" = NULL
    WHERE
        "id" IN (SELECT "id" FROM deleted_archives)
)
UPDATE
    "StepRun"
SET
    "input" = NULL,
    "output" = NULL,
    "error" = NULL
WHERE
    "id" IN (SELECT "id" FROM deleted_with_limit)
RETURNING
    (SELECT has_more FROM has_more) as has_more;

-- name: HasActiveWorkersForActionId :one
SELECT
    COUNT(DISTINCT w."id") AS "total"
FROM
    "Worker" w
JOIN
    "_ActionToWorker" atw ON w."id" = atw."B"
JOIN
    "Action" a ON atw."A" = a."id"
WHERE
    w."tenantId" = @tenantId::uuid
    AND a."actionId" = @actionId::text
    AND w."isActive" = true
    AND w."lastHeartbeatAt" > NOW() - INTERVAL '6 seconds';

-- name: ListChildWorkflowRunIds :many
SELECT
    "id"
FROM
    "WorkflowRun"
WHERE
    "parentStepRunId" = @stepRun::uuid
    AND "tenantId" = @tenantId::uuid
    AND "deletedAt" IS NULL;
