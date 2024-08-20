-- name: UpdateGetGroupKeyRun :one
UPDATE
    "GetGroupKeyRun"
SET
    "requeueAfter" = COALESCE(sqlc.narg('requeueAfter')::timestamp, "requeueAfter"),
    "startedAt" = COALESCE(sqlc.narg('startedAt')::timestamp, "startedAt"),
    "finishedAt" = COALESCE(sqlc.narg('finishedAt')::timestamp, "finishedAt"),
    "scheduleTimeoutAt" = COALESCE(sqlc.narg('scheduleTimeoutAt')::timestamp, "scheduleTimeoutAt"),
    "status" = CASE
        -- Final states are final, cannot be updated
        WHEN "status" IN ('SUCCEEDED', 'FAILED', 'CANCELLED') THEN "status"
        ELSE COALESCE(sqlc.narg('status'), "status")
    END,
    "input" = COALESCE(sqlc.narg('input')::jsonb, "input"),
    "output" = COALESCE(sqlc.narg('output')::text, "output"),
    "error" = COALESCE(sqlc.narg('error')::text, "error"),
    "cancelledAt" = COALESCE(sqlc.narg('cancelledAt')::timestamp, "cancelledAt"),
    "cancelledReason" = COALESCE(sqlc.narg('cancelledReason')::text, "cancelledReason")
WHERE
  "id" = @id::uuid AND
  "tenantId" = @tenantId::uuid
RETURNING "GetGroupKeyRun".*;

-- name: GetGroupKeyRunForEngine :many
SELECT
    DISTINCT ON (ggr."id")
    sqlc.embed(ggr),
    wr."id" AS "workflowRunId",
    wv."id" AS "workflowVersionId",
    wv."workflowId" AS "workflowId",
    a."actionId" AS "actionId"
FROM
    "GetGroupKeyRun" ggr
JOIN
    "WorkflowRun" wr ON ggr."workflowRunId" = wr."id"
JOIN
    "WorkflowVersion" wv ON wr."workflowVersionId" = wv."id"
JOIN
    "WorkflowConcurrency" wc ON wv."id" = wc."workflowVersionId"
JOIN
    "Action" a ON wc."getConcurrencyGroupId" = a."id" AND a."tenantId" = ggr."tenantId"
WHERE
    ggr."id" = ANY(@ids::uuid[]) AND
    ggr."deletedAt" IS NULL AND
    wr."deletedAt" IS NULL AND
    wv."deletedAt" IS NULL AND
    ggr."tenantId" = @tenantId::uuid;

-- name: ListGetGroupKeyRunsToReassign :many
WITH valid_workers AS (
    SELECT
        DISTINCT ON (w."id")
        w."id",
        100 AS "remainingSlots"
    FROM
        "Worker" w
    WHERE
        w."tenantId" = @tenantId::uuid
        AND w."lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
        AND w."isActive" = true
        AND w."isPaused" = false
    GROUP BY
        w."id"
),
total_max_runs AS (
    SELECT
        SUM("remainingSlots") AS "totalMaxRuns"
    FROM
        valid_workers
),
limit_max_runs AS (
    SELECT
        GREATEST("totalMaxRuns", 100) AS "limitMaxRuns"
    FROM
        total_max_runs
),
group_key_runs AS (
    SELECT
        ggr.*
    FROM
        "GetGroupKeyRun" ggr
    LEFT JOIN
        "Worker" w ON ggr."workerId" = w."id"
    WHERE
        ggr."tenantId" = @tenantId::uuid
        AND ggr."deletedAt" IS NULL
        AND ggr."status" = ANY(ARRAY['RUNNING', 'ASSIGNED']::"StepRunStatus"[])
        AND w."lastHeartbeatAt" < NOW() - INTERVAL '30 seconds'
    ORDER BY
        ggr."createdAt" ASC
    LIMIT
        (SELECT "limitMaxRuns" FROM limit_max_runs)
),
locked_group_key_runs AS (
    SELECT
        ggr."id", ggr."status", ggr."workerId"
    FROM
        group_key_runs ggr
    FOR UPDATE SKIP LOCKED
)
UPDATE
    "GetGroupKeyRun"
SET
    "status" = 'PENDING_ASSIGNMENT',
    "requeueAfter" = CURRENT_TIMESTAMP + INTERVAL '4 seconds',
    "updatedAt" = CURRENT_TIMESTAMP
FROM
    locked_group_key_runs
WHERE
    "GetGroupKeyRun"."id" = locked_group_key_runs."id"
RETURNING "GetGroupKeyRun".*;


