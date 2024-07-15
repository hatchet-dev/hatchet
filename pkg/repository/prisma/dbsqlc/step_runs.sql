-- name: GetStepRun :one
SELECT
    "StepRun".*
FROM
    "StepRun"
WHERE
    "id" = @id::uuid AND
    "tenantId" = @tenantId::uuid;

-- name: GetStepRunDataForEngine :one
SELECT
    sr."input",
    sr."output",
    sr."error",
    jrld."data" AS "jobRunLookupData"
FROM
    "StepRun" sr
JOIN
    "JobRun" jr ON sr."jobRunId" = jr."id"
JOIN
    "JobRunLookupData" jrld ON jr."id" = jrld."jobRunId"
WHERE
    sr."id" = @id::uuid AND
    sr."tenantId" = @tenantId::uuid;

-- name: GetStepRunForEngine :many
SELECT
    DISTINCT ON (sr."id")
    sr."id" AS "SR_id",
    sr."createdAt" AS "SR_createdAt",
    sr."updatedAt" AS "SR_updatedAt",
    sr."deletedAt" AS "SR_deletedAt",
    sr."tenantId" AS "SR_tenantId",
    sr."order" AS "SR_order",
    sr."workerId" AS "SR_workerId",
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
    -- TODO: everything below this line is cacheable and should be moved to a separate query
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
    jr."workflowRunId" AS "workflowRunId",
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
    );

-- name: UpdateStepRun :one
UPDATE
    "StepRun"
SET
    "requeueAfter" = COALESCE(sqlc.narg('requeueAfter')::timestamp, "requeueAfter"),
    "scheduleTimeoutAt" = CASE
        -- if this is a rerun, we clear the scheduleTimeoutAt
        WHEN sqlc.narg('rerun')::boolean THEN NULL
        ELSE COALESCE(sqlc.narg('scheduleTimeoutAt')::timestamp, "scheduleTimeoutAt")
    END,
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
    "retryCount" = COALESCE(sqlc.narg('retryCount')::int, "retryCount"),
    "semaphoreReleased" = COALESCE(sqlc.narg('semaphoreReleased')::boolean, "semaphoreReleased")
WHERE
  "id" = @id::uuid AND
  "tenantId" = @tenantId::uuid
RETURNING "StepRun".*;

-- name: UnlinkStepRunFromWorker :one
UPDATE
    "StepRun"
SET
    "workerId" = NULL
WHERE
    "id" = @stepRunId::uuid AND
    "tenantId" = @tenantId::uuid
RETURNING *;

