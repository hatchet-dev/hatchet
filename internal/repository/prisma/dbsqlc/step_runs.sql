-- name: GetStepRun :one
SELECT
    "StepRun".*
FROM
    "StepRun"
WHERE
    "id" = @id::uuid AND
    "tenantId" = @tenantId::uuid;

-- name: GetStepRunForEngine :many
SELECT
    sqlc.embed(sr),
    jrld."data" AS "jobRunLookupData",
    -- TODO: everything below this line is cacheable and should be moved to a separate query
    jr."id" AS "jobRunId",
    wr."id" AS "workflowRunId",
    s."id" AS "stepId",
    s."retries" AS "stepRetries",
    s."scheduleTimeout" AS "stepScheduleTimeout",
    s."readableId" AS "stepReadableId",
    s."customUserData" AS "stepCustomUserData",
    j."name" AS "jobName",
    j."id" AS "jobId",
    wv."id" AS "workflowVersionId",
    w."name" AS "workflowName",
    w."id" AS "workflowId",
    a."actionId" AS "actionId"
FROM
    "StepRun" sr
JOIN
    "Step" s ON sr."stepId" = s."id"
JOIN
    "Action" a ON s."actionId" = a."actionId"
JOIN
    "JobRun" jr ON sr."jobRunId" = jr."id"
JOIN
    "JobRunLookupData" jrld ON jr."id" = jrld."jobRunId"
JOIN
    "Job" j ON jr."jobId" = j."id"
JOIN 
    "WorkflowRun" wr ON jr."workflowRunId" = wr."id"
JOIN
    "WorkflowVersion" wv ON wr."workflowVersionId" = wv."id"
JOIN
    "Workflow" w ON wv."workflowId" = w."id"
WHERE
    sr."id" = ANY(@ids::uuid[]) AND
    (
        sqlc.narg('tenantId')::uuid IS NULL OR
        sr."tenantId" = sqlc.narg('tenantId')::uuid
    );

-- name: ListStartableStepRuns :many
WITH job_run AS (
    SELECT "status"
    FROM "JobRun"
    WHERE "id" = @jobRunId::uuid
)
SELECT 
    child_run."id" AS "id"
FROM 
    "StepRun" AS child_run
LEFT JOIN 
    "_StepRunOrder" AS step_run_order ON step_run_order."B" = child_run."id"
JOIN
    job_run ON true
WHERE 
    child_run."jobRunId" = @jobRunId::uuid
    AND child_run."status" = 'PENDING'
    AND job_run."status" = 'RUNNING'
    -- case on whether parentStepRunId is null
    AND (
        (sqlc.narg('parentStepRunId')::uuid IS NULL AND step_run_order."A" IS NULL) OR 
        (
            step_run_order."A" = sqlc.narg('parentStepRunId')::uuid
            AND NOT EXISTS (
                SELECT 1
                FROM "_StepRunOrder" AS parent_order
                JOIN "StepRun" AS parent_run ON parent_order."A" = parent_run."id"
                WHERE 
                    parent_order."B" = child_run."id"
                    AND parent_run."status" != 'SUCCEEDED'
            )
        )
    );

-- name: ListStepRuns :many
SELECT
    "StepRun"."id"
FROM
    "StepRun"
JOIN
    "JobRun" ON "StepRun"."jobRunId" = "JobRun"."id"
WHERE
    "StepRun"."tenantId" = @tenantId::uuid
    AND (
        sqlc.narg('status')::"StepRunStatus" IS NULL OR
        "StepRun"."status" = sqlc.narg('status')::"StepRunStatus"
    )
    AND (
        sqlc.narg('workflowRunId')::uuid IS NULL OR
        "JobRun"."workflowRunId" = sqlc.narg('workflowRunId')::uuid
    )
    AND (
        sqlc.narg('jobRunId')::uuid IS NULL OR
        "StepRun"."jobRunId" = sqlc.narg('jobRunId')::uuid
    )
    AND (
        sqlc.narg('tickerId')::uuid IS NULL OR
        "StepRun"."tickerId" = sqlc.narg('tickerId')::uuid
    );

-- name: UpdateStepRun :one
UPDATE
    "StepRun"
