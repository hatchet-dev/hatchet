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
    sqlc.embed(ggr),
    -- TODO: everything below this line is cacheable and should be moved to a separate query
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
    "Action" a ON wc."getConcurrencyGroupId" = a."id"
WHERE
    ggr."id" = ANY(@ids::uuid[]) AND
    ggr."tenantId" = @tenantId::uuid;

-- name: ListGetGroupKeyRunsToRequeue :many
SELECT
    ggr.*
FROM
    "GetGroupKeyRun" ggr
LEFT JOIN
    "Worker" w ON ggr."workerId" = w."id"
WHERE
    ggr."tenantId" = @tenantId::uuid
    AND ggr."requeueAfter" < NOW()
    AND ggr."workerId" IS NULL
    AND (ggr."status" = 'PENDING' OR ggr."status" = 'PENDING_ASSIGNMENT')
ORDER BY
    ggr."createdAt" ASC;

-- name: ListGetGroupKeyRunsToReassign :many
SELECT
    ggr.*
FROM
    "GetGroupKeyRun" ggr
LEFT JOIN
    "Worker" w ON ggr."workerId" = w."id"
WHERE
    ggr."tenantId" = @tenantId::uuid
    AND ((
        ggr."status" = 'RUNNING'
        AND w."lastHeartbeatAt" < NOW() - INTERVAL '60 seconds'
    ) OR (
        ggr."status" = 'ASSIGNED'
        AND w."lastHeartbeatAt" < NOW() - INTERVAL '5 seconds'
    ))
ORDER BY
    ggr."createdAt" ASC;

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
        ggr."id" = @getGroupKeyRunId::uuid AND
        ggr."tenantId" = @tenantId::uuid
    FOR UPDATE SKIP LOCKED
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
    FOR UPDATE SKIP LOCKED
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
    "updatedAt" = CURRENT_TIMESTAMP
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
    EXISTS (SELECT 1 FROM selected_ticker)
RETURNING "GetGroupKeyRun"."id", "GetGroupKeyRun"."tickerId";