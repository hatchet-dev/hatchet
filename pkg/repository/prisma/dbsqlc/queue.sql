-- name: UpsertQueue :exec
WITH queue_exists AS (
    SELECT
        1
    FROM
        "Queue"
    WHERE
        "tenantId" = @tenantId::uuid
        AND "name" = @name::text
), queue_to_update AS (
    SELECT
        *
    FROM
        "Queue"
    WHERE
        EXISTS (
            SELECT
                1
            FROM
                queue_exists
        )
        AND "tenantId" = @tenantId::uuid
        AND "name" = @name::text
    FOR UPDATE SKIP LOCKED
), update_queue AS (
    UPDATE
        "Queue"
    SET
        "lastActive" = NOW()
    FROM
        queue_to_update
    WHERE
        "Queue"."tenantId" = queue_to_update."tenantId"
        AND "Queue"."name" = queue_to_update."name"
)
INSERT INTO
    "Queue" (
        "tenantId",
        "name",
        "lastActive"
    )
SELECT
    @tenantId::uuid,
    @name::text,
    NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM queue_exists
)
ON CONFLICT ("tenantId", "name") DO NOTHING;

-- name: ListQueues :many
SELECT
    *
FROM
    "Queue"
WHERE
    "tenantId" = @tenantId::uuid
    AND "lastActive" > NOW() - INTERVAL '1 day';

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

-- name: CreateQueueItemsBulk :copyfrom
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
        $1,
        $2,
        $3,
        $4,
        $5,
        $6,
        $7,
        $8,
        $9,
        $10,
        $11
    );

-- name: GetQueuedCounts :many
SELECT
    "queue",
    COUNT(*) AS "count"
FROM
    "QueueItem" qi
WHERE
    qi."isQueued" = true
    AND qi."tenantId" = @tenantId::uuid
GROUP BY
    qi."queue";

-- name: GetMinUnprocessedQueueItemId :one
WITH priority_1 AS (
    SELECT
        "id"
    FROM
        "QueueItem"
    WHERE
        "isQueued" = 't'
        AND "tenantId" = @tenantId::uuid
        AND "queue" = @queue::text
        AND "priority" = 1
    ORDER BY
        "id" ASC
    LIMIT 1
),
priority_2 AS (
    SELECT
        "id"
    FROM
        "QueueItem"
    WHERE
        "isQueued" = 't'
        AND "tenantId" = @tenantId::uuid
        AND "queue" = @queue::text
        AND "priority" = 2
    ORDER BY
        "id" ASC
    LIMIT 1
),
priority_3 AS (
    SELECT
        "id"
    FROM
        "QueueItem"
    WHERE
        "isQueued" = 't'
        AND "tenantId" = @tenantId::uuid
        AND "queue" = @queue::text
        AND "priority" = 3
    ORDER BY
        "id" ASC
    LIMIT 1
),
priority_4 AS (
    SELECT
        "id"
    FROM
        "QueueItem"
    WHERE
        "isQueued" = 't'
        AND "tenantId" = @tenantId::uuid
        AND "queue" = @queue::text
        AND "priority" = 4
    ORDER BY
        "id" ASC
    LIMIT 1
)
SELECT
    COALESCE(MIN("id"), 0)::bigint AS "minId"
FROM (
    SELECT "id" FROM priority_1
    UNION ALL
    SELECT "id" FROM priority_2
    UNION ALL
    SELECT "id" FROM priority_3
    UNION ALL
    SELECT "id" FROM priority_4
) AS combined_priorities;

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

-- name: ListQueueItemsForQueue :many
SELECT
    sqlc.embed(qi),
    sr."status"
FROM
    "QueueItem" qi
JOIN
    "StepRun" sr ON qi."stepRunId" = sr."id"
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
    COALESCE(sqlc.narg('limit')::integer, 100);

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
    input."queue",
    true,
    input."data",
    input."tenantId",
    1
