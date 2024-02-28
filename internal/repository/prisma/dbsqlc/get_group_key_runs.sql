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
    AND (
        (
            -- either no worker assigned
            ggr."workerId" IS NULL
            AND (ggr."status" = 'PENDING' OR ggr."status" = 'PENDING_ASSIGNMENT')
        ) OR (
            -- or the worker is not heartbeating
            ggr."status" = 'ASSIGNED'
            AND w."lastHeartbeatAt" < NOW() - INTERVAL '5 seconds'
        )
    )
ORDER BY
    ggr."createdAt" ASC;