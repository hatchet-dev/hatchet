-- name: UpsertQueue :exec
INSERT INTO
    "Queue" (
        "tenantId",
        "name"
    )
VALUES
    (
        @tenantId::uuid,
        @name::text
    )
ON CONFLICT ("tenantId", "name") DO NOTHING;

-- name: ListQueues :many
SELECT
    *
FROM
    "Queue"
WHERE
    "tenantId" = @tenantId::uuid;

-- name: CreateQueueItem :exec
INSERT INTO
    "QueueItem" (
        "stepRunId",
        "stepId",
        "actionId",
        "scheduleTimeoutAt",
        "stepTimeout",
        "isQueued",
        "tenantId",
        "queue"
    )
VALUES
    (
        sqlc.narg('stepRunId')::uuid,
        sqlc.narg('stepId')::uuid,
        sqlc.narg('actionId')::text,
        sqlc.narg('scheduleTimeoutAt')::timestamp,
        sqlc.narg('stepTimeout')::text,
        true,
        @tenantId::uuid,
        @queue
    );

-- name: ListQueueItems :batchmany
SELECT
    *
FROM
    "QueueItem" qi
WHERE
    qi."isQueued" = true
    AND qi."tenantId" = @tenantId::uuid
    AND qi."queue" = @queue::text
    AND (
        sqlc.narg('gtId')::bigint IS NULL OR
        qi."id" >= sqlc.narg('gtId')::bigint
    )
ORDER BY
    qi."id" ASC
LIMIT
    100
FOR UPDATE SKIP LOCKED;

-- name: BulkQueueItems :exec
UPDATE
    "QueueItem" qi
SET
    "isQueued" = false
WHERE
    qi."id" = ANY(@ids::bigint[]);