SET
    "requeueAfter" = COALESCE(sqlc.narg('requeueAfter')::timestamp, "requeueAfter"),
    "scheduleTimeoutAt" = COALESCE(sqlc.narg('scheduleTimeoutAt')::timestamp, "scheduleTimeoutAt"),
    "startedAt" = COALESCE(sqlc.narg('startedAt')::timestamp, "startedAt"),
    "finishedAt" = CASE
        -- if this is a rerun, we clear the finishedAt
        WHEN sqlc.narg('rerun')::boolean THEN NULL
        ELSE  COALESCE(sqlc.narg('finishedAt')::timestamp, "finishedAt")
    END,
    "status" = CASE 
        -- if this is a rerun, we permit status updates
        WHEN sqlc.narg('rerun')::boolean THEN COALESCE(sqlc.narg('status'), "status")
        -- Final states are final, cannot be updated
        WHEN "status" IN ('SUCCEEDED', 'FAILED', 'CANCELLED') THEN "status"
        ELSE COALESCE(sqlc.narg('status'), "status")
    END,
    "input" = COALESCE(sqlc.narg('input')::jsonb, "input"),
    "output" = CASE
        -- if this is a rerun, we clear the output
        WHEN sqlc.narg('rerun')::boolean THEN NULL
        ELSE COALESCE(sqlc.narg('output')::jsonb, "output")
    END,
    "error" = CASE
        -- if this is a rerun, we clear the error
        WHEN sqlc.narg('rerun')::boolean THEN NULL
        ELSE COALESCE(sqlc.narg('error')::text, "error")
    END,
    "cancelledAt" = CASE
        -- if this is a rerun, we clear the cancelledAt
        WHEN sqlc.narg('rerun')::boolean THEN NULL
        ELSE COALESCE(sqlc.narg('cancelledAt')::timestamp, "cancelledAt")
    END,
    "cancelledReason" = CASE
        -- if this is a rerun, we clear the cancelledReason
        WHEN sqlc.narg('rerun')::boolean THEN NULL
        ELSE COALESCE(sqlc.narg('cancelledReason')::text, "cancelledReason")
    END,
    "retryCount" = COALESCE(sqlc.narg('retryCount')::int, "retryCount")
WHERE 
  "id" = @id::uuid AND
  "tenantId" = @tenantId::uuid
RETURNING "StepRun".*;

-- name: ResolveLaterStepRuns :many
WITH currStepRun AS (
  SELECT *
  FROM "StepRun"
  WHERE
    "id" = @stepRunId::uuid AND
    "tenantId" = @tenantId::uuid
)
UPDATE
    "StepRun" as sr
SET "status" = CASE
    -- When the given step run has failed or been cancelled, then all later step runs are cancelled
    WHEN (cs."status" = 'FAILED' OR cs."status" = 'CANCELLED') THEN 'CANCELLED'
    ELSE sr."status"
    END,
    -- When the previous step run timed out, the cancelled reason is set
    "cancelledReason" = CASE
    WHEN (cs."status" = 'CANCELLED' AND cs."cancelledReason" = 'TIMED_OUT'::text) THEN 'PREVIOUS_STEP_TIMED_OUT'
    WHEN (cs."status" = 'CANCELLED') THEN 'PREVIOUS_STEP_CANCELLED'
    ELSE NULL
    END
FROM
    currStepRun cs