FROM (
    SELECT
        unnest(cast(@queues::text[] as"InternalQueue"[])) AS "queue",
        unnest(@datas::json[]) AS "data",
        unnest(@tenantIds::uuid[]) AS "tenantId"
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

-- name: ListAllAvailableSlotsForWorkers :many
WITH worker_max_runs AS (
    SELECT
        "id",
        "maxRuns"
    FROM
        "Worker"
    WHERE
        "tenantId" = @tenantId::uuid
), worker_filled_slots AS (
    SELECT
        "workerId",
        COUNT("stepRunId") AS "filledSlots"
    FROM
        "SemaphoreQueueItem"
    WHERE
        "tenantId" = @tenantId::uuid
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

-- name: ListAllWorkerActions :many
SELECT
    a."actionId" AS actionId
FROM "Worker" w
LEFT JOIN "_ActionToWorker" aw ON w.id = aw."B"
LEFT JOIN "Action" a ON aw."A" = a.id
WHERE
    a."tenantId" = @tenantId::uuid AND
    w."id" = @workerId::uuid;

-- name: ListActionsForWorkers :many
SELECT
    w."id" as "workerId",
    a."actionId"
FROM
    "Worker" w
LEFT JOIN
    "_ActionToWorker" atw ON w."id" = atw."B"
LEFT JOIN
    "Action" a ON atw."A" = a."id"
WHERE
    w."tenantId" = @tenantId::uuid
    AND w."id" = ANY(@workerIds::uuid[])
    AND w."dispatcherId" IS NOT NULL
    AND w."lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
    AND w."isActive" = true
    AND w."isPaused" = false;

-- name: ListActionsForAvailableWorkers :many
SELECT
    w."id" as "workerId",
    a."actionId"
FROM
    "Worker" w
JOIN
    "_ActionToWorker" atw ON w."id" = atw."B"
JOIN
    "Action" a ON atw."A" = a."id"
WHERE
    w."tenantId" = @tenantId::uuid
    AND w."dispatcherId" IS NOT NULL
    AND w."lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
    AND w."isActive" = true
    AND w."isPaused" = false;

-- name: ListActiveWorkers :many
SELECT
    w."id",
    w."maxRuns"
FROM
    "Worker" w
WHERE
    w."tenantId" = @tenantId::uuid
    AND w."dispatcherId" IS NOT NULL
    AND w."lastHeartbeatAt" > NOW() - INTERVAL '5 seconds'
    AND w."isActive" = true
    AND w."isPaused" = false;

-- name: RetryStepRuns :many
WITH retries AS (
    SELECT
        *
    FROM
        "RetryQueueItem" rqi
    WHERE
        rqi."isQueued" = true
        AND rqi."tenantId" = @tenantId::uuid
        AND rqi."retryAfter" <= NOW()
    ORDER BY
        rqi."retryAfter" ASC
    LIMIT
        1000
    FOR UPDATE SKIP LOCKED
), updated_rqis AS (
    UPDATE
        "RetryQueueItem" rqi
    SET
        "isQueued" = false
    FROM
        retries
    WHERE
        rqi."stepRunId" = retries."stepRunId"
), srs AS (
    SELECT
        sr."id",
        sr."tenantId",
        sr."scheduleTimeoutAt",
        sr."retryCount",
        sr."internalRetryCount",
        s."actionId",
        s."id" AS "stepId",
        s."timeout" AS "stepTimeout",
        s."scheduleTimeout" AS "scheduleTimeout",
        wr."id" AS "workflowRunId"
    FROM
        retries
    JOIN
        "StepRun" sr ON retries."stepRunId" = sr."id"
    JOIN
        "Step" s ON sr."stepId" = s."id"
    JOIN
        "JobRun" jr ON sr."jobRunId" = jr."id"
    JOIN
        "WorkflowRun" wr ON jr."workflowRunId" = wr."id"
    WHERE
        sr."status" NOT IN ('SUCCEEDED', 'FAILED', 'CANCELLED')
), updated_step_runs AS (
    UPDATE "StepRun" sr
    SET
        "scheduleTimeoutAt" = CURRENT_TIMESTAMP + COALESCE(convert_duration_to_interval(srs."scheduleTimeout"), INTERVAL '5 minutes'),
        "updatedAt" = CURRENT_TIMESTAMP,
        "retryCount" = srs."retryCount" + 1
    FROM srs
    WHERE sr."id" = srs."id"
    RETURNING sr."id"
), updated_workflow_runs AS (
    UPDATE "WorkflowRun" wr
    SET
        "status" = 'QUEUED',
        "updatedAt" = CURRENT_TIMESTAMP
    FROM srs
    WHERE wr."id" = srs."workflowRunId"
    RETURNING wr."id"
)
SELECT * FROM retries;

-- name: CreateRetryQueueItem :exec
INSERT INTO
    "RetryQueueItem" (
        "stepRunId",
        "retryAfter",
        "tenantId",
        "isQueued"
    )
VALUES
    (
        @stepRunId::uuid,
        @retryAfter::timestamp,
        @tenantId::uuid,
        true
    );

-- name: GetMinMaxProcessedRetryQueueItems :one
SELECT
    COALESCE(MIN("retryAfter"), NOW())::timestamp AS "minRetryAfter",
    COALESCE(MAX("retryAfter"), NOW())::timestamp AS "maxRetryAfter"
FROM
    "RetryQueueItem"
WHERE
    "isQueued" = 'f'
    AND "tenantId" = @tenantId::uuid;

-- name: CleanupRetryQueueItems :exec
DELETE FROM "RetryQueueItem"
WHERE "isQueued" = 'f'
AND
    "retryAfter" >= @minRetryAfter::timestamp
    AND "retryAfter" <= @maxRetryAfter::timestamp
    AND "tenantId" = @tenantId::uuid;
