-- name: GetStepRun :one
SELECT
    "StepRun".*
FROM
    "StepRun"
WHERE
    "id" = @id::uuid AND
    "deletedAt" IS NULL AND
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
    a."actionId" AS "actionId",
    sticky."strategy" AS "stickyStrategy"
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

-- name: ListStartableStepRuns :many
WITH job_run AS (
    SELECT "status", "deletedAt"
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
    AND child_run."deletedAt" IS NULL
    AND job_run."deletedAt" IS NULL
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
    AND "StepRun"."deletedAt" IS NULL
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
        AND sr."deletedAt" IS NULL
        AND jr."deletedAt" IS NULL
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

-- name: AcquireWorkerSemaphoreSlotAndAssign :one
WITH valid_workers AS (
    SELECT
        w."id"
    FROM
        "Worker" w
    WHERE
        w."tenantId" = @tenantId::uuid
        AND w."dispatcherId" IS NOT NULL
        AND w."lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
        AND w."isActive" = true
        AND w."isPaused" = false
        AND (
            -- sticky worker selection
            sqlc.narg('workerId')::uuid IS NULL
            OR w."id" = sqlc.narg('workerId')::uuid
        )
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
        sr."id" = @stepRunId::uuid AND
        sr."deletedAt" IS NULL
    FOR UPDATE SKIP LOCKED
),
desired_workflow_labels AS (
    SELECT
        "key",
        "strValue",
        "intValue",
        "required",
        "weight",
        "comparator"
    FROM
        "StepDesiredWorkerLabel"
    WHERE
        "stepId" = (SELECT "stepId" FROM locked_step_runs)
),
evaluated_affinities AS (
    SELECT DISTINCT
        wa."key" AS worker_key,
        dwl."key" AS desired_key,
        dwl."weight",
        vw."id" as "workerId",
        dwl."required",
        COALESCE(dwl."intValue"::text, dwl."strValue") AS input_value,
        CASE
            WHEN wa."intValue" IS NOT NULL THEN wa."intValue"::text
            WHEN wa."strValue" IS NOT NULL THEN wa."strValue"
        END AS value,
        dwl."comparator",
        CASE
            WHEN dwl.comparator = 'EQUAL' AND
                 (wa."intValue" IS NOT NULL AND dwl."intValue" IS NOT NULL AND dwl."intValue" = wa."intValue") THEN 1
            WHEN dwl.comparator = 'EQUAL' AND
                 (wa."strValue" IS NOT NULL AND dwl."strValue" = wa."strValue") THEN 1
            WHEN dwl.comparator = 'NOT_EQUAL' AND
                 (wa."intValue" IS NOT NULL AND dwl."intValue" IS NOT NULL AND dwl."intValue" <> wa."intValue") THEN 1
            WHEN dwl.comparator = 'NOT_EQUAL' AND
                 (wa."strValue" IS NOT NULL AND dwl."strValue" <> wa."strValue") THEN 1
            WHEN dwl.comparator = 'GREATER_THAN' AND
                 (wa."intValue" IS NOT NULL AND dwl."intValue" IS NOT NULL AND dwl."intValue" > wa."intValue") THEN 1
            WHEN dwl.comparator = 'LESS_THAN' AND
                 (wa."intValue" IS NOT NULL AND dwl."intValue" IS NOT NULL AND dwl."intValue" < wa."intValue") THEN 1
            WHEN dwl.comparator = 'GREATER_THAN_OR_EQUAL' AND
                 (wa."intValue" IS NOT NULL AND dwl."intValue" IS NOT NULL AND dwl."intValue" >= wa."intValue") THEN 1
            WHEN dwl.comparator = 'LESS_THAN_OR_EQUAL' AND
                 (wa."intValue" IS NOT NULL AND dwl."intValue" IS NOT NULL AND dwl."intValue" <= wa."intValue") THEN 1
            ELSE 0
        END AS is_true
    FROM
        valid_workers vw
    LEFT JOIN "WorkerLabel" wa ON wa."workerId" = vw."id"
    LEFT JOIN desired_workflow_labels dwl ON wa."key" = dwl."key"
),
weighted_workers AS (
    SELECT
        ea."workerId",
        CASE
            WHEN COUNT(*) FILTER (WHERE ea."required" = TRUE AND (ea."desired_key" IS NULL OR ea."is_true" = 0)) > 0 THEN -99999
            ELSE COALESCE(SUM(CASE WHEN is_true = 1 THEN ea."weight" ELSE 0 END), 0)
        END AS total_weight,
        COUNT(wss."id") AS available_slots
    FROM
        evaluated_affinities ea
    LEFT JOIN "WorkerSemaphoreSlot" wss ON ea."workerId" = wss."workerId" AND wss."stepRunId" IS NULL
    GROUP BY
        ea."workerId"
),
selected_worker AS (
    SELECT
        vw."id",
        COALESCE(ww.total_weight, 0) AS total_weight
    FROM
        valid_workers vw
    LEFT JOIN weighted_workers ww ON vw."id" = ww."workerId"
    WHERE
        COALESCE(ww.total_weight, 0) >= 0
    ORDER BY
        COALESCE(ww.total_weight, 0) DESC,
        COALESCE(ww.available_slots, 0) DESC,
        RANDOM()
    LIMIT 1
),
selected_slot AS (
    SELECT
        wss."id" AS "slotId",
        wss."workerId" AS "workerId"
    FROM
        "WorkerSemaphoreSlot" wss
    JOIN
        selected_worker sw ON wss."workerId" = sw."id"
    WHERE
        wss."stepRunId" IS NULL
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
    jsonb_agg(
        jsonb_build_object(
            'key', dwl."key",
            'strValue', dwl."strValue",
            'intValue', dwl."intValue",
            'required', dwl."required",
            'weight', dwl."weight",
            'comparator', dwl."comparator",
            'is_true', ea."is_true"
        )
    ) AS desired_labels,
    jsonb_agg(
        jsonb_build_object(
            'key', wa."key",
            'strValue', wa."strValue",
            'intValue', wa."intValue"
        )
    ) AS worker_labels,
    COALESCE(COUNT(exhausted_rate_limits."key"), 0)::int as "exhaustedRateLimitCount",
    COALESCE(SUM(weighted_workers."available_slots"),0)::int as "remainingSlots"
FROM
    (SELECT 1 as filler) as filler_row_subquery -- always return a row
    LEFT JOIN updated_slot ON true
    LEFT JOIN selected_dispatcher ON true
    LEFT JOIN exhausted_rate_limits ON true
    LEFT JOIN weighted_workers ON total_weight >= 0
    LEFT JOIN
        evaluated_affinities ea ON updated_slot."workerId" = ea."workerId"
    LEFT JOIN
        desired_workflow_labels dwl ON ea."desired_key" = dwl."key"
    LEFT JOIN
        "WorkerLabel" wa ON ea."workerId" = wa."workerId" AND ea."worker_key" = wa."key"
GROUP BY
    updated_slot."workerId",
    updated_slot."stepRunId",
    selected_dispatcher."dispatcherId";

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
UPDATE
    "WorkflowRun"
SET
    "status" = 'PENDING',
    "updatedAt" = CURRENT_TIMESTAMP,
    "startedAt" = NULL,
    "finishedAt" = NULL,
    "duration" = NULL
WHERE
    "id" =  @workflowRunId::uuid
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
    "input" = NULL
WHERE
    sr."id" = ANY(@ids::uuid[]) AND
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
    sr."tenantId" = @tenantId::uuid AND
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
    FOR UPDATE SKIP LOCKED
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
