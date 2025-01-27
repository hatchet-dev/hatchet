-- name: UpsertQueues :exec
INSERT INTO
    v2_queue (
        tenant_id,
        name,
        last_active
    )
SELECT
    $1,
    unnest(@names::text[]) AS name,
    NOW()
ON CONFLICT (tenant_id, name) DO NOTHING;

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
        worker_id,
        COUNT(task_id) AS "filledSlots"
    FROM
        v2_semaphore_queue_item
    WHERE
        tenant_id = @tenantId::uuid
        AND worker_id = ANY(@workerIds::uuid[])
    GROUP BY
        worker_id
)
-- subtract the filled slots from the max runs to get the available slots
SELECT
    wmr."id",
    wmr."maxRuns" - COALESCE(wfs."filledSlots", 0) AS "availableSlots"
FROM
    worker_max_runs wmr
LEFT JOIN
    worker_filled_slots wfs ON wmr."id" = wfs.worker_id;

-- name: ListQueues :many
SELECT
    *
FROM
    v2_queue
WHERE
    tenant_id = @tenantId::uuid
    AND last_active > NOW() - INTERVAL '1 day';

-- name: ListQueueItemsForQueue :many
SELECT
    *
FROM
    v2_queue_item qi
WHERE
    qi.is_queued = true
    AND qi.tenant_id = @tenantId::uuid
    AND qi.queue = @queue::text
    AND (
        sqlc.narg('gtId')::bigint IS NULL OR
        qi.id >= sqlc.narg('gtId')::bigint
    )
    -- Added to ensure that the index is used
    AND qi.priority >= 1 AND qi.priority <= 4
ORDER BY
    qi.priority DESC,
    qi.id ASC
LIMIT
    COALESCE(sqlc.narg('limit')::integer, 100);

-- name: GetMinUnprocessedQueueItemId :one
WITH priority_1 AS (
    SELECT
        id
    FROM
        v2_queue_item
    WHERE
        is_queued = true
        AND tenant_id = @tenantId::uuid
        AND queue = @queue::text
        AND priority = 1
    ORDER BY
        id ASC
    LIMIT 1
),
priority_2 AS (
    SELECT
        id
    FROM
        v2_queue_item
    WHERE
        is_queued = true
        AND tenant_id = @tenantId::uuid
        AND queue = @queue::text
        AND priority = 2
    ORDER BY
        id ASC
    LIMIT 1
),
priority_3 AS (
    SELECT
        id
    FROM
        v2_queue_item
    WHERE
        is_queued = true
        AND tenant_id = @tenantId::uuid
        AND queue = @queue::text
        AND priority = 3
    ORDER BY
        id ASC
    LIMIT 1
),
priority_4 AS (
    SELECT
        id
    FROM
        v2_queue_item
    WHERE
        is_queued = true
        AND tenant_id = @tenantId::uuid
        AND queue = @queue::text
        AND priority = 4
    ORDER BY
        id ASC
    LIMIT 1
)
SELECT
    COALESCE(MIN(id), 0)::bigint AS "minId"
FROM (
    SELECT id FROM priority_1
    UNION ALL
    SELECT id FROM priority_2
    UNION ALL
    SELECT id FROM priority_3
    UNION ALL
    SELECT id FROM priority_4
) AS combined_priorities;

-- name: BulkQueueItems :exec
DELETE FROM
    v2_queue_item
WHERE
    id = ANY(@ids::bigint[]);

-- name: UpdateTasksToAssigned :many
WITH input AS (
    SELECT
        id,
        worker_id
    FROM
        (
            SELECT
                unnest(@taskIds::bigint[]) AS id,
                unnest(@workerIds::uuid[]) AS worker_id
        ) AS subquery
    ORDER BY id
), updated_tasks AS (
    SELECT
        t.id,
        t.retry_count,
        input.worker_id,
        t.tenant_id,
        CURRENT_TIMESTAMP + convert_duration_to_interval(t.step_timeout) AS timeout_at
    FROM
        input
    JOIN
        v2_task t ON t.id = input.id
    ORDER BY t.id
), assigned_tasks AS (
    INSERT INTO v2_semaphore_queue_item (
        task_id,
        retry_count,
        worker_id,
        tenant_id
    )
    SELECT
        t.id,
        t.retry_count,
        t.worker_id,
        @tenantId::uuid
    FROM
        updated_tasks t
    ON CONFLICT (task_id) DO NOTHING
    -- only return the task ids that were successfully assigned
    RETURNING task_id, worker_id
), timeout_insert AS (
    -- bulk insert into timeout queue items
    INSERT INTO
        v2_timeout_queue_item (
            task_id,
            retry_count,
            timeout_at,
            tenant_id,
            is_queued
        )
    SELECT
        t.id,
        t.retry_count,
        t.timeout_at,
        t.tenant_id,
        true
    FROM
        updated_tasks t
    JOIN
        assigned_tasks asr ON t.id = asr.task_id
    ON CONFLICT (task_id, retry_count) DO UPDATE
    SET
        timeout_at = EXCLUDED.timeout_at
    RETURNING
        task_id
)
SELECT
    asr.task_id,
    asr.worker_id
FROM
    assigned_tasks asr;

-- name: GetDesiredLabels :many
SELECT
    "key",
    "strValue",
    "intValue",
    "required",
    "weight",
    "comparator",
    "stepId"
FROM
    "StepDesiredWorkerLabel"
WHERE
    "stepId" = ANY(@stepIds::uuid[]);
