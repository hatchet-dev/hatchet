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
        "priority",
        "isQueued",
        "tenantId",
        "queue",
        "sticky",
        "desiredWorkerId"
    )
VALUES
    (
        sqlc.narg('stepRunId')::uuid,
        sqlc.narg('stepId')::uuid,
        sqlc.narg('actionId')::text,
        sqlc.narg('scheduleTimeoutAt')::timestamp,
        sqlc.narg('stepTimeout')::text,
        COALESCE(sqlc.narg('priority')::integer, 1),
        true,
        @tenantId::uuid,
        @queue,
        sqlc.narg('sticky')::"StickyStrategy",
        sqlc.narg('desiredWorkerId')::uuid
    );

-- name: GetMinMaxProcessedQueueItems :one
SELECT
    COALESCE(MIN("id"), 0)::bigint AS "minId",
    COALESCE(MAX("id"), 0)::bigint AS "maxId"
FROM
    "QueueItem"
WHERE
    "isQueued" = 'f'
    AND "tenantId" = @tenantId::uuid;

-- name: CleanupQueueItems :exec
DELETE FROM "QueueItem"
WHERE "isQueued" = 'f'
AND
    "id" >= @minId::bigint
    AND "id" <= @maxId::bigint
    AND "tenantId" = @tenantId::uuid;

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
    -- Added to ensure that the index is used
    AND qi."priority" >= 1 AND qi."priority" <= 4
ORDER BY
    qi."priority" DESC,
    qi."id" ASC
LIMIT
    COALESCE(sqlc.narg('limit')::integer, 100)
FOR UPDATE SKIP LOCKED;

-- name: BulkQueueItems :exec
UPDATE
    "QueueItem" qi
SET
    "isQueued" = false
WHERE
    qi."id" = ANY(@ids::bigint[]);

-- name: ListWorkerSemaphoreQueueItems :many
SELECT
    *
FROM
    "WorkerSemaphoreQueueItem" qi
WHERE
    "isProcessed" = false
    AND "workerId" = ANY(@workerIds::uuid[])
    AND (
        sqlc.narg('gtId')::bigint IS NULL OR
        qi."id" >= sqlc.narg('gtId')::bigint
    )
ORDER BY
    qi."id" ASC
FOR UPDATE SKIP LOCKED;

-- name: MarkWorkerSemaphoreQueueItemsProcessed :exec
UPDATE
    "WorkerSemaphoreQueueItem" qi
SET
    "isProcessed" = true
WHERE
    qi."id" = ANY(@ids::bigint[]);
