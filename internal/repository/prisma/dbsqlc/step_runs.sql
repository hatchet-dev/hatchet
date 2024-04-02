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
    DISTINCT ON (sr."id")
    sqlc.embed(sr),
    jrld."data" AS "jobRunLookupData",
    -- TODO: everything below this line is cacheable and should be moved to a separate query
    jr."id" AS "jobRunId",
    wr."id" AS "workflowRunId",
    s."id" AS "stepId",
    s."retries" AS "stepRetries",
    s."timeout" AS "stepTimeout",
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
    "Action" a ON s."actionId" = a."actionId" AND s."tenantId" = a."tenantId"
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
    DISTINCT ON (child_run."id")
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
    DISTINCT ON ("StepRun"."id")
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
WITH valid_workers AS (
    SELECT
        DISTINCT ON (w."id")
        w."id",
        COALESCE(SUM(ws."slots"), 100) AS "slots"
    FROM
        "Worker" w
    LEFT JOIN
        "WorkerSemaphore" ws ON ws."workerId" = w."id"
    WHERE
        w."tenantId" = @tenantId::uuid
        AND w."lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
    GROUP BY
        w."id"
),
-- Count the total number of slots across all workers
total_max_runs AS (
    SELECT
        SUM("slots") AS "totalMaxRuns"
    FROM
        valid_workers
),
limit_max_runs AS (
    SELECT
        GREATEST("totalMaxRuns", 100) AS "limitMaxRuns"
    FROM
        total_max_runs
),
step_runs AS (
    SELECT
        sr.*
    FROM
        "StepRun" sr
    LEFT JOIN
        "Worker" w ON sr."workerId" = w."id"
    JOIN
        "JobRun" jr ON sr."jobRunId" = jr."id"
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
            AND w."lastHeartbeatAt" < NOW() - INTERVAL '60 seconds'
        ))
        AND jr."status" = 'RUNNING'
        AND sr."input" IS NOT NULL
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
        sr."createdAt" ASC
    LIMIT
        (SELECT "limitMaxRuns" FROM limit_max_runs)
),
locked_step_runs AS (
    SELECT
        sr."id", sr."status", sr."workerId"
    FROM
        step_runs sr
    FOR UPDATE SKIP LOCKED
)
UPDATE
    "StepRun"
SET
    "status" = 'PENDING_ASSIGNMENT',
    -- requeue after now plus 4 seconds
    "requeueAfter" = CURRENT_TIMESTAMP + INTERVAL '4 seconds',
    "updatedAt" = CURRENT_TIMESTAMP
FROM 
    locked_step_runs
WHERE
    "StepRun"."id" = locked_step_runs."id"
RETURNING "StepRun"."id";

-- name: ListStepRunsToRequeue :many
WITH valid_workers AS (
    SELECT
        DISTINCT ON (w."id")
        w."id",
        COALESCE(SUM(ws."slots"), 100) AS "slots"
    FROM
        "Worker" w
    LEFT JOIN
        "WorkerSemaphore" ws ON w."id" = ws."workerId"
    WHERE
        w."tenantId" = @tenantId::uuid
        AND w."lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
    GROUP BY
        w."id"
),
-- Count the total number of maxRuns - runningStepRuns across all workers
total_max_runs AS (
    SELECT
        -- if maxRuns is null, then we assume the worker can run 100 step runs
        SUM("slots") AS "totalMaxRuns"
    FROM
        valid_workers
),
step_runs AS (
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
        AND sr."input" IS NOT NULL
        AND NOT EXISTS (
            SELECT 1
            FROM "_StepRunOrder" AS order_table
            JOIN "StepRun" AS prev_sr ON order_table."A" = prev_sr."id"
            WHERE 
                order_table."B" = sr."id"
                AND prev_sr."status" != 'SUCCEEDED'
        )
    ORDER BY
        sr."createdAt" ASC
    LIMIT
        COALESCE((SELECT "totalMaxRuns" FROM total_max_runs), 100)
),
locked_step_runs AS (
    SELECT
        sr."id", sr."status", sr."workerId"
    FROM
        step_runs sr
    FOR UPDATE SKIP LOCKED
)
UPDATE
    "StepRun"