-- name: ResolveLaterStepRuns :many
WITH RECURSIVE currStepRun AS (
  SELECT *
  FROM "StepRun"
  WHERE
    "id" = @stepRunId::uuid AND
    "tenantId" = @tenantId::uuid
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
    WHEN (SELECT "status" FROM currStepRun) IN ('FAILED', 'CANCELLED') THEN 'CANCELLED'
    ELSE sr."status"
    END,
    -- When the previous step run timed out, the cancelled reason is set
    "cancelledReason" = CASE
    -- When the step is in a final state, it cannot be updated
    WHEN sr."status" IN ('SUCCEEDED', 'FAILED', 'CANCELLED') THEN sr."cancelledReason"
    WHEN (SELECT "status" FROM currStepRun) = 'CANCELLED' AND (SELECT "cancelledReason" FROM currStepRun) = 'TIMED_OUT'::text THEN 'PREVIOUS_STEP_TIMED_OUT'
    WHEN (SELECT "status" FROM currStepRun) = 'FAILED' THEN 'PREVIOUS_STEP_FAILED'
    WHEN (SELECT "status" FROM currStepRun) = 'CANCELLED' THEN 'PREVIOUS_STEP_CANCELLED'
    ELSE NULL
    END
FROM
    childStepRuns csr
WHERE
    sr."id" = csr."id" AND
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

-- name: GetMaxRunsLimit :one
WITH valid_workers AS (
    SELECT
        w."id",
        COALESCE(w."maxRuns", 100) - COUNT(wss."id") AS "remainingSlots"
    FROM
        "Worker" w
    LEFT JOIN
        "WorkerSemaphoreSlot" wss ON w."id" = wss."workerId" AND wss."stepRunId" IS NOT NULL
    WHERE
        w."tenantId" = @tenantId::uuid
        AND w."lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
        -- necessary because isActive is set to false immediately when the stream closes
        AND w."isActive" = true
        AND w."isPaused" = false
    GROUP BY
        w."id", w."maxRuns"
    HAVING
        COALESCE(w."maxRuns", 100) - COUNT(wss."stepRunId") > 0
),
-- Count the total number of maxRuns - runningStepRuns across all workers
total_max_runs AS (
    SELECT
        SUM("remainingSlots") AS "totalMaxRuns"
    FROM
        valid_workers
)
SELECT
    GREATEST("totalMaxRuns", 100)::int AS "limitMaxRuns"
FROM
    total_max_runs;

-- name: ListStepRunsToReassign :many
WITH inactive_workers AS (
    SELECT
        w."id"
    FROM
        "Worker" w
    WHERE
        w."tenantId" = @tenantId::uuid
        AND w."lastHeartbeatAt" < NOW() - INTERVAL '30 seconds'
),
step_runs_to_reassign AS (
    SELECT "stepRunId"
    FROM "WorkerSemaphoreSlot"
    WHERE
        "workerId" = ANY(SELECT "id" FROM inactive_workers)
        AND "stepRunId" IS NOT NULL
    FOR UPDATE SKIP LOCKED
),
update_semaphore_steps AS (
    UPDATE "WorkerSemaphoreSlot" wss
    SET "stepRunId" = NULL
    FROM step_runs_to_reassign
    WHERE wss."stepRunId" = step_runs_to_reassign."stepRunId"
)
UPDATE
    "StepRun"
SET
    "status" = 'PENDING_ASSIGNMENT',
    -- place directly in the queue
    "requeueAfter" = CURRENT_TIMESTAMP,
    "updatedAt" = CURRENT_TIMESTAMP,
    -- unset the schedule timeout
    "scheduleTimeoutAt" = NULL
FROM
    step_runs_to_reassign
WHERE
    "StepRun"."id" = step_runs_to_reassign."stepRunId"
RETURNING "StepRun"."id";

-- name: ListStepRunsToTimeout :many
SELECT "id"
FROM "StepRun"
WHERE
    "status" = ANY(ARRAY['RUNNING', 'ASSIGNED']::"StepRunStatus"[])
    AND "timeoutAt" < NOW()
    AND "tenantId" = @tenantId::uuid
LIMIT 100;

-- name: ListStepRunsToRequeue :many
WITH step_runs AS (
    SELECT
        sr."id", sr."status", sr."workerId"
    FROM
        "StepRun" sr
    JOIN
        "JobRun" jr ON sr."jobRunId" = jr."id" AND jr."status" = 'RUNNING'
    WHERE
        sr."tenantId" = @tenantId::uuid
        AND sr."status" = ANY(ARRAY['PENDING', 'PENDING_ASSIGNMENT']::"StepRunStatus"[])
        AND sr."requeueAfter" < NOW()
        AND sr."input" IS NOT NULL
        AND NOT EXISTS (
            SELECT 1
            FROM "_StepRunOrder" AS order_table
            JOIN "StepRun" AS prev_sr ON order_table."A" = prev_sr."id"
            WHERE
                order_table."B" = sr."id"
                AND prev_sr."status" != 'SUCCEEDED'
        )
    FOR UPDATE SKIP LOCKED
    LIMIT
        sqlc.arg('limit')::int
)
UPDATE
    "StepRun"
SET
    "status" = 'PENDING_ASSIGNMENT',
    -- requeue after now plus 4 seconds
    "requeueAfter" = CURRENT_TIMESTAMP + INTERVAL '4 seconds',
    "updatedAt" = CURRENT_TIMESTAMP
FROM
    step_runs
WHERE
    "StepRun"."id" = step_runs."id"
RETURNING "StepRun"."id";

-- name: RefreshTimeoutBy :one
UPDATE
    "StepRun" sr
SET
    "timeoutAt" = CASE
        -- Only update timeoutAt if the step run is currently in RUNNING status
        WHEN sr."status" = 'RUNNING' THEN
            COALESCE(sr."timeoutAt", CURRENT_TIMESTAMP) + convert_duration_to_interval(sqlc.narg('incrementTimeoutBy')::text)
            ELSE sr."timeoutAt"
        END,
    "updatedAt" = CURRENT_TIMESTAMP
WHERE
    "id" = @stepRunId::uuid AND
    "tenantId" = @tenantId::uuid
RETURNING *;

-- name: ReleaseWorkerSemaphoreSlot :one
WITH step_run as (
  SELECT "workerId"
  FROM "StepRun"
  WHERE "id" = @stepRunId::uuid AND "tenantId" = @tenantId::uuid
)
UPDATE "WorkerSemaphoreSlot"
    SET "stepRunId" = NULL
WHERE "stepRunId" = @stepRunId::uuid
  AND "workerId" = (SELECT "workerId" FROM step_run)
RETURNING *;

-- name: AcquireWorkerSemaphoreSlotAndAssign :one
WITH valid_workers AS (
    SELECT w."id", COUNT(wss."id") AS "slots"
    FROM "Worker" w
    JOIN "WorkerSemaphoreSlot" wss ON w."id" = wss."workerId" AND wss."stepRunId" IS NULL
    WHERE
        w."tenantId" = @tenantId::uuid
        AND w."dispatcherId" IS NOT NULL
        AND w."lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
        AND w."isActive" = true
        AND w."isPaused" = false
        AND w."id" IN (
            SELECT "_ActionToWorker"."B"
            FROM "_ActionToWorker"
            INNER JOIN "Action" ON "Action"."id" = "_ActionToWorker"."A"
            WHERE "Action"."tenantId" = @tenantId AND "Action"."actionId" = @actionId::text
        )
    GROUP BY w."id"
),
locked_step_runs AS (
    SELECT
        sr."id", sr."status", sr."workerId", sr."stepId"
    FROM
        "StepRun" sr
    WHERE
        sr."id" = @stepRunId::uuid
    FOR UPDATE SKIP LOCKED
),
selected_slot AS (
    SELECT wss."id" AS "slotId", wss."workerId" AS "workerId"
    FROM "WorkerSemaphoreSlot" wss
    JOIN valid_workers w ON wss."workerId" = w."id"
    WHERE
        wss."stepRunId" IS NULL
        AND (SELECT COUNT(*) FROM locked_step_runs) > 0
    ORDER BY w."slots" DESC, RANDOM()
    FOR UPDATE SKIP LOCKED
    LIMIT 1
),
updated_slot AS (
    UPDATE "WorkerSemaphoreSlot"
    SET "stepRunId" = @stepRunId::uuid
    WHERE "id" = (SELECT "slotId" FROM selected_slot)
    AND "stepRunId" IS NULL
    RETURNING *
),
assign_step_run_to_worker AS (
	UPDATE
	    "StepRun"
	SET
	    "status" = 'ASSIGNED',
	    "workerId" = (SELECT "workerId" FROM updated_slot),
	    "tickerId" = NULL,
	    "updatedAt" = CURRENT_TIMESTAMP,
	    "timeoutAt" = CASE
	        WHEN sqlc.narg('stepTimeout')::text IS NOT NULL THEN
	            CURRENT_TIMESTAMP + convert_duration_to_interval(sqlc.narg('stepTimeout')::text)
	        ELSE CURRENT_TIMESTAMP + INTERVAL '5 minutes'
	    END
	WHERE
	    "id" = (SELECT "stepRunId" FROM updated_slot) AND
	    "status" = 'PENDING_ASSIGNMENT'
	RETURNING
	    "StepRun"."id", "StepRun"."workerId"
),
selected_dispatcher AS (
    SELECT "dispatcherId" FROM "Worker"
    WHERE "id" = (SELECT "workerId" FROM updated_slot)
),
step_rate_limits AS (
    SELECT
        rl."units" AS "units",
        rl."rateLimitKey" AS "rateLimitKey"
    FROM
        "StepRateLimit" rl
    JOIN locked_step_runs lsr ON rl."stepId" = lsr."stepId" -- only increment if we have a lsr
    JOIN updated_slot us ON us."stepRunId" = lsr."id" -- only increment if we have a slot
    WHERE
        rl."tenantId" = @tenantId::uuid
),
locked_rate_limits AS (
    SELECT
        srl.*,
        step_rate_limits."units"
    FROM
        step_rate_limits
    JOIN
        "RateLimit" srl ON srl."key" = step_rate_limits."rateLimitKey" AND srl."tenantId" = @tenantId::uuid
    FOR UPDATE
),
update_rate_limits AS (
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
    RETURNING srl.*
),
exhausted_rate_limits AS (
    SELECT
        srl."key"
    FROM
        update_rate_limits srl
    WHERE
        srl."value" < 0
)
SELECT
    updated_slot."workerId" as "workerId",
    updated_slot."stepRunId" as "stepRunId",
    selected_dispatcher."dispatcherId" as "dispatcherId",
    COALESCE(COUNT(exhausted_rate_limits."key"), 0)::int as "exhaustedRateLimitCount",
    COALESCE(SUM(valid_workers."slots"),0)::int as "remainingSlots"
FROM
    (SELECT 1 as filler) as filler_row_subquery -- always return a row
    LEFT JOIN updated_slot ON true
    LEFT JOIN selected_dispatcher ON true
    LEFT JOIN exhausted_rate_limits ON true
    LEFT JOIN valid_workers ON true
GROUP BY
    updated_slot."workerId",
    updated_slot."stepRunId",
    selected_dispatcher."dispatcherId";


-- name: CreateStepRunEvent :exec
WITH input_values AS (
    SELECT
        CURRENT_TIMESTAMP AS "timeFirstSeen",
        CURRENT_TIMESTAMP AS "timeLastSeen",
        @stepRunId::uuid AS "stepRunId",
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


-- name: ReplayStepRunResetWorkflowRun :one
WITH workflow_run_id AS (
    SELECT
        "workflowRunId"
    FROM
        "JobRun"
    WHERE
        "id" = @jobRunId::uuid
)
UPDATE
    "WorkflowRun"
SET
    "status" = 'RUNNING',
    "updatedAt" = CURRENT_TIMESTAMP,
    "startedAt" = NULL,
    "finishedAt" = NULL
WHERE
    "id" = (SELECT "workflowRunId" FROM workflow_run_id)
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

-- name: GetLaterStepRunsForReplay :many
WITH RECURSIVE currStepRun AS (
    SELECT *
    FROM "StepRun"
    WHERE
        "id" = @stepRunId::uuid AND
        "tenantId" = @tenantId::uuid
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
    childStepRuns csr ON sr."id" = csr."id"
WHERE
    sr."tenantId" = @tenantId::uuid;

-- name: ReplayStepRunResetLaterStepRuns :many
WITH RECURSIVE currStepRun AS (
    SELECT *
    FROM "StepRun"
    WHERE
        "id" = @stepRunId::uuid AND
        "tenantId" = @tenantId::uuid
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
    "input" = NULL
FROM
    childStepRuns csr
WHERE
    sr."id" = csr."id" AND
    sr."tenantId" = @tenantId::uuid
RETURNING sr.*;

-- name: ListNonFinalChildStepRuns :many
WITH RECURSIVE currStepRun AS (
    SELECT *
    FROM "StepRun"
    WHERE
        "id" = @stepRunId::uuid AND
        "tenantId" = @tenantId::uuid
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
-- Select all child step runs that are not in a final state
SELECT
    sr.*
FROM
    "StepRun" sr
JOIN
    childStepRuns csr ON sr."id" = csr."id"
WHERE
    sr."tenantId" = @tenantId::uuid AND
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
    "StepRun"."tenantId" = @tenantId::uuid
ORDER BY
    "StepRunResultArchive"."createdAt"
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
