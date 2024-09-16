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

-- name: GetMinMaxProcessedInternalQueueItems :one
SELECT
    COALESCE(MIN("id"), 0)::bigint AS "minId",
    COALESCE(MAX("id"), 0)::bigint AS "maxId"
FROM
    "InternalQueueItem"
WHERE
    "isQueued" = 'f'
    AND "tenantId" = @tenantId::uuid;

-- name: CleanupInternalQueueItems :exec
DELETE FROM "InternalQueueItem"
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

-- name: ListInternalQueueItems :many
SELECT
    *
FROM
    "InternalQueueItem" qi
WHERE
    qi."isQueued" = true
    AND qi."tenantId" = @tenantId::uuid
    AND qi."queue" = @queue::"InternalQueue"
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

-- name: MarkInternalQueueItemsProcessed :exec
UPDATE
    "InternalQueueItem" qi
SET
    "isQueued" = false
WHERE
    qi."id" = ANY(@ids::bigint[]);

-- name: CreateUniqueInternalQueueItemsBulk :exec
INSERT INTO
    "InternalQueueItem" (
        "queue",
        "isQueued",
        "data",
        "tenantId",
        "priority",
        "uniqueKey"
    )
SELECT
    @queue::"InternalQueue",
    true,
    input."data",
    @tenantId::uuid,
    1,
    input."uniqueKey"
FROM (
    SELECT
        unnest(@datas::json[]) AS "data",
        unnest(@uniqueKeys::text[]) AS "uniqueKey"
) AS input
ON CONFLICT DO NOTHING;

-- name: CreateInternalQueueItemsBulk :exec
INSERT INTO
    "InternalQueueItem" (
        "queue",
        "isQueued",
        "data",
        "tenantId",
        "priority"
    )
SELECT
    @queue::"InternalQueue",
    true,
    input."data",
    @tenantId::uuid,
    1
FROM (
    SELECT
        unnest(@datas::json[]) AS "data"
) AS input
ON CONFLICT DO NOTHING;

-- name: CreateTimeoutQueueItem :exec
INSERT INTO
    "InternalQueueItem" (
        "stepRunId",
        "retryCount",
        "timeoutAt",
        "tenantId",
        "isQueued"
    )
SELECT
    @stepRunId::uuid,
    @retryCount::integer,
    @timeoutAt::timestamp,
    @tenantId::uuid,
    true
ON CONFLICT DO NOTHING;

-- name: PopTimeoutQueueItems :many
WITH qis AS (
    SELECT
        "id",
        "stepRunId"
    FROM
        "TimeoutQueueItem"
    WHERE
        "isQueued" = true
        AND "tenantId" = @tenantId::uuid
        AND "timeoutAt" <= NOW()
    ORDER BY
        "timeoutAt" ASC
    LIMIT
        COALESCE(sqlc.narg('limit')::integer, 100)
    FOR UPDATE SKIP LOCKED
)
UPDATE
    "TimeoutQueueItem" qi
SET
    "isQueued" = false
FROM
    qis
WHERE
    qi."id" = qis."id"
RETURNING
    qis."stepRunId" AS "stepRunId";

-- name: RemoveTimeoutQueueItem :exec
DELETE FROM
    "TimeoutQueueItem"
WHERE
    "stepRunId" = @stepRunId::uuid
    AND "retryCount" = @retryCount::integer;

-- name: GetMinMaxProcessedTimeoutQueueItems :one
SELECT
    COALESCE(MIN("id"), 0)::bigint AS "minId",
    COALESCE(MAX("id"), 0)::bigint AS "maxId"
FROM
    "TimeoutQueueItem"
WHERE
    "isQueued" = 'f'
    AND "tenantId" = @tenantId::uuid;

-- name: CleanupTimeoutQueueItems :exec
DELETE FROM "TimeoutQueueItem"
WHERE "isQueued" = 'f'
AND
    "id" >= @minId::bigint
    AND "id" <= @maxId::bigint
    AND "tenantId" = @tenantId::uuid;

-- name: ListAvailableSlotsForWorkers :many
WITH worker_max_runs AS (
    SELECT
        "id",
        "maxRuns"
    FROM
        "Worker"
    WHERE
        "tenantId" = @tenantId::uuid
        AND "id" = ANY(@workerIds::uuid[])
), worker_filled_slots AS (
    SELECT
        "workerId",
        COUNT("stepRunId") AS "filledSlots"
    FROM
        "SemaphoreQueueItem"
    WHERE
        "tenantId" = @tenantId::uuid
        AND "workerId" = ANY(@workerIds::uuid[])
    GROUP BY
        "workerId"
)
-- subtract the filled slots from the max runs to get the available slots
SELECT
    wmr."id",
    wmr."maxRuns" - COALESCE(wfs."filledSlots", 0) AS "availableSlots"
FROM
    worker_max_runs wmr
LEFT JOIN
    worker_filled_slots wfs ON wmr."id" = wfs."workerId";