SET
    "status" = 'PENDING_ASSIGNMENT',
    -- requeue after now plus 4 seconds
    "requeueAfter" = CURRENT_TIMESTAMP + INTERVAL '4 seconds',
    "updatedAt" = CURRENT_TIMESTAMP
FROM 
    locked_step_runs
WHERE
    "StepRun"."id" = locked_step_runs."id"
RETURNING "StepRun"."id";

-- name: AssignStepRunToWorker :one
WITH selected_worker AS (
    SELECT
        w."id", w."dispatcherId", ws."slots"
    FROM
        "Worker" w
    LEFT JOIN 
        "WorkerSemaphore" ws ON w."id" = ws."workerId"
    WHERE
        w."tenantId" = @tenantId::uuid
        AND w."dispatcherId" IS NOT NULL
        AND w."lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
        AND w."id" IN (
            SELECT "_ActionToWorker"."B"
            FROM "_ActionToWorker"
            INNER JOIN "Action" ON "Action"."id" = "_ActionToWorker"."A"
            WHERE "Action"."tenantId" = @tenantId AND "Action"."actionId" = @actionId::text
        )
        AND (
            ws."workerId" IS NULL OR
            ws."slots" > 0
        )
    ORDER BY ws."slots" DESC NULLS FIRST
    LIMIT 1
),
step_run AS (
    SELECT
        "id", "workerId"
    FROM
        "StepRun"
    WHERE
        "id" = @stepRunId::uuid AND
        "tenantId" = @tenantId::uuid AND
        "status" = 'PENDING_ASSIGNMENT' AND
        EXISTS (SELECT 1 FROM selected_worker)
    FOR UPDATE
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
    "tickerId" = NULL,
    "updatedAt" = CURRENT_TIMESTAMP,
    "timeoutAt" = CASE
        WHEN sqlc.narg('stepTimeout')::text IS NOT NULL THEN
            CURRENT_TIMESTAMP + convert_duration_to_interval(sqlc.narg('stepTimeout')::text)
        ELSE CURRENT_TIMESTAMP + INTERVAL '5 minutes'
    END
WHERE
    "id" = @stepRunId::uuid AND
    "tenantId" = @tenantId::uuid AND
    "status" = 'PENDING_ASSIGNMENT' AND
    EXISTS (SELECT 1 FROM selected_worker)
RETURNING "StepRun"."id", "StepRun"."workerId", (SELECT "dispatcherId" FROM selected_worker) AS "dispatcherId";

-- name: UpdateWorkerSemaphore :one
WITH worker_id AS (
    SELECT
        "workerId"
    FROM
        "StepRun"
    WHERE
        "id" = @stepRunId::uuid AND "tenantId" = @tenantId::uuid
),
semaphore AS (
    SELECT
        "slots"
    FROM
        "WorkerSemaphore" ws, worker_id
    WHERE
        ws."workerId" = worker_id."workerId"
    FOR UPDATE
)
UPDATE
    "WorkerSemaphore" ws
SET
    "slots" = (semaphore."slots" + @inc::int)
FROM
    semaphore, worker_id
WHERE
    ws."workerId" = worker_id."workerId"
RETURNING ws.*;

-- name: UpdateStepRateLimits :many
WITH step_rate_limits AS (
    SELECT
        rl."units" AS "units",
        rl."rateLimitKey" AS "rateLimitKey"
    FROM
        "StepRateLimit" rl
    WHERE
        rl."stepId" = @stepId::uuid AND
        rl."tenantId" = @tenantId::uuid
), locked_rate_limits AS (
    SELECT
        srl.*,
        step_rate_limits."units"
    FROM
        step_rate_limits
    JOIN
        "RateLimit" srl ON srl."key" = step_rate_limits."rateLimitKey" AND srl."tenantId" = @tenantId::uuid
    FOR UPDATE
)
UPDATE
    "RateLimit" srl
SET
    "value" = get_refill_value(srl) - lrl."units",
    "lastRefill" = CASE
        WHEN NOW() - srl."lastRefill" >= srl."window"::INTERVAL THEN
            CURRENT_TIMESTAMP
        ELSE
            srl."lastRefill"
    END
FROM
    locked_rate_limits lrl
WHERE
    srl."tenantId" = lrl."tenantId" AND
    srl."key" = lrl."key"
RETURNING srl.*;