-- name: ListGetGroupKeyRunsToRequeue :many
WITH valid_workers AS (
    SELECT
        w."id",
        100 AS "remainingSlots"
    FROM
        "Worker" w
    WHERE
        w."tenantId" = @tenantId::uuid
        AND w."lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
        AND w."isActive" = true
    GROUP BY
        w."id"
),
total_max_runs AS (
    SELECT
        SUM("remainingSlots") AS "totalMaxRuns"
    FROM
        valid_workers
),
group_key_runs AS (
    SELECT
        ggr.*
    FROM
        "GetGroupKeyRun" ggr
    LEFT JOIN
        "Worker" w ON ggr."workerId" = w."id"
    WHERE
        ggr."tenantId" = @tenantId::uuid
        AND ggr."deletedAt" IS NULL
        AND ggr."requeueAfter" < NOW()
        AND ggr."status" = ANY(ARRAY['PENDING', 'PENDING_ASSIGNMENT']::"StepRunStatus"[])
    ORDER BY
        ggr."createdAt" ASC
    LIMIT
        COALESCE((SELECT "totalMaxRuns" FROM total_max_runs), 100)
),
locked_group_key_runs AS (
    SELECT
        ggr."id", ggr."status", ggr."workerId"
    FROM
        group_key_runs ggr
    FOR UPDATE SKIP LOCKED
)
UPDATE
    "GetGroupKeyRun"
SET
    "status" = 'PENDING_ASSIGNMENT',
    "requeueAfter" = CURRENT_TIMESTAMP + INTERVAL '4 seconds',
    "updatedAt" = CURRENT_TIMESTAMP
FROM
    locked_group_key_runs
WHERE
    "GetGroupKeyRun"."id" = locked_group_key_runs."id"
RETURNING "GetGroupKeyRun".*;

-- name: AssignGetGroupKeyRunToWorker :one
WITH get_group_key_run AS (
    SELECT
        ggr."id",
        ggr."status",
        a."id" AS "actionId"
    FROM
        "GetGroupKeyRun" ggr
    JOIN
        "WorkflowRun" wr ON ggr."workflowRunId" = wr."id"
    JOIN
        "WorkflowVersion" wv ON wr."workflowVersionId" = wv."id"
    JOIN
        "WorkflowConcurrency" wc ON wv."id" = wc."workflowVersionId"
    JOIN
        "Action" a ON wc."getConcurrencyGroupId" = a."id"
    WHERE
        wr."deletedAt" IS NULL AND
        ggr."deletedAt" IS NULL AND
        wv."deletedAt" IS NULL AND
        ggr."id" = @getGroupKeyRunId::uuid AND
        ggr."tenantId" = @tenantId::uuid
), valid_workers AS (
    SELECT
        w."id", w."dispatcherId"
    FROM
        "Worker" w, get_group_key_run
    WHERE
        w."tenantId" = @tenantId::uuid
        AND w."lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
        AND w."id" IN (
            SELECT "_ActionToWorker"."B"
            FROM "_ActionToWorker"
            INNER JOIN "Action" ON "Action"."id" = "_ActionToWorker"."A"
            WHERE "Action"."tenantId" = @tenantId AND "Action"."id" = get_group_key_run."actionId"
        )
    ORDER BY random()
), selected_worker AS (
    SELECT "id", "dispatcherId"
    FROM valid_workers
    LIMIT 1
)
UPDATE
    "GetGroupKeyRun"
SET
    "status" = 'ASSIGNED',
    "workerId" = (
        SELECT "id"
        FROM selected_worker
        LIMIT 1
    ),
    "updatedAt" = CURRENT_TIMESTAMP,
    "timeoutAt" = CURRENT_TIMESTAMP + INTERVAL '5 minutes'
WHERE
    "id" = @getGroupKeyRunId::uuid AND
    "tenantId" = @tenantId::uuid AND
    EXISTS (SELECT 1 FROM selected_worker)
RETURNING "GetGroupKeyRun"."id", "GetGroupKeyRun"."workerId", (SELECT "dispatcherId" FROM selected_worker) AS "dispatcherId";

-- name: AssignGetGroupKeyRunToTicker :one
WITH selected_ticker AS (
    SELECT
        t."id"
    FROM
        "Ticker" t
    WHERE
        t."lastHeartbeatAt" > NOW() - INTERVAL '6 seconds'
        AND t."isActive" = true
    ORDER BY random()
    LIMIT 1
)
UPDATE
    "GetGroupKeyRun"
SET
    "tickerId" = (
        SELECT "id"
        FROM selected_ticker
    )
WHERE
    "id" = @getGroupKeyRunId::uuid AND
    "tenantId" = @tenantId::uuid AND
    "deletedAt" IS NULL AND
    EXISTS (SELECT 1 FROM selected_ticker)
RETURNING "GetGroupKeyRun"."id", "GetGroupKeyRun"."tickerId";