WHERE
    sr."jobRunId" = (
        SELECT "jobRunId"
        FROM "StepRun"
        WHERE "id" = @stepRunId::uuid
    ) AND
    sr."order" > (
        SELECT "order"
        FROM "StepRun"
        WHERE "id" = @stepRunId::uuid
    ) AND
    sr."tenantId" = @tenantId::uuid
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
    WHERE "id" = @stepRunId::uuid AND "tenantId" = @tenantId::uuid
)
INSERT INTO "StepRunResultArchive" (
    "id",
    "createdAt",
    "updatedAt",
    "deletedAt",
    "stepRunId",
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
    step_run_data."input",
    step_run_data."output",
    step_run_data."error",
    step_run_data."startedAt",
    step_run_data."finishedAt",
    step_run_data."timeoutAt",
    step_run_data."cancelledAt",
    step_run_data."cancelledReason",
    step_run_data."cancelledError"
FROM step_run_data
RETURNING *;

-- name: ListStepRunsToReassign :many
SELECT
    sr.*
FROM
    "StepRun" sr
LEFT JOIN
    "Worker" w ON sr."workerId" = w."id"
JOIN
    "Step" s ON sr."stepId" = s."id"
WHERE
    sr."tenantId" = @tenantId::uuid
    AND ((
        sr."status" = 'RUNNING'
        AND w."lastHeartbeatAt" < NOW() - INTERVAL '60 seconds'
        AND s."retries" > sr."retryCount"
    ) OR (
        sr."status" = 'ASSIGNED'
        AND w."lastHeartbeatAt" < NOW() - INTERVAL '5 seconds'
    ))
    -- Step run cannot have a failed parent
    AND NOT EXISTS (
        SELECT 1
        FROM "_StepRunOrder" AS order_table
        JOIN "StepRun" AS prev_sr ON order_table."A" = prev_sr."id"
        WHERE 
            order_table."B" = sr."id"
            AND prev_sr."status" != 'SUCCEEDED'
    )
ORDER BY
    sr."createdAt" ASC;

-- name: ListStepRunsToRequeue :many
SELECT
    sr.*
FROM
    "StepRun" sr
LEFT JOIN
    "Worker" w ON sr."workerId" = w."id"
JOIN
    "JobRun" jr ON sr."jobRunId" = jr."id"
WHERE
    sr."tenantId" = @tenantId::uuid
    AND sr."requeueAfter" < NOW()
    AND (sr."status" = 'PENDING' OR sr."status" = 'PENDING_ASSIGNMENT')
    AND jr."status" = 'RUNNING'
    AND NOT EXISTS (
        SELECT 1
        FROM "_StepRunOrder" AS order_table
        JOIN "StepRun" AS prev_sr ON order_table."A" = prev_sr."id"
        WHERE 
            order_table."B" = sr."id"
            AND prev_sr."status" != 'SUCCEEDED'
    )
ORDER BY
    sr."createdAt" ASC;

-- name: AssignStepRunToWorker :one
WITH step_run AS (
    SELECT
        sr."id",
        sr."status",
        a."id" AS "actionId",
        s."timeout" AS "stepTimeout"
    FROM
        "StepRun" sr
    JOIN
        "Step" s ON sr."stepId" = s."id"
    JOIN
        "Action" a ON s."actionId" = a."actionId" AND a."tenantId" = @tenantId::uuid
    WHERE
        sr."id" = @stepRunId::uuid AND
        sr."tenantId" = @tenantId::uuid
),
valid_workers AS (
    SELECT
        w."id", w."dispatcherId"
    FROM
        "Worker" w, step_run
    WHERE
        w."tenantId" = @tenantId::uuid
        AND w."lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
        AND w."id" IN (
            SELECT "_ActionToWorker"."B"
            FROM "_ActionToWorker"
            INNER JOIN "Action" ON "Action"."id" = "_ActionToWorker"."A"
            WHERE "Action"."tenantId" = @tenantId AND "Action"."id" = step_run."actionId"
        )
        AND (
            w."maxRuns" IS NULL OR
            w."maxRuns" > (
                SELECT COUNT(*)
                FROM "StepRun" srs
                WHERE srs."workerId" = w."id" AND srs."status" = 'RUNNING'
            )
        )
    ORDER BY random()
),
selected_worker AS (
    SELECT "id", "dispatcherId"
    FROM valid_workers
    LIMIT 1
)
UPDATE
    "StepRun"
SET
    "status" = 'ASSIGNED',
    "workerId" = (
        SELECT "id"
        FROM selected_worker
        LIMIT 1
    ),
    "updatedAt" = CURRENT_TIMESTAMP,
    "timeoutAt" = CASE
        WHEN (SELECT "stepTimeout" FROM step_run) IS NOT NULL THEN
            CURRENT_TIMESTAMP + convert_duration_to_interval((SELECT "stepTimeout" FROM step_run))
        ELSE CURRENT_TIMESTAMP + INTERVAL '5 minutes'
    END
WHERE
    "id" = @stepRunId::uuid AND
    "tenantId" = @tenantId::uuid AND
    EXISTS (SELECT 1 FROM selected_worker)
RETURNING "StepRun"."id", "StepRun"."workerId", (SELECT "dispatcherId" FROM selected_worker) AS "dispatcherId";